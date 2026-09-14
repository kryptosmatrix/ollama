//go:build windows || darwin

package ui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/app/store"
)

// storeWithAutoApprove returns a Store on a fresh database with the
// auto-approve switch in the given position.
func storeWithAutoApprove(t *testing.T, on bool) *store.Store {
	t.Helper()
	s := &store.Store{DBPath: filepath.Join(t.TempDir(), "db.sqlite")}
	t.Cleanup(func() { s.Close() })
	setAutoApprove(t, s, on)
	return s
}

func setAutoApprove(t *testing.T, s *store.Store, on bool) {
	t.Helper()
	settings, err := s.Settings()
	if err != nil {
		t.Fatal(err)
	}
	settings.AutoApproveTools = on
	if err := s.SetSettings(settings); err != nil {
		t.Fatal(err)
	}
}

// TestAutoApproveSwitchSkipsThePrompt: with the switch on, a gated tool call
// proceeds at once, nothing is sent to the client, and nothing is left
// waiting for an answer that will never come.
func TestAutoApproveSwitchSkipsThePrompt(t *testing.T) {
	server, registry, _ := approvalServer()
	server.Store = storeWithAutoApprove(t, true)

	recorder := httptest.NewRecorder()
	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", map[string]any{"path": "/etc/passwd"})
	if err != nil {
		t.Fatalf("with the switch on the call must proceed: %v", err)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("no prompt may be sent while the switch is on, got %q", recorder.Body.String())
	}
	if pending := server.Approvals.Pending(); pending != 0 {
		t.Errorf("pending approvals = %d, want 0", pending)
	}
}

// TestAutoApproveSwitchOffAsksAgain: switching off takes effect on the next
// call, and the earlier automatic approvals left no blanket grant behind in
// the chat's state — a declined answer still stops the tool.
func TestAutoApproveSwitchOffAsksAgain(t *testing.T) {
	server, registry, gated := approvalServer()
	server.Store = storeWithAutoApprove(t, true)

	recorder := httptest.NewRecorder()
	if err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", nil); err != nil {
		t.Fatalf("with the switch on the call must proceed: %v", err)
	}

	setAutoApprove(t, server.Store, false)

	errs, _, id := awaitInBackground(t, server, registry, "chat-1")
	postApproval(t, server, "chat-1", map[string]any{"approvalId": id, "allow": false})
	if err := <-errs; err == nil {
		t.Fatal("after switching off, a declined call must not run")
	}
	if gated.executed != 0 {
		t.Errorf("the tool ran %d times after being declined", gated.executed)
	}
}

// TestAutoApproveWithoutAStoreStillAsks: a Server with no Store has no
// switch, and the existing approval path stands.
func TestAutoApproveWithoutAStoreStillAsks(t *testing.T) {
	server, registry, gated := approvalServer()
	server.Approvals.Timeout = 150 * time.Millisecond

	recorder := httptest.NewRecorder()
	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", nil)
	if err == nil {
		t.Fatal("without a store there is no switch; the call must wait for an answer")
	}
	if !strings.Contains(recorder.Body.String(), "tool_approval") {
		t.Errorf("the user was never asked: %q", recorder.Body.String())
	}
	if gated.executed != 0 {
		t.Error("the tool ran without an answer")
	}
}

// TestAutoApproveUnreadableSettingsStillAsks: a store that cannot be read
// must not be taken as permission. The database path is placed under a
// regular file, so the store cannot open it.
func TestAutoApproveUnreadableSettingsStillAsks(t *testing.T) {
	server, registry, gated := approvalServer()
	server.Approvals.Timeout = 150 * time.Millisecond

	blocker := filepath.Join(t.TempDir(), "not-a-directory")
	if err := os.WriteFile(blocker, []byte("x"), 0o600); err != nil {
		t.Fatal(err)
	}
	broken := &store.Store{DBPath: filepath.Join(blocker, "db.sqlite")}
	t.Cleanup(func() { broken.Close() })
	if _, err := broken.Settings(); err == nil {
		t.Fatal("the fixture store was expected to be unreadable")
	}
	server.Store = broken

	recorder := httptest.NewRecorder()
	err := server.awaitToolApproval(t.Context(), recorder, recorder, "chat-1", registry, "files__read", nil)
	if err == nil {
		t.Fatal("an unreadable store must not approve anything")
	}
	if !strings.Contains(recorder.Body.String(), "tool_approval") {
		t.Errorf("the user was never asked: %q", recorder.Body.String())
	}
	if gated.executed != 0 {
		t.Error("the tool ran on an unreadable store")
	}
}

