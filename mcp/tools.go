package mcp

import (
	"encoding/json"
	"errors"
	"fmt"
	"maps"
	"slices"
	"strings"

	"github.com/ollama/ollama/api"
)

// Tool is one tool as advertised by an MCP server, in Ollama's own terms. The
// protocol library's types never leave this package (see the plan's §5.1
// containment rule), so this is what every caller sees.
type Tool struct {
	// Server is the configured name of the server that advertises this tool.
	Server string
	// Name is the tool name as the server gave it, before namespacing.
	Name string
	// Title is an optional human-readable label.
	Title string
	// Description is the server's own text. It is untrusted input: it is
	// injected into the model's prompt, so a hostile server can attempt
	// instructions here. Callers render it as data, never as guidance.
	Description string
	// InputSchema is the server's JSON Schema for the tool's arguments,
	// exactly as received.
	InputSchema json.RawMessage
}

// QualifiedName is the name the model sees: "<server>__<tool>".
func (t Tool) QualifiedName() string {
	return QualifyName(t.Server, t.Name)
}

// maxDescriptionRunes caps how much server-supplied prose reaches the model's
// context. A server that wants more space than this is padding the prompt.
const maxDescriptionRunes = 4000

// maxRefDepth bounds local $ref resolution.
const maxRefDepth = 8

// Schema converts the tool into an Ollama tool definition.
//
// api.ToolFunctionParameters is a subset of JSON Schema: it can carry type,
// description, enum, items, nested properties, required and anyOf, and nothing
// else. Constraints it cannot carry are not discarded silently — they are
// summarised into the description so the model still sees them, because a model
// that cannot see "minimum: 1" will send zero and the call will fail at the
// server for reasons the model cannot diagnose.
//
// A schema that cannot be represented at all returns an error, and the caller
// must skip the tool rather than offer a broken one.
func (t Tool) Schema() (api.ToolFunction, error) {
	fn := api.ToolFunction{
		Name:        t.QualifiedName(),
		Description: sanitiseText(t.Description, maxDescriptionRunes),
	}

	root, err := parseSchemaObject(t.InputSchema)
	if err != nil {
		return api.ToolFunction{}, fmt.Errorf("tool %q: %w", t.QualifiedName(), err)
	}

	var lost lostConstraints
	params, err := convertRoot(root, &lost)
	if err != nil {
		return api.ToolFunction{}, fmt.Errorf("tool %q: %w", t.QualifiedName(), err)
	}
	fn.Parameters = params

	if note := lost.note(); note != "" {
		if fn.Description == "" {
			fn.Description = note
		} else {
			fn.Description = fn.Description + "\n\n" + note
		}
	}
	return fn, nil
}

// jsonSchema is the subset of JSON Schema this package reads. Anything not
// named here is either recorded as a lost constraint or ignored as metadata.
type jsonSchema struct {
	Ref         string                 `json:"$ref"`
	Defs        map[string]*jsonSchema `json:"$defs"`
	Definitions map[string]*jsonSchema `json:"definitions"`

	Type        json.RawMessage        `json:"type"`
	Description string                 `json:"description"`
	Enum        []any                  `json:"enum"`
	Properties  map[string]*jsonSchema `json:"properties"`
	Required    []string               `json:"required"`
	Items       *jsonSchema            `json:"items"`
	AnyOf       []*jsonSchema          `json:"anyOf"`

	// Recorded as lost constraints rather than represented.
	OneOf                []*jsonSchema `json:"oneOf"`
	AllOf                []*jsonSchema `json:"allOf"`
	Not                  *jsonSchema   `json:"not"`
	Const                any           `json:"const"`
	Default              any           `json:"default"`
	Format               string        `json:"format"`
	Pattern              string        `json:"pattern"`
	Minimum              *float64      `json:"minimum"`
	Maximum              *float64      `json:"maximum"`
	ExclusiveMinimum     *float64      `json:"exclusiveMinimum"`
	ExclusiveMaximum     *float64      `json:"exclusiveMaximum"`
	MultipleOf           *float64      `json:"multipleOf"`
	MinLength            *int          `json:"minLength"`
	MaxLength            *int          `json:"maxLength"`
	MinItems             *int          `json:"minItems"`
	MaxItems             *int          `json:"maxItems"`
	UniqueItems          *bool         `json:"uniqueItems"`
	AdditionalProperties json.RawMessage
}

