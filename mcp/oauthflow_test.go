package mcp

import (
	"context"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
	"golang.org/x/oauth2"
)

// authorizedMCPServer is a hosted MCP server that requires an authorization,
// together with the authorization server that issues one. It implements enough
// of RFC 9728, RFC 8414, RFC 7591, RFC 7636 and RFC 7009 to drive Ollama's own
// sign-in from end to end.
//
// It exists because every other test in this package proves one link of the
// chain. A discovery that works, a redirect that works and an exchange that
// works still do not add up to a sign-in: what breaks in practice is the joins
// — a redirect address that is declared but not bound, a verifier that is
// generated and not sent, a token that is obtained and not stored. Only a run
// through the whole flow catches those.
//
// Every request is recorded so the test can assert what Ollama actually sent
// rather than that it ended up somewhere plausible.
type authorizedMCPServer struct {
	*httptest.Server
	t *testing.T

	mu sync.Mutex
	// codes maps an issued authorization code to the PKCE challenge it was
	// issued against, so the token endpoint can verify the verifier.
	codes map[string]issuedCode
	// tokens are the access tokens this server will accept.
	tokens map[string]bool
	// refreshTokens map a refresh token to the client it was issued to.
	refreshTokens map[string]string
	clients       map[string]bool

	authorizeRequests []url.Values
	tokenRequests     []url.Values
	revocations       []url.Values
	registrations     []map[string]any
	bearerSeen        []string

	// nextTokenLifetime is how long the next issued access token is valid. Zero
	// means no expiry is declared.
	nextTokenLifetime time.Duration
	// breakAfterAuthorization makes the MCP endpoint fail once a valid token
	// arrives, so a test can separate "the sign-in worked" from "the
	// connection worked".
	breakAfterAuthorization bool
	// sendIssuer returns the RFC 9207 "iss" parameter in the authorization
	// response. advertiseIssuer declares support for it in the metadata. Real
	// services exist that do the first without the second.
	sendIssuer      bool
	advertiseIssuer bool
	// issuerOverride returns a different issuer in the authorization response
	// than the one this server publishes, which is the mix-up case.
	issuerOverride string
	// revocationOverride advertises a revocation endpoint somewhere other than
	// this server, so a test can put an address on a real network into the
	// metadata.
	revocationOverride string
	// noRevocation withholds this server's revocation endpoint from its
	// metadata, which is how a test builds an issuer that cannot revoke.
	noRevocation bool
	// listAlso names another authorization server to advertise after this one
	// in the protected-resource metadata. The protocol library signs in
	// against the first, so this one is only ever a bystander.
	listAlso string
	// issuedTokens are the access tokens this server has minted, in order.
	issuedTokens []string
}

type issuedCode struct {
	challenge   string
	method      string
	clientID    string
	redirectURI string
	resource    string
}

