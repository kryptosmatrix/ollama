package mcp

import (
	"encoding/json"
	"runtime"
	"slices"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"

	"github.com/ollama/ollama/api"
)

// schemaJSON renders a converted tool as the JSON the model would be shown,
// which is the form worth asserting on: it is what actually reaches the wire.
func schemaJSON(t *testing.T, tool Tool) string {
	t.Helper()
	fn, err := tool.Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	data, err := json.Marshal(fn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(data)
}

func tool(name string, schema string) Tool {
	return Tool{Server: "files", Name: name, Description: "does a thing", InputSchema: json.RawMessage(schema)}
}

func TestQualifyAndSplit(t *testing.T) {
	t.Run("round trip", func(t *testing.T) {
		qualified := QualifyName("files", "read_file")
		if qualified != "files__read_file" {
			t.Fatalf("QualifyName = %q", qualified)
		}
		server, name, ok := SplitQualifiedName(qualified)
		if !ok || server != "files" || name != "read_file" {
			t.Errorf("SplitQualifiedName(%q) = %q, %q, %v", qualified, server, name, ok)
		}
	})

	t.Run("a tool name containing the separator splits at the first one", func(t *testing.T) {
		server, name, ok := SplitQualifiedName("files__odd__name")
		if !ok {
			t.Fatal("expected a split")
		}
		if server != "files" || name != "odd__name" {
			t.Errorf("got server %q tool %q, want files / odd__name", server, name)
		}
	})

	t.Run("a first-party name has no separator and does not split", func(t *testing.T) {
		if _, _, ok := SplitQualifiedName("web_search"); ok {
			t.Error("web_search should not look like a namespaced MCP tool")
		}
	})

	t.Run("empty halves do not count as a split", func(t *testing.T) {
		for _, input := range []string{"__tool", "server__", "__"} {
			if _, _, ok := SplitQualifiedName(input); ok {
				t.Errorf("SplitQualifiedName(%q) should not report a valid split", input)
			}
		}
	})
}

func TestValidateToolName(t *testing.T) {
	cases := []struct {
		name    string
		tool    string
		wantErr string
	}{
		{name: "ordinary", tool: "read_file"},
		{name: "dotted", tool: "repo.search"},
		{name: "dotted and reserved by a first-party tool", tool: "browser.search", wantErr: "reserved"},
		{name: "hyphenated", tool: "read-file"},
		{name: "reserved first-party name", tool: "bash", wantErr: "reserved"},
		{name: "reserved regardless of case", tool: "Web_Search", wantErr: "reserved"},
		{name: "empty", tool: "", wantErr: "must be 1-128"},
		{name: "with a space", tool: "read file", wantErr: "must be 1-128"},
		{name: "with a slash", tool: "read/file", wantErr: "must be 1-128"},
		{name: "leading hyphen", tool: "-read", wantErr: "must be 1-128"},
		{name: "too long", tool: strings.Repeat("a", 129), wantErr: "must be 1-128"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			err := validateToolName("files", tc.tool)
			if tc.wantErr == "" {
				if err != nil {
					t.Fatalf("validateToolName(%q) = %v, want nil", tc.tool, err)
				}
				return
			}
			if err == nil {
				t.Fatalf("validateToolName(%q) = nil, want an error containing %q", tc.tool, tc.wantErr)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("validateToolName(%q) = %v, want it to contain %q", tc.tool, err, tc.wantErr)
			}
		})
	}
}

func TestSchemaSimpleObject(t *testing.T) {
	got := schemaJSON(t, tool("read_file", `{
	  "type": "object",
	  "properties": {
	    "path":     {"type": "string", "description": "file to read"},
	    "encoding": {"type": "string", "enum": ["utf8", "ascii"]},
	    "lines":    {"type": "integer"}
	  },
	  "required": ["path"]
	}`))

	want := `{"name":"files__read_file","description":"does a thing","parameters":{"type":"object","required":["path"],"properties":{"encoding":{"type":"string","enum":["utf8","ascii"]},"lines":{"type":"integer"},"path":{"type":"string","description":"file to read"}}}}`
	if diff := cmp.Diff(want, got); diff != "" {
		t.Errorf("schema mismatch (-want +got):\n%s", diff)
	}
}

func TestSchemaNoArguments(t *testing.T) {
	for _, raw := range []string{``, `null`, `{}`, `{"type":"object"}`, `{"type":"object","properties":{}}`} {
		t.Run(raw, func(t *testing.T) {
			fn, err := tool("ping", raw).Schema()
			if err != nil {
				t.Fatalf("a tool with no arguments should convert, got %v", err)
			}
			if fn.Parameters.Type != "object" {
				t.Errorf("parameters type = %q, want object", fn.Parameters.Type)
			}
			if fn.Parameters.Properties.Len() != 0 {
				t.Errorf("expected no properties, got %d", fn.Parameters.Properties.Len())
			}
		})
	}
}

