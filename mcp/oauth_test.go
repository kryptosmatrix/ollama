package mcp

import (
	"context"
	"errors"
	"net"
	"net/http"
	"net/url"
	"strings"
	"testing"
	"time"

	sdkauth "github.com/modelcontextprotocol/go-sdk/auth"
)

// authorizationURL builds what an authorization server would send the user to,
// carrying the state the redirect must echo back.
func authorizationURL(state string) string {
	return "https://auth.example.com/authorize?client_id=x&state=" + url.QueryEscape(state)
}

// callback plays the browser being redirected back, and returns the status the
// listener replied with.
func callback(t *testing.T, redirect *LoopbackRedirect, query string) int {
	t.Helper()
	response, err := http.Get(redirect.RedirectURL() + "?" + query)
	if err != nil {
		t.Fatalf("callback: %v", err)
	}
	defer response.Body.Close()
	return response.StatusCode
}

func startRedirect(t *testing.T) (*LoopbackRedirect, chan string) {
	t.Helper()
	redirect, err := StartLoopbackRedirect()
	if err != nil {
		t.Fatalf("StartLoopbackRedirect: %v", err)
	}
	t.Cleanup(func() { redirect.Close() })

	opened := make(chan string, 4)
	redirect.Open = func(rawURL string) error {
		opened <- rawURL
		return nil
	}
	redirect.Timeout = 5 * time.Second
	return redirect, opened
}

func TestRedirectListensOnLoopbackOnly(t *testing.T) {
	redirect, _ := startRedirect(t)

	address := redirect.RedirectURL()
	if !strings.HasPrefix(address, "http://127.0.0.1:") {
		t.Fatalf("RedirectURL = %q, want a loopback address; anything else puts an authorization code on the network", address)
	}
	if !strings.HasSuffix(address, "/callback") {
		t.Errorf("RedirectURL = %q, want a /callback path", address)
	}

	host, port, err := net.SplitHostPort(strings.TrimSuffix(strings.TrimPrefix(address, "http://"), "/callback"))
	if err != nil {
		t.Fatalf("split address: %v", err)
	}
	if host != "127.0.0.1" {
		t.Errorf("host = %q", host)
	}
	if port == "0" || port == "" {
		t.Errorf("port = %q, want one the operating system chose", port)
	}
}

// TestSignInRoundTrip is the whole flow the protocol library delegates: open
// the page, receive the redirect, return the code.
func TestSignInRoundTrip(t *testing.T) {
	redirect, opened := startRedirect(t)

	results := make(chan *sdkauth.AuthorizationResult, 1)
	errs := make(chan error, 1)
	go func() {
		result, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("the-state")})
		results <- result
		errs <- err
	}()

	select {
	case url := <-opened:
		if url != authorizationURL("the-state") {
			t.Errorf("opened %q, want the exact authorization URL", url)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("the browser was never opened")
	}

	if status := callback(t, redirect, "code=the-code&state=the-state&iss=https://auth.example.com"); status != http.StatusOK {
		t.Fatalf("callback status = %d", status)
	}

	if err := <-errs; err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	result := <-results
	if result.Code != "the-code" || result.State != "the-state" {
		t.Errorf("result = %+v", result)
	}
	if result.Iss != "https://auth.example.com" {
		t.Errorf("the issuer must be carried through, got %q", result.Iss)
	}
}

