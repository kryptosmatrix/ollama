package tools

import (
	"context"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/agent"
	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/mcp"
)

// scriptedClient plays a fixed sequence of model turns. The first turn asks for
// a tool, the second replies in prose, which is the shape of a real agent run.
type scriptedClient struct {
	turns []api.Message
	seen  []api.ChatRequest
}

func (c *scriptedClient) Chat(_ context.Context, req *api.ChatRequest, fn api.ChatResponseFunc) error {
	c.seen = append(c.seen, *req)

	index := min(len(c.seen)-1, len(c.turns)-1)
	message := c.turns[index]

	return fn(api.ChatResponse{Model: req.Model, Message: message, Done: true, DoneReason: "stop"})
}

type prompter struct {
	allow   bool
	prompts []agent.ApprovalRequest
}

func (p *prompter) PromptApproval(_ context.Context, req agent.ApprovalRequest) (agent.Approval, error) {
	p.prompts = append(p.prompts, req)
	if !p.allow {
		return agent.Approval{Reason: "the user declined"}, nil
	}
	var scopes []string
	for _, call := range req.Calls {
		scopes = append(scopes, call.ApprovalScope)
	}
	return agent.Approval{AllowScopes: scopes}, nil
}

// liveManager builds and connects the hand-written MCP server from the mcp
// package's testdata, approves it, and returns a connected manager. Nothing
// here is mocked: it is a real subprocess speaking the real protocol.
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

	spec := &mcp.ServerSpec{Command: binary}
	cfg := &mcp.Config{}
	cfg.Set("raw", spec)

	approvals := &mcp.Approvals{}
	stored, _ := cfg.Get("raw")
	approvals.Approve(stored, time.Now())

	manager := mcp.NewManager(mcp.Options{
		ConnectTimeout: 30 * time.Second,
		CallTimeout:    15 * time.Second,
		Approvals:      approvals,
	})
	t.Cleanup(func() { manager.Close() })

	manager.Connect(t.Context(), cfg)
	state, _ := manager.State("raw")
	if state.Status != mcp.StatusConnected {
		t.Fatalf("mcp server status = %q, err = %v", state.Status, state.Err)
	}
	return manager
}

func toolCallTurn(name string, args map[string]any) api.Message {
	arguments := api.NewToolCallFunctionArguments()
	for key, value := range args {
		arguments.Set(key, value)
	}
	return api.Message{
		Role: "assistant",
		ToolCalls: []api.ToolCall{{
			ID:       "call-1",
			Function: api.ToolCallFunction{Name: name, Arguments: arguments},
		}},
	}
}

