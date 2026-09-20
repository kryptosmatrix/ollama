//go:build windows || darwin

package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/ollama/ollama/app/ui/responses"
)

// discoveryServer returns a server that searches a home directory belonging to
// the test. Searching the real one would read the developer's own MCP clients
// — and report their configuration back through the API.
func discoveryServer(t *testing.T) (*Server, string) {
	t.Helper()
	home := t.TempDir()
	server, _ := mcpSignInServer(t)
	server.discoveryHome = home
	return server, home
}

// call drives the real mux, with the token the server requires. Going through
// the mux rather than calling the handler directly is what proves the route is
// registered at all.
func call(t *testing.T, server *Server, method, path string) *httptest.ResponseRecorder {
	t.Helper()
	request := httptest.NewRequest(method, path, strings.NewReader(""))
	request.AddCookie(&http.Cookie{Name: "token", Value: server.Token})
	recorder := httptest.NewRecorder()
	server.Handler().ServeHTTP(recorder, request)
	return recorder
}

func writeClaudeDesktopConfig(t *testing.T, home, body string) string {
	t.Helper()
	dir := filepath.Join(home, "Library", "Application Support", "Claude")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(dir, "claude_desktop_config.json")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	return path
}

func TestDiscoverRouteReportsAnotherClientsServers(t *testing.T) {
	server, home := discoveryServer(t)
	writeClaudeDesktopConfig(t, home, `{
	  "mcpServers": {"files": {"command": "uvx", "args": ["mcp-server-files"]}}
	}`)

	recorder := call(t, server, http.MethodGet, "/api/v1/mcp-discover")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", recorder.Code, recorder.Body.String())
	}

	var body responses.MCPDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\n%s", err, recorder.Body.String())
	}
	if len(body.Servers) != 1 {
		t.Fatalf("servers = %+v", body.Servers)
	}

	found := body.Servers[0]
	if found.Name != "files" || found.Command != "uvx" {
		t.Errorf("found = %+v", found)
	}
	if found.Runs != "uvx mcp-server-files" {
		t.Errorf("runs = %q; the page shows this before anything is added", found.Runs)
	}
	if len(found.Sources) != 1 || found.Sources[0] != "Claude Desktop" {
		t.Errorf("sources = %v", found.Sources)
	}

	// The page can only say where it looked if the API tells it.
	if len(body.Searched) == 0 {
		t.Error("no searched paths were reported")
	}
}

// Discovery offers; it never adds. A server found on this machine must not
// appear in the configured list until the user has asked for it.
func TestDiscoverRouteAddsNothing(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, home := discoveryServer(t)
	writeClaudeDesktopConfig(t, home, `{
	  "mcpServers": {"files": {"command": "uvx", "args": ["mcp-server-files"]}}
	}`)

	if recorder := call(t, server, http.MethodGet, "/api/v1/mcp-discover"); recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}

	if data, err := os.ReadFile(configPath); err == nil && strings.Contains(string(data), "mcp-server-files") {
		t.Errorf("discovery wrote to the configuration:\n%s", data)
	}
	if len(listServers(t, server)) != 0 {
		t.Errorf("a discovered server appeared in the configured list without being added")
	}
}

// A credential written literally in another application's file must not travel
// into Ollama's, and the response has to say so — the page shows that note
// beside the Add button, which is the only moment the user can act on it.
func TestDiscoverRouteDoesNotHandBackALiteralCredential(t *testing.T) {
	server, home := discoveryServer(t)
	writeClaudeDesktopConfig(t, home, `{
	  "mcpServers": {
	    "weather": {
	      "command": "uvx",
	      "args": ["weather-mcp"],
	      "env": {"WEATHER_API_KEY": "sk-live-real-secret"}
	    }
	  }
	}`)

	recorder := call(t, server, http.MethodGet, "/api/v1/mcp-discover")

	if strings.Contains(recorder.Body.String(), "sk-live-real-secret") {
		t.Fatalf("the credential was sent to the browser:\n%s", recorder.Body.String())
	}

	var body responses.MCPDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(body.Servers) != 1 {
		t.Fatalf("servers = %+v", body.Servers)
	}
	if got := body.Servers[0].Env["WEATHER_API_KEY"]; got != "${env:WEATHER_API_KEY}" {
		t.Errorf("env = %q", got)
	}
	if len(body.Servers[0].Notes) == 0 {
		t.Error("nothing told the user their credential was left behind")
	}
}

// The probe is a POST because it contacts things. A GET must not run it: a
// page load is not a decision to scan the machine.
func TestProbeRouteIsAPostAndAnswersWithAList(t *testing.T) {
	server, _ := discoveryServer(t)

	// A GET falls through to the frontend, which is a page and not a probe.
	// What must not happen is the probe answering a plain read.
	getRecorder := call(t, server, http.MethodGet, "/api/v1/mcp-probe")
	if strings.Contains(getRecorder.Header().Get("Content-Type"), "application/json") {
		t.Errorf("GET /api/v1/mcp-probe answered with JSON; reading a page must not start a probe:\n%s", getRecorder.Body.String())
	}

	recorder := call(t, server, http.MethodPost, "/api/v1/mcp-probe")
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d\n%s", recorder.Code, recorder.Body.String())
	}

	var body responses.MCPDiscoveryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\n%s", err, recorder.Body.String())
	}
	// What it found depends on the machine, so nothing is asserted about the
	// contents. What must hold is that the request answered with a list rather
	// than a failure, and that a probe which could not run says why in the body
	// instead of replacing the page with an error.
	if body.Servers == nil && body.Error == "" {
		t.Error("the probe answered with neither a list nor a reason")
	}
}