func parseSchemaObject(raw json.RawMessage) (*jsonSchema, error) {
	trimmed := strings.TrimSpace(string(raw))
	if trimmed == "" || trimmed == "null" {
		// A tool with no declared arguments is ordinary, not an error.
		return &jsonSchema{}, nil
	}

	var schema jsonSchema
	if err := json.Unmarshal(raw, &schema); err != nil {
		return nil, fmt.Errorf("input schema is not valid JSON Schema: %w", err)
	}

	// additionalProperties is bool-or-schema, so it is read separately to keep
	// the strict decode above from failing on the bool form.
	var probe map[string]json.RawMessage
	if err := json.Unmarshal(raw, &probe); err != nil {
		return nil, fmt.Errorf("input schema is not a JSON object: %w", err)
	}
	schema.AdditionalProperties = probe["additionalProperties"]
	return &schema, nil
}

// convertRoot maps the tool's top-level schema, which must describe an object.
func convertRoot(root *jsonSchema, lost *lostConstraints) (api.ToolFunctionParameters, error) {
	resolver := &refResolver{root: root}
	resolved, chain, err := resolver.resolve(root, 0, nil)
	if err != nil {
		return api.ToolFunctionParameters{}, err
	}

	types := schemaTypes(resolved.Type)
	if len(types) > 0 && !slices.Contains(types, "object") {
		return api.ToolFunctionParameters{}, fmt.Errorf(`input schema has type %q; only an object of arguments can be represented`, strings.Join(types, "|"))
	}
	if len(resolved.OneOf) > 0 || len(resolved.AllOf) > 0 || len(resolved.AnyOf) > 0 {
		return api.ToolFunctionParameters{}, errors.New("input schema uses oneOf, allOf or anyOf at the top level, which cannot be represented as a flat argument list")
	}

	params := api.ToolFunctionParameters{
		Type:       "object",
		Required:   resolved.Required,
		Properties: api.NewToolPropertiesMap(),
	}

	recordLost(resolved, "", lost)

	for _, name := range slices.Sorted(maps.Keys(resolved.Properties)) {
		property, err := convertProperty(resolver, resolved.Properties[name], name, 0, lost, chain)
		if err != nil {
			return api.ToolFunctionParameters{}, err
		}
		params.Properties.Set(name, property)
	}
	return params, nil
}

func convertProperty(resolver *refResolver, schema *jsonSchema, path string, depth int, lost *lostConstraints, ancestors []string) (api.ToolProperty, error) {
	if schema == nil {
		return api.ToolProperty{}, nil
	}
	if depth > maxRefDepth {
		lost.add(path, "nested beyond the depth this tool schema can express")
		// Sanitised like every other description that reaches the model. This
		// branch returned the server's raw text, which is the one place a
		// hostile or careless server could put control characters and
		// unbounded length into a prompt.
		return api.ToolProperty{Description: sanitiseText(schema.Description, maxDescriptionRunes)}, nil
	}

	resolved, chain, err := resolver.resolve(schema, depth, ancestors)
	if err != nil {
		// An unresolvable reference degrades the property to untyped rather
		// than losing the whole tool.
		lost.add(path, err.Error())
		return api.ToolProperty{Description: sanitiseText(schema.Description, maxDescriptionRunes)}, nil
	}

	property := api.ToolProperty{
		Type:        schemaTypes(resolved.Type),
		Description: sanitiseText(resolved.Description, maxDescriptionRunes),
		Enum:        resolved.Enum,
		Required:    resolved.Required,
	}
	if schemaTypeUnreadable(resolved.Type) {
		lost.add(path, "the server declared a type this tool definition could not read, so no type is stated here")
	}

	recordLost(resolved, path, lost)

	for _, alternative := range resolved.AnyOf {
		converted, err := convertProperty(resolver, alternative, path+".anyOf", depth+1, lost, chain)
		if err != nil {
			return api.ToolProperty{}, err
		}
		property.AnyOf = append(property.AnyOf, converted)
	}

	if resolved.Items != nil {
		items, err := convertProperty(resolver, resolved.Items, path+"[]", depth+1, lost, chain)
		if err != nil {
			return api.ToolProperty{}, err
		}
		property.Items = items
	}

	if len(resolved.Properties) > 0 {
		nested := api.NewToolPropertiesMap()
		for _, name := range slices.Sorted(maps.Keys(resolved.Properties)) {
			child, err := convertProperty(resolver, resolved.Properties[name], path+"."+name, depth+1, lost, chain)
			if err != nil {
				return api.ToolProperty{}, err
			}
			nested.Set(name, child)
		}
		property.Properties = nested
	}

	return property, nil
}

