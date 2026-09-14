package mcp

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

// The environment variables that turn this file on. It is inert without them,
// because a suite that reaches a third party over the network fails for reasons
// that have nothing to do with this code — a service being down, a corporate
// proxy, an expired certificate — and an upstream project cannot be asked to
// depend on somebody else's uptime.
//
// Everything else in this package runs against a server it controls. That is
// what makes the tests deterministic, and it is also their limit: a server
// written to satisfy this client agrees with whatever this client does. These
// run against one that has never heard of Ollama.
const (
	// hostedURLEnv names the MCP endpoint to connect to.
	hostedURLEnv = "OLLAMA_MCP_TEST_URL"
	// hostedSignInEnv permits an interactive browser sign-in. Without it a
	// server that wants one is reported and left alone, which is itself worth
	// proving against a real service.
	hostedSignInEnv = "OLLAMA_MCP_TEST_SIGNIN"
	// hostedToolEnv and hostedArgsEnv name a tool to call and its arguments as
	// JSON. Calling a stranger's tool is a side effect on their service, so it
	// happens only when a person has named one.
	hostedToolEnv = "OLLAMA_MCP_TEST_TOOL"
	hostedArgsEnv = "OLLAMA_MCP_TEST_ARGS"
)

func hostedSpec(t *testing.T) *ServerSpec {
	t.Helper()
	endpoint := strings.TrimSpace(os.Getenv(hostedURLEnv))
	if endpoint == "" {
		t.Skipf("set %s to a hosted MCP endpoint to run this", hostedURLEnv)
	}
	return &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: endpoint}
}

func hostedManager(t *testing.T) (*Manager, TokenStore) {
	t.Helper()
	// A file store in a temporary directory, never the platform default: a
	// token obtained here must not land in the operator's keychain beside
	// their real sign-ins.
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	manager := NewManager(Options{
		ConnectTimeout: 45 * time.Second,
		CallTimeout:    45 * time.Second,
		// The operator approved this server by naming it in the environment.
		// The ledger exists to stop Ollama running what a user has not read;
		// here the user typed the endpoint.
		Approvals: allowAll{},
		Tokens:    store,
	})
	t.Cleanup(func() { manager.Close() })
	return manager, store
}

// TestAHostedServerConnects is the run that no fake can stand in for: a real
// endpoint, real TLS, a real streamable HTTP session, and a tool list written
// by somebody who has never seen this code.
//
// Both outcomes are a pass. A server that answers the handshake proves the
// transport and the schema conversion against tools nobody here designed. A
// server that demands an authorization proves the other half — that Ollama
// reports it and stops, rather than failing obscurely or opening a browser at
// a machine nobody is sitting at.
func TestAHostedServerConnects(t *testing.T) {
	spec := hostedSpec(t)
	manager, store := hostedManager(t)

	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	manager.Connect(t.Context(), cfg)

	state, ok := manager.State(spec.Name)
	if !ok {
		t.Fatal("no state was recorded for the server")
	}
	t.Logf("endpoint: %s", spec.URL)
	t.Logf("status: %s", state.Status)
	if state.ServerInfo != "" {
		t.Logf("server: %s (protocol %s)", state.ServerInfo, state.ProtocolVersion)
	}
	if state.Err != nil {
		t.Logf("error: %v", state.Err)
	}

	switch state.Status {
	case StatusNeedsSignIn:
		if !SignInRequired(state.Err) {
			t.Errorf("err = %v, want it to report that a sign-in is needed", state.Err)
		}
		// Nothing was stored, because nothing was signed in to.
		if _, err := store.Load(spec.Name); !errors.Is(err, ErrNoToken) {
			t.Errorf("a token exists for a server that was never signed in to: %v", err)
		}
		t.Log("this server requires an authorization; run with " + hostedSignInEnv + "=1 to sign in")

	case StatusConnected:
		if len(state.Tools) == 0 && len(state.Skipped) == 0 {
			t.Error("the server offered nothing at all, so nothing here was exercised")
		}
		reportHostedTools(t, state)

		// The schemas are what actually reach a model. Converting somebody
		// else's real schemas is the part a hand-written fixture cannot prove.
		schemas, err := manager.Schemas()
		if err != nil {
			t.Fatalf("Schemas: %v", err)
		}
		if len(schemas) != len(state.Tools) {
			t.Errorf("%d schemas for %d tools", len(schemas), len(state.Tools))
		}
		for _, schema := range schemas {
			if schema.Function.Name == "" {
				t.Error("a converted tool has no name")
			}
			if _, err := json.Marshal(schema); err != nil {
				t.Errorf("a converted schema does not marshal: %v", err)
			}
		}
		callHostedTool(t, manager, state)

	default:
		t.Fatalf("status = %s: %v", state.Status, state.Err)
	}
}

