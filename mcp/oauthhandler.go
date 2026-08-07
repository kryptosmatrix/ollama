package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
	"golang.org/x/oauth2"
)

// ErrSignInRequired reports that a remote server wants an authorization the
// user has not given. It is returned instead of opening a browser, because a
// server connecting in the background must never be able to summon a sign-in
// window the user did not ask for.
var ErrSignInRequired = errors.New("this MCP server needs you to sign in")

// signInMode says whether this connection is allowed to open a browser.
type signInMode int

const (
	// signInDisallowed is every ordinary connection: at start-up, on a
	// configuration change, on a reconnect. A 401 fails with ErrSignInRequired.
	signInDisallowed signInMode = iota
	// signInAllowed is only ever an explicit act by the user — the connect
	// button, or `ollama mcp login`.
	signInAllowed
)

// clientName is what a server shows the user on its consent screen.
const clientName = "Ollama"

// unusableRedirect is the redirect address declared by a connection that is
// not allowed to sign in. Nothing listens on it: port 80 on loopback is not
// bound by this process, and the fetcher refuses before any browser is opened,
// so the address is never reached.
const unusableRedirect = "http://127.0.0.1/callback"

// oauthSession is a configured OAuth handler and the resources it holds.
type oauthSession struct {
	handler sdkauth.OAuthHandler
	// redirectURL is the address declared to the authorization server. In an
	// ordinary connection it is unusableRedirect and nothing is bound to it.
	redirectURL string
	// fetch is what the handler calls when the server wants an authorization.
	// It is the only route to a browser.
	fetch sdkauth.AuthorizationCodeFetcher
	// redirect is the loopback listener, or nil when this connection is not
	// allowed to sign in and none was started.
	redirect *LoopbackRedirect
	// close releases the redirect listener. It is always non-nil.
	close func()
}

// newOAuthSession builds the OAuth handler for a remote server.
//
// The protocol library performs metadata discovery, dynamic client
// registration, PKCE, the token exchange and refresh. Three things are ours:
// which tokens it starts from, where refreshed tokens are written, and whether
// a browser may be opened at all.
func newOAuthSession(server string, store TokenStore, mode signInMode, open func(string) error) (*oauthSession, error) {
	if store == nil {
		return nil, errors.New("oauth needs somewhere to keep tokens")
	}

	session := &oauthSession{close: func() {}}

	config := &sdkauth.AuthorizationCodeHandlerConfig{
		// Ollama is a native application with no client secret and no fixed
		// address, so registration is dynamic where the server supports it.
		DynamicClientRegistrationConfig: &sdkauth.DynamicClientRegistrationConfig{
			Metadata: &oauthex.ClientRegistrationMetadata{
				ClientName:              clientName,
				GrantTypes:              []string{"authorization_code", "refresh_token"},
				ResponseTypes:           []string{"code"},
				TokenEndpointAuthMethod: "none",
			},
		},
		// Ollama can store refresh tokens, so ask for one: without it the user
		// is sent back to the browser every time the access token expires.
		RequestRefreshToken: true,
		NewTokenSource:      savingTokenSource(server, store),
	}

	// A token already stored means the ordinary case: use it, and never open a
	// browser at all.
	if record, err := store.Load(server); err == nil {
		config.InitialTokenSource = oauth2.StaticTokenSource(record.Token)
	} else if !errors.Is(err, ErrNoToken) {
		return nil, err
	}

	switch mode {
	case signInAllowed:
		redirect, err := StartLoopbackRedirect()
		if err != nil {
			return nil, err
		}
		if open != nil {
			redirect.Open = open
		}
		session.redirect = redirect
		session.close = func() { redirect.Close() }
		session.redirectURL = redirect.RedirectURL()
		session.fetch = redirect.Fetch
	default:
		// No listener is started and no browser can be opened. The redirect
		// address is still required by the configuration, so it is declared
		// and never used: the fetcher refuses before anything reaches it.
		session.redirectURL = unusableRedirect
		session.fetch = refuseSignIn(server)
	}
	config.RedirectURL = session.redirectURL
	config.AuthorizationCodeFetcher = session.fetch
	config.DynamicClientRegistrationConfig.Metadata.RedirectURIs = []string{session.redirectURL}

	handler, err := sdkauth.NewAuthorizationCodeHandler(config)
	if err != nil {
		session.close()
		return nil, fmt.Errorf("prepare sign-in for %s: %w", server, err)
	}
	session.handler = handler
	return session, nil
}