func newAuthorizedMCPServer(t *testing.T, tools ...*sdk.Tool) *authorizedMCPServer {
	t.Helper()

	fake := &authorizedMCPServer{
		t:             t,
		codes:         map[string]issuedCode{},
		tokens:        map[string]bool{},
		refreshTokens: map[string]string{},
		clients:       map[string]bool{},
	}

	server := sdk.NewServer(&sdk.Implementation{Name: "hosted", Version: "1.0.0"}, nil)
	for _, tool := range tools {
		sdk.AddTool(server, tool, func(ctx context.Context, req *sdk.CallToolRequest, args map[string]any) (*sdk.CallToolResult, any, error) {
			return &sdk.CallToolResult{Content: []sdk.Content{&sdk.TextContent{Text: "called " + req.Params.Name}}}, nil, nil
		})
	}
	mcpHandler := sdk.NewStreamableHTTPHandler(func(*http.Request) *sdk.Server { return server }, nil)

	mux := http.NewServeMux()
	fake.Server = httptest.NewServer(mux)
	t.Cleanup(fake.Close)

	// The MCP endpoint itself, behind a bearer check. An unauthenticated
	// request gets the 401 and the WWW-Authenticate challenge that starts the
	// whole flow — this is the only way a client learns authorization is
	// required at all.
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		bearer := strings.TrimPrefix(r.Header.Get("Authorization"), "Bearer ")
		fake.mu.Lock()
		fake.bearerSeen = append(fake.bearerSeen, bearer)
		ok := bearer != "" && fake.tokens[bearer]
		fake.mu.Unlock()

		if !ok {
			w.Header().Set("WWW-Authenticate", fmt.Sprintf(
				`Bearer resource_metadata="%s/.well-known/oauth-protected-resource/mcp"`, fake.URL))
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		fake.mu.Lock()
		broken := fake.breakAfterAuthorization
		fake.mu.Unlock()
		if broken {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		mcpHandler.ServeHTTP(w, r)
	})

	mux.HandleFunc("/.well-known/oauth-protected-resource/mcp", func(w http.ResponseWriter, r *http.Request) {
		fake.mu.Lock()
		servers := []string{fake.URL}
		if fake.listAlso != "" {
			servers = []string{fake.URL, fake.listAlso}
		}
		fake.mu.Unlock()
		writeJSON(w, map[string]any{
			"resource":              fake.URL + "/mcp",
			"authorization_servers": servers,
			"scopes_supported":      []string{"mcp.read"},
		})
	})

	mux.HandleFunc("/.well-known/oauth-authorization-server", func(w http.ResponseWriter, r *http.Request) {
		meta := map[string]any{
			"issuer":                                fake.URL,
			"authorization_endpoint":                fake.URL + "/authorize",
			"token_endpoint":                        fake.URL + "/token",
			"registration_endpoint":                 fake.URL + "/register",
			"code_challenge_methods_supported":      []string{"S256"},
			"token_endpoint_auth_methods_supported": []string{"none"},
			"scopes_supported":                      []string{"mcp.read", "offline_access"},
			"grant_types_supported":                 []string{"authorization_code", "refresh_token"},
		}
		fake.mu.Lock()
		advertise := fake.advertiseIssuer
		withhold := fake.noRevocation
		fake.mu.Unlock()
		if !withhold {
			meta["revocation_endpoint"] = fake.revocationEndpoint()
		}
		if advertise {
			meta["authorization_response_iss_parameter_supported"] = true
		}
		writeJSON(w, meta)
	})

	// RFC 7591 dynamic client registration.
	mux.HandleFunc("/register", func(w http.ResponseWriter, r *http.Request) {
		var metadata map[string]any
		if err := json.NewDecoder(r.Body).Decode(&metadata); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fake.mu.Lock()
		fake.registrations = append(fake.registrations, metadata)
		clientID := fmt.Sprintf("client-%d", len(fake.registrations))
		fake.clients[clientID] = true
		fake.mu.Unlock()

		w.WriteHeader(http.StatusCreated)
		writeJSON(w, map[string]any{
			"client_id":                  clientID,
			"client_id_issued_at":        1700000000,
			"token_endpoint_auth_method": "none",
			"redirect_uris":              metadata["redirect_uris"],
		})
	})

	// The authorization endpoint. There is no consent screen: this stands in
	// for a user who is already signed in at the service and approves at once,
	// which is also the timing that catches a redirect listener bound too late.
	mux.HandleFunc("/authorize", func(w http.ResponseWriter, r *http.Request) {
		query := r.URL.Query()
		fake.mu.Lock()
		fake.authorizeRequests = append(fake.authorizeRequests, query)
		code := fmt.Sprintf("code-%d", len(fake.authorizeRequests))
		fake.codes[code] = issuedCode{
			challenge:   query.Get("code_challenge"),
			method:      query.Get("code_challenge_method"),
			clientID:    query.Get("client_id"),
			redirectURI: query.Get("redirect_uri"),
			resource:    query.Get("resource"),
		}
		fake.mu.Unlock()

		redirect, err := url.Parse(query.Get("redirect_uri"))
		if err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		back := redirect.Query()
		back.Set("code", code)
		back.Set("state", query.Get("state"))
		fake.mu.Lock()
		if fake.sendIssuer {
			issuer := fake.URL
			if fake.issuerOverride != "" {
				issuer = fake.issuerOverride
			}
			back.Set("iss", issuer)
		}
		fake.mu.Unlock()
		redirect.RawQuery = back.Encode()
		http.Redirect(w, r, redirect.String(), http.StatusFound)
	})

	mux.HandleFunc("/token", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fake.mu.Lock()
		fake.tokenRequests = append(fake.tokenRequests, r.PostForm)
		fake.mu.Unlock()

		switch r.PostForm.Get("grant_type") {
		case "authorization_code":
			fake.exchangeCode(w, r.PostForm)
		case "refresh_token":
			fake.refresh(w, r.PostForm)
		default:
			oauthError(w, "unsupported_grant_type")
		}
	})

	mux.HandleFunc("/revoke", func(w http.ResponseWriter, r *http.Request) {
		if err := r.ParseForm(); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		fake.mu.Lock()
		fake.revocations = append(fake.revocations, r.PostForm)
		token := r.PostForm.Get("token")
		if client, ok := fake.refreshTokens[token]; ok {
			delete(fake.refreshTokens, token)
			// RFC 7009 §2.1: revoking a refresh token SHOULD invalidate the
			// access tokens issued from it.
			for access := range fake.tokens {
				if strings.HasPrefix(access, client+":") {
					delete(fake.tokens, access)
				}
			}
		}
		delete(fake.tokens, token)
		fake.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	})

	return fake
}

