package cmd

import (
	"path/filepath"
	"strings"
	"testing"
)

// signInEnv isolates the token store as well, so a test can never read or write
// the developer's own credentials.
func signInEnv(t *testing.T) (configPath, approvalsPath, tokensPath string) {
	t.Helper()
	configPath, approvalsPath = mcpEnv(t)
	tokensPath = filepath.Join(filepath.Dir(configPath), "mcp-tokens.json")
	t.Setenv("OLLAMA_MCP_TOKENS", tokensPath)
	return configPath, approvalsPath, tokensPath
}

func TestMCPLoginAndLogoutAreRegistered(t *testing.T) {
	var found []string
	for _, cmd := range MCPCmd().Commands() {
		found = append(found, cmd.Name())
	}
	for _, want := range []string{"login", "logout"} {
		if !slicesContains(found, want) {
			t.Errorf("ollama mcp has no %q command; it is unreachable however well it works: %v", want, found)
		}
	}
}

func slicesContains(haystack []string, needle string) bool {
	for _, item := range haystack {
		if item == needle {
			return true
		}
	}
	return false
}

func TestMCPLoginRefusesALocalServer(t *testing.T) {
	signInEnv(t)
	if out, err := runMCP(t, "", "add", "files", "uvx", "mcp-server-files"); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}

	out, err := runMCP(t, "", "login", "files")
	if err == nil {
		t.Fatalf("a stdio server has nothing to sign in to, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "files") {
		t.Errorf("err = %v, want it to name the server", err)
	}
}

func TestMCPLoginRefusesAnUnknownServer(t *testing.T) {
	signInEnv(t)
	if _, err := runMCP(t, "", "login", "nothing"); err == nil {
		t.Fatal("signing in to a server that does not exist must fail")
	}
}

// TestMCPLoginSaysWhereTheTokenWillBeKept is the disclosure that has to come
// before the browser, not after. The file store is weaker than the operating
// system's keychain, and someone signing in to a third-party service is
// entitled to know that while they can still decide not to.
//
// The approval is withdrawn first, so the sign-in stops at the approval gate
// and nothing is ever contacted — which is also the second thing this test
// proves. `ollama mcp add` approves what the user typed at their own keyboard,
// so a freshly added server is already approved and would be dialled.
func TestMCPLoginSaysWhereTheTokenWillBeKeptAndStopsAtApproval(t *testing.T) {
	_, _, tokensPath := signInEnv(t)
	if out, err := runMCP(t, "", "add", "hosted", "--url", "https://mcp.example.com/v1"); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	if out, err := runMCP(t, "", "revoke", "hosted"); err != nil {
		t.Fatalf("revoke: %v\n%s", err, out)
	}

	out, err := runMCP(t, "", "login", "hosted")
	if err == nil {
		t.Fatalf("an unapproved server must not be signed in to, got:\n%s", out)
	}
	if !strings.Contains(err.Error(), "approved") {
		t.Errorf("err = %v, want it to say the server is not approved", err)
	}
	if !strings.Contains(out, tokensPath) {
		t.Errorf("the user was not told where their token would be kept before the browser opened, got:\n%s", out)
	}
	if !strings.Contains(out, "readable by any program running as you") {
		t.Errorf("the store's protection was not stated, got:\n%s", out)
	}
}

func TestMCPLogoutOfAServerNeverSignedIn(t *testing.T) {
	signInEnv(t)
	if out, err := runMCP(t, "", "add", "hosted", "--url", "https://mcp.example.com/v1"); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}

	// Nothing is stored, so nothing is contacted and nothing fails: the user's
	// intent — that no token remain — is already satisfied.
	if _, err := runMCP(t, "", "logout", "hosted"); err != nil {
		t.Fatalf("logout: %v", err)
	}
}

func TestMCPLogoutRefusesALocalServer(t *testing.T) {
	signInEnv(t)
	if out, err := runMCP(t, "", "add", "files", "uvx", "mcp-server-files"); err != nil {
		t.Fatalf("add: %v\n%s", err, out)
	}
	if _, err := runMCP(t, "", "logout", "files"); err == nil {
		t.Fatal("a local server is not signed in to, so signing out of it is not a thing that can happen")
	}
}
