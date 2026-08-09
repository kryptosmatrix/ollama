package mcp

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// writeClientConfig puts a foreign MCP client's configuration where discovery
// looks for it, inside a home directory belonging to the test.
func writeClientConfig(t *testing.T, home, label, body string) string {
	t.Helper()
	for _, source := range configSources(home) {
		if source.Label != label {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(source.Path), 0o700); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(source.Path, []byte(body), 0o600); err != nil {
			t.Fatalf("write: %v", err)
		}
		return source.Path
	}
	t.Fatalf("no configuration source called %q on %s", label, runtime.GOOS)
	return ""
}

func discovered(t *testing.T, servers []DiscoveredServer, name string) DiscoveredServer {
	t.Helper()
	for _, server := range servers {
		if server.Name == name {
			return server
		}
	}
	t.Fatalf("no discovered server called %q in %+v", name, servers)
	return DiscoveredServer{}
}

func TestDiscoverReadsAnotherClientsServers(t *testing.T) {
	home := t.TempDir()
	path := writeClientConfig(t, home, "Claude Desktop", `{
	  "mcpServers": {
	    "files": {"command": "uvx", "args": ["mcp-server-files"]},
	    "hosted": {"url": "https://mcp.example.com/v1"}
	  }
	}`)

	found, searched := DiscoverConfigured(home)
	if len(found) != 2 {
		t.Fatalf("found %d servers, want 2: %+v", len(found), found)
	}

	files := discovered(t, found, "files")
	if files.Spec == nil || files.Spec.Command != "uvx" {
		t.Errorf("spec = %+v", files.Spec)
	}
	if files.Runs != "uvx mcp-server-files" {
		t.Errorf("runs = %q", files.Runs)
	}
	if len(files.Sources) != 1 || files.Sources[0] != "Claude Desktop" {
		t.Errorf("sources = %v", files.Sources)
	}
	if len(files.Paths) != 1 || files.Paths[0] != path {
		t.Errorf("paths = %v", files.Paths)
	}
	if files.Origin != OriginConfig {
		t.Errorf("origin = %q", files.Origin)
	}

	// Nothing is enabled or approved by being found.
	if files.Problem != "" {
		t.Errorf("problem = %q", files.Problem)
	}

	if hosted := discovered(t, found, "hosted"); hosted.Spec == nil || hosted.Spec.URL != "https://mcp.example.com/v1" {
		t.Errorf("hosted spec = %+v", hosted.Spec)
	}

	// Every path looked at is reported, existing or not, so the coverage of
	// the search is visible rather than implied.
	if len(searched) < 4 {
		t.Errorf("only %d paths reported as searched: %v", len(searched), searched)
	}
	var sawPath bool
	for _, candidate := range searched {
		if candidate == path {
			sawPath = true
		}
	}
	if !sawPath {
		t.Errorf("the file that was read is not in the searched list: %v", searched)
	}
}