func TestSchemaNestedObjectsAndArrays(t *testing.T) {
	got := schemaJSON(t, tool("write", `{
	  "type": "object",
	  "properties": {
	    "files": {
	      "type": "array",
	      "items": {
	        "type": "object",
	        "properties": {
	          "path":    {"type": "string"},
	          "content": {"type": "string"}
	        },
	        "required": ["path"]
	      }
	    }
	  }
	}`))

	if !strings.Contains(got, `"items"`) {
		t.Errorf("array items were dropped:\n%s", got)
	}
	if !strings.Contains(got, `"content"`) || !strings.Contains(got, `"path"`) {
		t.Errorf("nested object properties were dropped:\n%s", got)
	}
	if !strings.Contains(got, `"required":["path"]`) {
		t.Errorf("nested required list was dropped:\n%s", got)
	}
}

func TestSchemaResolvesLocalRefs(t *testing.T) {
	// This is the shape emitted by pydantic and zod, which most real MCP
	// servers are built on. Without resolution the tool is unusable.
	got := schemaJSON(t, tool("create_issue", `{
	  "type": "object",
	  "$defs": {
	    "Priority": {"type": "string", "enum": ["low", "high"], "description": "how urgent"}
	  },
	  "properties": {
	    "title":    {"type": "string"},
	    "priority": {"$ref": "#/$defs/Priority"}
	  }
	}`))

	if !strings.Contains(got, `"enum":["low","high"]`) {
		t.Errorf("$ref to $defs was not resolved:\n%s", got)
	}
	if !strings.Contains(got, "how urgent") {
		t.Errorf("resolved definition lost its description:\n%s", got)
	}
}

func TestSchemaResolvesLegacyDefinitionsKeyword(t *testing.T) {
	got := schemaJSON(t, tool("legacy", `{
	  "type": "object",
	  "definitions": {"Colour": {"type": "string", "enum": ["red"]}},
	  "properties": {"colour": {"$ref": "#/definitions/Colour"}}
	}`))

	if !strings.Contains(got, `"enum":["red"]`) {
		t.Errorf("$ref to definitions was not resolved:\n%s", got)
	}
}

func TestSchemaCyclicRefDegradesInsteadOfHanging(t *testing.T) {
	// A self-referential definition must not loop forever and must not lose
	// the whole tool; it degrades to an untyped property with a stated reason.
	fn, err := tool("tree", `{
	  "type": "object",
	  "$defs": {"Node": {"type": "object", "properties": {"child": {"$ref": "#/$defs/Node"}}}},
	  "properties": {"root": {"$ref": "#/$defs/Node"}}
	}`).Schema()
	if err != nil {
		t.Fatalf("a cyclic schema should degrade, not fail: %v", err)
	}
	if !strings.Contains(fn.Description, "cyclic") {
		t.Errorf("the model should be told the schema was cyclic, got description:\n%s", fn.Description)
	}
}

func TestSchemaRemoteRefIsNotFollowed(t *testing.T) {
	fn, err := tool("remote", `{
	  "type": "object",
	  "properties": {"thing": {"$ref": "https://example.com/schema.json#/Thing"}}
	}`).Schema()
	if err != nil {
		t.Fatalf("a remote ref should degrade, not fail: %v", err)
	}
	if !strings.Contains(fn.Description, "not followed") {
		t.Errorf("a remote $ref must be refused and reported, got description:\n%s", fn.Description)
	}
}

func TestSchemaReportsConstraintsItCannotCarry(t *testing.T) {
	fn, err := tool("search", `{
	  "type": "object",
	  "properties": {
	    "limit": {"type": "integer", "minimum": 1, "maximum": 100, "default": 10},
	    "query": {"type": "string", "minLength": 3, "pattern": "^[a-z ]+$"},
	    "mode":  {"const": "fast"}
	  },
	  "additionalProperties": false
	}`).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}

	for _, want := range []string{
		"minimum 1", "maximum 100", "defaults to 10",
		"at least 3 characters", "must match ^[a-z ]+$",
		`must equal fast`,
		"no properties beyond those listed",
	} {
		if !strings.Contains(fn.Description, want) {
			t.Errorf("description should carry %q so the model can honour it; got:\n%s", want, fn.Description)
		}
	}

	if !strings.HasPrefix(fn.Description, "does a thing") {
		t.Errorf("the server's own description should come first, got:\n%s", fn.Description)
	}
}