// refResolver resolves local "#/$defs/Name" and "#/definitions/Name" pointers
// against the tool's own schema. Remote references are not followed: a tool
// definition must not cause a network fetch.
type refResolver struct {
	root *jsonSchema
}

// resolve follows a property's references and returns both the schema it lands
// on and the chain of references followed to get there, extending the chain it
// was given.
//
// The chain is the path from the root to this property, not a list of every
// reference the tool has ever used. It was once a field on the resolver, which
// meant it accumulated across siblings: the second property to reference a
// shared definition found it already there and was reported as a cycle, when
// sharing one definition between two fields is ordinary schema practice. What
// that property got was no type, no enum, and a note telling the model its
// reference was cyclic — a false statement about the server's own schema.
//
// Threading it down the nesting path instead keeps the real diagnosis: a
// definition that contains a property referring back to itself closes a cycle
// along one path, and is still caught and named as cyclic rather than degrading
// into an unhelpful note about depth.
func (r *refResolver) resolve(schema *jsonSchema, depth int, ancestors []string) (*jsonSchema, []string, error) {
	current := schema
	chain := ancestors

	for hops := 0; current != nil && current.Ref != ""; hops++ {
		if hops > maxRefDepth || depth > maxRefDepth {
			return nil, chain, fmt.Errorf("reference %q nests too deeply to resolve", schema.Ref)
		}
		if slices.Contains(chain, current.Ref) {
			return nil, chain, fmt.Errorf("reference %q is cyclic", current.Ref)
		}
		// Appended to a copy. The correctness of sibling sharing comes from
		// threading the path, not from this copy — and an attempt to falsify
		// the copy passed, because nothing here retains a chain long enough
		// for two siblings writing into one backing array to be observed. It
		// is kept as a guard against that changing, and recorded as
		// unfalsified rather than counted as a protection.
		chain = append(slices.Clone(chain), current.Ref)

		target, err := r.lookup(current.Ref)
		if err != nil {
			return nil, chain, err
		}
		current = target
	}
	if current == nil {
		return &jsonSchema{}, chain, nil
	}
	return current, chain, nil
}

func (r *refResolver) lookup(ref string) (*jsonSchema, error) {
	name, ok := strings.CutPrefix(ref, "#/$defs/")
	if ok {
		if target := r.root.Defs[name]; target != nil {
			return target, nil
		}
		return nil, fmt.Errorf("reference %q has no matching definition", ref)
	}
	name, ok = strings.CutPrefix(ref, "#/definitions/")
	if ok {
		if target := r.root.Definitions[name]; target != nil {
			return target, nil
		}
		return nil, fmt.Errorf("reference %q has no matching definition", ref)
	}
	return nil, fmt.Errorf("reference %q is not a local definition and is not followed", ref)
}

// schemaTypes reads the "type" keyword, which is a string or an array of them.
// schemaTypeUnreadable reports a "type" keyword that is present but is neither
// a string nor an array of strings. It is distinct from an absent type: the
// server said something about this property and Ollama could not read it, and
// treating that as "no constraint" tells the model the opposite of the truth.
func schemaTypeUnreadable(raw json.RawMessage) bool {
	if len(raw) == 0 {
		return false
	}
	var single string
	if json.Unmarshal(raw, &single) == nil {
		return false
	}
	var many []string
	return json.Unmarshal(raw, &many) != nil
}

func schemaTypes(raw json.RawMessage) api.PropertyType {
	if len(raw) == 0 {
		return nil
	}
	var single string
	if err := json.Unmarshal(raw, &single); err == nil {
		if single == "" {
			return nil
		}
		return api.PropertyType{single}
	}
	var many []string
	if err := json.Unmarshal(raw, &many); err == nil {
		return api.PropertyType(many)
	}
	return nil
}

// lostConstraints accumulates schema keywords that Ollama's tool type cannot
// carry, so they can be told to the model in prose instead of vanishing.
type lostConstraints struct {
	entries []string
}

func (l *lostConstraints) add(path, detail string) {
	where := path
	if where == "" {
		where = "(arguments)"
	}
	l.entries = append(l.entries, fmt.Sprintf("%s: %s", where, detail))
}

func (l *lostConstraints) note() string {
	if len(l.entries) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("Argument rules this tool definition cannot express, still enforced by the server:")
	for _, entry := range l.entries {
		b.WriteString("\n- ")
		b.WriteString(entry)
	}
	return b.String()
}