// A literal credential in another application's file must not be copied into
// Ollama's. mcp.json is readable by anything running as this user, and a second
// copy of a secret is a second place it can leak from.
func TestDiscoverDoesNotCopyALiteralCredential(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Claude Desktop", `{
	  "mcpServers": {
	    "weather": {
	      "command": "uvx",
	      "args": ["weather-mcp"],
	      "env": {"WEATHER_API_KEY": "sk-live-real-secret", "WEATHER_UNITS": "metric"}
	    }
	  }
	}`)

	weather := discoveredFrom(t, home, "weather")
	if weather.Spec == nil {
		t.Fatalf("the server was not usable: %s", weather.Problem)
	}
	if got := weather.Spec.Env["WEATHER_API_KEY"]; got != "${env:WEATHER_API_KEY}" {
		t.Errorf("the credential was carried across as %q", got)
	}
	// An ordinary value is left exactly as it was.
	if got := weather.Spec.Env["WEATHER_UNITS"]; got != "metric" {
		t.Errorf("an ordinary value was rewritten: %q", got)
	}
	if !strings.Contains(strings.Join(weather.Notes, " "), "WEATHER_API_KEY") {
		t.Errorf("the user is not told what was left behind: %v", weather.Notes)
	}
	// And what is offered would actually pass Ollama's own validation.
	if err := validateServer(weather.Name, weather.Spec); err != nil {
		t.Errorf("the sanitised server is still invalid: %v", err)
	}
}

func discoveredFrom(t *testing.T, home, name string) DiscoveredServer {
	t.Helper()
	found, _ := DiscoverConfigured(home)
	return discovered(t, found, name)
}

func TestDiscoverKeepsAServerSwitchedOffElsewhereSwitchedOff(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Claude Desktop", `{
	  "mcpServers": {"paused": {"command": "uvx", "args": ["x"], "disabled": true}}
	}`)

	paused := discoveredFrom(t, home, "paused")
	if paused.Spec == nil || !paused.Spec.Disabled {
		t.Fatalf("a disabled server arrived enabled: %+v", paused.Spec)
	}
	if !strings.Contains(strings.Join(paused.Notes, " "), "switched off") {
		t.Errorf("notes do not mention it: %v", paused.Notes)
	}
}

// VS Code calls the same object "servers". A discovery that only knew one
// spelling would silently find nothing in half the files it read.
func TestDiscoverReadsTheVSCodeSpelling(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "VS Code", `{
	  "servers": {"docs": {"command": "npx", "args": ["-y", "docs-mcp"]}}
	}`)

	docs := discoveredFrom(t, home, "docs")
	if docs.Spec == nil || docs.Spec.Command != "npx" {
		t.Fatalf("spec = %+v (problem %q)", docs.Spec, docs.Problem)
	}
	if len(docs.Sources) != 1 || docs.Sources[0] != "VS Code" {
		t.Errorf("sources = %v", docs.Sources)
	}
}

// A name another client accepted may not be one Ollama accepts. Renaming is
// fine; renaming quietly is not.
func TestDiscoverRenamesWhatItMustAndSaysSo(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Cursor", `{
	  "mcpServers": {"my files!": {"command": "uvx", "args": ["f"]}}
	}`)

	found, _ := DiscoverConfigured(home)
	if len(found) != 1 {
		t.Fatalf("found %+v", found)
	}
	renamed := found[0]
	if renamed.Name != "my-files" {
		t.Errorf("name = %q, want my-files", renamed.Name)
	}
	if !strings.Contains(strings.Join(renamed.Notes, " "), "my files!") {
		t.Errorf("the rename is not explained: %v", renamed.Notes)
	}
	if err := validateServer(renamed.Name, renamed.Spec); err != nil {
		t.Errorf("the renamed server is invalid: %v", err)
	}
}

// An entry that cannot be added is reported rather than dropped. Vanishing
// without explanation reads as a discovery that missed it.
func TestDiscoverExplainsWhatItCannotOffer(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Claude Desktop", `{
	  "mcpServers": {"empty": {"args": ["nothing-to-run"]}}
	}`)

	found, _ := DiscoverConfigured(home)
	if len(found) != 1 {
		t.Fatalf("found %+v", found)
	}
	if found[0].Problem == "" {
		t.Errorf("an unusable server was offered as usable: %+v", found[0])
	}
	if found[0].Spec != nil {
		t.Errorf("an unusable server carried a spec: %+v", found[0].Spec)
	}
}

func TestDiscoverSurvivesAFileItCannotRead(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Claude Desktop", "{not json")
	writeClientConfig(t, home, "Cursor", `{"mcpServers": {"ok": {"command": "uvx", "args": ["x"]}}}`)

	found, _ := DiscoverConfigured(home)
	if server := discovered(t, found, "ok"); server.Spec == nil {
		t.Errorf("one bad file stopped the others being read")
	}
	var reported bool
	for _, server := range found {
		if strings.Contains(server.Problem, "not valid JSON") {
			reported = true
		}
	}
	if !reported {
		t.Errorf("the unreadable file was skipped silently: %+v", found)
	}
}

func TestDiscoverFindsNothingInAnEmptyHome(t *testing.T) {
	found, searched := DiscoverConfigured(t.TempDir())
	if len(found) != 0 {
		t.Errorf("found %+v", found)
	}
	if len(searched) == 0 {
		t.Errorf("nothing was reported as searched")
	}
}

// --- the loopback probe ---------------------------------------------------

func TestLoopbackPortReadsOnlyReachableAddresses(t *testing.T) {
	for _, tc := range []struct {
		address string
		port    int
		ok      bool
	}{
		{"*:3000", 3000, true},
		{"127.0.0.1:8080", 8080, true},
		{"[::1]:9000", 9000, true},
		{"[::]:9100", 9100, true},
		// Bound to one external address: not reachable at 127.0.0.1, so
		// probing it would be contacting something that cannot answer.
		{"192.168.1.10:5000", 0, false},
		{"nonsense", 0, false},
	} {
		port, ok := loopbackPort(tc.address)
		if ok != tc.ok || port != tc.port {
			t.Errorf("loopbackPort(%q) = %d, %v; want %d, %v", tc.address, port, ok, tc.port, tc.ok)
		}
	}
}

func TestParseLsofNamesTheProcessBehindEachPort(t *testing.T) {
	// lsof -F output: p<pid>, c<command>, n<address>.
	output := strings.Join([]string{
		"p101", "cnode", "n*:3000", "n127.0.0.1:3001",
		"p202", "cPython", "n[::1]:8000",
		"p303", "cprivate", "n10.0.0.5:9999",
	}, "\n")

	listeners := parseLsof(output)
	if len(listeners) != 3 {
		t.Fatalf("got %+v", listeners)
	}
	if listeners[0].Port != 3000 || listeners[0].Process != "node" {
		t.Errorf("first = %+v", listeners[0])
	}
	if listeners[2].Port != 8000 || listeners[2].Process != "Python" {
		t.Errorf("third = %+v", listeners[2])
	}
	for _, listener := range listeners {
		if listener.Port == 9999 {
			t.Errorf("a socket bound to an external address was listed: %+v", listener)
		}
	}
}

// The probe must accept only something that actually answers the MCP
// handshake. A port being open is not evidence of anything.
func TestHandshakeRefusesSomethingThatIsNotAnMCPServer(t *testing.T) {
	notMCP := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{"hello": "I am a web service"})
	}))
	defer notMCP.Close()

	if _, ok := mcpHandshake(context.Background(), notMCP.URL+"/mcp"); ok {
		t.Error("a plain web service was reported as an MCP server")
	}
}

func TestHandshakeAcceptsARealServerAndCountsItsTools(t *testing.T) {
	url := startTestMCPServer(t)

	tools, ok := mcpHandshake(context.Background(), url)
	if !ok {
		t.Fatalf("a real MCP server was not recognised at %s", url)
	}
	if tools != 1 {
		t.Errorf("tools = %d, want 1", tools)
	}
}

// startTestMCPServer runs a real MCP server over streamable HTTP, so the probe
// is proved against the protocol rather than against a stub of it.
func startTestMCPServer(t *testing.T) string {
	t.Helper()
	server := sdk.NewServer(&sdk.Implementation{Name: "local", Version: "1.0.0"}, nil)
	sdk.AddTool(server, &sdk.Tool{
		Name:        "echo",
		Description: "returns what it is given",
	}, func(ctx context.Context, req *sdk.CallToolRequest, args map[string]any) (*sdk.CallToolResult, any, error) {
		return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "ok"}}}, nil, nil
	})

	mux := http.NewServeMux()
	mux.Handle("/mcp", sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil))
	httpServer := httptest.NewServer(mux)
	t.Cleanup(httpServer.Close)
	return httpServer.URL + "/mcp"
}

// TestProbeLoopbackFindsARunningServer is the whole probe, end to end: a real
// MCP server on a real port, found by enumerating this machine's listening
// sockets and handshaking with them. Everything else about the probe can pass
// while it finds nothing.
func TestProbeLoopbackFindsARunningServer(t *testing.T) {
	if testing.Short() {
		t.Skip("contacts every listening port on this machine")
	}
	url := startTestMCPServer(t)

	found, err := ProbeLoopback(t.Context())
	if err != nil {
		t.Skipf("this machine cannot enumerate listening sockets: %v", err)
	}

	for _, server := range found {
		if server.Runs != url {
			continue
		}
		if server.Origin != OriginListening {
			t.Errorf("origin = %q", server.Origin)
		}
		if server.Spec == nil || server.Spec.URL != url {
			t.Errorf("spec = %+v", server.Spec)
		}
		if !strings.Contains(strings.Join(server.Notes, " "), "1 tools") {
			t.Errorf("notes do not report what it offers: %v", server.Notes)
		}
		if len(server.Sources) != 1 || !strings.Contains(server.Sources[0], "listening on port") {
			t.Errorf("sources = %v", server.Sources)
		}
		return
	}
	t.Fatalf("the running server at %s was not found; probe returned %+v", url, found)
}

// One server registered in several applications is one server. Listing it twice
// invites adding it twice, and the second add collides with the first on its
// name.
func TestDiscoverMergesOneServerRegisteredTwice(t *testing.T) {
	home := t.TempDir()
	claude := writeClientConfig(t, home, "Claude Code", `{
	  "mcpServers": {"agora-memory": {"command": "/Users/x/.local/bin/agora-memory-mcp"}}
	}`)
	cursor := writeClientConfig(t, home, "Cursor", `{
	  "mcpServers": {"agora-memory": {"command": "/Users/x/.local/bin/agora-memory-mcp"}}
	}`)

	found, _ := DiscoverConfigured(home)
	if len(found) != 1 {
		t.Fatalf("got %d entries, want one merged: %+v", len(found), found)
	}

	// Both applications are named, and both files: which two is what tells a
	// user they are looking at their own setup rather than a duplicate.
	if len(found[0].Sources) != 2 {
		t.Fatalf("sources = %v", found[0].Sources)
	}
	if !slices.Contains(found[0].Sources, "Claude Code") || !slices.Contains(found[0].Sources, "Cursor") {
		t.Errorf("sources = %v", found[0].Sources)
	}
	if !slices.Contains(found[0].Paths, claude) || !slices.Contains(found[0].Paths, cursor) {
		t.Errorf("paths = %v", found[0].Paths)
	}
}

// Two registrations that would write different configurations are different
// servers, however alike their names. Merging them would hide one of them.
func TestDiscoverKeepsRegistrationsThatDiffer(t *testing.T) {
	home := t.TempDir()
	writeClientConfig(t, home, "Claude Code", `{
	  "mcpServers": {"files": {"command": "uvx", "args": ["mcp-server-files", "--root", "/a"]}}
	}`)
	writeClientConfig(t, home, "Cursor", `{
	  "mcpServers": {"files": {"command": "uvx", "args": ["mcp-server-files", "--root", "/b"]}}
	}`)

	found, _ := DiscoverConfigured(home)
	if len(found) != 2 {
		t.Fatalf("got %d entries, want both: %+v", len(found), found)
	}
}