func TestSchemaRejectsWhatItCannotRepresent(t *testing.T) {
	cases := []struct {
		name    string
		schema  string
		wantErr string
	}{
		{
			name:    "a scalar input schema",
			schema:  `{"type": "string"}`,
			wantErr: "only an object of arguments",
		},
		{
			name:    "alternation at the root",
			schema:  `{"oneOf": [{"type": "object"}, {"type": "object"}]}`,
			wantErr: "top level",
		},
		{
			name:    "anyOf at the root",
			schema:  `{"anyOf": [{"type": "object"}]}`,
			wantErr: "top level",
		},
		{
			name:    "input schema that is not JSON",
			schema:  `{"type": `,
			wantErr: "input schema",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := tool("broken", tc.schema).Schema()
			if err == nil {
				t.Fatal("expected an error so the caller skips the tool rather than offering a broken one")
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("error = %v, want it to contain %q", err, tc.wantErr)
			}
			if !strings.Contains(err.Error(), "files__broken") {
				t.Errorf("error should name the tool, got %v", err)
			}
		})
	}
}

func TestSchemaIsDeterministic(t *testing.T) {
	// Tool schemas are hashed to detect a server changing its tools underneath
	// the user, so conversion must not depend on Go's map ordering.
	schema := `{
	  "type": "object",
	  "properties": {
	    "zulu": {"type": "string"}, "alpha": {"type": "string"},
	    "mike": {"type": "string"}, "delta": {"type": "string"},
	    "kilo": {"type": "string"}, "bravo": {"type": "string"}
	  }
	}`
	first := schemaJSON(t, tool("sorted", schema))
	for range 20 {
		if got := schemaJSON(t, tool("sorted", schema)); got != first {
			t.Fatalf("conversion is not deterministic:\n%s\nvs\n%s", first, got)
		}
	}
	if !strings.Contains(first, `"alpha"`) {
		t.Fatalf("unexpected output: %s", first)
	}
	if strings.Index(first, `"alpha"`) > strings.Index(first, `"zulu"`) {
		t.Errorf("properties should be emitted in sorted order:\n%s", first)
	}
}

func TestDescriptionIsTreatedAsUntrustedInput(t *testing.T) {
	t.Run("control characters are stripped", func(t *testing.T) {
		hostile := Tool{
			Server:      "files",
			Name:        "read",
			Description: "harmless\x00\x07\x1b[31m text\rhidden",
			InputSchema: json.RawMessage(`{"type":"object"}`),
		}
		fn, err := hostile.Schema()
		if err != nil {
			t.Fatalf("Schema: %v", err)
		}
		for _, forbidden := range []string{"\x00", "\x07", "\x1b", "\r"} {
			if strings.Contains(fn.Description, forbidden) {
				t.Errorf("control character %q survived sanitisation: %q", forbidden, fn.Description)
			}
		}
		if !strings.Contains(fn.Description, "harmless") {
			t.Errorf("legitimate text was lost: %q", fn.Description)
		}
	})

	t.Run("newlines and tabs survive", func(t *testing.T) {
		got := sanitiseText("one\ntwo\tthree", 100)
		if got != "one\ntwo\tthree" {
			t.Errorf("sanitiseText dropped ordinary whitespace: %q", got)
		}
	})

	t.Run("an over-long description is truncated", func(t *testing.T) {
		long := strings.Repeat("padding ", 5000)
		got := sanitiseText(long, maxDescriptionRunes)
		if len([]rune(got)) <= maxDescriptionRunes {
			t.Fatalf("expected a truncation marker to be appended, got %d runes", len([]rune(got)))
		}
		if !strings.HasSuffix(got, "(truncated by Ollama)") {
			t.Errorf("truncation should be visible to the model, got tail %q", got[len(got)-40:])
		}
		if len([]rune(got)) > maxDescriptionRunes+40 {
			t.Errorf("truncated description is still %d runes", len([]rune(got)))
		}
	})

	t.Run("a description in a nested property is sanitised too", func(t *testing.T) {
		nested := Tool{
			Server:      "files",
			Name:        "nested",
			InputSchema: json.RawMessage("{\"type\":\"object\",\"properties\":{\"a\":{\"type\":\"string\",\"description\":\"deep\\u0000\\u001btext\"}}}"),
		}
		fn, err := nested.Schema()
		if err != nil {
			t.Fatalf("Schema: %v", err)
		}
		data, err := json.Marshal(fn)
		if err != nil {
			t.Fatalf("marshal: %v", err)
		}
		for _, forbidden := range []string{"\\u0000", "\\u001b"} {
			if strings.Contains(string(data), forbidden) {
				t.Errorf("a nested description kept control character %s:\n%s", forbidden, data)
			}
		}
		if !strings.Contains(string(data), "deeptext") {
			t.Errorf("nested description text was lost:\n%s", data)
		}
	})
}

