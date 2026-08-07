package mcp

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"sync"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// DefaultAuthorizationTimeout bounds how long Ollama waits for the user to
// finish signing in. A sign-in nobody completes must not hold a listener open
// on the user's machine for ever.
const DefaultAuthorizationTimeout = 5 * time.Minute

// LoopbackRedirect is the browser half of the authorization code flow: it
// receives the redirect the authorization server sends back.
//
// The protocol library performs the rest — metadata discovery, client
// registration, PKCE, the token exchange and refresh — and explicitly leaves
// the redirect to the caller. This is that caller.
//
// It listens on 127.0.0.1 on a port the operating system chooses. A fixed port
// would be guessable by anything else on the machine, and a non-loopback bind
// would put an authorization code on the network.
type LoopbackRedirect struct {
	// Open launches the user's browser. It is a field so tests can drive the
	// flow without one, and so that opening a browser is always an explicit
	// act of this type rather than a side effect elsewhere.
	Open func(rawURL string) error

	// Timeout bounds one sign-in. Zero means DefaultAuthorizationTimeout.
	Timeout time.Duration

	// ExpectedIssuer is the authorization server's issuer identifier. When it
	// is set, an "iss" parameter in the callback is compared against it and a
	// mismatch ends the sign-in.
	//
	// RFC 9207 §2.4 requires a client to extract and compare the iss parameter
	// whenever it is present — not only when the server advertised that it
	// would send one. A mismatch is the mix-up attack the parameter exists to
	// catch, and it must be caught wherever it arrives.
	ExpectedIssuer string

	// IgnoreIssuer stops the "iss" parameter being passed on to the protocol
	// library after it has been checked here.
	//
	// It is set when the authorization server's metadata does not advertise
	// authorization_response_iss_parameter_supported, because the library
	// treats an unadvertised issuer as fatal and refuses the whole sign-in.
	// Real services do send one without advertising it. The check above still
	// runs, so this withholds the value from a library that would choke on it
	// rather than skipping the validation the RFC asks for.
	//
	// The zero value passes the issuer on, so a caller that has not looked at
	// the metadata gets the library's stricter behaviour rather than the laxer
	// one.
	IgnoreIssuer bool

	listener net.Listener
	server   *http.Server

	mu       sync.Mutex
	expected string
	failure  string
	results  chan sdkauth.AuthorizationResult
	closed   bool
}

// StartLoopbackRedirect binds the redirect listener.
//
// It is started before the flow begins because the redirect address has to be
// known in order to build the authorization request. Close must be called.
func StartLoopbackRedirect() (*LoopbackRedirect, error) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return nil, fmt.Errorf("listen for the sign-in redirect: %w", err)
	}

	redirect := &LoopbackRedirect{
		Open:     openInBrowser,
		listener: listener,
		results:  make(chan sdkauth.AuthorizationResult, 1),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/callback", redirect.handleCallback)
	redirect.server = &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}
	go redirect.server.Serve(listener)

	return redirect, nil
}

// RedirectURL is the address the authorization server must send the user back
// to. It is only ever a loopback address.
func (r *LoopbackRedirect) RedirectURL() string {
	return "http://" + r.listener.Addr().String() + "/callback"
}

// Fetch opens the authorization URL in the user's browser and waits for the
// redirect. It satisfies the protocol library's AuthorizationCodeFetcher.
//
// The browser is opened here and nowhere else, so nothing but an explicit
// sign-in can cause one to open — never a model, a tool result, or a
// configuration file.
func (r *LoopbackRedirect) Fetch(ctx context.Context, args *sdkauth.AuthorizationArgs) (*sdkauth.AuthorizationResult, error) {
	if args == nil || strings.TrimSpace(args.URL) == "" {
		return nil, errors.New("no authorization URL to open")
	}

	// The state the authorization server will echo back. A callback carrying
	// anything else is not this flow's, and must not end the wait.
	parsed, err := url.Parse(args.URL)
	if err != nil {
		return nil, fmt.Errorf("authorization URL is not usable: %w", err)
	}
	r.mu.Lock()
	r.expected = parsed.Query().Get("state")
	r.mu.Unlock()

	if r.Open != nil {
		if err := r.Open(args.URL); err != nil {
			return nil, fmt.Errorf("open the sign-in page: %w", err)
		}
	}

	timeout := r.Timeout
	if timeout <= 0 {
		timeout = DefaultAuthorizationTimeout
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case result := <-r.results:
		if result.Code == "" {
			r.mu.Lock()
			reason := r.failure
			r.mu.Unlock()
			if reason != "" {
				return nil, errors.New(reason)
			}
			return nil, errors.New("the sign-in was not completed")
		}
		return &result, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-timer.C:
		return nil, fmt.Errorf("no sign-in within %s", timeout)
	}
}

