package mcp

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

// fakeAuthServer is a resource server and an authorization server in one, the
// shape a small hosted MCP service actually takes. It records every revocation
// request so a test can assert what was sent, which is the only way to tell a
// real revocation from a local delete dressed up as one.
type fakeAuthServer struct {
	*httptest.Server

	mu       sync.Mutex
	revoked  []url.Values
	revokes  bool // whether the metadata advertises a revocation endpoint
	rejectAt int  // status to answer revocation with; 0 means 200
}

func newFakeAuthServer(t *testing.T, revokes bool) *fakeAuthServer {
	t.Helper()
	fake := &fakeAuthServer{revokes: revokes}
	mux := http.NewServeMux()
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)

	// RFC 9728: the MCP endpoint says which authorization servers govern it.
	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, map[string]any{
			"resource":              fake.URL + "/mcp",
			"authorization_servers": []string{fake.URL},
		})
	})
	// RFC 8414: the authorization server says where its endpoints are.
	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		meta := map[string]any{
			"issuer":                                fake.URL,
			"authorization_endpoint":                fake.URL + "/authorize",
			"token_endpoint":                        fake.URL + "/token",
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
		}
		if fake.revokes {
			meta["revocation_endpoint"] = fake.URL + "/revoke"
		}
		writeJSON(w, meta)
	})
	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fake.mu.Lock()
		fake.revoked = append(fake.revoked, r.PostForm)
		status := fake.rejectAt
		fake.mu.Unlock()

		if status != 0 {
			w.WriteHeader(status)
			return
		}
		w.WriteHeader(http.StatusOK)
	})
	return fake
}

func writeJSON(w http.ResponseWriter, body any) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(body)
}

func (f *fakeAuthServer) revocations() []url.Values {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.revoked...)
}

func (f *fakeAuthServer) spec() *ServerSpec {
	return &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: f.URL + "/mcp"}
}

func signInManager(t *testing.T, store TokenStore) *Manager {
	t.Helper()
	m := NewManager(Options{
		Tokens:    store,
		Approvals: allowAll{},
	})
	t.Cleanup(func() { m.Close() })
	return m
}

