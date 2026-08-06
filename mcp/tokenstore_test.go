package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func tokenStore(t *testing.T) (*FileTokenStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "mcp-tokens.json")
	return &FileTokenStore{Path: path}, path
}

func sampleToken() *oauth2.Token {
	return &oauth2.Token{
		AccessToken:  "access-abc123",
		TokenType:    "Bearer",
		RefreshToken: "refresh-def456",
		Expiry:       time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
	}
}

func TestTokensPath(t *testing.T) {
	t.Run("environment override wins", func(t *testing.T) {
		want := filepath.Join(t.TempDir(), "custom.json")
		t.Setenv(TokensPathEnv, want)
		got, err := TokensPath()
		if err != nil {
			t.Fatalf("TokensPath: %v", err)
		}
		if got != want {
			t.Errorf("TokensPath() = %q, want %q", got, want)
		}
	})

	t.Run("sits beside the config and the ledger", func(t *testing.T) {
		dir := t.TempDir()
		t.Setenv(TokensPathEnv, "")
		t.Setenv("XDG_CONFIG_HOME", dir)
		got, err := TokensPath()
		if err != nil {
			t.Fatalf("TokensPath: %v", err)
		}
		if want := filepath.Join(dir, "ollama", "mcp-tokens.json"); got != want {
			t.Errorf("TokensPath() = %q, want %q", got, want)
		}
	})
}

func TestTokenRoundTrip(t *testing.T) {
	store, _ := tokenStore(t)
	want := sampleToken()

	if err := store.Save("hosted", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.AccessToken != want.AccessToken {
		t.Errorf("AccessToken = %q", got.AccessToken)
	}
	if got.RefreshToken != want.RefreshToken {
		t.Errorf("the refresh token must survive, or the user is signed out at the first expiry: %q", got.RefreshToken)
	}
	if got.TokenType != want.TokenType {
		t.Errorf("TokenType = %q", got.TokenType)
	}
	if !got.Expiry.Equal(want.Expiry) {
		t.Errorf("Expiry = %v, want %v; a lost expiry means the token is never refreshed", got.Expiry, want.Expiry)
	}
}

func TestTokensArePerServer(t *testing.T) {
	store, _ := tokenStore(t)

	if err := store.Save("one", &oauth2.Token{AccessToken: "first"}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("two", &oauth2.Token{AccessToken: "second"}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	first, err := store.Load("one")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if first.AccessToken != "first" {
		t.Errorf("one = %q; one server's credential must never be handed to another", first.AccessToken)
	}
	second, _ := store.Load("two")
	if second.AccessToken != "second" {
		t.Errorf("two = %q", second.AccessToken)
	}

	servers, err := store.Servers()
	if err != nil {
		t.Fatalf("Servers: %v", err)
	}
	if len(servers) != 2 || servers[0] != "one" || servers[1] != "two" {
		t.Errorf("Servers() = %v, want a stable sorted list", servers)
	}
}

func TestLoadWithoutAToken(t *testing.T) {
	store, _ := tokenStore(t)

	t.Run("no store at all", func(t *testing.T) {
		_, err := store.Load("hosted")
		if !errors.Is(err, ErrNoToken) {
			t.Errorf("err = %v, want ErrNoToken; not being signed in is not a failure to read", err)
		}
	})

	t.Run("a store without that server", func(t *testing.T) {
		if err := store.Save("other", sampleToken()); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
			t.Errorf("err = %v, want ErrNoToken", err)
		}
	})
}

func TestDelete(t *testing.T) {
	store, _ := tokenStore(t)
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if err := store.Delete("hosted"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token survived deletion: %v", err)
	}
	if err := store.Delete("hosted"); err != nil {
		t.Errorf("deleting an absent token is not a failure; the intent is satisfied: %v", err)
	}

	// The secret must be gone from the file, not merely unreachable through
	// the API.
	path, _ := store.path()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(data), "access-abc123") || strings.Contains(string(data), "refresh-def456") {
		t.Error("a deleted token is still in the file")
	}
}

func TestStoreIsPrivate(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix file modes are not meaningful on windows")
	}
	store, path := tokenStore(t)
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Errorf("token store mode = %o, want 600 — file permissions are the only thing protecting these", got)
	}
	dirInfo, err := os.Stat(filepath.Dir(path))
	if err != nil {
		t.Fatalf("stat dir: %v", err)
	}
	if got := dirInfo.Mode().Perm(); got != 0o700 {
		t.Errorf("directory mode = %o, want 700", got)
	}
}

