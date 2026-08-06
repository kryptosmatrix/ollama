package mcp

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// registryFixture serves the recorded registry responses. The tests never touch
// the live registry: a suite that depends on a third party's uptime and current
// contents is a suite that fails for reasons unrelated to this code.
func registryFixture(t *testing.T) (*RegistryClient, *[]string) {
	t.Helper()

	var requested []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requested = append(requested, r.URL.String())

		name := "page1.json"
		switch r.URL.Query().Get("cursor") {
		case "cursor-two":
			name = "page2.json"
		case "malformed":
			name = "malformed.json"
		case "missing":
			http.Error(w, "gone", http.StatusNotFound)
			return
		}

		data, err := os.ReadFile(filepath.Join("testdata", "registry", name))
		if err != nil {
			t.Errorf("read fixture: %v", err)
			http.Error(w, "fixture", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	}))
	t.Cleanup(server.Close)

	client := NewRegistryClient(server.URL)
	client.HTTP = server.Client()
	return client, &requested
}

func TestRegistrySearchReadsAPage(t *testing.T) {
	client, requested := registryFixture(t)

	page, err := client.Search(t.Context(), "weather", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Servers) != 6 {
		t.Fatalf("got %d servers", len(page.Servers))
	}
	if page.NextCursor != "cursor-two" {
		t.Errorf("NextCursor = %q", page.NextCursor)
	}

	first := page.Servers[0]
	if first.Name != "io.github.example/weather" || first.Title != "Weather" {
		t.Errorf("first entry = %+v", first)
	}
	if first.Repository == nil || first.Repository.URL == "" {
		t.Error("the repository must survive, it is the only provenance on offer")
	}

	if len(*requested) != 1 {
		t.Fatalf("requests = %v", *requested)
	}
	query := (*requested)[0]
	for _, want := range []string{"/v0/servers?", "search=weather", "limit=30", "version=latest"} {
		if !strings.Contains(query, want) {
			t.Errorf("request %q should contain %q", query, want)
		}
	}
}

func TestRegistrySearchPaginates(t *testing.T) {
	client, requested := registryFixture(t)

	page, err := client.Search(t.Context(), "", "cursor-two")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}
	if len(page.Servers) != 1 || page.Servers[0].Name != "io.github.example/last" {
		t.Fatalf("page = %+v", page.Servers)
	}
	if page.NextCursor != "" {
		t.Errorf("the last page must report no cursor, got %q", page.NextCursor)
	}
	if !strings.Contains((*requested)[0], "cursor=cursor-two") {
		t.Errorf("the cursor was not sent: %q", (*requested)[0])
	}
	if strings.Contains((*requested)[0], "search=") {
		t.Errorf("an empty query should not be sent: %q", (*requested)[0])
	}
}

func TestRegistrySearchReportsFailures(t *testing.T) {
	client, _ := registryFixture(t)

	t.Run("a malformed response", func(t *testing.T) {
		if _, err := client.Search(t.Context(), "", "malformed"); err == nil {
			t.Fatal("expected an error")
		}
	})

	t.Run("an error status", func(t *testing.T) {
		_, err := client.Search(t.Context(), "", "missing")
		if err == nil {
			t.Fatal("expected an error")
		}
		if !strings.Contains(err.Error(), "404") {
			t.Errorf("error = %v, want it to name the status", err)
		}
	})

	t.Run("an unreachable registry", func(t *testing.T) {
		offline := NewRegistryClient("http://127.0.0.1:1")
		if _, err := offline.Search(t.Context(), "", ""); err == nil {
			t.Fatal("expected an error")
		}
	})
}

func TestPublisher(t *testing.T) {
	cases := map[string]string{
		"io.github.example/weather": "io.github.example",
		"ac.inference.sh/mcp":       "ac.inference.sh",
		"bare":                      "bare",
	}
	for name, want := range cases {
		if got := (RegistryEntry{Name: name}).Publisher(); got != want {
			t.Errorf("Publisher(%q) = %q, want %q", name, got, want)
		}
	}
}

// TestResolveProducesTheExactCommandLine is the heart of the install gate. The
// user approves what Summary() renders, so what Resolve builds is what they are
// agreeing to run.
func TestResolveProducesTheExactCommandLine(t *testing.T) {
	client, _ := registryFixture(t)
	page, err := client.Search(t.Context(), "", "")
	if err != nil {
		t.Fatalf("Search: %v", err)
	}

	byName := map[string]RegistryEntry{}
	for _, entry := range page.Servers {
		byName[entry.Name] = entry
	}

	cases := []struct {
		entry   string
		summary string
	}{
		{"io.github.example/weather", "npx -y @example/weather-mcp@1.2.0"},
		{"io.github.example/pythonic", "uvx example-mcp==0.3.0 --read-only"},
		{"io.github.example/containerised", "docker run --rm -i --network none ghcr.io/example/mcp:latest"},
		{"ac.inference.sh/mcp", "https://mcp.inference.sh/v1"},
	}
	for _, tc := range cases {
		t.Run(tc.entry, func(t *testing.T) {
			spec, err := byName[tc.entry].Resolve()
			if err != nil {
				t.Fatalf("Resolve: %v", err)
			}
			if got := spec.Summary(); got != tc.summary {
				t.Errorf("Summary() = %q, want %q", got, tc.summary)
			}
		})
	}
}

