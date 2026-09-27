//go:build darwin || windows

package instructionsbaseline_test

import (
 "bytes"
 "encoding/json"
 "io"
 "log/slog"
 "net/http"
 "net/http/httptest"
 "path/filepath"
 "sync/atomic"
 "testing"
 "time"

 "github.com/ollama/ollama/app/store"
 "github.com/ollama/ollama/app/ui"
)

func instructionBaselineServer(t *testing.T) (*httptest.Server, *atomic.Int32) {
 t.Helper()
 var inference atomic.Int32
 upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
  if r.URL.Path == "/api/chat" { inference.Add(1) }
  if r.URL.Path == "/" || r.URL.Path == "/api/version" {
   w.Header().Set("Content-Type", "application/json")
   io.WriteString(w, `{"version":"synthetic-baseline"}`)
   return
  }
  http.Error(w, "synthetic upstream: unsupported endpoint", http.StatusNotFound)
 }))
 t.Cleanup(upstream.Close)
 t.Setenv("OLLAMA_HOST", upstream.URL)
 t.Setenv("OLLAMA_CORS", "false")
 s := &store.Store{DBPath: filepath.Join(t.TempDir(), "baseline.sqlite")}
 t.Cleanup(func() { if err := s.Close(); err != nil { t.Error(err) } })
 // Public production constructor only; no TTS test helper or speech endpoint.
 server := &ui.Server{Store:s, Token:"synthetic-g1-test-token", Dev:false,
  Logger:slog.New(slog.NewTextHandler(io.Discard,nil))}
 endpoint := httptest.NewServer(server.Handler())
 t.Cleanup(endpoint.Close)
 return endpoint, &inference
}

func instructionBaselineRequest(t *testing.T, endpoint, method, path string, authenticated bool, body []byte) (int, []byte) {
 t.Helper()
 req, err := http.NewRequest(method, endpoint+path, bytes.NewReader(body))
 if err != nil { t.Fatal(err) }
 req.Header.Set("Content-Type", "application/json")
 if authenticated { req.AddCookie(&http.Cookie{Name:"token",Value:"synthetic-g1-test-token"}) }
 response, err := (&http.Client{Timeout:15*time.Second}).Do(req)
 if err != nil { t.Fatal(err) }
 defer response.Body.Close()
 data, err := io.ReadAll(response.Body)
 if err != nil { t.Fatal(err) }
 return response.StatusCode, data
}

func TestG1BaselineExistingAuthenticatedSettingsControl(t *testing.T) {
 server, inference := instructionBaselineServer(t)
 status, _ := instructionBaselineRequest(t,server.URL,http.MethodGet,"/api/v1/settings",false,nil)
 if status != http.StatusForbidden { t.Fatalf("unauthenticated control status=%d; want 403",status) }
 status, body := instructionBaselineRequest(t,server.URL,http.MethodGet,"/api/v1/settings",true,nil)
 var decoded struct { Settings json.RawMessage `json:"settings"` }
 if status != http.StatusOK || json.Unmarshal(body,&decoded) != nil || len(decoded.Settings)==0 {
  t.Fatalf("authenticated existing settings control failed: status=%d body=%s",status,body)
 }
 if inference.Load()!=0 { t.Fatalf("settings control invoked inference %d times",inference.Load()) }
 t.Log("G1_EXISTING_HTTP_CONTROL=PASS; real authenticated router and synthetic store; inference=0")
}

// This pre-feature diagnostic stops at the first absent production boundary.
// It cannot prove restart or request delivery when saving is unavailable.
func TestG1BaselineInstructionSaveBoundary(t *testing.T) {
 server, inference := instructionBaselineServer(t)
 const text = "Use Australian English. Preserve this exact G1 baseline value — café."
 body, err := json.Marshal(map[string]any{"expected_revision":"0","edit":map[string]any{"enabled":true,"text":text,"binding":nil}})
 if err != nil { t.Fatal(err) }
 status, response := instructionBaselineRequest(t,server.URL,http.MethodPut,"/api/v1/instructions",true,body)
 var profile struct { Revision string `json:"revision"`; Enabled bool `json:"enabled"`; Text string `json:"text"` }
 decoded := json.Unmarshal(response,&profile)==nil
 if inference.Load()!=0 { t.Fatalf("instruction save unexpectedly invoked inference %d times",inference.Load()) }
 if status!=http.StatusOK || !decoded || profile.Revision=="" || profile.Revision=="0" || !profile.Enabled || profile.Text!=text {
  t.Fatalf("G1_SAVE_BOUNDARY_MISSING: authenticated PUT status=%d decoded=%t revision=%q; response=%s; restart/delivery not reached",status,decoded,profile.Revision,response)
 }
}
