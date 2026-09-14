package mcp

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
	"github.com/modelcontextprotocol/go-sdk/oauthex"
)

// ErrSignedOutLocallyOnly reports that the stored token was deleted from this
// machine but could not be withdrawn at the authorization server, so it may
// still be valid there. The user is told, because the remedy is theirs: revoke
// Ollama's access in the service's own account settings.
var ErrSignedOutLocallyOnly = errors.New("signed out on this machine, but the token could not be revoked at the server")

// signInHTTPTimeout bounds each discovery and revocation request.
const signInHTTPTimeout = 30 * time.Second

// SignIn connects a remote server, opening a browser if the server asks for an
// authorization. It is the only path in this package that may open one: every
// other connection — at start-up, on a configuration change, on a reconnect —
// refuses with ErrSignInRequired rather than summoning a window nobody asked
// for.
//
// It is therefore only ever called from an explicit act of the user's: the
// connect button, or `ollama mcp login`.
func (m *Manager) SignIn(ctx context.Context, spec *ServerSpec) (ServerState, error) {
	if spec == nil {
		return ServerState{}, errors.New("no server to sign in to")
	}
	if spec.transport() != TransportHTTP {
		return ServerState{}, fmt.Errorf("%s runs on this machine and has nothing to sign in to", spec.Name)
	}
	if m.opts.Tokens == nil {
		return ServerState{}, errors.New("there is nowhere to keep a token, so signing in would be lost immediately")
	}
	// Signing in still runs through the approval gate. Approval is what says
	// this server may be contacted at all, and a sign-in contacts it.
	if !m.approves(spec) {
		return ServerState{}, fmt.Errorf("%s has not been approved to run: %s", spec.Name, spec.Summary())
	}

	// One sign-in at a time per server. A second would open a second browser
	// window and a second redirect listener, and the callback that arrived
	// second would be answering a request that no longer exists.
	if !m.beginSignIn(spec.Name) {
		return ServerState{}, fmt.Errorf("a sign-in to %s is already in progress", spec.Name)
	}
	defer m.endSignIn(spec.Name)

	// Any live session holds a handler built without a browser. Ending it first
	// means the sign-in replaces the connection rather than sitting beside it.
	m.closeSession(spec.Name)
	m.setState(&ServerState{Name: spec.Name, Spec: spec, Status: StatusConnecting})

	// A sign-in is its own attempt, and the closeSession above has already moved
	// the epoch on, so a connect that was dialling when the user pressed the
	// button cannot land on top of the sign-in's result.
	if err := m.dial(ctx, spec, signInAllowed, m.beginAttempt(spec.Name)); err != nil {
		state := &ServerState{Name: spec.Name, Spec: spec, Status: StatusFailed, Err: err}
		if SignInRequired(err) {
			state.Status = StatusNeedsSignIn
		}
		m.setState(state)
		return *state, err
	}

	state, _ := m.State(spec.Name)
	return state, nil
}

// SignOut withdraws a server's stored authorization.
//
// Forgetting a token locally is not signing out: the token stays valid at the
// service until it expires, and a user who has just clicked "sign out" would
// reasonably believe otherwise. So the token is revoked at the authorization
// server first, and it is deleted here whether or not that succeeded — a token
// that could not be revoked must at least stop being used from this machine.
// When revocation did not happen, ErrSignedOutLocallyOnly is returned so the
// surfaces can say so and point the user at their account settings.
func (m *Manager) SignOut(ctx context.Context, spec *ServerSpec) error {
	if spec == nil {
		return errors.New("no server to sign out of")
	}
	if spec.transport() != TransportHTTP {
		return fmt.Errorf("%s runs on this machine and is not signed in to", spec.Name)
	}
	if m.opts.Tokens == nil {
		return nil
	}

	// Disconnect first. A live session holds the token in memory and would go
	// on using it after it had been deleted from disk. The state goes with it:
	// a session ended while its state still said connected would leave the
	// server's tools on offer with nothing behind them.
	m.closeSession(spec.Name)
	m.setState(&ServerState{Name: spec.Name, Spec: spec, Status: StatusNeedsSignIn})

	record, err := m.opts.Tokens.Load(spec.Name)
	if errors.Is(err, ErrNoToken) {
		return nil
	}
	if err != nil {
		return err
	}

	revokeErr := revokeSignIn(ctx, spec, record)
	if err := m.opts.Tokens.Delete(spec.Name); err != nil {
		return err
	}

	if revokeErr != nil {
		m.opts.Logger.Warn("mcp token could not be revoked at the server", "server", spec.Name, "error", revokeErr)
		return fmt.Errorf("%w: %w", ErrSignedOutLocallyOnly, revokeErr)
	}
	return nil
}

