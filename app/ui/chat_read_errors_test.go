//go:build windows || darwin

package ui

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/app/store"
	"github.com/ollama/ollama/app/types/not"
)

const chatReadStageEnv = "OLLAMA_CHAT_READ_TEST_STAGE"

// R-01 through R-06. This regression uses the existing authenticated test
// server fixture and the real HTTP router, with process-separated storage.
func TestChatReadIntegrityAcrossProcesses(t *testing.T) {
	if phase := os.Getenv(chatReadStageEnv); phase != "" {
		runChatReadPhase(t, phase)
		if !t.Failed() {
			fmt.Println("CHAT_READ_PHASE=" + phase)
		}
		return
	}
	executable, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	home := t.TempDir()
	for _, phase := range []string{"seed", "fault", "recover", "positive"} {
		t.Run(phase, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			cmd := exec.CommandContext(ctx, executable, "-test.run=^TestChatReadIntegrityAcrossProcesses$", "-test.v", "-test.timeout=25s")
			cmd.Env = chatReadEnvironment(home, phase)
			output, err := cmd.CombinedOutput()
			t.Logf("phase=%s\n%s", phase, output)
			if err != nil {
				t.Fatalf("phase %s failed: %v", phase, err)
			}
			if !strings.Contains(string(output), "CHAT_READ_PHASE="+phase) {
				t.Fatalf("phase %s did not execute its completion assertion", phase)
			}
		})
	}
}

func chatReadEnvironment(home, phase string) []string {
	changes := map[string]string{
		"HOME": home, "USERPROFILE": home, "LOCALAPPDATA": home,
		"OLLAMA_HOST": "http://127.0.0.1:1", "OLLAMA_CORS": "false",
		chatReadStageEnv: phase,
	}
	result := make([]string, 0, len(os.Environ())+len(changes))
	for _, item := range os.Environ() {
		key, _, _ := strings.Cut(item, "=")
		if _, replaced := changes[key]; !replaced {
			result = append(result, item)
		}
	}
	for key, value := range changes {
		result = append(result, key+"="+value)
	}
	return result
}

func chatReadFixture() *store.Chat {
	chat := store.NewChat("preserved-chat")
	chat.Title = "Saved conversation — not a replacement"
	chat.BrowserState = json.RawMessage(`{"saved":"retained"}`)
	chat.Messages = []store.Message{
		store.NewMessage("user", "original user — café", &store.MessageOptions{
			Attachments: []store.File{{Filename: "evidence.txt", Data: []byte("original attachment\nsecond line\n")}},
		}),
		store.NewMessage("assistant", "original assistant", &store.MessageOptions{
			Model: "storage-contract-model",
			ToolCalls: []store.ToolCall{{Type: "function", Function: store.ToolFunction{
				Name: "retained_tool", Arguments: `{"query":"retained"}`,
				Result: map[string]any{"value": "retained result"},
			}}},
		}),
	}
	return chat
}