func recordLost(schema *jsonSchema, path string, lost *lostConstraints) {
	var details []string

	if schema.Const != nil {
		details = append(details, fmt.Sprintf("must equal %v", schema.Const))
	}
	if schema.Default != nil {
		details = append(details, fmt.Sprintf("defaults to %v", schema.Default))
	}
	if schema.Format != "" {
		details = append(details, fmt.Sprintf("format %s", schema.Format))
	}
	if schema.Pattern != "" {
		details = append(details, fmt.Sprintf("must match %s", schema.Pattern))
	}
	if schema.Minimum != nil {
		details = append(details, fmt.Sprintf("minimum %v", *schema.Minimum))
	}
	if schema.Maximum != nil {
		details = append(details, fmt.Sprintf("maximum %v", *schema.Maximum))
	}
	if schema.ExclusiveMinimum != nil {
		details = append(details, fmt.Sprintf("greater than %v", *schema.ExclusiveMinimum))
	}
	if schema.ExclusiveMaximum != nil {
		details = append(details, fmt.Sprintf("less than %v", *schema.ExclusiveMaximum))
	}
	if schema.MultipleOf != nil {
		details = append(details, fmt.Sprintf("multiple of %v", *schema.MultipleOf))
	}
	if schema.MinLength != nil {
		details = append(details, fmt.Sprintf("at least %d characters", *schema.MinLength))
	}
	if schema.MaxLength != nil {
		details = append(details, fmt.Sprintf("at most %d characters", *schema.MaxLength))
	}
	if schema.MinItems != nil {
		details = append(details, fmt.Sprintf("at least %d items", *schema.MinItems))
	}
	if schema.MaxItems != nil {
		details = append(details, fmt.Sprintf("at most %d items", *schema.MaxItems))
	}
	if schema.UniqueItems != nil && *schema.UniqueItems {
		details = append(details, "items must be unique")
	}
	if len(schema.OneOf) > 0 && path != "" {
		details = append(details, "exactly one of several shapes")
	}
	if len(schema.AllOf) > 0 && path != "" {
		details = append(details, "must satisfy several schemas at once")
	}
	if schema.Not != nil {
		details = append(details, "constrained by a negated schema")
	}
	// additionalProperties is a boolean or a schema. Only "false" was reported
	// before, so a server that said "anything extra must look like this" had
	// its constraint dropped in silence — and silence here reads as "anything
	// extra is fine", which is the opposite of what was said.
	switch raw := strings.TrimSpace(string(schema.AdditionalProperties)); raw {
	case "", "true":
	case "false":
		details = append(details, "no properties beyond those listed")
	default:
		details = append(details, "extra properties are allowed but must match a shape this tool definition cannot express")
	}

	if len(details) > 0 {
		lost.add(path, strings.Join(details, ", "))
	}
}

// sanitiseText prepares server-supplied prose for the model's context. Control
// characters are stripped so a description cannot forge message structure, and
// the text is truncated so a server cannot consume the context window.
func sanitiseText(text string, maxRunes int) string {
	// Cut before doing any work. Everything below allocates in proportion to
	// what it is given, and what it is given arrives from an MCP server with no
	// length agreed anywhere. A generous multiple of the cap is kept so the
	// stripping below still has room to remove characters without the result
	// coming up short.
	if trimmed, cut := truncateRunes(text, maxRunes*4); cut {
		text = trimmed
	}

	var b strings.Builder
	b.Grow(len(text))
	for _, r := range text {
		switch {
		case r == '\n' || r == '\t':
			b.WriteRune(r)
		case r == '\r':
			// Normalised away; a bare carriage return only serves to hide text.
		case r < 0x20 || r == 0x7f:
			// Other control characters are dropped.
		default:
			b.WriteRune(r)
		}
	}

	out := strings.TrimSpace(b.String())
	runes := []rune(out)
	if len(runes) > maxRunes {
		out = string(runes[:maxRunes]) + "… (truncated by Ollama)"
	}
	return out
}

// truncateRunes cuts a string to at most limit runes without walking past that
// point. It exists so a hostile description does not get copied twice at full
// size before being thrown away — a hundred-megabyte description would
// otherwise allocate the builder and then a rune slice of the whole thing to
// keep four thousand characters.
func truncateRunes(text string, limit int) (string, bool) {
	count := 0
	for index := range text {
		if count == limit {
			return text[:index], true
		}
		count++
	}
	return text, false
}

// maps_Keys is a tiny local helper so this file does not depend on the ordering
// behaviour of a shared utility; sorted iteration keeps tool schemas stable
// between runs, which matters because they are hashed for change detection.
func maps_Keys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	return keys
}
