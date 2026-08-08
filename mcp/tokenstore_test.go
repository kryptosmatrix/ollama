package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"golang.org/x/oauth2"
)

func tokenStore(t *testing.T) (*FileTokenStore, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "mcp-tokens.json")
	return &FileTokenStore{Path: path}, path
}

func sampleSignIn() *SignInRecord {
	return &SignInRecord{
		Token: &oauth2.Token{
			AccessToken:  "access-abc123",
			TokenType:    "Bearer",
			RefreshToken: "refresh-def456",
			Expiry:       time.Date(2026, 8, 6, 12, 0, 0, 0, time.UTC),
		},
		ClientID: "client-xyz789",
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
	want := sampleSignIn()

	if err := store.Save("hosted", want); err != nil {
		t.Fatalf("Save: %v", err)
	}
	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != want.Token.AccessToken {
		t.Errorf("AccessToken = %q", got.Token.AccessToken)
	}
	if got.Token.RefreshToken != want.Token.RefreshToken {
		t.Errorf("the refresh token must survive, or the user is signed out at the first expiry: %q", got.Token.RefreshToken)
	}
	if got.Token.TokenType != want.Token.TokenType {
		t.Errorf("TokenType = %q", got.Token.TokenType)
	}
	if !got.Token.Expiry.Equal(want.Token.Expiry) {
		t.Errorf("Expiry = %v, want %v; a lost expiry means the token is never refreshed", got.Token.Expiry, want.Token.Expiry)
	}
	if got.ClientID != want.ClientID {
		t.Errorf("ClientID = %q, want %q; without it the sign-in can be forgotten but never revoked", got.ClientID, want.ClientID)
	}
}

// TestARefreshWithoutAClientIDKeepsTheRecordedOne guards the revocability of a
// sign-in across refreshes. The identifier is issued once, at registration; a
// later save that does not carry it must not erase it, or the sign-in becomes
// unrevocable at the first token refresh.
func TestARefreshWithoutAClientIDKeepsTheRecordedOne(t *testing.T) {
	store, _ := tokenStore(t)
	if err := store.Save("hosted", sampleSignIn()); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "refreshed"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	got, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if got.Token.AccessToken != "refreshed" {
		t.Errorf("AccessToken = %q, want the refreshed one", got.Token.AccessToken)
	}
	if got.ClientID != "client-xyz789" {
		t.Errorf("ClientID = %q, want the one recorded at sign-in to survive a refresh", got.ClientID)
	}
}

func TestTokensArePerServer(t *testing.T) {
	store, _ := tokenStore(t)

	if err := store.Save("one", &SignInRecord{Token: &oauth2.Token{AccessToken: "first"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}
	if err := store.Save("two", &SignInRecord{Token: &oauth2.Token{AccessToken: "second"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	first, err := store.Load("one")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if first.Token.AccessToken != "first" {
		t.Errorf("one = %q; one server's credential must never be handed to another", first.Token.AccessToken)
	}
	second, _ := store.Load("two")
	if second.Token.AccessToken != "second" {
		t.Errorf("two = %q", second.Token.AccessToken)
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
		if err := store.Save("other", sampleSignIn()); err != nil {
			t.Fatalf("Save: %v", err)
		}
		if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
			t.Errorf("err = %v, want ErrNoToken", err)
		}
	})
}

func TestDelete(t *testing.T) {
	store, _ := tokenStore(t)
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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
	if err := store.Save("hosted", sampleSignIn()); err == nil {
		t.Error("saving over a store that could not be read would discard whatever was in it")
	}
}

func TestRefusesToStoreNothing(t *testing.T) {
	store, _ := tokenStore(t)

	if err := store.Save("", sampleSignIn()); err == nil {
		t.Error("a token needs a server name")
	}
	if err := store.Save("hosted", nil); err == nil {
		t.Error("a nil token must be refused rather than written as an empty credential")
	}
	if err := store.Save("hosted", &SignInRecord{}); err == nil {
		t.Error("a record with no token must be refused; storing it would look like being signed in")
	}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{}}); err == nil {
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
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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
	if err := store.Save("hosted", sampleSignIn()); err != nil {
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

// TestConcurrentSavesDoNotLoseATokenInThisProcess. Save and Delete read the
// whole file, change one entry and write it back. Without serialisation two
// saves for *different* servers interleave and the second overwrites the
// first's change — measured before the fix: eight concurrent saves for eight
// servers left one token on disk.
//
// It closes the window inside one process. Between processes it stays open, and
// the store's comment says so: the desktop app and a terminal can still lose
// one, and closing that needs a lock file with all the staleness that implies.
func TestConcurrentSavesDoNotLoseATokenInThisProcess(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	names := []string{"one", "two", "three", "four", "five", "six", "seven", "eight"}

	var wg sync.WaitGroup
	for _, name := range names {
		wg.Add(1)
		go func() {
			defer wg.Done()
			if err := store.Save(name, &SignInRecord{Token: &oauth2.Token{AccessToken: "t-" + name}}); err != nil {
				t.Errorf("Save %s: %v", name, err)
			}
		}()
	}
	wg.Wait()

	got, err := store.Servers()
	if err != nil {
		t.Fatalf("Servers: %v", err)
	}
	if len(got) != len(names) {
		t.Errorf("%d of %d tokens survived concurrent saves: %v", len(got), len(names), got)
	}
	for _, name := range names {
		record, err := store.Load(name)
		if err != nil {
			t.Errorf("Load %s: %v", name, err)
			continue
		}
		if record.Token.AccessToken != "t-"+name {
			t.Errorf("%s holds %q", name, record.Token.AccessToken)
		}
	}
}

// TestAConcurrentlyUsedTokenSourceIsNotARace. A token source is consulted from
// whatever goroutine is making a request, and this one keeps the last access
// token it wrote so it does not rewrite the store on every call.
func TestAConcurrentlyUsedTokenSourceIsNotARace(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	source := &persistingTokenSource{
		server: "hosted", store: store, clientID: "client-1",
		source: oauth2.StaticTokenSource(&oauth2.Token{AccessToken: "same"}),
	}

	var wg sync.WaitGroup
	for range 8 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 50 {
				if _, err := source.Token(); err != nil {
					t.Errorf("Token: %v", err)
					return
				}
			}
		}()
	}
	wg.Wait()

	record, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if record.Token.AccessToken != "same" {
		t.Errorf("AccessToken = %q", record.Token.AccessToken)
	}
}

// TestAStdioServerThatFailsItsHandshakeIsNotLeft records a REFUTED finding
// rather than a repair, because the answer is worth keeping.
//
// The review expected a subprocess that starts and then fails the MCP handshake
// to survive until the manager closes, since the stdio release function is a
// no-op. It does not: the protocol library kills it. This is the same guarantee
// that led to deleting a per-server reaping mechanism in Phase 2 after three
// attempts to falsify it all passed — and it is worth a standing test, because
// the next reader will have the same suspicion.
func TestAStdioServerThatFailsItsHandshakeIsNotLeft(t *testing.T) {
	if _, err := exec.LookPath("pgrep"); err != nil {
		t.Skip("pgrep is needed to look for a surviving process")
	}
	marker := "ollama-mcp-handshake-probe-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	spec := &ServerSpec{
		Name:    "leaky",
		Command: "sh",
		Args:    []string{"-c", "echo " + marker + " >&2; echo not-json; sleep 30"},
	}

	manager := NewManager(Options{ConnectTimeout: 3 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() {
		manager.Close()
		exec.Command("pkill", "-f", marker).Run()
	})

	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State(spec.Name); state.Status == StatusConnected {
		t.Fatal("setup: this server was supposed to fail its handshake")
	}

	time.Sleep(500 * time.Millisecond)
	out, _ := exec.Command("pgrep", "-fl", marker).Output()
	if alive := strings.TrimSpace(string(out)); alive != "" {
		t.Errorf("a subprocess survived a failed handshake:\n%s", alive)
	}
}