// TestAStrayCallbackCannotEndTheSignIn is the reason the state is checked at
// the listener as well as by the protocol library. Anything on the machine can
// reach a loopback port; a request carrying the wrong state must be refused
// and must leave the real sign-in still waiting.
func TestAStrayCallbackCannotEndTheSignIn(t *testing.T) {
	redirect, opened := startRedirect(t)

	results := make(chan *sdkauth.AuthorizationResult, 1)
	errs := make(chan error, 1)
	go func() {
		result, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("the-state")})
		results <- result
		errs <- err
	}()
	<-opened

	for _, stray := range []string{
		"code=injected&state=wrong",
		"code=injected",
		"code=injected&state=",
	} {
		if status := callback(t, redirect, stray); status != http.StatusBadRequest {
			t.Errorf("callback %q returned %d, want 400", stray, status)
		}
	}

	select {
	case err := <-errs:
		t.Fatalf("a stray callback ended the sign-in: %v, %+v", err, <-results)
	case <-time.After(200 * time.Millisecond):
	}

	// The real redirect must still complete.
	if status := callback(t, redirect, "code=the-code&state=the-state"); status != http.StatusOK {
		t.Fatalf("the real callback returned %d", status)
	}
	if err := <-errs; err != nil {
		t.Fatalf("Fetch: %v", err)
	}
	if result := <-results; result.Code != "the-code" {
		t.Errorf("got the wrong code: %+v", result)
	}
}

func TestASecondCallbackIsIgnored(t *testing.T) {
	redirect, opened := startRedirect(t)

	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened

	callback(t, redirect, "code=first&state=s")
	if err := <-errs; err != nil {
		t.Fatalf("Fetch: %v", err)
	}

	// A replay of the same redirect must not be accepted after the flow ended.
	if status := callback(t, redirect, "code=second&state=s"); status != http.StatusBadRequest {
		t.Errorf("a replayed callback returned %d, want 400", status)
	}
}

// TestSignInStopsWhenTheCallerGivesUp covers the other way a sign-in ends: not
// the timeout, but the caller abandoning it — the app quitting, or a connect
// giving up. Without this the wait runs to its own timeout, holding a loopback
// port and a goroutine for minutes after the thing that wanted them is gone.
func TestSignInStopsWhenTheCallerGivesUp(t *testing.T) {
	redirect, opened := startRedirect(t)
	// Long enough that only the cancellation can end this wait.
	redirect.Timeout = time.Minute

	ctx, cancel := context.WithCancel(t.Context())
	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(ctx, &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened
	cancel()

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("an abandoned sign-in must fail rather than wait for its own timeout")
		}
		if !errors.Is(err, context.Canceled) {
			t.Errorf("err = %v, want the cancellation reported", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch ignored the cancellation and waited on its own timeout")
	}
}

func TestSignInTimesOut(t *testing.T) {
	redirect, opened := startRedirect(t)
	redirect.Timeout = 150 * time.Millisecond

	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("an unfinished sign-in must fail rather than wait for ever")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not time out")
	}
}

func TestSignInStopsWhenCancelled(t *testing.T) {
	redirect, opened := startRedirect(t)
	ctx, cancel := context.WithCancel(t.Context())

	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(ctx, &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened

	cancel()
	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("a cancelled sign-in must not succeed")
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not return when cancelled")
	}
}

// TestClosingStopsListening proves the port does not outlive the sign-in.
func TestClosingStopsListening(t *testing.T) {
	redirect, _ := startRedirect(t)
	address := redirect.RedirectURL()

	if _, err := http.Get(address + "?state=x"); err != nil {
		t.Fatalf("the listener should be up before Close: %v", err)
	}
	if err := redirect.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}
	if err := redirect.Close(); err != nil {
		t.Fatalf("Close must be safe to call twice: %v", err)
	}

	client := &http.Client{Timeout: 2 * time.Second}
	if _, err := client.Get(address + "?state=x"); err == nil {
		t.Error("the listener outlived Close; a port left open on the user's machine is an invitation")
	}
}

func TestFetchRefusesWithoutAnAuthorizationURL(t *testing.T) {
	redirect, opened := startRedirect(t)

	for _, args := range []*sdkauth.AuthorizationArgs{nil, {URL: ""}, {URL: "   "}} {
		if _, err := redirect.Fetch(t.Context(), args); err == nil {
			t.Errorf("expected an error for %+v", args)
		}
	}
	select {
	case url := <-opened:
		t.Errorf("a browser was opened for an unusable request: %q", url)
	default:
	}
}