func TestResolvePrefersAHostedEndpoint(t *testing.T) {
	// Running nothing on the user's machine is safer than running something,
	// so a publisher offering both gets the remote.
	entry := RegistryEntry{
		Name:     "io.github.example/both",
		Remotes:  []RegistryRemote{{Type: "streamable-http", URL: "https://mcp.example.com/v1"}},
		Packages: []RegistryPackage{{RegistryType: "npm", Identifier: "@example/thing"}},
	}
	spec, err := entry.Resolve()
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if spec.transport() != TransportHTTP {
		t.Errorf("transport = %q, want http", spec.transport())
	}
}

func TestResolveRefusesWhatItCannotBuild(t *testing.T) {
	client, _ := registryFixture(t)
	page, _ := client.Search(t.Context(), "", "")
	byName := map[string]RegistryEntry{}
	for _, entry := range page.Servers {
		byName[entry.Name] = entry
	}

	for _, name := range []string{"io.github.example/exotic", "io.github.example/empty"} {
		t.Run(name, func(t *testing.T) {
			spec, err := byName[name].Resolve()
			if err == nil {
				t.Fatalf("expected a refusal, got %+v", spec)
			}
			if !errors.Is(err, ErrUnresolvable) {
				t.Errorf("error = %v, want ErrUnresolvable", err)
			}
		})
	}
}

// TestResolveNeverWritesASecret is the rule that keeps a registry entry from
// putting a credential into the configuration file. The registry says what a
// server needs; the value comes from the user's environment at connect time.
func TestResolveNeverWritesASecret(t *testing.T) {
	client, _ := registryFixture(t)
	page, _ := client.Search(t.Context(), "", "")

	for _, entry := range page.Servers {
		spec, err := entry.Resolve()
		if err != nil {
			continue
		}
		for name, value := range spec.Env {
			if !isEnvRef(value) {
				t.Errorf("%s: env %q = %q, want an ${env:NAME} reference", entry.Name, name, value)
			}
		}
		for name, value := range spec.Headers {
			if !isEnvRef(value) {
				t.Errorf("%s: header %q = %q, want an ${env:NAME} reference", entry.Name, name, value)
			}
		}
	}
}

// TestResolvedSpecsPassTheSameValidationAsTypedOnes proves a registry entry
// cannot smuggle in something the configuration layer would refuse from a
// human. Whatever the registry says, the result has to survive the same checks.
func TestResolvedSpecsPassTheSameValidationAsTypedOnes(t *testing.T) {
	client, _ := registryFixture(t)
	page, _ := client.Search(t.Context(), "", "")

	cfg := &Config{}
	var added int
	for _, entry := range page.Servers {
		spec, err := entry.Resolve()
		if err != nil {
			continue
		}
		name := strings.ReplaceAll(entry.Publisher(), ".", "-")
		cfg.Set(name, spec)
		added++
	}
	if added == 0 {
		t.Fatal("nothing resolved, so nothing was validated")
	}
	if problems := cfg.Problems(); len(problems) != 0 {
		t.Errorf("resolved specs failed validation: %v", problems)
	}
}

func TestEnvReferenceNameIsUsable(t *testing.T) {
	cases := map[string]string{
		"WEATHER_API_KEY": "WEATHER_API_KEY",
		"Authorization":   "AUTHORIZATION",
		"x-api-key":       "X_API_KEY",
		"1password":       "MCP_1PASSWORD",
		"":                "MCP_",
	}
	for input, want := range cases {
		if got := envReferenceName(input); got != want {
			t.Errorf("envReferenceName(%q) = %q, want %q", input, got, want)
		}
	}
}

func TestNewRegistryClientDefaultsToTheOfficialRegistry(t *testing.T) {
	if got := NewRegistryClient("").BaseURL; got != DefaultRegistryURL {
		t.Errorf("BaseURL = %q, want %q", got, DefaultRegistryURL)
	}
	if got := NewRegistryClient("https://mirror.example.com/").BaseURL; got != "https://mirror.example.com" {
		t.Errorf("BaseURL = %q, want the trailing slash trimmed", got)
	}
}
