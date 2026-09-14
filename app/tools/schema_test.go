//go:build windows || darwin

package tools

import (
	"encoding/json"
	"reflect"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/ollama/ollama/api"
)

// firstPartyTools is every tool the desktop app registers on a user's behalf,
// mirroring app/ui/ui.go: the three browser tools for models that support them,
// and web_search/web_fetch for models that do not. All of them reach the model
// through Registry.OllamaTools.
func firstPartyTools() []Tool {
	browser := NewBrowser(nil)
	return []Tool{
		NewBrowserSearch(browser),
		NewBrowserOpen(browser),
		NewBrowserFind(browser),
		&WebSearch{},
		&WebFetch{},
	}
}

func toolFunction(t *testing.T, definitions api.Tools, name string) api.ToolFunction {
	t.Helper()
	for _, definition := range definitions {
		if definition.Function.Name == name {
			return definition.Function
		}
	}
	t.Fatalf("no definition for %q in %d tools", name, len(definitions))
	return api.ToolFunction{}
}

// TestBrowserOpenKeepsItsAlternativeIDTypes is the discriminating test for this
// change: it is the one assertion here that a tool routed through the derived
// path cannot satisfy.
//
// browser.open's "id" is either a URL to open or the index of a link on the
// page being viewed, and Execute branches on exactly that. Alternatives are
// expressed as anyOf, and anyOf is one of the things the map form cannot carry,
// so rebuilding this tool from a map leaves "id" with no type at all and the
// model guessing which of the two it may send.
//
// The assertions are deliberately not on the required list, the property names
// or the descriptions. The derived path preserves all three, so asserting them
// would prove nothing about which path was taken — the mistake recorded against
// F46 in docs/_design/proof/phase3c-falsification.txt, where a falsification
// run left the test green.
func TestBrowserOpenKeepsItsAlternativeIDTypes(t *testing.T) {
	registry := NewRegistry()
	registry.Register(NewBrowserOpen(NewBrowser(nil)))

	function := toolFunction(t, registry.OllamaTools(), "browser.open")
	rendered := function.Parameters.String()

	id, ok := function.Parameters.Properties.Get("id")
	if !ok {
		t.Fatalf("browser.open has no id property:\n%s", rendered)
	}

	offered := make([]string, 0, len(id.AnyOf))
	for _, alternative := range id.AnyOf {
		offered = append(offered, alternative.Type.String())
	}
	slices.Sort(offered)

	if got := strings.Join(offered, ","); got != "integer,string" {
		t.Errorf("id offers %q; the model must be told it may send either a URL (string) or a link index (integer):\n%s", got, rendered)
	}
}

// TestFirstPartyToolsSupplyTheirOwnDefinition is the activation check. Doing
// this for one tool and not the rest would leave the others quietly rebuilt
// from their map.
func TestFirstPartyToolsSupplyTheirOwnDefinition(t *testing.T) {
	for _, tool := range firstPartyTools() {
		if _, ok := tool.(OllamaTool); !ok {
			t.Errorf("%s does not implement OllamaTool, so its definition is rebuilt from the map form and loses whatever the map cannot express", tool.Name())
		}
	}
}

// TestFirstPartyToolDefinitionsAndSchemaMapsAgree keeps the two forms of a tool
// from drifting apart. The map is derived from the definition, so they can only
// disagree if a tool goes back to writing one of them out by hand — which is
// how web_search came to advertise a default of 3 while asking for 5.
func TestFirstPartyToolDefinitionsAndSchemaMapsAgree(t *testing.T) {
	for _, tool := range firstPartyTools() {
		faithful, ok := tool.(OllamaTool)
		if !ok {
			continue // reported by TestFirstPartyToolsSupplyTheirOwnDefinition
		}
		function := faithful.ToolFunction()

		if function.Name != tool.Name() {
			t.Errorf("%s: definition names it %q", tool.Name(), function.Name)
		}
		if function.Description != tool.Description() {
			t.Errorf("%s: definition describes it as %q but Description says %q", tool.Name(), function.Description, tool.Description())
		}

		encoded, err := json.Marshal(function.Parameters)
		if err != nil {
			t.Fatalf("%s: marshal parameters: %v", tool.Name(), err)
		}
		var want map[string]any
		if err := json.Unmarshal(encoded, &want); err != nil {
			t.Fatalf("%s: decode parameters: %v", tool.Name(), err)
		}
		if got := tool.Schema(); !reflect.DeepEqual(got, want) {
			t.Errorf("%s: Schema map and definition disagree\n got: %v\nwant: %v", tool.Name(), got, want)
		}
	}
}

// TestToolsStateTheirDefaultsToTheModel covers the loss this change was asked
// to fix.
//
// api.ToolFunction has no way to carry a JSON Schema "default", so web_search's
// hand-written `"default": 3` could not have reached the model down either
// path — it was written down and dropped. Prose in the property's description
// is the only form the model can see.
//
// This does not discriminate the faithful path from the derived one, because
// descriptions survive both. What it proves is that the definition carries the
// default at all, which it previously did not, and that the number the model is
// told is the number Execute actually uses.
func TestToolsStateTheirDefaultsToTheModel(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&WebSearch{})
	registry.Register(NewBrowserSearch(NewBrowser(nil)))
	definitions := registry.OllamaTools()

	for _, testCase := range []struct {
		tool     string
		property string
		want     int
	}{
		{"web_search", "max_results", defaultWebSearchResults},
		{"browser.search", "topn", defaultBrowserSearchResults},
	} {
		function := toolFunction(t, definitions, testCase.tool)
		property, ok := function.Parameters.Properties.Get(testCase.property)
		if !ok {
			t.Errorf("%s has no %s property:\n%s", testCase.tool, testCase.property, function.Parameters.String())
			continue
		}
		if !strings.Contains(property.Description, strconv.Itoa(testCase.want)) {
			t.Errorf("%s.%s does not tell the model it gets %d when the argument is omitted: %q",
				testCase.tool, testCase.property, testCase.want, property.Description)
		}
	}
}

// TestBrowserSearchHonoursTopn guards the fix that made declaring topn honest.
//
// Tool arguments arrive decoded from JSON, where every number is a float64.
// Execute read topn as an int alone, so the assertion never matched and the
// model's choice was discarded — declaring the parameter to the model while
// ignoring what it sent would have been a facade.
func TestBrowserSearchHonoursTopn(t *testing.T) {
	for _, testCase := range []struct {
		name string
		args map[string]any
		want int
	}{
		{"decoded from json", map[string]any{"topn": float64(2)}, 2},
		{"passed as a go int", map[string]any{"topn": 2}, 2},
		{"omitted", map[string]any{}, defaultBrowserSearchResults},
		{"not a number", map[string]any{"topn": "two"}, defaultBrowserSearchResults},
		{"zero", map[string]any{"topn": float64(0)}, defaultBrowserSearchResults},
		{"negative", map[string]any{"topn": float64(-1)}, defaultBrowserSearchResults},
		// Honouring topn must not hand the model a larger request than this
		// tool has ever made. Every search it ran before topn was declared
		// asked for maxBrowserSearchResults.
		{"above the ceiling", map[string]any{"topn": float64(100)}, maxBrowserSearchResults},
	} {
		t.Run(testCase.name, func(t *testing.T) {
			if got := browserSearchTopn(testCase.args); got != testCase.want {
				t.Errorf("topn = %d, want %d for %v", got, testCase.want, testCase.args["topn"])
			}
		})
	}
}
