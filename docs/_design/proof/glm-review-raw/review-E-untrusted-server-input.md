## Findings

### 1. HIGH — `refResolver.visited` is never cleared, so legitimate shared `$ref` is reported as cyclic

**File:** `mcp/tools.go`, lines 238–250

The `visited` slice on `refResolver` is accumulated across every call to `resolve` for the lifetime of the resolver (created once at line 144 in `convertRoot`). It is never reset between properties. When two sibling properties both reference the same definition — a common pattern, e.g. `from` and `to` both pointing at `#/$defs/currency` — the first property's resolution adds the ref to `visited`, and the second property's resolution finds it there and returns "reference is cyclic." The error propagates through `convertProperty` → `convertRoot` → `Schema`, causing the entire tool to be rejected.

**Scenario:** A server advertises a tool with schema:
```json
{
  "type": "object",
  "properties": {
    "from": {"$ref": "#/$defs/currency"},
    "to": {"$ref": "#/$defs/currency"}
  },
  "$defs": {"currency": {"type": "string", "enum": ["USD","EUR"]}}
}
```
The tool is silently dropped because the second `$ref` is falsely flagged as cyclic.

---

### 2. HIGH — Argument injection via unvalidated package identifiers

**File:** `mcp/registry.go`, lines 237–254

`resolvePackage` inserts the registry-supplied `pkg.Identifier` directly into the argument list without validating that it does not start with `-` and without a `--` separator. For npm, the result is `npx -y <identifier> <args...>`. A malicious registry entry can set `identifier` to a value that npx/npm interprets as a flag.

**Scenario:** A registry entry with `registryType: "npm"`, `identifier: "--call=rm -rf /tmp"` produces the command line `npx -y --call=rm -rf /tmp`, which npx interprets as "run the command `rm -rf /tmp`" rather than installing a package of that name. The same applies to `uvx <identifier>` (pypi) and `docker run --rm -i <identifier>` (oci). The user is asked to approve the command line, but a non-technical user may not recognise flag injection.

---

### 3. MEDIUM — `additionalProperties` as a schema is silently dropped without being recorded as a lost constraint

**File:** `mcp/tools.go`, line 382

```go
if len(schema.AdditionalProperties) > 0 && string(schema.AdditionalProperties) == "false" {
```

This only records the constraint when `additionalProperties` is exactly the literal `false`. When `additionalProperties` is a schema (e.g. `{"type": "string"}`), the condition is false and the constraint is neither represented in the output schema nor recorded in the lost-constraints note. The code's own design principle (lines 49–52) says constraints that cannot be carried must be summarised into the description so the model sees them. This one vanishes silently.

**Scenario:** A server declares `additionalProperties: {"type": "string"}` to allow arbitrary extra string properties. The model sees no mention of this constraint, sends extra properties of the wrong type, and the call fails at the server with no diagnostic the model can reason about.

---

### 4. MEDIUM — Description is not sanitised when nesting depth is exceeded

**File:** `mcp/tools.go`, line 182

```go
if depth > maxRefDepth {
    lost.add(path, "nested beyond the depth this tool schema can express")
    return api.ToolProperty{Description: schema.Description}, nil
}
```

The raw `schema.Description` is returned without calling `sanitiseText`. Compare with line 190 (the unresolvable-ref branch), which correctly calls `sanitiseText(schema.Description, maxDescriptionRunes)`. A deeply nested property's description bypasses control-character stripping and length truncation.

**Scenario:** A server nests a property beyond depth 8 and sets that property's `description` to a string containing `\r` characters or 100 KB of padding. The control characters or unbounded text reach the model's context.

---

### 5. MEDIUM — `envReferenceName` collisions cause silent loss of header/variable references

**File:** `mcp/registry.go`, lines 304–319

`envReferenceName` replaces every non-alphanumeric character with `_` and upper-cases. Two distinct header names can map to the same environment variable name. Because `declaredValues` (line 288–299) uses the original name as the map key but `envReferenceName(name)` as the value, two headers that collide after sanitisation produce two entries in the map with different keys but the same `${env:...}` value — which is not itself a collision. However, if two headers have the *same* original name (after trimming), the second silently overwrites the first in the map.

More importantly, if two different headers map to the same env reference name, the user is told to set one environment variable but it is used for two different headers, and the second header's value is never independently configurable.

**Scenario:** A remote declares headers `X-Custom-Auth` and `X_Custom_Auth`. Both map to `${env:X_CUSTOM_AUTH}`. The user sets one env var; both headers receive the same value. If they were meant to carry different credentials, one is silently wrong.

---

### 6. MEDIUM — `sanitiseText` processes the entire input before truncating, allowing memory amplification

**File:** `mcp/tools.go`, lines 394–416

`sanitiseText` iterates over every rune in the input (line 397), builds the full sanitised string in a `strings.Builder` (pre-grown to `len(text)` at line 396), and only then truncates to `maxRunes` at line 411–413 via `[]rune(out)`, which allocates a second copy of the full string. There is no bound on the input size before this function is called — tool descriptions arrive from the MCP server with no length cap.

**Scenario:** A hostile server sends a 100 MB tool description. `sanitiseText` allocates ~100 MB for the builder and ~300 MB for the `[]rune` slice before discarding all but 4000 runes. This is a memory-exhaustion vector.

---

### 7. LOW — `schemaTypes` silently treats a malformed `type` as "no type"

**File:** `mcp/tools.go`, lines 283–299

If the `type` keyword is neither a string nor an array of strings (e.g. `{"type": 123}` or `{"type": {"foo": "bar"}}`), `schemaTypes` returns `nil`, which is treated identically to an absent `type`. The tool is offered with no type constraint rather than being rejected for an invalid schema.

**Scenario:** A server sends `{"type": "objekt"}` (typo). `schemaTypes` returns `["objekt"]`, which is passed through as the property type. The model may produce arguments of the wrong type. (This is a different issue — a valid string but unknown type is passed through. The nil-return issue applies to non-string, non-array types like `{"type": 123}`.)

---

### No defect found

- **Name separator collision / impersonation:** The `NameSeparator = "__"` scheme combined with `SplitQualifiedName` splitting on the first occurrence is sound as long as server names cannot contain `__` (enforced in config.go, not shown). The `reservedToolNames` check on the qualified name (line 65) is currently dead code since no reserved name contains `__`, but the comment documents this as intentional future-proofing, not a bug.

- **Secret embedding in registry resolution:** `declaredValues` correctly emits `${env:NAME}` references rather than literals, so no secret is ever written into the configuration.