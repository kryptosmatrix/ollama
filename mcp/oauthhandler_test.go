package mcp

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

func testStore(t *testing.T) TokenStore {
	t.Helper()
	return &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
}

// TestAnOrdinaryConnectionCannotOpenABrowser is the property that keeps a
// sign-in an act of the user's. Servers connect at start-up, on a
// configuration change and on reconnects; none of those may summon a browser
// window, however the server behaves.
//
// This goes through the session's own fetcher, which is what the handler calls
// on a 401, rather than through refuseSignIn directly: the property is that an
// ordinary connection is wired to a refusal, not merely that a refusal exists.
func TestAnOrdinaryConnectionCannotOpenABrowser(t *testing.T) {
	session, err := newOAuthSession("hosted", testStore(t), signInDisallowed)
	if err != nil {
		t.Fatalf("newOAuthSession: %v", err)
	}
	t.Cleanup(session.close)

	if session.handler == nil {
		t.Fatal("an http server should still get a handler; only the browser is withheld")
	}
	if session.fetch == nil {
		t.Fatal("no fetcher was configured, so nothing decides whether a browser opens")
	}

	result, err := session.fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: "https://auth.example.com/authorize"})
	if err == nil {
		t.Fatalf("an ordinary connection must not begin a sign-in, got %+v", result)
	}
	if !SignInRequired(err) {
		t.Errorf("err = %v, want it to report that a sign-in is needed", err)
	}
	if !errors.Is(err, ErrSignInRequired) {
		t.Errorf("err = %v, want ErrSignInRequired", err)
	}
	if got := err.Error(); !strings.Contains(got, "hosted") {
		t.Errorf("err = %q, want it to name the server so the user knows which one", got)
	}
}

// TestAnOrdinaryConnectionOpensNoListener proves the browser is not merely
// unused but unreachable: no redirect port is bound at all.
func TestAnOrdinaryConnectionOpensNoListener(t *testing.T) {
	session, err := newOAuthSession("hosted", testStore(t), signInDisallowed)
	if err != nil {
		t.Fatalf("newOAuthSession: %v", err)
	}
	defer session.close()

	if session.redirectURL != unusableRedirect {
		t.Fatalf("redirectURL = %q, want the unusable placeholder %q; a real one means a listener was started", session.redirectURL, unusableRedirect)
	}
	// And nothing answers on it. If a listener had been started this process
	// would be bound to the address it declared.
	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(session.redirectURL); err == nil {
		resp.Body.Close()
		t.Error("something is listening for a redirect during an ordinary connection")
	}
}

// TestAnExplicitSignInOpensAListener is the other half: when the user asks, the
// redirect is bound and answering before the browser is sent anywhere. A
// sign-in that opens a browser first and binds afterwards loses the callback
// when the user is already signed in at the authorization server and is
// redirected straight back.
func TestAnExplicitSignInOpensAListener(t *testing.T) {
	session, err := newOAuthSession("hosted", testStore(t), signInAllowed)
	if err != nil {
		t.Fatalf("newOAuthSession: %v", err)
	}
	defer session.close()

	if session.handler == nil {
		t.Fatal("no handler was built")
	}
	if session.redirectURL == unusableRedirect || session.redirectURL == "" {
		t.Fatalf("redirectURL = %q, want a real loopback address the server can redirect to", session.redirectURL)
	}
	if !strings.HasPrefix(session.redirectURL, "http://127.0.0.1:") {
		t.Errorf("redirectURL = %q, want an ephemeral port on loopback", session.redirectURL)
	}

	// It answers, so the callback cannot arrive before the listener is ready.
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(session.redirectURL)
	if err != nil {
		t.Fatalf("nothing is listening on the declared redirect: %v", err)
	}
	resp.Body.Close()
}

// TestClosingASignInReleasesTheListener keeps a refused or abandoned sign-in
// from leaving a port bound for the life of the process.
func TestClosingASignInReleasesTheListener(t *testing.T) {
	session, err := newOAuthSession("hosted", testStore(t), signInAllowed)
	if err != nil {
		t.Fatalf("newOAuthSession: %v", err)
	}
	address := session.redirectURL
	session.close()

	client := &http.Client{Timeout: 2 * time.Second}
	if resp, err := client.Get(address); err == nil {
		resp.Body.Close()
		t.Error("the redirect listener is still bound after close")
	}
}

func TestOAuthNeedsSomewhereToKeepTokens(t *testing.T) {
	if _, err := newOAuthSession("hosted", nil, signInDisallowed); err == nil {
		t.Fatal("a handler without a token store would sign the user in and then lose it")
	}
}