// TestMCPToolReachesAServerThroughTheAgentHarness is the signal-chain proof for
// the CLI surface: a model tool call travels through the real agent session, the
// real registry, this adapter, the manager, and out to a real MCP server
// subprocess — and the server's answer comes back in the run's messages.
//
// Every link is production code. A facade at any one of them fails this test.
func TestMCPToolReachesAServerThroughTheAgentHarness(t *testing.T) {
	manager := liveManager(t)

	registry := &agent.Registry{}
	registered, err := RegisterMCP(registry, manager)
	if err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}
	if len(registered) == 0 {
		t.Fatal("no tools were registered")
	}
	if _, ok := registry.Get("raw__echo"); !ok {
		t.Fatalf("raw__echo was not registered; registry has %v", registry.Names())
	}

	approver := &prompter{allow: true}
	session := &agent.Session{
		Client: &scriptedClient{turns: []api.Message{
			toolCallTurn("raw__echo", map[string]any{"text": "through the harness"}),
			{Role: "assistant", Content: "done"},
		}},
		Tools:            registry,
		ApprovalPrompter: approver,
		ApprovalState:    &agent.ApprovalState{},
	}

	result, err := session.Run(t.Context(), agent.RunOptions{
		Model:         "test-model",
		NewMessages:   []api.Message{{Role: "user", Content: "use the echo tool"}},
		MaxToolRounds: 3,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	var toolContent string
	for _, message := range result.Messages {
		if message.Role == "tool" {
			toolContent = message.Content
		}
	}
	if toolContent == "" {
		t.Fatalf("no tool message in the run; messages were %+v", result.Messages)
	}
	if !strings.Contains(toolContent, "echo: through the harness") {
		t.Errorf("tool message = %q, want the subprocess's own reply", toolContent)
	}

	if len(approver.prompts) != 1 {
		t.Errorf("the user was prompted %d times, want once", len(approver.prompts))
	} else if scope := approver.prompts[0].Calls[0].ApprovalScope; scope != "raw__echo" {
		t.Errorf("approval scope = %q, want the tool scoped to its server", scope)
	}
}

// TestDeniedApprovalStopsTheCall proves the approval gate is load-bearing: when
// the user declines, the server must not be called at all, and the model must
// be told.
func TestDeniedApprovalStopsTheCall(t *testing.T) {
	manager := liveManager(t)

	registry := &agent.Registry{}
	if _, err := RegisterMCP(registry, manager); err != nil {
		t.Fatalf("RegisterMCP: %v", err)
	}

	approver := &prompter{allow: false}
	session := &agent.Session{
		Client: &scriptedClient{turns: []api.Message{
			toolCallTurn("raw__echo", map[string]any{"text": "should never arrive"}),
			{Role: "assistant", Content: "understood"},
		}},
		Tools:            registry,
		ApprovalPrompter: approver,
		ApprovalState:    &agent.ApprovalState{},
	}

	result, err := session.Run(t.Context(), agent.RunOptions{
		Model:         "test-model",
		NewMessages:   []api.Message{{Role: "user", Content: "use the echo tool"}},
		MaxToolRounds: 3,
	})
	if err != nil {
		t.Fatalf("Run: %v", err)
	}

	for _, message := range result.Messages {
		if message.Role == "tool" && strings.Contains(message.Content, "echo: should never arrive") {
			t.Fatal("the tool ran despite the user declining")
		}
	}
	if len(approver.prompts) == 0 {
		t.Error("the user was never asked")
	}
}

func TestMCPToolShape(t *testing.T) {
	manager := liveManager(t)

	tool, err := NewMCP(manager, mcp.Tool{Server: "raw", Name: "echo", Description: "returns its argument"})
	if err != nil {
		t.Fatalf("NewMCP: %v", err)
	}

	if tool.Name() != "raw__echo" {
		t.Errorf("Name() = %q, want the namespaced name", tool.Name())
	}
	if !agent.ToolRequiresApproval(tool, nil) {
		t.Error("an MCP tool must require approval: approving a server is agreement to run it, not to let the model drive it")
	}
	scoped, ok := any(tool).(agent.ScopedTool)
	if !ok {
		t.Fatal("an MCP tool must scope its approval")
	}
	if got := scoped.ApprovalScope(nil); got != "raw__echo" {
		t.Errorf("ApprovalScope = %q, want the tool scoped to its own server", got)
	}
	if tool.Schema().Name != "raw__echo" {
		t.Errorf("Schema().Name = %q", tool.Schema().Name)
	}
}

func TestMCPToolSurfacesAServerReportedFailure(t *testing.T) {
	manager := liveManager(t)

	tool, err := NewMCP(manager, mcp.Tool{Server: "raw", Name: "fail"})
	if err != nil {
		t.Fatalf("NewMCP: %v", err)
	}

	_, err = tool.Execute(t.Context(), agent.ToolContext{}, nil)
	if err == nil {
		t.Fatal("a server-reported failure must reach the harness as an error so it is rendered as a failed call")
	}
	if !strings.Contains(err.Error(), "the tool failed") {
		t.Errorf("error = %v, want the server's own message preserved", err)
	}
}