func TestFetchReportsAFailureToOpenTheBrowser(t *testing.T) {
	redirect, _ := startRedirect(t)
	redirect.Open = func(string) error { return errNoBrowser }

	if _, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("s")}); err == nil {
		t.Fatal("if the user cannot be shown the page, the flow must not silently wait")
	}
}

var errNoBrowser = &net.AddrError{Err: "no browser", Addr: ""}

// TestOpenInBrowserRefusesAnUnexpectedScheme guards against handing an
// arbitrary URL from an authorization server to the operating system's opener.
func TestOpenInBrowserRefusesAnUnexpectedScheme(t *testing.T) {
	for _, rawURL := range []string{
		"file:///etc/passwd",
		"ftp://example.com",
		"javascript:alert(1)",
		"vscode://extension/install",
	} {
		if err := openInBrowser(rawURL); err == nil {
			t.Errorf("openInBrowser(%q) should refuse a non-web scheme", rawURL)
		}
	}
}

// TestFetchSatisfiesTheProtocolLibrary keeps the signature honest: this type
// exists to be handed to the library as its AuthorizationCodeFetcher, and a
// change that broke that would otherwise only show up at the call site.
func TestFetchSatisfiesTheProtocolLibrary(t *testing.T) {
	redirect, _ := startRedirect(t)
	var fetcher sdkauth.AuthorizationCodeFetcher = redirect.Fetch
	if fetcher == nil {
		t.Fatal("Fetch is not usable as an AuthorizationCodeFetcher")
	}
}

// TestARefusedSignInSaysWhy supersedes TestARefusedSignInIsReported, which
// asserted that the user was shown "not completed" — the very behaviour the
// review identified as the defect. A test can encode a bug as firmly as the
// code does, and this one did.
//
// TestARefusedSignInSaysWhy is a defect a cross-substrate review found. The
// error branch of the callback delivered an empty result rather than a failure
// with a reason, so Fetch fell back to the generic "the sign-in was not
// completed" and the authorization server's own error and error_description —
// access_denied, invalid_scope, a message naming the missing permission — were
// shown on the browser page and then thrown away.
//
// A user whose sign-in is refused has no other source for why.
func TestARefusedSignInSaysWhy(t *testing.T) {
	redirect, opened := startRedirect(t)
	redirect.Timeout = 5 * time.Second

	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened

	if status := callback(t, redirect, "error=access_denied&error_description=the+workspace+administrator+has+not+approved+Ollama&state=s"); status != http.StatusOK {
		t.Fatalf("the refusal page returned %d", status)
	}

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("a refused sign-in must fail")
		}
		if !strings.Contains(err.Error(), "workspace administrator") {
			t.Errorf("err = %v, want the server's own reason; it is the only place the user can learn why", err)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not return")
	}
}

// TestARefusalReasonIsSanitised keeps a third party's text from arriving
// unbounded. The error_description is written by the authorization server and
// ends up in a message shown to the user.
func TestARefusalReasonIsSanitised(t *testing.T) {
	redirect, opened := startRedirect(t)
	redirect.Timeout = 5 * time.Second

	errs := make(chan error, 1)
	go func() {
		_, err := redirect.Fetch(t.Context(), &sdkauth.AuthorizationArgs{URL: authorizationURL("s")})
		errs <- err
	}()
	<-opened

	hostile := url.QueryEscape("refused\r\n\r\nHTTP/1.1 200 OK\x00" + strings.Repeat("x", 20000))
	callback(t, redirect, "error=access_denied&error_description="+hostile+"&state=s")

	select {
	case err := <-errs:
		if err == nil {
			t.Fatal("a refused sign-in must fail")
		}
		if strings.ContainsAny(err.Error(), "\r\x00") {
			t.Error("control characters from the authorization server reached the error message")
		}
		if len(err.Error()) > 6000 {
			t.Errorf("the reason is %d bytes; a server's text must be bounded", len(err.Error()))
		}
	case <-time.After(5 * time.Second):
		t.Fatal("Fetch did not return")
	}
}