func TestOAuthUsesAStoredTokenWithoutAnySignIn(t *testing.T) {
	store := testStore(t)
	if err := store.Save("hosted", &SignInRecord{Token: &oauth2.Token{AccessToken: "stored", TokenType: "Bearer"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	// Disallowed mode: no listener, no browser. A stored token must still make
	// a usable handler, which is the ordinary case for an already-signed-in
	// server at start-up.
	session, err := newOAuthSession("hosted", store, signInDisallowed)
	if err != nil {
		t.Fatalf("newOAuthSession: %v", err)
	}
	defer session.close()

	source, err := session.handler.TokenSource(t.Context())
	if err != nil {
		t.Fatalf("TokenSource: %v", err)
	}
	if source == nil {
		t.Fatal("a stored token must produce a token source, or the server is asked to sign in again every launch")
	}
	token, err := source.Token()
	if err != nil {
		t.Fatalf("Token: %v", err)
	}
	if token.AccessToken != "stored" {
		t.Errorf("AccessToken = %q, want the stored one", token.AccessToken)
	}
}

func TestOAuthReportsAnUnreadableStore(t *testing.T) {
	if _, err := newOAuthSession("hosted", &brokenStore{}, signInDisallowed); err == nil {
		t.Fatal("a store that cannot be read must fail loudly rather than silently signing the user out")
	}
}

type brokenStore struct{}

func (brokenStore) Load(string) (*SignInRecord, error) { return nil, errors.New("disk on fire") }
func (brokenStore) Save(string, *SignInRecord) error   { return errors.New("disk on fire") }
func (brokenStore) Delete(string) error                { return errors.New("disk on fire") }
func (brokenStore) Description() string                { return "a broken store" }

// fakeSource hands out a new token each time, the way a refreshing source does.
type fakeSource struct {
	tokens []*oauth2.Token
	calls  int
}

func (f *fakeSource) Token() (*oauth2.Token, error) {
	if f.calls >= len(f.tokens) {
		return nil, fmt.Errorf("no more tokens")
	}
	token := f.tokens[f.calls]
	f.calls++
	return token, nil
}

// TestARefreshedTokenIsWrittenDown is what keeps a user signed in across
// restarts. A refresh that lives only in memory means they are signed in until
// Ollama closes and then sent back to the browser, which reads as the sign-in
// not having worked.
func TestARefreshedTokenIsWrittenDown(t *testing.T) {
	store := testStore(t)
	inner := &fakeSource{tokens: []*oauth2.Token{
		{AccessToken: "first", RefreshToken: "r1"},
		{AccessToken: "second", RefreshToken: "r2"},
	}}
	source := &persistingTokenSource{server: "hosted", store: store, clientID: "registered-client", source: inner}

	if _, err := source.Token(); err != nil {
		t.Fatalf("Token: %v", err)
	}
	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Token.AccessToken != "first" {
		t.Errorf("AccessToken = %q, want the first token written down", stored.Token.AccessToken)
	}

	if _, err := source.Token(); err != nil {
		t.Fatalf("Token: %v", err)
	}
	stored, err = store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Token.AccessToken != "second" {
		t.Errorf("AccessToken = %q, want the refreshed token to replace it", stored.Token.AccessToken)
	}
	if stored.Token.RefreshToken != "r2" {
		t.Errorf("RefreshToken = %q, want the new refresh token too", stored.Token.RefreshToken)
	}
	if stored.ClientID != "registered-client" {
		t.Errorf("ClientID = %q, want the sign-in's client identifier carried through the refresh; without it the sign-in cannot be revoked", stored.ClientID)
	}
}

// TestAnUnchangedTokenIsNotRewritten guards against writing the store on every
// request. A token source is consulted constantly; rewriting a file each time
// would be a great deal of disk for no change.
func TestAnUnchangedTokenIsNotRewritten(t *testing.T) {
	counting := &countingStore{TokenStore: testStore(t)}
	same := &oauth2.Token{AccessToken: "same"}
	source := &persistingTokenSource{
		server: "hosted",
		store:  counting,
		source: &fakeSource{tokens: []*oauth2.Token{same, same, same}},
	}

	for range 3 {
		if _, err := source.Token(); err != nil {
			t.Fatalf("Token: %v", err)
		}
	}
	if counting.saves != 1 {
		t.Errorf("saves = %d, want 1; an unchanged token must not be rewritten on every request", counting.saves)
	}
}

type countingStore struct {
	TokenStore
	saves int
}

func (c *countingStore) Save(server string, record *SignInRecord) error {
	c.saves++
	return c.TokenStore.Save(server, record)
}

func TestSavingTokenSourceStoresWhatItIsGiven(t *testing.T) {
	store := testStore(t)
	build := savingTokenSource("hosted", store)

	source, err := build(t.Context(), &oauth2.Config{ClientID: "registered-client"}, &oauth2.Token{AccessToken: "handed-over", RefreshToken: "r"})
	if err != nil {
		t.Fatalf("build: %v", err)
	}
	if source == nil {
		t.Fatal("no token source was built")
	}

	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Token.AccessToken != "handed-over" {
		t.Errorf("the token from the exchange was not written down: %q", stored.Token.AccessToken)
	}
	// The client identifier is visible only at this moment. Dynamic
	// registration issues a fresh one on every registration, so a sign-out that
	// registered again would identify a different client and revoke nothing.
	if stored.ClientID != "registered-client" {
		t.Errorf("ClientID = %q, want the identifier the token was issued to; without it the sign-in can be forgotten but never revoked", stored.ClientID)
	}
}

func TestSavingTokenSourceIgnoresAnEmptyToken(t *testing.T) {
	store := testStore(t)
	build := savingTokenSource("hosted", store)

	if _, err := build(t.Context(), &oauth2.Config{}, nil); err != nil {
		t.Fatalf("build: %v", err)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("an empty token must not be written, got %v", err)
	}
}

func TestOAuthHandlerOnlyForRemoteServers(t *testing.T) {
	store := testStore(t)

	local := &ServerSpec{Name: "files", Command: "uvx"}
	session, err := oauthHandlerFor(local, store, signInDisallowed)
	if err != nil {
		t.Fatalf("oauthHandlerFor: %v", err)
	}
	if session != nil {
		t.Error("a stdio server has no authorization to do; a handler there is noise")
	}

	remote := &ServerSpec{Name: "hosted", URL: "https://mcp.example.com/v1"}
	session, err = oauthHandlerFor(remote, store, signInDisallowed)
	if err != nil {
		t.Fatalf("oauthHandlerFor: %v", err)
	}
	if session == nil {
		t.Fatal("a remote server must get a handler; only the server can say whether it needs authorization, and it says so with a 401")
	}
	session.close()
}

func TestSignInRequiredUnwraps(t *testing.T) {
	wrapped := fmt.Errorf("connect: %w", fmt.Errorf("transport: %w", ErrSignInRequired))
	if !SignInRequired(wrapped) {
		t.Error("the caller must be able to recognise a sign-in requirement however deeply the transport wraps it")
	}
	if SignInRequired(errors.New("something else")) {
		t.Error("an unrelated error must not read as a sign-in requirement")
	}
	if SignInRequired(nil) {
		t.Error("nil is not a sign-in requirement")
	}
}

// TestAnHTTPTransportCarriesTheOAuthHandler is the wiring itself. Everything
// else in this file describes a handler that nothing uses unless it reaches the
// transport, and the failure is silent: a server needing authorization would
// answer 401 and simply never connect.
func TestAnHTTPTransportCarriesTheOAuthHandler(t *testing.T) {
	spec := &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: "https://mcp.example.com/v1"}
	transport, release, err := newTransport(t.Context(), spec, transportOptions{
		tokens: testStore(t),
		signIn: signInDisallowed,
	})
	if err != nil {
		t.Fatalf("newTransport: %v", err)
	}
	defer release()

	streamable, ok := transport.(*sdk.StreamableClientTransport)
	if !ok {
		t.Fatalf("transport = %T, want a streamable http transport", transport)
	}
	if streamable.OAuthHandler == nil {
		t.Error("the http transport has no OAuth handler, so a server that asks for an authorization can never be signed in to")
	}
}

// TestAnHTTPTransportWithoutATokenStoreHasNoHandler keeps a build with no
// token store from signing a user in and then losing it.
func TestAnHTTPTransportWithoutATokenStoreHasNoHandler(t *testing.T) {
	spec := &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: "https://mcp.example.com/v1"}
	transport, release, err := newTransport(t.Context(), spec, transportOptions{})
	if err != nil {
		t.Fatalf("newTransport: %v", err)
	}
	defer release()

	streamable, ok := transport.(*sdk.StreamableClientTransport)
	if !ok {
		t.Fatalf("transport = %T, want a streamable http transport", transport)
	}
	if streamable.OAuthHandler != nil {
		t.Error("a handler was built with nowhere to keep the token it would obtain")
	}
}

// TestEveryTransportReturnsSomethingToRelease lets a caller defer the release
// without checking it, which is what stops a redirect listener leaking on the
// paths nobody thinks about.
func TestEveryTransportReturnsSomethingToRelease(t *testing.T) {
	cases := map[string]*ServerSpec{
		"stdio":   {Name: "files", Command: "go"},
		"http":    {Name: "hosted", Type: TransportHTTP, URL: "https://mcp.example.com/v1"},
		"neither": {Name: "broken"},
	}
	for name, spec := range cases {
		t.Run(name, func(t *testing.T) {
			_, release, _ := newTransport(t.Context(), spec, transportOptions{tokens: testStore(t)})
			if release == nil {
				t.Fatal("release is nil, so every caller must check before deferring and one of them will not")
			}
			release()
		})
	}
}