func runChatReadPhase(t *testing.T, phase string) {
	t.Helper()
	// The child exercises Store's normal platform path after HOME is isolated
	// before package initialisation; DBPath is not overridden.
	s := &store.Store{}
	t.Cleanup(func() {
		if err := s.Close(); err != nil {
			t.Error(err)
		}
	})
	if phase == "seed" {
		if err := s.SetChat(*chatReadFixture()); err != nil {
			t.Fatal(err)
		}
		return
	}
	if _, err := s.Settings(); err != nil {
		t.Fatal(err)
	}
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(home, "Library", "Application Support", "Ollama", "db.sqlite")
	if runtime.GOOS == "windows" {
		path = filepath.Join(os.Getenv("LOCALAPPDATA"), "Ollama", "db.sqlite")
	}
	observer, err := sql.Open("sqlite3", path+"?mode=rw&_busy_timeout=5000")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		if err := observer.Close(); err != nil {
			t.Error(err)
		}
	})
	var hits atomic.Int32
	requests := make(chan api.ChatRequest, 16)
	model := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.Header().Set("Content-Type", "application/json")
		switch r.URL.Path {
		case "/api/show":
			fmt.Fprintln(w, `{"capabilities":["completion"]}`)
		case "/api/chat":
			var request api.ChatRequest
			if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
				http.Error(w, err.Error(), http.StatusBadRequest)
				return
			}
			select {
			case requests <- request:
			default:
				http.Error(w, "unexpected request overflow", http.StatusInternalServerError)
				return
			}
			text := ""
			for _, message := range request.Messages {
				if message.Role == "user" {
					text = message.Content
				}
			}
			if err := json.NewEncoder(w).Encode(map[string]any{
				"model": request.Model, "message": map[string]string{"role": "assistant", "content": "reply:" + text}, "done": true,
			}); err != nil {
				t.Errorf("encode inference fixture response: %v", err)
			}
		default:
			http.Error(w, "unexpected model endpoint", http.StatusInternalServerError)
		}
	}))
	t.Cleanup(model.Close)
	t.Setenv("OLLAMA_HOST", model.URL)
	server := ttsServer(t, &countingSynth{})
	server.Store = s
	server.Logger = slog.New(slog.NewTextHandler(io.Discard, nil))
	if server.Dev {
		t.Fatal("authenticated test fixture unexpectedly enables development mode")
	}
	endpoint := httptest.NewServer(server.Handler())
	t.Cleanup(endpoint.Close)
	client := &http.Client{Timeout: 10 * time.Second}
	post := func(id, prompt string, authenticated bool) (int, []byte) {
		t.Helper()
		body := mustChatReadJSON(t, map[string]string{"model": "storage-contract-model", "prompt": prompt})
		req, err := http.NewRequest(http.MethodPost, endpoint.URL+"/api/v1/chat/"+id, strings.NewReader(body))
		if err != nil {
			t.Fatal(err)
		}
		req.Header.Set("Content-Type", "application/json")
		if authenticated {
			req.AddCookie(&http.Cookie{Name: "token", Value: server.Token})
		}
		response, err := client.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		data, readErr := io.ReadAll(response.Body)
		closeErr := response.Body.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("read response: %v; close: %v", readErr, closeErr)
		}
		return response.StatusCode, data
	}
	switch phase {
	case "fault":
		beforeAuth := chatReadSnapshot(t, observer)
		status, body := post("preserved-chat", "unauthorised request", false)
		if status != http.StatusForbidden || hits.Load() != 0 || !bytes.Equal(beforeAuth, chatReadSnapshot(t, observer)) {
			t.Fatalf("authentication refusal changed data or reached inference: status=%d body=%s", status, body)
		}
		result, err := observer.Exec("UPDATE messages SET stream = 'invalid_boolean' WHERE chat_id = ? AND role = 'user'", "preserved-chat")
		if err != nil {
			t.Fatal(err)
		}
		n, err := result.RowsAffected()
		if err != nil || n != 1 {
			t.Fatalf("fault not installed on exactly one original row: rows=%d err=%v", n, err)
		}
		_, loadErr := s.ChatWithOptions("preserved-chat", true)
		if loadErr == nil {
			t.Fatal("injected malformed boolean did not cause a loader error")
		}
		if errors.Is(loadErr, not.Found) {
			t.Errorf("R-01: malformed existing chat misclassified as absent: %v", loadErr)
		}
		before := chatReadSnapshot(t, observer)
		status, body = post("preserved-chat", "must not replace history", true)
		if status != http.StatusInternalServerError {
			t.Errorf("R-04: status=%d body=%s; want 500", status, body)
		}
		var failure map[string]string
		if err := json.Unmarshal(body, &failure); err != nil || failure["error"] == "" {
			t.Errorf("R-04: missing explicit error response: %s", body)
		}
		if hits.Load() != 0 {
			t.Errorf("R-04: inference reached %d times after failed load", hits.Load())
		}
		after := chatReadSnapshot(t, observer)
		if !bytes.Equal(before, after) {
			t.Errorf("R-03: persisted records changed after failed read\nbefore=%s\nafter=%s", before, after)
		}
		t.Logf("R-03/R-04: http=%d model_requests=%d all_records_unchanged=%t", status, hits.Load(), bytes.Equal(before, after))
	case "recover":
		result, err := observer.Exec("UPDATE messages SET stream = 0 WHERE chat_id = ? AND stream = 'invalid_boolean'", "preserved-chat")
		if err != nil {
			t.Fatal(err)
		}
		n, err := result.RowsAffected()
		if err != nil || n != 1 {
			t.Fatalf("R-03: original malformed row did not survive: rows=%d err=%v", n, err)
		}
		chat, err := s.ChatWithOptions("preserved-chat", true)
		if err != nil || chat == nil {
			t.Fatalf("read saved chat: %v", err)
		}
		want := chatReadFixture()
		if chat.Title != want.Title || len(chat.Messages) != 2 || chat.Messages[0].Content != want.Messages[0].Content || chat.Messages[1].Content != want.Messages[1].Content {
			t.Fatalf("R-03: saved conversation was not restored: %#v", chat)
		}
		if len(chat.Messages[0].Attachments) != 1 || !bytes.Equal(chat.Messages[0].Attachments[0].Data, want.Messages[0].Attachments[0].Data) || len(chat.Messages[1].ToolCalls) != 1 || chat.Messages[1].ToolCalls[0].Function.Name != "retained_tool" {
			t.Fatal("R-03: attachment or tool record not restored")
		}
		status, body := post("preserved-chat", "continue after recovery", true)
		if status != http.StatusOK || !bytes.Contains(body, []byte(`"eventName":"done"`)) {
			t.Fatalf("R-05: recovery continuation failed: status=%d body=%s", status, body)
		}
		chat, err = s.ChatWithOptions("preserved-chat", true)
		if err != nil || chat == nil || len(chat.Messages) != 4 || chat.Messages[0].Content != want.Messages[0].Content || chat.Messages[3].Content != "reply:continue after recovery" {
			t.Fatalf("R-05: restored state was not functionally used: chat=%#v err=%v", chat, err)
		}
		wantBodies := []string{want.Messages[0].Content, want.Messages[1].Content, "continue after recovery", "reply:continue after recovery"}
		for i, text := range wantBodies {
			if chat.Messages[i].Content != text {
				t.Errorf("R-05: recovered message %d=%q; want %q", i, chat.Messages[i].Content, text)
			}
		}
		if hits.Load() == 0 {
			t.Fatal("R-05: successful recovery did not activate the inference service counter")
		}
		select {
		case request := <-requests:
			found := false
			for _, message := range request.Messages {
				found = found || strings.Contains(message.Content, want.Messages[0].Content)
			}
			if !found {
				t.Fatal("R-05: restored history did not reach the model-request consumer")
			}
		default:
			t.Fatal("R-05: no model request was observed")
		}
	case "positive":
		if existing, err := s.ChatWithOptions("new-control-chat", true); existing != nil || !errors.Is(err, not.Found) {
			t.Fatalf("R-06: new-chat control was not genuinely absent: chat=%#v err=%v", existing, err)
		}
		for _, text := range []string{"first independent prompt", "second independent prompt — different"} {
			status, body := post("new-control-chat", text, true)
			if status != http.StatusOK || !bytes.Contains(body, []byte(`"eventName":"done"`)) {
				t.Fatalf("R-05/R-06: valid continuation failed: status=%d body=%s", status, body)
			}
		}
		chat, err := s.ChatWithOptions("new-control-chat", true)
		if err != nil || chat == nil || len(chat.Messages) != 4 {
			t.Fatalf("R-05/R-06: positive conversation missing: chat=%#v err=%v", chat, err)
		}
		if chat.ID != "new-control-chat" || hits.Load() == 0 {
			t.Fatal("R-06: requested chat identity or inference-counter activation missing")
		}
		want := []string{"first independent prompt", "reply:first independent prompt", "second independent prompt — different", "reply:second independent prompt — different"}
		for i, text := range want {
			if chat.Messages[i].Content != text {
				t.Errorf("R-05/R-06: message %d=%q; want %q", i, chat.Messages[i].Content, text)
			}
		}
	default:
		t.Fatalf("unknown integrity phase %q", phase)
	}
}