// refuseSignIn is the fetcher used by every connection that is not an explicit
// sign-in. It fails rather than opening a browser.
func refuseSignIn(server string) sdkauth.AuthorizationCodeFetcher {
	return func(context.Context, *sdkauth.AuthorizationArgs) (*sdkauth.AuthorizationResult, error) {
		return nil, fmt.Errorf("%w: %s", ErrSignInRequired, server)
	}
}

// savingTokenSource wraps the library's token source in one that writes every
// token back to the store, including the ones obtained by refreshing.
//
// Without this a token lives only in memory: the user is signed in until Ollama
// restarts and then is sent back to the browser, which looks like the sign-in
// not having worked.
//
// The freshly exchanged token is not written here. It was, and two attempts to
// falsify that write both passed: the transport consults the token source
// immediately after the exchange, so persistingTokenSource has already stored
// it by the time anything can observe the difference — including on the path
// where the connection then fails. Two writers for one fact, one of which
// cannot be tested, is worse than one that can.
func savingTokenSource(server string, store TokenStore) func(context.Context, *oauth2.Config, *oauth2.Token) (oauth2.TokenSource, error) {
	return func(ctx context.Context, config *oauth2.Config, token *oauth2.Token) (oauth2.TokenSource, error) {
		// config.ClientID is the identifier the library was issued when it
		// registered. This is the only moment it is visible to us, and without
		// it the sign-in can never be revoked.
		clientID := ""
		if config != nil {
			clientID = config.ClientID
		}
		return &persistingTokenSource{
			server:   server,
			store:    store,
			clientID: clientID,
			source:   config.TokenSource(ctx, token),
		}, nil
	}
}

// persistingTokenSource saves a token whenever the one underneath renews it.
type persistingTokenSource struct {
	server   string
	store    TokenStore
	clientID string
	source   oauth2.TokenSource
	last     string
}

func (p *persistingTokenSource) Token() (*oauth2.Token, error) {
	token, err := p.source.Token()
	if err != nil {
		return nil, err
	}
	// Only write when it has actually changed. A token source is consulted on
	// every request, and rewriting the file each time would be a great deal of
	// disk for no change.
	if token != nil && token.AccessToken != "" && token.AccessToken != p.last {
		if err := p.store.Save(p.server, &SignInRecord{Token: token, ClientID: p.clientID}); err != nil {
			return nil, err
		}
		p.last = token.AccessToken
	}
	return token, nil
}

// SignInRequired reports whether an error means the user must sign in, however
// deeply it is wrapped by the transport.
func SignInRequired(err error) bool {
	return errors.Is(err, ErrSignInRequired)
}

// issuerExpectation looks up what a server's authorization server has said
// about the RFC 9207 "iss" parameter: its issuer identifier, and whether it has
// committed to returning one.
//
// Both answers matter, for different reasons. The identifier is what a present
// iss is compared against — RFC 9207 §2.4 requires that comparison whenever the
// parameter arrives, which is the mix-up defence. Whether it was advertised
// decides only whether the value is then passed on to the protocol library,
// which refuses a sign-in outright when it sees an unadvertised one. Real
// services send an iss without advertising it: Sentry's hosted MCP server does.
//
// A discovery that fails answers ("", true): nothing to compare against, and
// the library's stricter handling left in place, so not knowing never quietly
// relaxes anything.
func issuerExpectation(ctx context.Context, serverURL string) (issuer string, advertised bool) {
	ctx, cancel := context.WithTimeout(ctx, signInHTTPTimeout)
	defer cancel()

	client := &http.Client{Timeout: signInHTTPTimeout}
	metadata, err := protectedResourceMetadata(ctx, serverURL, client)
	if err != nil || metadata == nil {
		return "", true
	}
	for _, candidate := range metadata.AuthorizationServers {
		server, err := sdkauth.GetAuthServerMetadata(ctx, candidate, client)
		if err != nil {
			return "", true
		}
		if server != nil {
			return server.Issuer, server.AuthorizationResponseIssParameterSupported
		}
	}
	return "", true
}