// fakeModel answers the two calls the chat handler makes to the Ollama
// server: /api/show, reporting the tools capability, and /api/chat, which
// first asks for the MCP echo tool and, once a tool result is in the
// conversation, finishes with plain text.
func fakeModel(t *testing.T) *httptest.Server {
	t.Helper()
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/show":
			w.Header().Set("Content-Type", "application/json")
			fmt.Fprint(w, `{"capabilities":["tools"]}`)
		case "/api/chat":
			var req struct {
				Model    string `json:"model"`
				Messages []struct {
					Role string `json:"role"`
				} `json:"messages"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/x-ndjson")
			for _, m := range req.Messages {
				if m.Role == "tool" {
					fmt.Fprintf(w, `{"model":%q,"created_at":"2026-09-14T08:00:00Z","message":{"role":"assistant","content":"all done"},"done":true,"done_reason":"stop"}`+"\n", req.Model)
					return
				}
			}
			fmt.Fprintf(w, `{"model":%q,"created_at":"2026-09-14T08:00:00Z","message":{"role":"assistant","content":"","tool_calls":[{"function":{"index":0,"name":"raw__echo","arguments":{"text":"hi"}}}]},"done":true,"done_reason":"stop"}`+"\n", req.Model)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(ts.Close)
	return ts
}

// runChat drives the production chat handler for one user message against
// the fake model and a live MCP fixture, and returns the event stream.
func runChat(t *testing.T, server *Server) string {
	t.Helper()
	t.Setenv("OLLAMA_HOST", fakeModel(t).URL)

	body := `{"model":"test-model","prompt":"please echo hi"}`
	request := httptest.NewRequest(http.MethodPost, "/api/v1/chat/new", strings.NewReader(body))
	request.SetPathValue("id", "new")
	recorder := httptest.NewRecorder()
	if err := server.chat(recorder, request); err != nil {
		t.Fatalf("chat: %v\n%s", err, recorder.Body.String())
	}
	return recorder.Body.String()
}

// TestChatLoopRunsMCPToolWithoutAskingWhenSwitchedOn is the activation
// evidence for the switch: through the real chat handler, with a real MCP
// server subprocess, the tool the model asked for runs and its result reaches
// the model and the client, and no approval prompt is ever sent.
func TestChatLoopRunsMCPToolWithoutAskingWhenSwitchedOn(t *testing.T) {
	server := mcpServer(t)
	server.Store = storeWithAutoApprove(t, true)
	server.Approvals.Timeout = 300 * time.Millisecond

	stream := runChat(t, server)
	if strings.Contains(stream, "tool_approval") {
		t.Errorf("an approval prompt was sent while the switch is on:\n%s", stream)
	}
	if !strings.Contains(stream, "echo: hi") {
		t.Errorf("the MCP tool did not run, or its result never reached the client:\n%s", stream)
	}
	if !strings.Contains(stream, "all done") {
		t.Errorf("the model never received the tool result and finished:\n%s", stream)
	}
	if strings.Contains(stream, "was not run") {
		t.Errorf("the tool was refused:\n%s", stream)
	}
}

// TestChatLoopAsksBeforeMCPToolWhenSwitchedOff guards the other direction
// through the same handler: with the switch off, the prompt is sent, and
// with nobody answering the tool is refused rather than run.
func TestChatLoopAsksBeforeMCPToolWhenSwitchedOff(t *testing.T) {
	server := mcpServer(t)
	server.Store = storeWithAutoApprove(t, false)
	server.Approvals.Timeout = 300 * time.Millisecond

	stream := runChat(t, server)
	if !strings.Contains(stream, "tool_approval") {
		t.Errorf("the user was never asked:\n%s", stream)
	}
	if strings.Contains(stream, "echo: hi") {
		t.Errorf("the MCP tool ran without approval:\n%s", stream)
	}
	if !strings.Contains(stream, "was not run") {
		t.Errorf("the unanswered call was not refused:\n%s", stream)
	}
}