// exchangeCode is the PKCE check. A client that did not send a verifier, or
// sent one that does not hash to the challenge it committed to, gets nothing.
func (f *authorizedMCPServer) exchangeCode(w http.ResponseWriter, form url.Values) {
	f.mu.Lock()
	defer f.mu.Unlock()

	issued, ok := f.codes[form.Get("code")]
	if !ok {
		oauthError(w, "invalid_grant")
		return
	}
	// A code is good once.
	delete(f.codes, form.Get("code"))

	if issued.method != "S256" {
		oauthError(w, "invalid_request")
		return
	}
	verifier := form.Get("code_verifier")
	if verifier == "" || pkceChallenge(verifier) != issued.challenge {
		oauthError(w, "invalid_grant")
		return
	}
	if form.Get("redirect_uri") != issued.redirectURI {
		oauthError(w, "invalid_grant")
		return
	}

	f.issueLocked(w, issued.clientID)
}

func (f *authorizedMCPServer) refresh(w http.ResponseWriter, form url.Values) {
	f.mu.Lock()
	defer f.mu.Unlock()

	client, ok := f.refreshTokens[form.Get("refresh_token")]
	if !ok {
		oauthError(w, "invalid_grant")
		return
	}
	delete(f.refreshTokens, form.Get("refresh_token"))
	f.issueLocked(w, client)
}

// issueLocked mints an access token and a refresh token. The caller holds mu.
func (f *authorizedMCPServer) issueLocked(w http.ResponseWriter, clientID string) {
	issue := len(f.issuedTokens) + 1
	access := fmt.Sprintf("%s:access-%d", clientID, issue)
	refresh := fmt.Sprintf("%s:refresh-%d", clientID, issue)
	f.issuedTokens = append(f.issuedTokens, access)
	f.tokens[access] = true
	f.refreshTokens[refresh] = clientID

	body := map[string]any{
		"access_token":  access,
		"token_type":    "Bearer",
		"refresh_token": refresh,
	}
	// A negative lifetime is issued as a negative expires_in, which is how this
	// server hands out an already-expired token. Only zero means "no expiry
	// declared".
	if f.nextTokenLifetime != 0 {
		body["expires_in"] = int(f.nextTokenLifetime.Seconds())
	}
	writeJSON(w, body)
}

func oauthError(w http.ResponseWriter, code string) {
	w.WriteHeader(http.StatusBadRequest)
	writeJSON(w, map[string]any{"error": code})
}

func pkceChallenge(verifier string) string {
	sum := sha256.Sum256([]byte(verifier))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// revocationEndpoint is this server's own /revoke unless a test has pointed it
// somewhere else.
func (f *authorizedMCPServer) revocationEndpoint() string {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.revocationOverride != "" {
		return f.revocationOverride
	}
	return f.URL + "/revoke"
}

func (f *authorizedMCPServer) spec() *ServerSpec {
	return &ServerSpec{Name: "hosted", Type: TransportHTTP, URL: f.URL + "/mcp"}
}

// browser stands in for the user's browser. It follows the authorization
// server's redirect, which lands the authorization code on Ollama's own
// loopback listener exactly as a real browser would.
//
// If the listener were not bound before the browser was sent anywhere, this is
// where it would fail — which is the point of driving it this way rather than
// posting to the callback directly.
func (f *authorizedMCPServer) browser() func(string) error {
	return func(rawURL string) error {
		client := &http.Client{Timeout: 10 * time.Second}
		resp, err := client.Get(rawURL)
		if err != nil {
			return err
		}
		defer resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			return fmt.Errorf("the sign-in page answered %s", resp.Status)
		}
		return nil
	}
}

// issuedAccessTokens is every access token minted so far, in order.
func (f *authorizedMCPServer) issuedAccessTokens() []string {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]string(nil), f.issuedTokens...)
}

func (f *authorizedMCPServer) snapshot() (authorize, token, revoke []url.Values, registrations []map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]url.Values(nil), f.authorizeRequests...),
		append([]url.Values(nil), f.tokenRequests...),
		append([]url.Values(nil), f.revocations...),
		append([]map[string]any(nil), f.registrations...)
}