func mustChatReadJSON(t *testing.T, value any) string {
	t.Helper()
	data, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// Raw queries preserve IDs, values and blobs independently of the production
// getter. Empty tables are failures, not vacuous equality proofs.
func chatReadSnapshot(t *testing.T, db *sql.DB) []byte {
	t.Helper()
	state := make(map[string][][]any)
	for _, table := range []string{"chats", "messages", "attachments", "tool_calls"} {
		rows, err := db.Query("SELECT * FROM " + table + " ORDER BY id")
		if err != nil {
			t.Fatal(err)
		}
		columns, err := rows.Columns()
		if err != nil {
			rows.Close()
			t.Fatal(err)
		}
		for rows.Next() {
			values := make([]any, len(columns))
			pointers := make([]any, len(columns))
			for i := range values {
				pointers[i] = &values[i]
			}
			if err := rows.Scan(pointers...); err != nil {
				rows.Close()
				t.Fatal(err)
			}
			state[table] = append(state[table], values)
		}
		readErr := rows.Err()
		closeErr := rows.Close()
		if readErr != nil || closeErr != nil {
			t.Fatalf("snapshot %s: read=%v close=%v", table, readErr, closeErr)
		}
		if len(state[table]) == 0 {
			t.Errorf("R-03: fixture table %s unexpectedly empty", table)
		}
	}
	return []byte(mustChatReadJSON(t, state))
}
