//go:build windows || darwin

package ui

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/app/tools"
	"github.com/ollama/ollama/mcp"
)

// approvalTool requires approval and records whether it ever ran. Whether it
// ran is the assertion that matters: a declined call must not execute.
type approvalTool struct {
	name     string
	executed int
}

func (a *approvalTool) Name() string           { return a.name }
func (a *approvalTool) Description() string    { return "needs approval" }
func (a *approvalTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (a *approvalTool) Prompt() string         { return "" }
func (a *approvalTool) Execute(context.Context, map[string]any) (any, string, error) {
	a.executed++
	return nil, "ran", nil
}
func (a *approvalTool) RequiresApproval(map[string]any) bool { return true }
func (a *approvalTool) ApprovalScope(map[string]any) string  { return a.name }

type openTool struct{ name string }

func (o *openTool) Name() string           { return o.name }
func (o *openTool) Description() string    { return "no approval needed" }
func (o *openTool) Schema() map[string]any { return map[string]any{"type": "object"} }
func (o *openTool) Prompt() string         { return "" }
func (o *openTool) Execute(context.Context, map[string]any) (any, string, error) {
	return nil, "ran", nil
}

func approvalServer() (*Server, *tools.Registry, *approvalTool) {
	gated := &approvalTool{name: "files__read"}
	registry := tools.NewRegistry()
	registry.Register(gated)
	registry.Register(&openTool{name: "web_search"})
	return &Server{Approvals: tools.NewApprovals()}, registry, gated
}

// TestApprovalGateLetsUngatedToolsThrough guards the existing behaviour: the
// first-party tools do not opt into approval and must keep running untouched.
func TestApprovalGateLetsUngatedToolsThrough(t *testing.T) {
	server, registry, _ := approvalServer()
	recorder := httptest.NewRecorder()

	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "web_search", nil)
	if err != nil {
		t.Fatalf("an ungated tool must not be gated: %v", err)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("nothing should have been sent to the client, got %q", recorder.Body.String())
	}
}

func TestApprovalGateIgnoresUnknownTools(t *testing.T) {
	server, registry, _ := approvalServer()
	recorder := httptest.NewRecorder()
	if err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "nope", nil); err != nil {
		t.Fatalf("an unknown tool is refused later by Execute with a clearer message: %v", err)
	}
}

// awaitInBackground starts the gate and returns once the client has been sent
// the approval event, along with the identifier the client would answer with.
func awaitInBackground(t *testing.T, server *Server, registry *tools.Registry, chatID string) (<-chan error, *httptest.ResponseRecorder, string) {
	t.Helper()
	recorder := httptest.NewRecorder()
	errs := make(chan error, 1)
	go func() {
		errs <- server.awaitToolApproval(t.Context(), recorder, recorder, chatID, registry, "files__read", map[string]any{"path": "/etc/passwd"})
	}()

	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		if server.Approvals.Pending() > 0 && recorder.Body.Len() > 0 {
			break
		}
		time.Sleep(5 * time.Millisecond)
	}

	var event struct {
		EventName     string         `json:"eventName"`
		ToolName      string         `json:"toolName"`
		ApprovalID    string         `json:"approvalId"`
		ApprovalScope string         `json:"approvalScope"`
		ApprovalArgs  map[string]any `json:"approvalArgs"`
	}
	line := strings.TrimSpace(recorder.Body.String())
	if line == "" {
		t.Fatal("the client was never told an approval was needed")
	}
	if err := json.Unmarshal([]byte(strings.Split(line, "\n")[0]), &event); err != nil {
		t.Fatalf("approval event is not decodable: %v\n%s", err, line)
	}
	if event.EventName != "tool_approval" {
		t.Fatalf("eventName = %q, want tool_approval", event.EventName)
	}
	if event.ToolName != "files__read" || event.ApprovalScope != "files__read" {
		t.Errorf("event = %+v", event)
	}
	if event.ApprovalArgs["path"] != "/etc/passwd" {
		t.Errorf("the arguments must reach the user, got %+v", event.ApprovalArgs)
	}
	if !strings.HasPrefix(event.ApprovalID, chatID+":") {
		t.Errorf("approvalId = %q, want it to carry its chat", event.ApprovalID)
	}
	return errs, recorder, event.ApprovalID
}