func TestSchemaHandlesUnionTypes(t *testing.T) {
	got := schemaJSON(t, tool("maybe", `{
	  "type": "object",
	  "properties": {"value": {"type": ["string", "null"]}}
	}`))
	if !strings.Contains(got, `["string","null"]`) {
		t.Errorf("a union type should survive conversion:\n%s", got)
	}
}

// TestTwoPropertiesMayShareOneDefinition is a defect a cross-substrate review
// found and a probe confirmed: the resolver's visited list was created once per
// tool and never reset, so the second sibling to reference a definition was
// reported as a cycle.
//
// Sharing a definition between two fields is ordinary schema practice — two
// currency fields, two dates, two identifiers of the same kind. The property
// was not dropped, which would at least have been visible; it was emitted with
// no type and no enum, and the note attached to the description said the
// reference was cyclic, which was not true.
func TestTwoPropertiesMayShareOneDefinition(t *testing.T) {
	tool := Tool{
		Server:      "hosted",
		Name:        "convert",
		Description: "convert an amount",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {
				"from": {"$ref": "#/$defs/currency"},
				"to":   {"$ref": "#/$defs/currency"}
			},
			"$defs": {"currency": {"type": "string", "enum": ["USD", "EUR"]}}
		}`),
	}

	schema, err := tool.Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	for _, name := range []string{"from", "to"} {
		property, ok := schema.Parameters.Properties.Get(name)
		if !ok {
			t.Fatalf("property %q is missing", name)
		}
		if !slices.Contains(property.Type, "string") {
			t.Errorf("property %q has type %v, want string; a shared definition must resolve for every property that uses it", name, property.Type)
		}
		if len(property.Enum) != 2 {
			t.Errorf("property %q has enum %v, want both values", name, property.Enum)
		}
	}
	// And nothing was reported as lost, because nothing was.
	if strings.Contains(schema.Description, "cyclic") {
		t.Errorf("the description claims a cycle that does not exist: %q", schema.Description)
	}
}

// TestAGenuineCycleIsStillCaught is the other half. Resetting the chain per
// resolution must not stop a schema that really does refer to itself from being
// caught — it is bounded by depth, and what the model is told must be true.
func TestAGenuineCycleIsStillCaught(t *testing.T) {
	tool := Tool{
		Server:      "hosted",
		Name:        "walk",
		Description: "walk a tree",
		InputSchema: json.RawMessage(`{
			"type": "object",
			"properties": {"node": {"$ref": "#/$defs/node"}},
			"$defs": {"node": {"$ref": "#/$defs/node"}}
		}`),
	}

	schema, err := tool.Schema()
	if err != nil {
		// Refusing the tool is an acceptable answer too; what must not happen
		// is hanging or a stack overflow.
		return
	}
	if _, ok := schema.Parameters.Properties.Get("node"); !ok {
		t.Error("the property vanished entirely rather than being reported")
	}
	if !strings.Contains(schema.Description, "cyclic") {
		t.Errorf("a genuine cycle was not reported to the model: %q", schema.Description)
	}
}

// controlBytes is a carriage return and a NUL, built rather than written so the
// source file stays free of them.
var controlBytes = string([]byte{13, 0})

// TestADeepPropertyDescriptionIsSanitised. The depth-limit branch returned the
// server's raw description while the branch beside it sanitised. It was the one
// place a careless or hostile server could put control characters and unbounded
// text into a model's prompt.
func TestADeepPropertyDescriptionIsSanitised(t *testing.T) {
	// The description has to sit at every level, not only the innermost: the
	// depth-limit branch returns the schema it was handed when the limit is
	// crossed, and an intermediate object with no description of its own would
	// exercise nothing. An earlier version of this test nested a description
	// twelve deep, never reached the branch, and passed against the defect.
	nasty := "deep " + controlBytes + strings.Repeat("x", 9000)
	inner := map[string]any{"type": "object", "description": nasty}
	for range 12 {
		inner = map[string]any{
			"type":        "object",
			"description": nasty,
			"properties":  map[string]any{"a": inner},
		}
	}
	raw, err := json.Marshal(inner)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	fn, err := (Tool{Server: "hosted", Name: "deep", Description: "d", InputSchema: raw}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	encoded, err := json.Marshal(fn)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.ContainsAny(string(encoded), controlBytes) {
		t.Error("a raw control character from a deep property reached the model")
	}
	for _, escaped := range []string{"\\u0000", "\\r"} {
		if strings.Contains(string(encoded), escaped) {
			t.Errorf("an escaped control character reached the model: %.200s", encoded)
		}
	}
	// Total size is legitimately large here — every level carries a
	// description and each is capped separately. What must not happen is one
	// description arriving uncapped, so the cap is checked per description
	// rather than in aggregate.
	var check func(props *api.ToolPropertiesMap)
	check = func(props *api.ToolPropertiesMap) {
		if props == nil {
			return
		}
		for name, property := range props.All() {
			// The cap plus the visible truncation marker sanitiseText appends.
			const allowance = 32
			if got := len([]rune(property.Description)); got > maxDescriptionRunes+allowance {
				t.Errorf("property %q has a description of %d runes, over the %d cap", name, got, maxDescriptionRunes)
			}
			check(property.Properties)
		}
	}
	check(fn.Parameters.Properties)
}

// TestATypeTheServerDeclaredButOllamaCannotReadIsSaidSoFar. A "type" that is
// neither a string nor an array of strings was treated exactly like an absent
// one, so the model was told nothing at all — the opposite of the truth. The
// server said something; Ollama could not read it, and that is what the model
// now hears.
func TestATypeTheServerDeclaredButOllamaCannotReadIsSaidSoFar(t *testing.T) {
	fn, err := (Tool{
		Server: "hosted", Name: "odd", Description: "d",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"n":{"type":123}}}`),
	}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if !strings.Contains(fn.Description, "could not read") {
		t.Errorf("the model was not told the type was unreadable: %q", fn.Description)
	}

	plain, err := (Tool{
		Server: "hosted", Name: "plain", Description: "d",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"n":{"type":"string"}}}`),
	}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if strings.Contains(plain.Description, "could not read") {
		t.Errorf("a readable type was reported as unreadable: %q", plain.Description)
	}
}

// TestAdditionalPropertiesAsASchemaIsReported. Only the literal false was
// reported before, so a server saying "anything extra must look like this" had
// its constraint dropped — and silence there reads to the model as "anything
// extra is fine", which is the opposite of what was said.
func TestAdditionalPropertiesAsASchemaIsReported(t *testing.T) {
	withSchema, err := (Tool{Server: "hosted", Name: "t", Description: "d",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":{"type":"string"}}`)}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if !strings.Contains(withSchema.Description, "extra properties are allowed") {
		t.Errorf("the constraint was dropped: %q", withSchema.Description)
	}

	withFalse, err := (Tool{Server: "hosted", Name: "t", Description: "d",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":false}`)}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if !strings.Contains(withFalse.Description, "no properties beyond those listed") {
		t.Errorf("the false case stopped being reported: %q", withFalse.Description)
	}

	// And "true" says nothing, because it constrains nothing.
	withTrue, err := (Tool{Server: "hosted", Name: "t", Description: "d",
		InputSchema: json.RawMessage(`{"type":"object","properties":{"a":{"type":"string"}},"additionalProperties":true}`)}).Schema()
	if err != nil {
		t.Fatalf("Schema: %v", err)
	}
	if strings.Contains(withTrue.Description, "extra properties") {
		t.Errorf("a permissive additionalProperties was reported as a constraint: %q", withTrue.Description)
	}
}