// TestSignOutRevokesAtTheServer is what makes signing out mean something. A
// token deleted here but left valid at the service is still a credential a
// user believes they have withdrawn.
func TestSignOutRevokesAtTheServer(t *testing.T) {
	fake := newFakeAuthServer(t, true)
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	if err := store.Save("hosted", &SignInRecord{
		Token:    &oauth2.Token{AccessToken: "access-1", RefreshToken: "refresh-1"},
		ClientID: "registered-client",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := signInManager(t, store)
	if err := manager.SignOut(t.Context(), fake.spec()); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	revocations := fake.revocations()
	if len(revocations) != 1 {
		t.Fatalf("the server received %d revocations, want 1; a sign-out that does not reach the server is only a local delete", len(revocations))
	}
	form := revocations[0]
	// The refresh token, not the access token: revoking it withdraws the whole
	// sign-in rather than one short-lived credential.
	if got := form.Get("token"); got != "refresh-1" {
		t.Errorf("revoked token = %q, want the refresh token", got)
	}
	if got := form.Get("token_type_hint"); got != "refresh_token" {
		t.Errorf("token_type_hint = %q, want refresh_token", got)
	}
	if got := form.Get("client_id"); got != "registered-client" {
		t.Errorf("client_id = %q; a public client identifies itself in the body, and a server that requires one refuses without it", got)
	}

	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token is still stored after signing out: %v", err)
	}
}

func TestSignOutRevokesTheAccessTokenWhenThereIsNoRefreshToken(t *testing.T) {
	fake := newFakeAuthServer(t, true)
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "access-only"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := signInManager(t, store)
	if err := manager.SignOut(t.Context(), fake.spec()); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	revocations := fake.revocations()
	if len(revocations) != 1 {
		t.Fatalf("revocations = %d, want 1", len(revocations))
	}
	if got := revocations[0].Get("token"); got != "access-only" {
		t.Errorf("revoked token = %q, want the access token", got)
	}
	if got := revocations[0].Get("token_type_hint"); got != "access_token" {
		t.Errorf("token_type_hint = %q, want access_token", got)
	}
}

// TestSignOutForgetsTheTokenEvenWhenRevocationFails is the other half of the
// bargain. A token that could not be revoked must at least stop being used
// from this machine — but the user is told, because the remedy is theirs.
func TestSignOutForgetsTheTokenEvenWhenRevocationFails(t *testing.T) {
	fake := newFakeAuthServer(t, true)
	fake.rejectAt = http.StatusInternalServerError
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "access-1"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := signInManager(t, store)
	err := manager.SignOut(t.Context(), fake.spec())
	if err == nil {
		t.Fatal("a failed revocation must be reported; the user believes the credential is withdrawn")
	}
	if !errors.Is(err, ErrSignedOutLocallyOnly) {
		t.Errorf("err = %v, want ErrSignedOutLocallyOnly so a surface can say what actually happened", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token survived a sign-out; it must stop being used from this machine either way: %v", err)
	}
}

func TestSignOutOfAServerThatOffersNoRevocation(t *testing.T) {
	fake := newFakeAuthServer(t, false)
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "access-1"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := signInManager(t, store)
	err := manager.SignOut(t.Context(), fake.spec())
	if !errors.Is(err, ErrSignedOutLocallyOnly) {
		t.Fatalf("err = %v, want ErrSignedOutLocallyOnly: a server with no revocation endpoint cannot be signed out of, and saying otherwise is a lie", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token was not forgotten: %v", err)
	}
	if len(fake.revocations()) != 0 {
		t.Error("something was posted to a revocation endpoint that was never advertised")
	}
}

func TestSignOutOfAServerNeverSignedIn(t *testing.T) {
	fake := newFakeAuthServer(t, true)
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	manager := signInManager(t, store)

	if err := manager.SignOut(t.Context(), fake.spec()); err != nil {
		t.Fatalf("signing out of a server that was never signed in is not a failure: %v", err)
	}
	if len(fake.revocations()) != 0 {
		t.Error("a server that was never signed in was sent a revocation")
	}
}

// TestSignOutEndsTheSessionAndTheState guards against a sign-out that revokes
// the credential while leaving the connection up. The tools would stay on
// offer and every call through them would fail with an authorization error the
// user could not explain.
func TestSignOutEndsTheSessionAndTheState(t *testing.T) {
	authServer := newFakeAuthServer(t, true)
	spec := authServer.spec()
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}

	fake := newFakeServer(t, simpleTool("alpha"))
	clientTransport, serverTransport := sdk.NewInMemoryTransports()
	session, err := fake.server.Connect(t.Context(), serverTransport, nil)
	if err != nil {
		t.Fatalf("fake server connect: %v", err)
	}
	t.Cleanup(func() { session.Close() })

	manager := NewManager(Options{
		Tokens:    store,
		Approvals: allowAll{},
		newTransport: func(context.Context, *ServerSpec, transportOptions) (sdk.Transport, func(), error) {
			return clientTransport, func() {}, nil
		},
	})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State(spec.Name); state.Status != StatusConnected {
		t.Fatalf("status = %q, want connected before signing out", state.Status)
	}
	if len(manager.Tools()) == 0 {
		t.Fatal("no tools were offered, so this test cannot show them being withdrawn")
	}

	if err := manager.SignOut(t.Context(), spec); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	state, _ := manager.State(spec.Name)
	if state.Status != StatusNeedsSignIn {
		t.Errorf("status = %q, want needs-sign-in", state.Status)
	}
	if got := manager.Tools(); len(got) != 0 {
		t.Errorf("%d tools are still on offer after signing out", len(got))
	}
}

func TestSignInRefusesWhatItCannotSignIntoTo(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}

	t.Run("a local server", func(t *testing.T) {
		manager := signInManager(t, store)
		_, err := manager.SignIn(t.Context(), &ServerSpec{Name: "files", Command: "uvx"})
		if err == nil {
			t.Fatal("a stdio server has nothing to sign in to")
		}
		if !strings.Contains(err.Error(), "files") {
			t.Errorf("err = %v, want it to name the server", err)
		}
	})

	t.Run("with nowhere to keep the token", func(t *testing.T) {
		manager := signInManager(t, nil)
		if _, err := manager.SignIn(t.Context(), &ServerSpec{Name: "hosted", URL: "https://mcp.example.com/v1"}); err == nil {
			t.Fatal("signing in with no token store would sign the user in and lose it immediately")
		}
	})
}

// TestSignInStillNeedsApproval keeps the sign-in button from being a way round
// the approval gate. Approval is what says a server may be contacted at all,
// and a sign-in contacts it.
func TestSignInStillNeedsApproval(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	reached := false
	manager := NewManager(Options{
		Tokens:    store,
		Approvals: &Approvals{},
		newTransport: func(context.Context, *ServerSpec, transportOptions) (sdk.Transport, func(), error) {
			reached = true
			return nil, func() {}, errors.New("must not be called")
		},
	})
	t.Cleanup(func() { manager.Close() })

	spec := &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: "https://mcp.example.com/v1"}
	if _, err := manager.SignIn(t.Context(), spec); err == nil {
		t.Fatal("an unapproved server must not be signed in to")
	}
	if reached {
		t.Error("an unapproved server was contacted by the sign-in path")
	}
}

// TestSignInIsTheOnlyPathThatMayOpenABrowser is the whole distinction, checked
// where the two paths meet: the mode reaches the transport, and it differs.
func TestSignInIsTheOnlyPathThatMayOpenABrowser(t *testing.T) {
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	spec := &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: "https://mcp.example.com/v1"}

	var mu sync.Mutex
	var modes []signInMode
	manager := NewManager(Options{
		Tokens:    store,
		Approvals: allowAll{},
		newTransport: func(_ context.Context, _ *ServerSpec, opts transportOptions) (sdk.Transport, func(), error) {
			mu.Lock()
			modes = append(modes, opts.signIn)
			mu.Unlock()
			return nil, func() {}, errors.New("no transport in this test")
		},
	})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	manager.Connect(t.Context(), cfg)

	mu.Lock()
	ordinary := append([]signInMode(nil), modes...)
	modes = nil
	mu.Unlock()

	if len(ordinary) == 0 {
		t.Fatal("the ordinary path never reached the transport")
	}
	for i, mode := range ordinary {
		if mode != signInDisallowed {
			t.Errorf("attempt %d connected with mode %v; an ordinary connection must never be allowed to open a browser", i, mode)
		}
	}

	manager.SignIn(t.Context(), spec)

	mu.Lock()
	explicit := append([]signInMode(nil), modes...)
	mu.Unlock()

	if len(explicit) != 1 {
		t.Fatalf("the sign-in reached the transport %d times, want 1; a browser sign-in must not be retried", len(explicit))
	}
	if explicit[0] != signInAllowed {
		t.Errorf("an explicit sign-in connected with mode %v, so it could never open a browser", explicit[0])
	}
}