func flowManager(t *testing.T, fake *authorizedMCPServer) (*Manager, *FileTokenStore) {
	t.Helper()
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	manager := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser:    fake.browser(),
	})
	t.Cleanup(func() { manager.Close() })
	return manager, store
}

// TestTheWholeSignInFlow drives authorize → callback → exchange → connect
// against a server that actually refuses an unauthenticated request, and
// checks what was sent at every step.
func TestTheWholeSignInFlow(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	manager, store := flowManager(t, fake)
	spec := fake.spec()

	state, err := manager.SignIn(t.Context(), spec)
	if err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	if state.Status != StatusConnected {
		t.Fatalf("status = %q (%v), want connected", state.Status, state.Err)
	}
	if len(state.Tools) != 1 || state.Tools[0].Name != "alpha" {
		t.Errorf("tools = %+v, want the server's one tool; a sign-in that authorizes but does not connect is not finished", state.Tools)
	}

	authorize, token, _, registrations := fake.snapshot()

	if len(registrations) != 1 {
		t.Fatalf("registrations = %d, want 1", len(registrations))
	}
	if got := registrations[0]["client_name"]; got != clientName {
		t.Errorf("client_name = %v, want %q; this is what the service shows the user on its consent screen", got, clientName)
	}

	if len(authorize) != 1 {
		t.Fatalf("authorization requests = %d, want 1", len(authorize))
	}
	request := authorize[0]
	if got := request.Get("code_challenge_method"); got != "S256" {
		t.Errorf("code_challenge_method = %q, want S256", got)
	}
	if request.Get("code_challenge") == "" {
		t.Error("no PKCE challenge was sent; without one an intercepted code can be redeemed by anyone")
	}
	if request.Get("state") == "" {
		t.Error("no state was sent; without one the callback cannot be tied to this sign-in")
	}
	// The redirect must be loopback: an authorization code arriving over the
	// network is an authorization code anyone on it can take.
	redirect := request.Get("redirect_uri")
	if !strings.HasPrefix(redirect, "http://127.0.0.1:") {
		t.Errorf("redirect_uri = %q, want a loopback address", redirect)
	}
	if resource := request.Get("resource"); resource != spec.URL {
		t.Errorf("resource = %q, want %q; the token must be bound to the server it is for", resource, spec.URL)
	}

	// The exchange. The fake refuses a verifier that does not hash to the
	// committed challenge, so reaching a connected state already proves one was
	// sent and matched — this asserts it explicitly so the reason is legible.
	if len(token) != 1 {
		t.Fatalf("token requests = %d, want 1", len(token))
	}
	verifier := token[0].Get("code_verifier")
	if verifier == "" {
		t.Fatal("no PKCE verifier was sent to the token endpoint")
	}
	if pkceChallenge(verifier) != request.Get("code_challenge") {
		t.Error("the verifier does not match the challenge that was committed to")
	}

	// And the token was written down, with the identifier it was issued to.
	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Token.AccessToken == "" {
		t.Error("no access token was stored, so the next launch would sign in again")
	}
	if stored.Token.RefreshToken == "" {
		t.Error("no refresh token was stored; the user is sent back to the browser at the first expiry")
	}
	if stored.ClientID != "client-1" {
		t.Errorf("ClientID = %q, want the registered client; without it the sign-in cannot be revoked", stored.ClientID)
	}
}

// TestAStoredSignInConnectsWithoutABrowser is the whole point of storing one.
// A second launch must reach the same connected state with no sign-in at all.
func TestAStoredSignInConnectsWithoutABrowser(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	manager, store := flowManager(t, fake)
	spec := fake.spec()

	if _, err := manager.SignIn(t.Context(), spec); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	authorizeAfterSignIn, _, _, _ := fake.snapshot()

	// A second manager, as a restart would build, over the same store — and
	// with a browser that fails if it is opened at all.
	restarted := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser: func(string) error {
			t.Error("an ordinary connection opened a browser")
			return fmt.Errorf("no browser may be opened here")
		},
	})
	t.Cleanup(func() { restarted.Close() })

	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	restarted.Connect(t.Context(), cfg)

	state, _ := restarted.State(spec.Name)
	if state.Status != StatusConnected {
		t.Fatalf("status = %q (%v), want connected from the stored token alone", state.Status, state.Err)
	}

	authorizeAfterRestart, _, _, _ := fake.snapshot()
	if len(authorizeAfterRestart) != len(authorizeAfterSignIn) {
		t.Errorf("the restart began another authorization: %d requests, was %d", len(authorizeAfterRestart), len(authorizeAfterSignIn))
	}
}