func postApproval(t *testing.T, server *Server, chatID string, body map[string]any) *httptest.ResponseRecorder {
	t.Helper()
	encoded, err := json.Marshal(body)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/"+chatID+"/approval", strings.NewReader(string(encoded)))
	request.SetPathValue("id", chatID)
	recorder := httptest.NewRecorder()
	if err := server.chatApproval(recorder, request); err != nil {
		// The handler reports the status itself; the error is informational.
		t.Logf("chatApproval: %v", err)
	}
	return recorder
}

// TestApprovalRoundTrip is the whole point of Phase 3b: a tool call blocks, the
// user is asked over the open stream, the answer arrives on a separate request,
// and the call proceeds.
func TestApprovalRoundTrip(t *testing.T) {
	server, registry, _ := approvalServer()
	errs, _, approvalID := awaitInBackground(t, server, registry, "chat-1")

	recorder := postApproval(t, server, "chat-1", map[string]any{"approvalId": approvalID, "allow": true})
	if recorder.Code != http.StatusOK {
		t.Fatalf("approval POST status = %d, body %s", recorder.Code, recorder.Body.String())
	}

	select {
	case err := <-errs:
		if err != nil {
			t.Fatalf("an allowed call must proceed: %v", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting call never resumed; the chat would hang")
	}
	if server.Approvals.Pending() != 0 {
		t.Errorf("pending = %d after answering", server.Approvals.Pending())
	}
}

func TestDeclinedApprovalStopsTheCall(t *testing.T) {
	server, registry, gated := approvalServer()
	errs, _, approvalID := awaitInBackground(t, server, registry, "chat-1")

	postApproval(t, server, "chat-1", map[string]any{"approvalId": approvalID, "allow": false})

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("a declined call must not proceed")
		}
		if !strings.Contains(err.Error(), "declined") {
			t.Errorf("error = %v, want it to say the user declined", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the waiting call never resumed")
	}
	if gated.executed != 0 {
		t.Error("the tool ran despite being declined")
	}
}

func TestRememberedApprovalIsNotAskedAgain(t *testing.T) {
	server, registry, _ := approvalServer()
	errs, _, approvalID := awaitInBackground(t, server, registry, "chat-1")

	postApproval(t, server, "chat-1", map[string]any{"approvalId": approvalID, "allow": true, "remember": true})
	if err := <-errs; err != nil {
		t.Fatalf("Await: %v", err)
	}

	// The second call must not ask.
	recorder := httptest.NewRecorder()
	if err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", nil); err != nil {
		t.Fatalf("a remembered scope must not be asked about again: %v", err)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("the user was asked a second time: %q", recorder.Body.String())
	}

	// A different chat has not agreed to anything.
	other := httptest.NewRecorder()
	done := make(chan error, 1)
	go func() {
		done <- server.awaitToolApproval(t.Context(), other, other, "chat-2", registry, "files__read", nil)
	}()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if other.Body.Len() > 0 {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Error("a second chat inherited the first chat's approval")
}

func TestApprovalFromAnotherChatIsRefused(t *testing.T) {
	server, registry, gated := approvalServer()
	errs, _, approvalID := awaitInBackground(t, server, registry, "chat-1")

	recorder := postApproval(t, server, "chat-2", map[string]any{"approvalId": approvalID, "allow": true})
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400: one chat must not answer another chat's question", recorder.Code)
	}

	select {
	case err := <-errs:
		t.Fatalf("the waiting call was released by another chat's answer: %v", err)
	case <-time.After(200 * time.Millisecond):
	}
	if gated.executed != 0 {
		t.Error("the tool ran on another chat's answer")
	}
}

func TestApprovalForSomethingNotWaitingIsAConflict(t *testing.T) {
	server, _, _ := approvalServer()
	recorder := postApproval(t, server, "chat-1", map[string]any{"approvalId": "chat-1:gone", "allow": true})
	if recorder.Code != http.StatusConflict {
		t.Errorf("status = %d, want 409 for an answer nobody is waiting for", recorder.Code)
	}
}

func TestApprovalRequiresAnIdentifier(t *testing.T) {
	server, _, _ := approvalServer()
	recorder := postApproval(t, server, "chat-1", map[string]any{"allow": true})
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", recorder.Code)
	}
}