// handleCallback receives the authorization server's redirect.
//
// A callback whose state does not match this flow's is refused and, crucially,
// does not end the wait: anything on the machine can reach a loopback port, so
// a stray or hostile request must not be able to abort a sign-in in progress.
func (r *LoopbackRedirect) handleCallback(w http.ResponseWriter, req *http.Request) {
	query := req.URL.Query()

	r.mu.Lock()
	expected := r.expected
	r.mu.Unlock()

	if expected == "" || query.Get("state") != expected {
		http.Error(w, "This sign-in did not come from Ollama.", http.StatusBadRequest)
		return
	}

	if description := query.Get("error"); description != "" {
		detail := query.Get("error_description")
		if detail == "" {
			detail = description
		}
		writeSignInPage(w, "Sign-in was refused: "+detail)
		r.deliver(sdkauth.AuthorizationResult{State: query.Get("state")})
		return
	}

	code := query.Get("code")
	if code == "" {
		http.Error(w, "No authorization code was returned.", http.StatusBadRequest)
		return
	}

	// RFC 9207 §2.4: an iss that is present must be compared with the issuer
	// of the server the request was sent to, and a mismatch means the response
	// must be rejected and the flow must not continue. This is the mix-up
	// attack the parameter exists for, so the comparison happens whether or not
	// the server advertised that it would send one.
	issuer := query.Get("iss")
	if issuer != "" && r.ExpectedIssuer != "" && issuer != r.ExpectedIssuer {
		http.Error(w, "This sign-in came back from the wrong authorization server.", http.StatusBadRequest)
		r.deliverFailure(fmt.Sprintf("the authorization response came from %q, not from %q", issuer, r.ExpectedIssuer))
		return
	}
	if r.IgnoreIssuer {
		issuer = ""
	}

	writeSignInPage(w, "Signed in. You can close this window and return to Ollama.")
	r.deliver(sdkauth.AuthorizationResult{
		Code:  code,
		State: query.Get("state"),
		Iss:   issuer,
	})
}

// deliverFailure ends the wait with a reason, so a refused sign-in says what
// was wrong rather than timing out or reporting only that it did not finish.
func (r *LoopbackRedirect) deliverFailure(reason string) {
	r.mu.Lock()
	r.failure = reason
	r.mu.Unlock()
	r.deliver(sdkauth.AuthorizationResult{})
}

// deliver hands the result to the waiter, once. The channel is buffered so a
// callback never blocks, and a second callback is dropped rather than queued.
func (r *LoopbackRedirect) deliver(result sdkauth.AuthorizationResult) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.closed {
		return
	}
	select {
	case r.results <- result:
		// The flow is finished; a later callback carries nothing new, and the
		// state is cleared so a replay cannot be accepted.
		r.expected = ""
	default:
	}
}

// Close stops listening. It is safe to call more than once.
//
// The listener exists only for the duration of a sign-in: leaving a port open
// on the user's machine after the flow has finished is an invitation.
func (r *LoopbackRedirect) Close() error {
	r.mu.Lock()
	if r.closed {
		r.mu.Unlock()
		return nil
	}
	r.closed = true
	r.expected = ""
	r.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := r.server.Shutdown(ctx); err != nil {
		return r.listener.Close()
	}
	return nil
}

func writeSignInPage(w http.ResponseWriter, message string) {
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	// Plain text rather than HTML: nothing here is worth the risk of rendering
	// a value that came from an authorization server.
	fmt.Fprintln(w, message)
}

// openInBrowser launches the user's browser at the authorization URL.
//
// Only http and https are launched. A URL from an authorization server is
// third-party input, and handing an arbitrary scheme to the operating system's
// opener is how one ends up launching something that is not a browser.
func openInBrowser(rawURL string) error {
	parsed, err := url.Parse(rawURL)
	if err != nil {
		return fmt.Errorf("authorization URL is not usable: %w", err)
	}
	if parsed.Scheme != "https" && parsed.Scheme != "http" {
		return fmt.Errorf("refusing to open a %q URL", parsed.Scheme)
	}

	switch runtime.GOOS {
	case "darwin":
		return exec.Command("open", rawURL).Start()
	case "windows":
		return exec.Command("rundll32", "url.dll,FileProtocolHandler", rawURL).Start()
	default:
		return exec.Command("xdg-open", rawURL).Start()
	}
}