// TestAnExpiredTokenIsRefreshedAndWrittenDown covers the join between the
// protocol library's refresh and Ollama's store. A refresh that lives only in
// memory keeps the user signed in until Ollama restarts and then sends them
// back to the browser, which reads as the sign-in not having worked.
//
// The first token is issued already expired, so the refresh happens during the
// connect itself rather than at some later call. An earlier version of this
// test compared the stored token before and after a tool call and failed —
// because by the time a sign-in returns, the refresh has already happened. It
// now checks against what the server actually minted, which does not depend on
// when the refresh falls.
func TestAnExpiredTokenIsRefreshedAndWrittenDown(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	fake.nextTokenLifetime = -time.Minute

	manager, store := flowManager(t, fake)
	spec := fake.spec()

	if _, err := manager.SignIn(t.Context(), spec); err != nil {
		t.Fatalf("SignIn: %v", err)
	}

	minted := fake.issuedAccessTokens()
	if len(minted) < 2 {
		t.Fatalf("the server minted %d tokens, want at least 2; an expired token was accepted without being refreshed", len(minted))
	}

	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Token.AccessToken != minted[len(minted)-1] {
		t.Errorf("stored token = %q, want the most recently issued %q; a refresh that is not written down is lost at the next launch",
			stored.Token.AccessToken, minted[len(minted)-1])
	}
	if stored.Token.AccessToken == minted[0] {
		t.Error("the expired token is still the stored one")
	}
	if stored.ClientID != "client-1" {
		t.Errorf("ClientID = %q, want it kept across the refresh; without it the sign-in cannot be revoked", stored.ClientID)
	}

	_, tokenRequests, _, _ := fake.snapshot()
	var refreshes int
	for _, request := range tokenRequests {
		if request.Get("grant_type") == "refresh_token" {
			refreshes++
		}
	}
	if refreshes == 0 {
		t.Error("no refresh_token grant reached the token endpoint, so the new token came from somewhere else")
	}
}

// TestATokenSurvivesAConnectionThatFailsAfterTheExchange separates the two
// halves of a sign-in. The user has been to the browser and the service has
// issued a credential; if the connection then fails — the server restarts, the
// network drops — that credential must still be on disk. Losing it sends them
// back through the browser to obtain something they already have.
func TestATokenSurvivesAConnectionThatFailsAfterTheExchange(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	fake.breakAfterAuthorization = true

	manager, store := flowManager(t, fake)

	if _, err := manager.SignIn(t.Context(), fake.spec()); err == nil {
		t.Fatal("the connection was meant to fail after the exchange")
	}

	_, tokenRequests, _, _ := fake.snapshot()
	if len(tokenRequests) == 0 {
		t.Fatal("no token was ever exchanged, so this test proves nothing about keeping one")
	}

	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("the token from a completed sign-in was lost when the connection failed: %v", err)
	}
	if stored.Token.AccessToken == "" {
		t.Error("an empty token was stored")
	}
	if stored.ClientID == "" {
		t.Error("the client identifier was lost, so this sign-in can never be revoked")
	}
}

// TestSigningOutRevokesTheSignInThatWasJustMade closes the loop: the token this
// flow obtained is the token that gets withdrawn, and the server stops
// accepting it.
func TestSigningOutRevokesTheSignInThatWasJustMade(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	manager, store := flowManager(t, fake)
	spec := fake.spec()

	if _, err := manager.SignIn(t.Context(), spec); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}

	if err := manager.SignOut(t.Context(), spec); err != nil {
		t.Fatalf("SignOut: %v", err)
	}

	_, _, revocations, _ := fake.snapshot()
	if len(revocations) != 1 {
		t.Fatalf("revocations = %d, want 1", len(revocations))
	}
	if got := revocations[0].Get("token"); got != stored.Token.RefreshToken {
		t.Errorf("revoked %q, want the refresh token this sign-in obtained", got)
	}
	if got := revocations[0].Get("client_id"); got != stored.ClientID {
		t.Errorf("client_id = %q, want the client the token was issued to (%q)", got, stored.ClientID)
	}

	// The server has stopped accepting it, which is what revocation means.
	fake.mu.Lock()
	stillValid := fake.tokens[stored.Token.AccessToken]
	fake.mu.Unlock()
	if stillValid {
		t.Error("the server still accepts the access token after a sign-out")
	}

	if state, _ := manager.State(spec.Name); state.Status != StatusNeedsSignIn {
		t.Errorf("status = %q, want needs-sign-in after signing out", state.Status)
	}
}