func TestApprovalTimeoutRefusesRatherThanHanging(t *testing.T) {
	server, registry, gated := approvalServer()
	server.Approvals.Timeout = 150 * time.Millisecond

	recorder := httptest.NewRecorder()
	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", nil)
	if err == nil {
		t.Fatal("an unanswered approval must eventually refuse rather than hold the chat open")
	}
	if gated.executed != 0 {
		t.Error("the tool ran after a timeout")
	}
}

// mcpServer builds a Server with a live MCP manager attached, exactly as
// app/cmd/app does, so the registration and gating paths can be exercised
// against a real server subprocess.
func mcpServer(t *testing.T) *Server {
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

	manager := mcp.NewManager(mcp.Options{ConnectTimeout: 30 * time.Second, Approvals: approvals})
	t.Cleanup(func() { manager.Close() })
	manager.Connect(t.Context(), cfg)
	if state, _ := manager.State("raw"); state.Status != mcp.StatusConnected {
		t.Fatalf("mcp server status = %q, err = %v", state.Status, state.Err)
	}

	return &Server{MCP: manager, Approvals: tools.NewApprovals()}
}

// TestChatRegistryOffersMCPTools is the activation evidence for the desktop
// app: the registry the chat handler builds for a request must contain the
// tools of connected MCP servers. Without it every layer beneath could be
// perfect and the app would show nothing.
func TestChatRegistryOffersMCPTools(t *testing.T) {
	server := mcpServer(t)
	registry := tools.NewRegistry()
	server.registerMCPTools(registry)

	if _, ok := registry.Get("raw__echo"); !ok {
		t.Fatalf("the MCP tool is missing from the request's registry; it has %v", registry.ToolNames())
	}
	for _, refused := range []string{"raw__bash", "raw__scalar_input"} {
		if _, present := registry.Get(refused); present {
			t.Errorf("%q must not be offered", refused)
		}
	}
}

func TestChatRegistryWithoutAnMCPManager(t *testing.T) {
	server := &Server{Approvals: tools.NewApprovals()}
	registry := tools.NewRegistry()
	server.registerMCPTools(registry)
	if len(registry.ToolNames()) != 0 {
		t.Errorf("a server with no manager must register nothing, got %v", registry.ToolNames())
	}
}

// TestMCPToolsAreGatedInTheApp is the security-critical composition: an MCP
// tool registered in the app must go through the approval path, not run the
// moment the model names it.
func TestMCPToolsAreGatedInTheApp(t *testing.T) {
	server := mcpServer(t)
	server.Approvals.Timeout = 300 * time.Millisecond
	registry := tools.NewRegistry()
	server.registerMCPTools(registry)

	recorder := httptest.NewRecorder()
	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "raw__echo", map[string]any{"text": "hi"})
	if err == nil {
		t.Fatal("an MCP tool ran in the app without being approved")
	}
	if !strings.Contains(recorder.Body.String(), "tool_approval") {
		t.Errorf("the user was never asked: %q", recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "raw__echo") {
		t.Errorf("the prompt did not name the tool: %q", recorder.Body.String())
	}
}
