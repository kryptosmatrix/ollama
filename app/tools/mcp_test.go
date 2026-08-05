//go:build windows || darwin

package tools

import (
	"encoding/json"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/mcp"
)

// liveManager builds and connects the hand-written MCP server from the mcp
// package's testdata and approves it, so these tests run against a real
// subprocess speaking the real protocol rather than a stand-in.
func liveManager(t *testing.T) *mcp.Manager {
	t.Helper()

	name := "rawserver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "github.com/ollama/ollama/mcp/testdata/rawserver")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build rawserver: %v\n%s", err, output)
	}

	cfg := &mcp.Config{}
	cfg.Set("raw", &mcp.ServerSpec{Command: binary})
	stored, _ := cfg.Get("raw")

	approvals := &mcp.Approvals{}
	approvals.Approve(stored, time.Now())

	manager := mcp.NewManager(mcp.Options{ConnectTimeout: 30 * time.Second, CallTimeout: 15 * time.Second, Approvals: approvals})
	t.Cleanup(func() { manager.Close() })
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != mcp.StatusConnected {
		t.Fatalf("mcp server status = %q, err = %v", state.Status, state.Err)
	}
	return manager
}

func TestRegisterMCPOffersTheServersTools(t *testing.T) {
	manager := liveManager(t)
	registry := NewRegistry()

	registered, err := RegisterMCP(registry, manager)
	if err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	if len(registered) == 0 {
		t.Fatal("nothing was registered")
	}

	tool, ok := registry.Get("raw__echo")
	if !ok {
		t.Fatalf("raw__echo is missing; registry has %v", registry.ToolNames())
	}
	if tool.Name() != "raw__echo" {
		t.Errorf("Name() = %q", tool.Name())
	}

	// Tools the manager refused must not appear, whatever the server claimed.
	for _, refused := range []string{"raw__bash", "raw__scalar_input", "bash"} {
		if _, present := registry.Get(refused); present {
			t.Errorf("%q must not be offered to the model", refused)
		}
	}
}

func TestRegisterMCPWithNoManager(t *testing.T) {
	registry := NewRegistry()
	registered, err := RegisterMCP(registry, nil)
	if err != nil || len(registered) != 0 {
		t.Fatalf("a nil manager should register nothing without failing, got %v, %v", registered, err)
	}
	if len(registry.ToolNames()) != 0 {
		t.Errorf("registry = %v", registry.ToolNames())
	}
}

func TestMCPToolRequiresApprovalAndIsScoped(t *testing.T) {
	manager := liveManager(t)
	registry := NewRegistry()
	if _, err := RegisterMCP(registry, manager); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	tool, _ := registry.Get("raw__echo")

	if !ToolRequiresApproval(tool, nil) {
		t.Error("an MCP tool must be gated: approving a server is agreement to run it, not to let the model drive it")
	}
	if got := ToolApprovalScope(tool, nil); got != "raw__echo" {
		t.Errorf("ApprovalScope = %q, want the tool scoped to its own server", got)
	}
}

// TestMCPToolKeepsItsFaithfulSchema is the reason OllamaTool exists.
//
// The app's map form of a schema carries only a property's type and
// description, so an MCP tool routed through it would reach the model without
// its enums, nested objects or array items — the parts that make a call valid.
// The assertions below are deliberately on those, not on the required list or
// the descriptions: those survive the lossy path too, so asserting them would
// prove nothing about which path was taken.
func TestMCPToolKeepsItsFaithfulSchema(t *testing.T) {
	manager := liveManager(t)

	rich := mcp.Tool{
		Server:      "raw",
		Name:        "create_issue",
		Description: "opens an issue",
		InputSchema: json.RawMessage(`{
		  "type": "object",
		  "properties": {
		    "priority": {"type": "string", "enum": ["low", "high"]},
		    "labels":   {"type": "array", "items": {"type": "string"}},
		    "author":   {"type": "object", "properties": {"name": {"type": "string"}}}
		  },
		  "required": ["priority"]
		}`),
	}
	adapted, err := NewMCP(manager, rich)
	if err != nil {
		t.Fatalf("NewMCP: %v", err)
	}

	registry := NewRegistry()
	registry.Register(adapted)

	definitions := registry.OllamaTools()
	if len(definitions) != 1 {
		t.Fatalf("got %d definitions", len(definitions))
	}
	encoded, err := json.Marshal(definitions[0].Function)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	rendered := string(encoded)

	// Each of these is dropped by the derived path and kept by the faithful one.
	if !strings.Contains(rendered, `"enum":["low","high"]`) {
		t.Errorf("the enum was lost, so the model cannot see the allowed values:\n%s", rendered)
	}
	if !strings.Contains(rendered, `"items"`) {
		t.Errorf("the array item type was lost:\n%s", rendered)
	}
	if !strings.Contains(rendered, `"name"`) {
		t.Errorf("the nested object's properties were lost:\n%s", rendered)
	}
}

func TestOllamaToolsFallsBackForOrdinaryTools(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&WebSearch{})

	definitions := registry.OllamaTools()
	if len(definitions) != 1 {
		t.Fatalf("got %d definitions", len(definitions))
	}
	function := definitions[0].Function
	if function.Name != "web_search" {
		t.Errorf("Name = %q", function.Name)
	}
	if function.Parameters.Properties.Len() == 0 {
		t.Error("a tool with only a schema map must still get its properties")
	}
	if len(function.Parameters.Required) == 0 {
		t.Error("the required list must survive the map form")
	}
}

func TestOllamaToolsIsStablyOrdered(t *testing.T) {
	registry := NewRegistry()
	registry.Register(&WebFetch{})
	registry.Register(&WebSearch{})

	first := registry.OllamaTools()
	for range 20 {
		next := registry.OllamaTools()
		for i := range first {
			if first[i].Function.Name != next[i].Function.Name {
				t.Fatalf("tool order is not stable: %v then %v", first, next)
			}
		}
	}
	if first[0].Function.Name != "web_fetch" {
		t.Errorf("expected sorted order, got %q first", first[0].Function.Name)
	}
}

func TestMCPToolCallsTheServer(t *testing.T) {
	manager := liveManager(t)
	registry := NewRegistry()
	if _, err := RegisterMCP(registry, manager); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	_, content, err := registry.Execute(t.Context(), "raw__echo", map[string]any{"text": "into the app"})
	if err != nil {
		t.Fatalf("Execute: %v", err)
	}
	if content != "echo: into the app" {
		t.Errorf("content = %q, want the subprocess's own reply", content)
	}
}

func TestMCPToolSurfacesAServerReportedFailure(t *testing.T) {
	manager := liveManager(t)
	registry := NewRegistry()
	if _, err := RegisterMCP(registry, manager); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	_, _, err := registry.Execute(t.Context(), "raw__fail", nil)
	if err == nil {
		t.Fatal("a server-reported failure must reach the app as an error")
	}
	if !strings.Contains(err.Error(), "the tool failed") {
		t.Errorf("error = %v, want the server's own message preserved", err)
	}
}