// TestACallbackWithTheWrongStateIsNotThisSignIn is the flow-level form of the
// state check. Anything on the machine can reach a loopback port, so a stray
// callback must neither complete a sign-in nor abort one — and the real one
// arriving afterwards must still work.
func TestACallbackWithTheWrongStateIsNotThisSignIn(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}

	manager := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser: func(rawURL string) error {
			parsed, err := url.Parse(rawURL)
			if err != nil {
				return err
			}
			redirect := parsed.Query().Get("redirect_uri")

			// An impostor gets in first with a code of its own choosing.
			client := &http.Client{Timeout: 10 * time.Second}
			impostor := redirect + "?code=stolen&state=not-this-flow"
			resp, err := client.Get(impostor)
			if err != nil {
				return err
			}
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				t.Errorf("a callback carrying another flow's state was accepted (%s)", resp.Status)
			}

			// Then the real browser completes the sign-in.
			return fake.browser()(rawURL)
		},
	})
	t.Cleanup(func() { manager.Close() })

	state, err := manager.SignIn(t.Context(), fake.spec())
	if err != nil {
		t.Fatalf("SignIn: %v; a stray callback must not be able to abort a sign-in in progress", err)
	}
	if state.Status != StatusConnected {
		t.Fatalf("status = %q (%v), want connected", state.Status, state.Err)
	}

	_, tokenRequests, _, _ := fake.snapshot()
	for _, request := range tokenRequests {
		if request.Get("code") == "stolen" {
			t.Fatal("the impostor's code was redeemed")
		}
	}
}

// TestASignInThatIsNeverCompletedGivesUp keeps an abandoned sign-in from
// holding a loopback port for the life of the process.
func TestASignInThatIsNeverCompletedGivesUp(t *testing.T) {
	fake := newAuthorizedMCPServer(t)
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}

	manager := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		// The user opens the page and never finishes.
		OpenBrowser: func(string) error { return nil },
	})
	t.Cleanup(func() { manager.Close() })

	ctx, cancel := context.WithTimeout(t.Context(), 2*time.Second)
	defer cancel()

	if _, err := manager.SignIn(ctx, fake.spec()); err == nil {
		t.Fatal("a sign-in nobody completes must not succeed")
	}
	if _, err := store.Load("hosted"); err == nil {
		t.Error("a token was stored for a sign-in that never completed")
	}
	// And the slot is free again, so the user can try once more.
	if manager.SigningIn("hosted") {
		t.Error("the sign-in slot was not released, so a second attempt is refused for ever")
	}
}

// TestAServerThatNeedsASignInSaysSoRatherThanFailing is the ordinary path
// against a server that requires authorization: no browser, no failure, and a
// status that names the one thing that resolves it.
func TestAServerThatNeedsASignInSaysSoRatherThanFailing(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}

	manager := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser: func(string) error {
			t.Error("an ordinary connection opened a browser")
			return fmt.Errorf("no browser may be opened here")
		},
	})
	t.Cleanup(func() { manager.Close() })

	spec := fake.spec()
	cfg := &Config{}
	cfg.Set(spec.Name, spec)
	manager.Connect(t.Context(), cfg)

	state, _ := manager.State(spec.Name)
	if state.Status != StatusNeedsSignIn {
		t.Fatalf("status = %q (%v), want needs-sign-in", state.Status, state.Err)
	}
	if !SignInRequired(state.Err) {
		t.Errorf("err = %v, want it to report that a sign-in is needed", state.Err)
	}

	authorize, _, _, registrations := fake.snapshot()
	if len(authorize) != 0 {
		t.Errorf("an ordinary connection began %d authorizations", len(authorize))
	}
	if len(registrations) != 0 {
		t.Errorf("an ordinary connection registered a client %d times", len(registrations))
	}
}

// TestASignInSurvivesAnUnadvertisedIssuer is a regression test for a real
// service. Sentry's hosted MCP server returns the RFC 9207 "iss" parameter in
// its authorization response while its metadata does not advertise
// authorization_response_iss_parameter_supported, and the protocol library
// refuses the entire sign-in when that happens — after the user has been to
// the browser and come back.
//
// RFC 9207 asks a client not to *rely* on an unadvertised iss. It does not ask
// it to reject one, and rejecting it makes the server unusable for no gain:
// a server that never committed to sending an issuer offers no mix-up defence
// whether it sends one or not. So an iss that nobody promised is dropped at the
// callback.
func TestASignInSurvivesAnUnadvertisedIssuer(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	fake.sendIssuer = true
	fake.advertiseIssuer = false

	manager, store := flowManager(t, fake)
	state, err := manager.SignIn(t.Context(), fake.spec())
	if err != nil {
		t.Fatalf("SignIn: %v\nthis is the Sentry failure: a server that sends an issuer it never advertised must not break the sign-in", err)
	}
	if state.Status != StatusConnected {
		t.Fatalf("status = %q (%v), want connected", state.Status, state.Err)
	}
	if _, err := store.Load("hosted"); err != nil {
		t.Errorf("nothing was stored: %v", err)
	}
}

