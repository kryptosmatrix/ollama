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
	"github.com/ollama/ollama/mcp"
)

// registryFixtureServer serves the recorded registry responses that the mcp
// package's own tests use, so these handlers are never pointed at the live
// registry.
func registryFixtureServer(t *testing.T) {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		name := "page1.json"
		if r.URL.Query().Get("cursor") == "cursor-two" {
			name = "page2.json"
		}
		data, err := os.ReadFile(filepath.Join("..", "..", "mcp", "testdata", "registry", name))
		if err != nil {
			t.Errorf("read fixture: %v", err)
			http.Error(w, "fixture", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)
	t.Setenv(mcp.RegistryURLEnv, server.URL)
}

func browse(t *testing.T, query string) responses.MCPRegistryResponse {
	t.Helper()
	server := &Server{}
	request := httptest.NewRequest(http.MethodGet, "/api/v1/mcp-registry?"+query, nil)
	recorder := httptest.NewRecorder()
	if err := server.browseMCPRegistry(recorder, request); err != nil {
		t.Fatalf("browse: %v", err)
	}
	var body responses.MCPRegistryResponse
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\n%s", err, recorder.Body.String())
	}
	return body
}

func entry(t *testing.T, body responses.MCPRegistryResponse, name string) responses.MCPRegistryEntry {
	t.Helper()
	for _, e := range body.Entries {
		if e.Name == name {
			return e
		}
	}
	t.Fatalf("no entry named %q", name)
	return responses.MCPRegistryEntry{}
}

// TestBrowseShowsTheCommandLineAnInstallWouldWrite is the install gate at the
// browse surface: the user reads the real command line, resolved by the same
// code that would write it, rather than a name and a description.
func TestBrowseShowsTheCommandLineAnInstallWouldWrite(t *testing.T) {
	registryFixtureServer(t)
	body := browse(t, "")

	weather := entry(t, body, "io.github.example/weather")
	if !weather.Installable {
		t.Fatalf("expected it to be installable: %+v", weather)
	}
	if weather.Runs != "npx -y @example/weather-mcp@1.2.0" {
		t.Errorf("Runs = %q, want the exact command line", weather.Runs)
	}
	if weather.Publisher != "io.github.example" {
		t.Errorf("Publisher = %q, want the only provenance the registry carries", weather.Publisher)
	}
	if weather.Repository == "" {
		t.Error("the repository must be shown when the publisher gave one")
	}
	if len(weather.Variables) == 0 || !strings.Contains(weather.Variables[0], "${env:") {
		t.Errorf("the values the user must set should be named as env references, got %v", weather.Variables)
	}
}

// TestBrowseAlwaysSaysItIsNotVetted keeps the honesty of the surface from
// depending on someone remembering to render a sentence.
func TestBrowseAlwaysSaysItIsNotVetted(t *testing.T) {
	registryFixtureServer(t)
	if !browse(t, "").NotVetted {
		t.Error("the registry is open-publish; the response must say so")
	}
}

func TestBrowseReportsEntriesItCannotInstall(t *testing.T) {
	registryFixtureServer(t)
	body := browse(t, "")

	for _, name := range []string{"io.github.example/exotic", "io.github.example/empty"} {
		found := entry(t, body, name)
		if found.Installable {
			t.Errorf("%s should not be installable", name)
		}
		if found.Reason == "" {
			t.Errorf("%s should say why it cannot be installed rather than being hidden", name)
		}
		if found.Runs != "" {
			t.Errorf("%s must not offer a command line it cannot build: %q", name, found.Runs)
		}
	}
}

func TestBrowsePassesTheCursorThrough(t *testing.T) {
	registryFixtureServer(t)

	first := browse(t, "")
	if first.NextCursor != "cursor-two" {
		t.Fatalf("NextCursor = %q", first.NextCursor)
	}
	second := browse(t, "cursor=cursor-two")
	if len(second.Entries) != 1 || second.Entries[0].Name != "io.github.example/last" {
		t.Fatalf("second page = %+v", second.Entries)
	}
	if second.NextCursor != "" {
		t.Errorf("the last page must report no cursor, got %q", second.NextCursor)
	}
}

