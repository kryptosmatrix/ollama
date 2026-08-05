package mcp

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
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