// TestAnAdvertisedIssuerIsStillChecked is the other half, and the reason the
// fix is conditional rather than a blanket drop. When a server does commit to
// returning an issuer, that value is the mix-up defence and must keep being
// validated — dropping it everywhere would trade one broken server for a
// silently weakened check on every compliant one.
func TestAnAdvertisedIssuerIsStillChecked(t *testing.T) {
	t.Run("advertised and sent", func(t *testing.T) {
		fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
		fake.sendIssuer = true
		fake.advertiseIssuer = true

		manager, _ := flowManager(t, fake)
		state, err := manager.SignIn(t.Context(), fake.spec())
		if err != nil {
			t.Fatalf("SignIn: %v", err)
		}
		if state.Status != StatusConnected {
			t.Fatalf("status = %q (%v), want connected", state.Status, state.Err)
		}
	})

	t.Run("advertised and withheld", func(t *testing.T) {
		fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
		fake.sendIssuer = false
		fake.advertiseIssuer = true

		manager, _ := flowManager(t, fake)
		if _, err := manager.SignIn(t.Context(), fake.spec()); err == nil {
			t.Fatal("a server that advertised the issuer parameter and then withheld it must not be accepted; that is the mix-up case the check exists for")
		}
	})
}

// TestIssuerExpectationAnswersStrictlyWhenItCannotTell covers the lookup the
// callback depends on. Its answers are the whole decision: what a present
// issuer is compared against, and whether the value is then passed on to a
// library that refuses an unadvertised one.
func TestIssuerExpectationAnswersStrictlyWhenItCannotTell(t *testing.T) {
	t.Run("advertised", func(t *testing.T) {
		fake := newAuthorizedMCPServer(t)
		fake.advertiseIssuer = true
		issuer, advertised := issuerExpectation(t.Context(), fake.spec().URL)
		if !advertised {
			t.Error("a server that advertises the issuer parameter must have it passed on and checked by the library too")
		}
		if issuer != fake.URL {
			t.Errorf("issuer = %q, want %q; without it there is nothing to compare a callback against", issuer, fake.URL)
		}
	})

	t.Run("not advertised", func(t *testing.T) {
		fake := newAuthorizedMCPServer(t)
		issuer, advertised := issuerExpectation(t.Context(), fake.spec().URL)
		if advertised {
			t.Error("a server that never advertised it would have its issuer passed to a library that refuses the sign-in over it")
		}
		// Still learned, because RFC 9207 asks for the comparison whenever an
		// issuer arrives — not only when it was promised.
		if issuer != fake.URL {
			t.Errorf("issuer = %q, want %q; an unadvertised issuer still has to be compared against something", issuer, fake.URL)
		}
	})

	t.Run("no metadata at all", func(t *testing.T) {
		silent := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
		}))
		defer silent.Close()
		issuer, advertised := issuerExpectation(t.Context(), silent.URL+"/mcp")
		if !advertised {
			t.Error("not knowing must answer strictly; guessing the laxer way turns an unreadable metadata document into a silently weakened check")
		}
		if issuer != "" {
			t.Errorf("issuer = %q, want empty when nothing could be read", issuer)
		}
	})
}

// TestAMismatchedIssuerEndsTheSignIn is the mix-up attack RFC 9207 exists for,
// and it is the case an earlier version of this fix got wrong: the issuer was
// dropped whenever the server had not advertised it, so a callback naming a
// different authorization server was accepted without a murmur. Section 2.4
// requires the comparison whenever the parameter is present.
func TestAMismatchedIssuerEndsTheSignIn(t *testing.T) {
	for name, advertise := range map[string]bool{
		"advertised":     true,
		"not advertised": false,
	} {
		t.Run(name, func(t *testing.T) {
			fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
			fake.sendIssuer = true
			fake.advertiseIssuer = advertise
			fake.issuerOverride = "https://not-this-server.example.com"

			manager, store := flowManager(t, fake)
			if _, err := manager.SignIn(t.Context(), fake.spec()); err == nil {
				t.Fatal("a response from a different authorization server must end the sign-in")
			}
			if _, err := store.Load("hosted"); err == nil {
				t.Error("a token was stored from a mixed-up authorization response")
			}
		})
	}
}