func reportHostedTools(t *testing.T, state ServerState) {
	t.Helper()
	t.Logf("%d tools offered, %d refused", len(state.Tools), len(state.Skipped))
	for _, tool := range state.Tools {
		t.Logf("  tool %s", tool.QualifiedName())
	}
	for _, skipped := range state.Skipped {
		// Not a failure: refusing a tool Ollama cannot honestly offer is the
		// designed behaviour. It is logged because which real tools get
		// refused, and why, is the useful thing to learn from a run like this.
		t.Logf("  refused %s: %s", skipped.Name, skipped.Reason)
	}
}

// callHostedTool calls one tool, and only when a person has named it. A tool
// call reaches into somebody else's service and may do something there.
func callHostedTool(t *testing.T, manager *Manager, state ServerState) {
	t.Helper()
	name := strings.TrimSpace(os.Getenv(hostedToolEnv))
	if name == "" {
		t.Log("set " + hostedToolEnv + " (and " + hostedArgsEnv + ") to call one of these tools")
		return
	}

	args := map[string]any{}
	if raw := strings.TrimSpace(os.Getenv(hostedArgsEnv)); raw != "" {
		if err := json.Unmarshal([]byte(raw), &args); err != nil {
			t.Fatalf("%s is not JSON: %v", hostedArgsEnv, err)
		}
	}

	qualified := QualifyName(state.Name, name)
	result, err := manager.Call(t.Context(), qualified, args)
	if err != nil {
		t.Fatalf("Call %s: %v", qualified, err)
	}
	if result.Content == "" && !result.IsError {
		t.Error("the call returned nothing at all; an empty result is indistinguishable from a silent failure")
	}
	t.Logf("called %s (isError=%v), %d bytes of content", qualified, result.IsError, len(result.Content))
	t.Logf("result: %s", truncateForLog(result.Content))
}

func truncateForLog(s string) string {
	const limit = 400
	if len(s) <= limit {
		return s
	}
	return s[:limit] + "… (truncated)"
}

// TestAHostedSignIn is the whole authorization flow against a real
// authorization server: discovery, registration, PKCE, a browser the operator
// finishes by hand, the exchange, and a connection that then works.
//
// It signs out at the end, so the run leaves no credential behind on the
// machine and none valid at the service. That is not tidiness — a test that
// leaves a live token lying around has widened what an operator agreed to.
func TestAHostedSignIn(t *testing.T) {
	spec := hostedSpec(t)
	if strings.TrimSpace(os.Getenv(hostedSignInEnv)) == "" {
		t.Skipf("set %s=1 to allow this to open a browser and sign in", hostedSignInEnv)
	}
	manager, store := hostedManager(t)

	t.Logf("a browser will open for %s; finish signing in there", spec.URL)
	state, err := manager.SignIn(t.Context(), spec)
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if state.Status != StatusConnected {
		t.Fatalf("status = %s: %v", state.Status, state.Err)
	}
	t.Logf("signed in: %s (protocol %s), %d tools", state.ServerInfo, state.ProtocolVersion, len(state.Tools))
	reportHostedTools(t, state)

	stored, err := store.Load(spec.Name)
	if err != nil {
		t.Fatalf("the sign-in succeeded but nothing was stored: %v", err)
	}
	if stored.Token.AccessToken == "" {
		t.Error("an empty access token was stored")
	}
	// Logged as presence, never as value: a real credential must not end up in
	// a test log, a CI transcript or a paste of one.
	t.Logf("stored: access token %d chars, refresh token present=%v, client id present=%v, expiry=%v",
		len(stored.Token.AccessToken), stored.Token.RefreshToken != "", stored.ClientID != "", stored.Token.Expiry)
	if stored.ClientID == "" {
		t.Error("no client identifier was recorded, so this sign-in could never be revoked")
	}

	// A second manager over the same store connects with no browser at all,
	// which is what a restart does.
	restarted := NewManager(Options{
		ConnectTimeout: 45 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser: func(string) error {
			t.Error("an ordinary connection opened a browser")
			return errors.New("no browser may be opened here")
		},
	})
	t.Cleanup(func() { restarted.Close() })
	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	restarted.Connect(t.Context(), cfg)
	if again, _ := restarted.State(spec.Name); again.Status != StatusConnected {
		t.Errorf("a stored token did not reconnect: %s (%v)", again.Status, again.Err)
	}

	// And sign out, which revokes at the service and then forgets locally.
	signOutErr := manager.SignOut(t.Context(), spec)
	switch {
	case signOutErr == nil:
		t.Log("signed out: the token was revoked at the server and deleted here")
	case errors.Is(signOutErr, ErrSignedOutLocallyOnly):
		// Not a failure of this code: plenty of authorization servers publish
		// no revocation endpoint. It is logged loudly because it is exactly
		// what the user is told, and because it means a live token remains at
		// the service until it expires.
		t.Logf("signed out locally only — the token may still be valid at the service: %v", signOutErr)
	default:
		t.Fatalf("SignOut: %v", signOutErr)
	}

	if _, err := store.Load(spec.Name); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token is still stored after signing out: %v", err)
	}
}