// revokeSignIn asks the authorization server to invalidate the stored token
// (RFC 7009).
//
// The refresh token is revoked in preference to the access token: RFC 7009 §2.1
// says a server receiving a refresh token SHOULD also invalidate the access
// tokens issued from it, so revoking the refresh token withdraws the whole
// sign-in rather than one short-lived credential.
func revokeSignIn(ctx context.Context, spec *ServerSpec, record *SignInRecord) error {
	if record == nil || record.Token == nil {
		return errors.New("no token to revoke")
	}

	client := &http.Client{Timeout: signInHTTPTimeout}
	endpoint, err := revocationEndpoint(ctx, spec.URL, record.Issuer, client)
	if err != nil {
		return err
	}
	if endpoint == "" {
		return fmt.Errorf("%s offers no revocation endpoint", spec.Name)
	}
	// The protocol library validates the scheme of this URL but deliberately
	// leaves it out of its https-or-loopback list, unlike the token and
	// authorization endpoints. A revocation request carries the refresh token
	// in its body, so a metadata document naming an http endpoint would put a
	// live credential on the wire in cleartext.
	if err := requireSecureEndpoint(endpoint); err != nil {
		return err
	}

	token, hint := record.Token.RefreshToken, "refresh_token"
	if token == "" {
		token, hint = record.Token.AccessToken, "access_token"
	}
	if token == "" {
		return errors.New("no token to revoke")
	}

	form := url.Values{
		"token":           {token},
		"token_type_hint": {hint},
	}
	// Ollama is a public client: it has no secret, so it identifies itself in
	// the request body with the identifier it was issued at registration.
	// Without that identifier a server that requires one refuses, which is why
	// it is stored alongside the token.
	if record.ClientID != "" {
		form.Set("client_id", record.ClientID)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, strings.NewReader(form.Encode()))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	// RFC 7009 §2.2: the server answers 200 both for a token it revoked and for
	// one it does not recognise, since an already-invalid token satisfies the
	// request.
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("revocation refused with %s", resp.Status)
	}
	return nil
}

// requireSecureEndpoint refuses to send a token anywhere a network can read it.
// Loopback is allowed because a server running on this machine has no network
// to be read on, and because that is where tests and local development live.
func requireSecureEndpoint(endpoint string) error {
	parsed, err := url.Parse(endpoint)
	if err != nil {
		return fmt.Errorf("revocation endpoint %q is not a usable URL: %w", endpoint, err)
	}
	if parsed.Scheme == "https" {
		return nil
	}
	if parsed.Scheme == "http" && isLoopbackHost(parsed.Hostname()) {
		return nil
	}
	return fmt.Errorf("refusing to send a token to %q: a revocation endpoint must use https", endpoint)
}

// revocationEndpoint discovers where a server's tokens are revoked: the MCP
// endpoint names its authorization servers, and an authorization server's
// metadata names its revocation endpoint. An empty string with no error means
// the server does not offer one.
func revocationEndpoint(ctx context.Context, serverURL, issued string, client *http.Client) (string, error) {
	// When the record says which server issued the token, ask that one and
	// nothing else. A protected resource may name several, and revoking at the
	// wrong one is worse than not revoking: RFC 7009 §2.2 has a server answer
	// 200 for a token it has never heard of, so the user would be told they
	// were signed out while the token stayed live at the server that issued it.
	if issued = strings.TrimSpace(issued); issued != "" {
		server, err := sdkauth.GetAuthServerMetadata(ctx, issued, client)
		if err != nil {
			return "", err
		}
		if server == nil {
			return "", fmt.Errorf("%s publishes no metadata", issued)
		}
		return server.RevocationEndpoint, nil
	}

	// No issuer recorded — a record written before this was kept. Falling back
	// to the resource's own list is the old behaviour, and its weakness is
	// exactly why the issuer is now stored.
	metadata, err := protectedResourceMetadata(ctx, serverURL, client)
	if err != nil {
		return "", err
	}
	if metadata == nil {
		return "", fmt.Errorf("%s publishes no authorization metadata", serverURL)
	}

	var problems []error
	for _, issuer := range metadata.AuthorizationServers {
		server, err := sdkauth.GetAuthServerMetadata(ctx, issuer, client)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		if server != nil && server.RevocationEndpoint != "" {
			return server.RevocationEndpoint, nil
		}
	}
	// Every authorization server was reachable and none revokes: that is an
	// answer, not a failure. Only report an error if none could be read at all.
	if len(problems) > 0 && len(problems) == len(metadata.AuthorizationServers) {
		return "", errors.Join(problems...)
	}
	return "", nil
}

// protectedResourceMetadata fetches RFC 9728 metadata for an MCP endpoint,
// trying the path-inserted well-known location before the root one, as the MCP
// specification requires.
func protectedResourceMetadata(ctx context.Context, serverURL string, client *http.Client) (*oauthex.ProtectedResourceMetadata, error) {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return nil, err
	}

	var candidates []string
	if path := strings.Trim(parsed.Path, "/"); path != "" {
		inserted := *parsed
		inserted.Path = "/.well-known/oauth-protected-resource/" + path
		inserted.RawQuery = ""
		candidates = append(candidates, inserted.String())
	}
	root := *parsed
	root.Path = "/.well-known/oauth-protected-resource"
	root.RawQuery = ""
	candidates = append(candidates, root.String())

	var problems []error
	for _, candidate := range candidates {
		metadata, err := oauthex.GetProtectedResourceMetadata(ctx, candidate, serverURL, client)
		if err != nil {
			problems = append(problems, err)
			continue
		}
		return metadata, nil
	}
	return nil, errors.Join(problems...)
}
