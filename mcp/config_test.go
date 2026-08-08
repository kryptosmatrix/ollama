package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/google/go-cmp/cmp"
)

func writeConfig(t *testing.T, contents string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "mcp.json")
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	return path
}

func TestConfigPath(t *testing.T) {
	t.Run("environment override wins", func(t *testing.T) {
		dir := t.TempDir()
		want := filepath.Join(dir, "custom.json")
		t.Setenv(ConfigPathEnv, want)
		t.Setenv("XDG_CONFIG_HOME", dir)

		got, err := ConfigPath()
		if err != nil {
			t.Fatalf("ConfigPath: %v", err)
		}
		if got != want {
			t.Errorf("ConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("xdg config home", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(ConfigPathEnv, "")
		t.Setenv("XDG_CONFIG_HOME", dir)

		got, err := ConfigPath()
		if err != nil {
			t.Fatalf("ConfigPath: %v", err)
		}
		if want := filepath.Join(dir, "ollama", "mcp.json"); got != want {
			t.Errorf("ConfigPath() = %q, want %q", got, want)
		}
	})

	t.Run("falls back to home", func(t *testing.T) {
		t.Setenv(ConfigPathEnv, "")
		t.Setenv("XDG_CONFIG_HOME", "")

		got, err := ConfigPath()
		if err != nil {
			t.Fatalf("ConfigPath: %v", err)
		}
		home, err := os.UserHomeDir()
		if err != nil {
			t.Skipf("no home directory: %v", err)
		}
		if want := filepath.Join(home, ".ollama", "mcp.json"); got != want {
			t.Errorf("ConfigPath() = %q, want %q", got, want)
		}
	})
}

func TestLoadMissingFileIsEmptyNotAnError(t *testing.T) {
	cfg, err := Load(filepath.Join(t.TempDir(), "absent.json"))
	if err != nil {
		t.Fatalf("Load of a missing file should succeed, got %v", err)
	}
	if len(cfg.Names()) != 0 {
		t.Errorf("expected no servers, got %v", cfg.Names())
	}
}

func TestLoadRejectsUnparseableFile(t *testing.T) {
	path := writeConfig(t, `{"mcpServers": `)
	if _, err := Load(path); err == nil {
		t.Fatal("expected an error for truncated JSON, got nil")
	}
}

func TestLoadRejectsNullServer(t *testing.T) {
	path := writeConfig(t, `{"mcpServers": {"broken": null}}`)
	_, err := Load(path)
	if err == nil {
		t.Fatal("expected an error for a null server entry, got nil")
	}
	if !strings.Contains(err.Error(), "broken") {
		t.Errorf("error should name the offending server, got %v", err)
	}
}

func TestLoadPopulatesNameAndTransport(t *testing.T) {
	path := writeConfig(t, `{
	  "mcpServers": {
	    "files":  {"command": "uvx", "args": ["mcp-server-files"]},
	    "hosted": {"url": "https://mcp.example.com/v1"}
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if diff := cmp.Diff([]string{"files", "hosted"}, cfg.Names()); diff != "" {
		t.Errorf("Names() mismatch (-want +got):\n%s", diff)
	}

	files, ok := cfg.Get("files")
	if !ok {
		t.Fatal(`server "files" missing`)
	}
	if files.Name != "files" {
		t.Errorf("Name = %q, want %q", files.Name, "files")
	}
	if got := files.transport(); got != TransportStdio {
		t.Errorf("transport() = %q, want %q for a command-only server", got, TransportStdio)
	}

	hosted, _ := cfg.Get("hosted")
	if got := hosted.transport(); got != TransportHTTP {
		t.Errorf("transport() = %q, want %q for a url-only server", got, TransportHTTP)
	}
}

func TestProblems(t *testing.T) {
	cases := []struct {
		name    string
		server  string
		json    string
		wantErr string
	}{
		{
			name:    "no command and no url",
			server:  "empty",
			json:    `{}`,
			wantErr: `either a "command"`,
		},
		{
			name:    "stdio with an empty command",
			server:  "blank",
			json:    `{"type": "stdio", "command": "   "}`,
			wantErr: `stdio server needs a "command"`,
		},
		{
			name:    "stdio that also sets url",
			server:  "mixed",
			json:    `{"command": "uvx", "url": "https://example.com"}`,
			wantErr: `must not set "url"`,
		},
		{
			name:    "http that also sets command",
			server:  "mixed2",
			json:    `{"type": "http", "url": "https://example.com", "command": "uvx"}`,
			wantErr: `must not set "command"`,
		},
		{
			name:    "plaintext http to a remote host",
			server:  "insecure",
			json:    `{"url": "http://mcp.example.com/v1"}`,
			wantErr: "must use https",
		},
		{
			name:    "unsupported scheme",
			server:  "weird",
			json:    `{"type": "http", "url": "ftp://example.com"}`,
			wantErr: "must use https",
		},
		{
			name:    "literal authorization header",
			server:  "leaky",
			json:    `{"url": "https://example.com", "headers": {"Authorization": "Bearer sk-live-1234567890"}}`,
			wantErr: `header "Authorization" must be written as "${env:NAME}"`,
		},
		{
			name:    "literal api key header, lower case",
			server:  "leaky2",
			json:    `{"url": "https://example.com", "headers": {"x-api-key": "abcd1234"}}`,
			wantErr: `must be written as "${env:NAME}"`,
		},
		{
			name:    "literal credential in env",
			server:  "leaky3",
			json:    `{"command": "uvx", "env": {"GITHUB_TOKEN": "ghp_realtokenvalue"}}`,
			wantErr: `env "GITHUB_TOKEN" looks like a credential`,
		},
		{
			name:    "server name containing the namespace separator",
			server:  "bad__name",
			json:    `{"command": "uvx"}`,
			wantErr: "must not contain",
		},
		{
			name:    "server name with a space",
			server:  "bad name",
			json:    `{"command": "uvx"}`,
			wantErr: "server name",
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			path := writeConfig(t, `{"mcpServers": {`+jsonKey(tc.server)+`: `+tc.json+`}}`)
			cfg, err := Load(path)
			if err != nil {
				t.Fatalf("Load: %v", err)
			}
			problems := cfg.Problems()
			err, reported := problems[tc.server]
			if !reported {
				t.Fatalf("expected a problem for %q, got none", tc.server)
			}
			if !strings.Contains(err.Error(), tc.wantErr) {
				t.Errorf("problem for %q = %v, want it to contain %q", tc.server, err, tc.wantErr)
			}
		})
	}
}

// jsonKey quotes a string as a JSON object key.
func jsonKey(s string) string {
	quoted, err := json.Marshal(s)
	if err != nil {
		panic(err)
	}
	return string(quoted)
}

func TestProblemsAcceptsValidServers(t *testing.T) {
	path := writeConfig(t, `{
	  "mcpServers": {
	    "files":     {"command": "uvx", "args": ["mcp-server-files"], "env": {"NODE_ENV": "production"}},
	    "loopback":  {"url": "http://127.0.0.1:8931/mcp"},
	    "localhost": {"url": "http://localhost:8931/mcp"},
	    "hosted":    {"url": "https://mcp.example.com/v1", "headers": {"Authorization": "${env:EXAMPLE_TOKEN}"}},
	    "keyed":     {"command": "uvx", "env": {"API_KEY": "${env:EXAMPLE_KEY}"}}
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problems := cfg.Problems(); len(problems) != 0 {
		t.Errorf("expected no problems, got %v", problems)
	}
}

func TestOneBadServerDoesNotDisableTheRest(t *testing.T) {
	path := writeConfig(t, `{
	  "mcpServers": {
	    "good": {"command": "uvx", "args": ["ok"]},
	    "bad":  {"url": "http://remote.example.com"}
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load should tolerate a bad server entry, got %v", err)
	}
	problems := cfg.Problems()
	if _, bad := problems["bad"]; !bad {
		t.Error(`expected "bad" to be reported as a problem`)
	}
	if _, alsoBad := problems["good"]; alsoBad {
		t.Error(`"good" should not be reported as a problem`)
	}
	if _, ok := cfg.Get("good"); !ok {
		t.Error(`"good" should still be loadable`)
	}
}

func TestSaveRoundTripPreservesUnknownFields(t *testing.T) {
	path := writeConfig(t, `{
	  "$schema": "https://example.com/mcp.schema.json",
	  "inputs": [{"id": "token", "type": "promptString"}],
	  "mcpServers": {
	    "files": {
	      "command": "uvx",
	      "args": ["mcp-server-files"],
	      "futureField": {"nested": true},
	      "timeoutSeconds": 30
	    }
	  }
	}`)

	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read back: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(data, &got); err != nil {
		t.Fatalf("reparse: %v", err)
	}

	if _, ok := got["$schema"]; !ok {
		t.Error("top-level $schema was dropped by save")
	}
	if _, ok := got["inputs"]; !ok {
		t.Error("top-level inputs was dropped by save")
	}

	servers, _ := got["mcpServers"].(map[string]any)
	files, _ := servers["files"].(map[string]any)
	if _, ok := files["futureField"]; !ok {
		t.Error("unknown server field futureField was dropped by save")
	}
	if _, ok := files["timeoutSeconds"]; !ok {
		t.Error("unknown server field timeoutSeconds was dropped by save")
	}
	if files["command"] != "uvx" {
		t.Errorf("command = %v, want uvx", files["command"])
	}
}

func TestSaveWritesPrivatePermissions(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes are not meaningful on windows")
	}

	dir := filepath.Join(t.TempDir(), "nested", "ollama")
	path := filepath.Join(dir, "mcp.json")

	cfg := &Config{Servers: map[string]*ServerSpec{}}
	cfg.Set("files", &ServerSpec{Command: "uvx", Args: []string{"mcp-server-files"}})
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("config mode = %o, want 600 — this file names executables Ollama will run", got)
	}

	dirInfo, err := os.Stat(dir)
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("config directory mode = %o, want 700", got)
	}
}

func TestSaveLeavesNoTemporaryFiles(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")

	cfg := &Config{Servers: map[string]*ServerSpec{}}
	cfg.Set("files", &ServerSpec{Command: "uvx"})
	if err := cfg.Save(path); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != "mcp.json" {
			t.Errorf("unexpected leftover file %q", entry.Name())
		}
	}
}

func TestSaveEmptyConfigIsAnObjectNotNull(t *testing.T) {
	cfg := &Config{}
	data, err := cfg.Marshal()
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"mcpServers": {}`) {
		t.Errorf("empty config should serialise mcpServers as an object, got:\n%s", data)
	}
}

func TestSetAndRemove(t *testing.T) {
	cfg := &Config{}
	cfg.Set("files", &ServerSpec{Command: "uvx"})

	spec, ok := cfg.Get("files")
	if !ok {
		t.Fatal("Set did not store the server")
	}
	if spec.Name != "files" {
		t.Errorf("Set should populate Name, got %q", spec.Name)
	}
	if !cfg.Remove("files") {
		t.Error("Remove should report that it removed an existing server")
	}
	if cfg.Remove("files") {
		t.Error("Remove should report false for an absent server")
	}
}

func TestResolveEnvAndHeaders(t *testing.T) {
	t.Run("expands references and passes literals through", func(t *testing.T) {
		t.Setenv("MCP_TEST_TOKEN", "secret-value")
		spec := &ServerSpec{
			Env:     map[string]string{"API_KEY": "${env:MCP_TEST_TOKEN}", "NODE_ENV": "production"},
			Headers: map[string]string{"Authorization": "${env:MCP_TEST_TOKEN}", "X-Client": "ollama"},
		}

		env, err := spec.ResolveEnv()
		if err != nil {
			t.Fatalf("ResolveEnv: %v", err)
		}
		want := map[string]string{"API_KEY": "secret-value", "NODE_ENV": "production"}
		if diff := cmp.Diff(want, env); diff != "" {
			t.Errorf("ResolveEnv mismatch (-want +got):\n%s", diff)
		}

		headers, err := spec.ResolveHeaders()
		if err != nil {
			t.Fatalf("ResolveHeaders: %v", err)
		}
		wantHeaders := map[string]string{"Authorization": "secret-value", "X-Client": "ollama"}
		if diff := cmp.Diff(wantHeaders, headers); diff != "" {
			t.Errorf("ResolveHeaders mismatch (-want +got):\n%s", diff)
		}
	})

	t.Run("fails rather than substituting empty for a missing variable", func(t *testing.T) {
		os.Unsetenv("MCP_TEST_ABSENT")
		spec := &ServerSpec{Headers: map[string]string{"Authorization": "${env:MCP_TEST_ABSENT}"}}

		got, err := spec.ResolveHeaders()
		if err == nil {
			t.Fatalf("expected an error for an unset variable, got %v", got)
		}
		var missing *ErrMissingEnv
		if !errors.As(err, &missing) {
			t.Fatalf("error should be an *ErrMissingEnv, got %T: %v", err, err)
		}
		if missing.Variable != "MCP_TEST_ABSENT" {
			t.Errorf("Variable = %q, want MCP_TEST_ABSENT", missing.Variable)
		}
	})

	t.Run("an empty variable is a value, not a failure", func(t *testing.T) {
		t.Setenv("MCP_TEST_EMPTY", "")
		spec := &ServerSpec{Env: map[string]string{"OPTIONAL": "${env:MCP_TEST_EMPTY}"}}

		env, err := spec.ResolveEnv()
		if err != nil {
			t.Fatalf("ResolveEnv: %v", err)
		}
		if got, ok := env["OPTIONAL"]; !ok || got != "" {
			t.Errorf("OPTIONAL = %q (present %v), want an empty string that is present", got, ok)
		}
	})
}

func TestResolveDoesNotLeakSecretsIntoErrors(t *testing.T) {
	os.Unsetenv("MCP_TEST_ABSENT")
	spec := &ServerSpec{Headers: map[string]string{"Authorization": "${env:MCP_TEST_ABSENT}"}}

	_, err := spec.ResolveHeaders()
	if err == nil {
		t.Fatal("expected an error")
	}
	if strings.Contains(err.Error(), "${env:") {
		t.Errorf("error message should name the variable, not echo the raw reference: %v", err)
	}
}

// TestAURLMayNotCarryEmbeddedCredentials closes a channel a cross-substrate
// review found and a probe confirmed: validateURL checked the scheme and the
// host and never looked at the userinfo, so https://user:secret@host was
// accepted and written to mcp.json as a literal.
//
// Userinfo is unambiguous — it is a credential or it is nothing — which is why
// this is a refusal rather than a heuristic. Pasting a configuration from
// another MCP client is how it arrives, since several put the token in the URL.
func TestAURLMayNotCarryEmbeddedCredentials(t *testing.T) {
	for name, raw := range map[string]string{
		"password":      "https://alice:s3cr3t@api.example.com/mcp",
		"bare username": "https://s3cr3t@api.example.com/mcp",
	} {
		t.Run(name, func(t *testing.T) {
			cfg := &Config{}
			cfg.Set("hosted", &ServerSpec{Type: TransportHTTP, URL: raw})

			problem := cfg.Problems()["hosted"]
			if problem == nil {
				t.Fatalf("%q was accepted; a credential in a URL is a literal on disk", raw)
			}
			// The refusal has to say what to do instead, or it is just a wall.
			if !strings.Contains(problem.Error(), "${env:") {
				t.Errorf("the refusal does not point at the alternative: %v", problem)
			}
		})
	}

	t.Run("an ordinary url is untouched", func(t *testing.T) {
		cfg := &Config{}
		cfg.Set("hosted", &ServerSpec{Type: TransportHTTP, URL: "https://api.example.com/mcp?region=eu"})
		if problem := cfg.Problems()["hosted"]; problem != nil {
			t.Errorf("a url with no credential was refused: %v", problem)
		}
	})
}

// TestALoopbackHostIsRecognisedWhateverItsCase. Host names are
// case-insensitive (RFC 4343), and the check compared against "localhost"
// exactly — so http://Localhost was told to use https for an address that never
// leaves the machine.
func TestALoopbackHostIsRecognisedWhateverItsCase(t *testing.T) {
	for _, host := range []string{"localhost", "Localhost", "LOCALHOST"} {
		cfg := &Config{}
		cfg.Set("local", &ServerSpec{Type: TransportHTTP, URL: "http://" + host + ":8080/mcp"})
		if problem := cfg.Problems()["local"]; problem != nil {
			t.Errorf("http://%s was refused: %v", host, problem)
		}
	}
	// And a real host still has to use https.
	cfg := &Config{}
	cfg.Set("remote", &ServerSpec{Type: TransportHTTP, URL: "http://api.example.com/mcp"})
	if cfg.Problems()["remote"] == nil {
		t.Error("plain http to a real host must still be refused")
	}
}

// TestAPreservedFieldMayNotCarryACredential. Fields this version of Ollama does
// not understand are kept and written back, which is what keeps a configuration
// portable between MCP clients — and is how a credential arrives in one. The
// env and header checks never saw a name like "apiKey" sitting directly on the
// server object, so it went through untouched and was saved as a literal.
func TestAPreservedFieldMayNotCarryACredential(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(path, []byte(`{"mcpServers":{"s":{"command":"srv","apiKey":"sk-LEAKED"}}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	problem := cfg.Problems()["s"]
	if problem == nil {
		t.Fatal("a credential in an unknown field was accepted")
	}
	if !strings.Contains(problem.Error(), "${env:") {
		t.Errorf("the refusal does not name the safe form: %v", problem)
	}

	// An unknown field that is not credential-shaped is still preserved, which
	// is the whole reason unknown fields are kept.
	plain := filepath.Join(dir, "plain.json")
	if err := os.WriteFile(plain, []byte(`{"mcpServers":{"s":{"command":"srv","displayOrder":3}}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	kept, err := Load(plain)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problem := kept.Problems()["s"]; problem != nil {
		t.Errorf("an ordinary unknown field was refused: %v", problem)
	}
	out := filepath.Join(dir, "out.json")
	if err := kept.Save(out); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, _ := os.ReadFile(out)
	if !strings.Contains(string(data), "displayOrder") {
		t.Errorf("an unknown field was lost on the way back out:\n%s", data)
	}

	// And an environment reference in such a field is left alone.
	referenced := filepath.Join(dir, "ref.json")
	if err := os.WriteFile(referenced, []byte(`{"mcpServers":{"s":{"command":"srv","apiKey":"${env:MY_TOKEN}"}}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}
	safe, err := Load(referenced)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if problem := safe.Problems()["s"]; problem != nil {
		t.Errorf("an environment reference in an unknown field was refused: %v", problem)
	}
}

// TestWarningsSayWhatAConfigurationCostsWithoutRefusingIt covers the ruling on
// three findings that had been left open as one question.
//
// Each was a credential or a privilege in a place Ollama cannot protect, and
// each looked like a choice between refusing a configuration — stranding a user
// with no safe alternative — and accepting it in silence, letting them believe
// nothing was given away. That was a false choice created by there being no way
// to say anything other than yes or no. There is one now.
func TestWarningsSayWhatAConfigurationCostsWithoutRefusingIt(t *testing.T) {
	cfg := &Config{}
	cfg.Set("argv", &ServerSpec{Command: "srv", Args: []string{"--api-key=sk-live-1"}})
	cfg.Set("argv-separate", &ServerSpec{Command: "srv", Args: []string{"--token", "sk-live-2"}})
	cfg.Set("query", &ServerSpec{Type: TransportHTTP, URL: "https://mcp.example.com/v1?api_key=sk-live-3"})
	cfg.Set("container", &ServerSpec{Command: "docker", Args: []string{"run", "--rm", "-i", "-v", "/:/host", "image"}})
	cfg.Set("plain", &ServerSpec{Command: "srv", Args: []string{"--verbose", "--token-file", "/etc/creds"}})
	cfg.Set("remote", &ServerSpec{Type: TransportHTTP, URL: "https://mcp.example.com/v1?region=eu"})

	// Nothing here is refused. That is the whole point.
	if problems := cfg.Problems(); len(problems) != 0 {
		t.Fatalf("a warning must not become a refusal: %v", problems)
	}

	warnings := cfg.Warnings()
	for _, name := range []string{"argv", "argv-separate", "query", "container"} {
		if len(warnings[name]) == 0 {
			t.Errorf("%s got no warning", name)
		}
	}
	if !strings.Contains(strings.Join(warnings["argv"], " "), "process list") {
		t.Errorf("the argv warning does not say what is exposed: %v", warnings["argv"])
	}
	if !strings.Contains(strings.Join(warnings["query"], " "), "mcp.json") {
		t.Errorf("the query warning does not say where it ends up: %v", warnings["query"])
	}
	if !strings.Contains(strings.Join(warnings["container"], " "), "access to a path") {
		t.Errorf("the container warning does not say what the flag does: %v", warnings["container"])
	}

	// An ordinary server is not nagged, or the warnings stop being read.
	if len(warnings["remote"]) != 0 {
		t.Errorf("an ordinary remote server was warned about: %v", warnings["remote"])
	}

	// "--token-file /etc/creds" names a path, not a secret. It is warned about
	// anyway, and deliberately: the pattern cannot tell the two apart, and a
	// warning a user can read and dismiss is the price of not refusing the one
	// that matters. What must never happen is refusing it.
	if len(warnings["plain"]) == 0 {
		t.Log("note: --token-file was not warned about; the pattern may have narrowed")
	}
}