func TestStoreLeavesNoTemporaryFiles(t *testing.T) {
	store, path := tokenStore(t)
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	entries, err := os.ReadDir(filepath.Dir(path))
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, entry := range entries {
		if entry.Name() != filepath.Base(path) {
			t.Errorf("a temporary file was left behind with a token in it: %q", entry.Name())
		}
	}
}

func TestACorruptStoreIsReported(t *testing.T) {
	store, path := tokenStore(t)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := os.WriteFile(path, []byte(`{"tokens": `), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	_, err := store.Load("hosted")
	if err == nil || errors.Is(err, ErrNoToken) {
		t.Fatalf("err = %v; a store that cannot be read must not read as 'not signed in', which would sign the user out of everything and then overwrite it", err)
	}
	if err := store.Save("hosted", sampleToken()); err == nil {
		t.Error("saving over a store that could not be read would discard whatever was in it")
	}
}

func TestRefusesToStoreNothing(t *testing.T) {
	store, _ := tokenStore(t)

	if err := store.Save("", sampleToken()); err == nil {
		t.Error("a token needs a server name")
	}
	if err := store.Save("hosted", nil); err == nil {
		t.Error("a nil token must be refused rather than written as an empty credential")
	}
	if err := store.Save("hosted", &oauth2.Token{}); err == nil {
		t.Error("an empty token must be refused; storing it would look like being signed in")
	}
}

// TestTokensNeverReachTheConfiguration is the rule the whole store exists to
// keep. The configuration file is shared, pasted between machines and
// hand-edited; a credential must never end up in it.
func TestTokensNeverReachTheConfiguration(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	tokensPath := filepath.Join(dir, "mcp-tokens.json")

	cfg := &Config{}
	cfg.Set("hosted", &ServerSpec{
		Type:    TransportHTTP,
		URL:     "https://mcp.example.com/v1",
		Headers: map[string]string{"Authorization": "${env:EXAMPLE_TOKEN}"},
	})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	store := &FileTokenStore{Path: tokensPath}
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}

	config, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatalf("read config: %v", err)
	}
	for _, secret := range []string{"access-abc123", "refresh-def456"} {
		if strings.Contains(string(config), secret) {
			t.Errorf("a token reached mcp.json: %s", secret)
		}
	}

	// And the reverse: the token store carries no server configuration, so a
	// leaked store does not also disclose what the user runs.
	tokens, err := os.ReadFile(tokensPath)
	if err != nil {
		t.Fatalf("read tokens: %v", err)
	}
	if strings.Contains(string(tokens), "mcp.example.com") {
		t.Error("the token store is carrying server configuration it does not need")
	}
}

func TestDescriptionSaysWhereTokensLiveAndHowWeakItIs(t *testing.T) {
	store, path := tokenStore(t)
	description := store.Description()

	if !strings.Contains(description, path) {
		t.Errorf("Description should name the file: %q", description)
	}
	// Someone signing in to a third-party service is entitled to know their
	// credential is protected by file permissions and nothing more.
	if !strings.Contains(description, "readable by any program running as you") {
		t.Errorf("Description must not imply the operating system's keychain: %q", description)
	}
}

func TestFileTokenStoreSatisfiesTheInterface(t *testing.T) {
	var store TokenStore = &FileTokenStore{Path: filepath.Join(t.TempDir(), "t.json")}
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if _, err := store.Load("hosted"); err != nil {
		t.Fatalf("Load: %v", err)
	}
	if err := store.Delete("hosted"); err != nil {
		t.Fatalf("Delete: %v", err)
	}
	if store.Description() == "" {
		t.Error("a store must be able to say where it keeps things")
	}
}

func TestStoredShapeIsExplicit(t *testing.T) {
	// The persisted form is written field by field rather than by embedding
	// oauth2.Token, so a change to that type cannot silently alter the file.
	store, path := tokenStore(t)
	if err := store.Save("hosted", sampleToken()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}

	var file struct {
		Tokens map[string]map[string]any `json:"tokens"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		t.Fatalf("parse: %v", err)
	}
	stored, ok := file.Tokens["hosted"]
	if !ok {
		t.Fatalf("no entry for hosted: %s", data)
	}
	for _, field := range []string{"accessToken", "tokenType", "refreshToken", "expiry"} {
		if _, present := stored[field]; !present {
			t.Errorf("field %q is missing from the stored shape: %v", field, stored)
		}
	}
}