// TestSigningOutRefusesToPostATokenOverPlainHTTP drives the check through the
// real sign-out rather than calling the predicate directly. An earlier version
// of this test exercised requireSecureEndpoint on its own and passed happily
// with the call removed from revokeSignIn — a predicate nothing consults is not
// a protection.
func TestSigningOutRefusesToPostATokenOverPlainHTTP(t *testing.T) {
	fake := newAuthorizedMCPServer(t, simpleTool("alpha"))
	fake.revocationOverride = "http://revocation.example.com/revoke"

	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	if err := store.Save("hosted", &SignInRecord{
		Token:    &oauth2.Token{AccessToken: "access-1", RefreshToken: "refresh-1"},
		ClientID: "client-1",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	manager := NewManager(Options{ConnectTimeout: 20 * time.Second, Approvals: allowAll{}, Tokens: store})
	t.Cleanup(func() { manager.Close() })

	err := manager.SignOut(t.Context(), fake.spec())
	if !errors.Is(err, ErrSignedOutLocallyOnly) {
		t.Fatalf("err = %v, want ErrSignedOutLocallyOnly; the token must not be posted to an http endpoint", err)
	}
	if !strings.Contains(err.Error(), "https") {
		t.Errorf("err = %v, want it to say why the endpoint was refused", err)
	}
	// The token is still forgotten locally, as every sign-out does.
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token survived the sign-out: %v", err)
	}
}

// TestRevocationGoesToTheServerThatIssuedTheToken. A protected resource may
// name several authorization servers. Revocation walked that list and used the
// first one advertising a revocation endpoint, while the record did not say
// which had issued the token.
//
// The scenario that exposes it: the issuing server offers no revocation
// endpoint and a second one on the list does. The old path posted the token to
// the second — a server that never issued it — and RFC 7009 §2.2 has such a
// server answer 200 for a token it has never heard of, so the user was told
// they were signed out while the token stayed live at the server that issued
// it. Being told the truth, that it could not be revoked, is strictly better
// than being told a comfortable lie.
func TestRevocationGoesToTheServerThatIssuedTheToken(t *testing.T) {
	issuer := newAuthorizedMCPServer(t, simpleTool("alpha"))
	issuer.noRevocation = true // the issuing server offers no revocation endpoint

	// A bystander that does advertise one, and issued nothing.
	var strangerMu sync.Mutex
	var strangerCalls int
	var stranger *httptest.Server
	stranger = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/.well-known/oauth-authorization-server":
			writeJSON(w, map[string]any{
				"issuer":                           stranger.URL,
				"authorization_endpoint":           stranger.URL + "/authorize",
				"token_endpoint":                   stranger.URL + "/token",
				"revocation_endpoint":              stranger.URL + "/revoke",
				"code_challenge_methods_supported": []string{"S256"},
			})
		case "/revoke":
			strangerMu.Lock()
			strangerCalls++
			strangerMu.Unlock()
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	}))
	defer stranger.Close()
	issuer.listAlso = stranger.URL

	store := &FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	manager := NewManager(Options{
		ConnectTimeout: 20 * time.Second,
		Approvals:      allowAll{},
		Tokens:         store,
		OpenBrowser:    issuer.browser(),
	})
	t.Cleanup(func() { manager.Close() })

	if _, err := manager.SignIn(t.Context(), issuer.spec()); err != nil {
		t.Fatalf("SignIn: %v", err)
	}
	stored, err := store.Load("hosted")
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if stored.Issuer != issuer.URL {
		t.Fatalf("Issuer = %q, want %q recorded at sign-in", stored.Issuer, issuer.URL)
	}

	err = manager.SignOut(t.Context(), issuer.spec())
	if !errors.Is(err, ErrSignedOutLocallyOnly) {
		t.Fatalf("err = %v, want ErrSignedOutLocallyOnly: the issuing server cannot revoke, and saying otherwise is the lie", err)
	}

	strangerMu.Lock()
	seen := strangerCalls
	strangerMu.Unlock()
	if seen != 0 {
		t.Errorf("the token was posted to a server that never issued it (%d times)", seen)
	}
	if _, err := store.Load("hosted"); !errors.Is(err, ErrNoToken) {
		t.Errorf("the token survived the sign-out: %v", err)
	}
}