// TestAHugeDescriptionIsCutBeforeItIsCopied. sanitiseText walked every rune of
// its input, built the whole sanitised string, then made a second full copy as
// runes to keep the first four thousand. A description arrives from an MCP
// server with no length agreed anywhere, so a hostile one would be copied twice
// at full size before nearly all of it was discarded.
func TestAHugeDescriptionIsCutBeforeItIsCopied(t *testing.T) {
	huge := strings.Repeat("x", 8_000_000)

	var before, after runtime.MemStats
	runtime.GC()
	runtime.ReadMemStats(&before)
	got := sanitiseText(huge, maxDescriptionRunes)
	runtime.ReadMemStats(&after)

	if runes := []rune(got); len(runes) > maxDescriptionRunes+32 {
		t.Errorf("the result is %d runes, over the cap", len(runes))
	}
	if !strings.Contains(got, "truncated by Ollama") {
		t.Error("the result does not say it was truncated")
	}

	// The input is 8MB. Copying it even once would show here; the allowance is
	// generous so this measures the difference between "cut first" and "copy
	// the lot twice", not an exact byte count.
	if allocated := after.TotalAlloc - before.TotalAlloc; allocated > 2_000_000 {
		t.Errorf("sanitising an 8MB description allocated %d bytes; it must cut before it copies", allocated)
	}
}
