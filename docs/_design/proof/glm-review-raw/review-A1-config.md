## Findings

### HIGH — Credentials in URL userinfo bypass the env-ref requirement
**File:** `mcp/config.go`, `validateURL`, lines 445–467.

`validateURL` parses the URL and checks only scheme and host. It never inspects `parsed.User`. A server spec such as `"url": "https://alice:s3cr3t@api.example.com"` passes validation and is written to `mcp.json` verbatim by `Save`. The password is therefore stored as a plaintext literal on disk, defeating the rule that credentials may appear only as `${env:NAME}` references. The same applies to credentials embedded in the query string (e.g. `?api_key=…`), which is a common convention for the very header names (`x-api-key`, `api-key`) the code tries to protect. Scenario: a user pastes a config from another MCP client that puts the API token in the URL; Ollama accepts it and persists the secret as a literal in `~/.ollama/mcp.json`.

### MEDIUM — Credentials in unknown/extra server fields are not validated
**File:** `mcp/config.go`, `UnmarshalJSON` (lines 111–133) and `validateServer` (lines 382–418).

Unknown fields are deliberately preserved in `extra` and round-tripped to disk, but `validateServer` only runs the credential check (`secretishEnvKey`) against `spec.Env` keys and the sensitive-header check against `spec.Headers`. A field stored directly on the server object such as `"token"`, `"apiKey"`, or `"password"` is kept in `extra`, never matched against `secretishEnvKey`, and saved as a literal. Scenario: a pasted config contains `{"mcpServers":{"s":{"command":"x","apiKey":"sk-..."}}}`; `Problems()` returns empty and `Save` writes the literal key to disk.

### MEDIUM — No env-reference support for `Command`/`Args`, so credentials there must be literals in the process argument list
**File:** `mcp/config.go`, `validateServer` and `resolveValue` usage.

`resolveValue`/`resolveMap` are applied only to `Env` and `Headers`. `Args` are passed through untouched (no `${env:…}` expansion is performed in this package), and `validateServer` never inspects `Args` for secretish content. The threat model explicitly calls out the process argument list as a place credentials must not reach (visible via `ps`). Scenario: a stdio server that needs a token on its command line, e.g. `"args": ["--api-key=sk-..."]`, has no safe form available — the user must place a literal, which is then visible in the child's argv.

### LOW — `isLoopbackHost` treats `localhost` case-sensitively
**File:** `mcp/config.go`, `isLoopbackHost`, lines 469–477.

The only string comparison is `host == "localhost"`. A URL such as `http://LOCALHOST:8080` or `http://Localhost` fails the string compare and `net.ParseIP` returns nil, so `isLoopbackHost` returns false and `validateURL` rejects it with "must use https". Scenario: a user points a local server at `http://Localhost` and is incorrectly told to use https for a loopback address.

---

No defect found in: the atomic write sequence in `writeFilePrivate` (temp + sync + rename, with cleanup on every failure path); the `extra`-field round-trip logic in `MarshalJSON`/`UnmarshalJSON` (known fields correctly take precedence over `extra`); the empty/missing-file handling in `Load`; the `mcpServers` extraction in `parseConfig`.