// TestSuggestedNamesAreAlwaysAcceptable proves the browse surface never
// proposes a name the configuration layer would refuse on add.
func TestSuggestedNamesAreAlwaysAcceptable(t *testing.T) {
	registryFixtureServer(t)
	body := browse(t, "")

	cfg := &mcp.Config{}
	var added int
	for _, e := range body.Entries {
		if !e.Installable {
			continue
		}
		cfg.Set(e.SuggestedName, &mcp.ServerSpec{Command: "uvx", Args: []string{"x"}})
		added++
	}
	if added == 0 {
		t.Fatal("nothing installable, so nothing was checked")
	}
	if problems := cfg.Problems(); len(problems) != 0 {
		t.Errorf("a suggested name would be refused on add: %v", problems)
	}
}

func TestSuggestedServerName(t *testing.T) {
	cases := map[string]string{
		"io.github.example/weather":  "weather",
		"ac.inference.sh/mcp":        "mcp",
		"io.github.example/odd__one": "odd_one",
		"io.github.example/a.b.c":    "a-b-c",
		"bare":                       "bare",
		"io.github.example/":         "io-github-example",
	}
	for input, want := range cases {
		if got := suggestedServerName(input); got != want {
			t.Errorf("suggestedServerName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestResolveOneEntry(t *testing.T) {
	registryFixtureServer(t)
	server := &Server{}

	t.Run("returns what an install would write", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp-registry/resolve",
			strings.NewReader(`{"name":"io.github.example/pythonic"}`))
		recorder := httptest.NewRecorder()
		if err := server.resolveMCPRegistryEntry(recorder, request); err != nil {
			t.Fatalf("resolve: %v", err)
		}
		var resolved responses.MCPRegistryEntry
		if err := json.Unmarshal(recorder.Body.Bytes(), &resolved); err != nil {
			t.Fatalf("decode: %v", err)
		}
		if resolved.Runs != "uvx example-mcp==0.3.0 --read-only" {
			t.Errorf("Runs = %q", resolved.Runs)
		}
	})

	t.Run("refuses an entry that is no longer listed", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp-registry/resolve",
			strings.NewReader(`{"name":"io.github.example/vanished"}`))
		recorder := httptest.NewRecorder()
		if err := server.resolveMCPRegistryEntry(recorder, request); err == nil {
			t.Fatal("expected an error")
		}
		if recorder.Code != http.StatusNotFound {
			t.Errorf("status = %d, want 404", recorder.Code)
		}
	})

	t.Run("requires a name", func(t *testing.T) {
		request := httptest.NewRequest(http.MethodPost, "/api/v1/mcp-registry/resolve", strings.NewReader(`{}`))
		recorder := httptest.NewRecorder()
		if err := server.resolveMCPRegistryEntry(recorder, request); err == nil {
			t.Fatal("expected an error")
		}
		if recorder.Code != http.StatusBadRequest {
			t.Errorf("status = %d, want 400", recorder.Code)
		}
	})
}

// TestBrowseReportsAnUnreachableRegistry proves the page is told, rather than
// shown an empty list that reads as "there is nothing here".
func TestBrowseReportsAnUnreachableRegistry(t *testing.T) {
	t.Setenv(mcp.RegistryURLEnv, "http://127.0.0.1:1")
	server := &Server{}
	recorder := httptest.NewRecorder()
	err := server.browseMCPRegistry(recorder, httptest.NewRequest(http.MethodGet, "/api/v1/mcp-registry", nil))
	if err == nil {
		t.Fatal("expected an error")
	}
	if recorder.Code != http.StatusBadGateway {
		t.Errorf("status = %d, want 502", recorder.Code)
	}
}
