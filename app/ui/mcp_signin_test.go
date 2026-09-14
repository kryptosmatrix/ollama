//go:build windows || darwin

package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/app/tools"
	"github.com/ollama/ollama/app/ui/responses"
	"github.com/ollama/ollama/mcp"
	"golang.org/x/oauth2"
)

// mcpSignInServer returns a Server whose manager has an isolated token store,
// so no test can read or write the developer's own credentials.
func mcpSignInServer(t *testing.T) (*Server, *mcp.FileTokenStore) {
	t.Helper()
	approvalsPath, err := mcp.ApprovalsPath()
	if err != nil {
		t.Fatalf("approvals path: %v", err)
	}
	store := &mcp.FileTokenStore{Path: filepath.Join(t.TempDir(), "mcp-tokens.json")}
	manager := mcp.NewManager(mcp.Options{
		ConnectTimeout: 30 * time.Second,
		Approvals:      mcp.ApprovalsFile(approvalsPath, nil),
		Tokens:         store,
	})
	t.Cleanup(func() { manager.Close() })
	return &Server{MCP: manager, Approvals: tools.NewApprovals()}, store
}

func writeRemoteServer(t *testing.T, configPath, name string, approve bool) *mcp.ServerSpec {
	return writeRemoteServerAt(t, configPath, name, "https://mcp.example.com/v1", approve)
}

// silentServer is an MCP endpoint that publishes no authorization metadata. It
// stands in for a server whose token cannot be revoked, without the test
// needing the internet: a test that resolves a real hostname fails for reasons
// that have nothing to do with this code.
func silentServer(t *testing.T) string {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	t.Cleanup(server.Close)
	return server.URL + "/mcp"
}

func writeRemoteServerAt(t *testing.T, configPath, name, url string, approve bool) *mcp.ServerSpec {
	t.Helper()
	cfg, err := mcp.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	spec := &mcp.ServerSpec{Type: mcp.TransportHTTP, URL: url}
	cfg.Set(name, spec)
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	if approve {
		approvalsPath, _ := mcp.ApprovalsPath()
		approvals, err := mcp.LoadApprovals(approvalsPath)
		if err != nil {
			t.Fatalf("load approvals: %v", err)
		}
		stored, _ := cfg.Get(name)
		approvals.Approve(stored, time.Now())
		if err := approvals.Save(approvalsPath); err != nil {
			t.Fatalf("save approvals: %v", err)
		}
	}
	stored, _ := cfg.Get(name)
	return stored
}

// TestSignInRouteStillNeedsApproval keeps the sign-in button from being a way
// round the approval gate. A sign-in contacts the server, and approval is what
// says a server may be contacted at all.
func TestSignInRouteStillNeedsApproval(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, _ := mcpSignInServer(t)
	writeRemoteServer(t, configPath, "hosted", false)

	recorder, err := callMCP(t, server, http.MethodPost, "/api/v1/mcp/hosted/signin", nil, map[string]string{"name": "hosted"})
	if err == nil {
		t.Fatal("an unapproved server must not be signed in to")
	}
	if recorder.Code != http.StatusForbidden {
		t.Errorf("status = %d, want 403", recorder.Code)
	}
	if server.MCP.SigningIn("hosted") {
		t.Error("a sign-in was started for a server that has not been approved")
	}
}

func TestSignInRouteRefusesALocalServer(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, _ := mcpSignInServer(t)

	cfg, err := mcp.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Set("files", &mcp.ServerSpec{Command: "uvx", Args: []string{"mcp-server-files"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	recorder, err := callMCP(t, server, http.MethodPost, "/api/v1/mcp/files/signin", nil, map[string]string{"name": "files"})
	if err == nil {
		t.Fatal("a stdio server has nothing to sign in to")
	}
	if recorder.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want 400", recorder.Code)
	}
}

func TestSignInRouteRefusesAnUnknownServer(t *testing.T) {
	mcpFiles(t)
	server, _ := mcpSignInServer(t)

	recorder, err := callMCP(t, server, http.MethodPost, "/api/v1/mcp/nothing/signin", nil, map[string]string{"name": "nothing"})
	if err == nil {
		t.Fatal("signing in to a server that does not exist must fail")
	}
	if recorder.Code != http.StatusNotFound {
		t.Errorf("status = %d, want 404", recorder.Code)
	}
}

// TestTheListSaysWhereACredentialWouldGo is the disclosure the page depends on.
// The page cannot say where a token is kept unless the API tells it, and a
// user signing in to a third-party service is entitled to know before they
// create one.
func TestTheListSaysWhereACredentialWouldGo(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, store := mcpSignInServer(t)
	writeRemoteServer(t, configPath, "hosted", true)

	cfg, err := mcp.Load(configPath)
	if err != nil {
		t.Fatalf("load config: %v", err)
	}
	cfg.Set("files", &mcp.ServerSpec{Command: "uvx", Args: []string{"mcp-server-files"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	servers := serversByName(t, server)
	remote, local := servers["hosted"], servers["files"]

	if !remote.CanSignIn {
		t.Error("a remote server must be shown as one that can be signed in to")
	}
	if remote.TokenStore != store.Description() {
		t.Errorf("tokenStore = %q, want the store actually in use (%q)", remote.TokenStore, store.Description())
	}
	if remote.SignedIn {
		t.Error("nothing is stored, so the server must not read as signed in")
	}
	// A local server never receives a credential, so offering one would be a
	// question with no meaning.
	if local.CanSignIn || local.TokenStore != "" {
		t.Errorf("a stdio server was offered a sign-in: %+v", local)
	}
}

func TestTheListReportsAStoredSignIn(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, store := mcpSignInServer(t)
	writeRemoteServer(t, configPath, "hosted", true)

	if err := store.Save("hosted", &mcp.SignInRecord{
		Token:    &oauth2.Token{AccessToken: "stored"},
		ClientID: "registered-client",
	}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	if got := serversByName(t, server)["hosted"]; !got.SignedIn {
		t.Error("a server with a stored token must read as signed in, or the page offers a sign-in that is already done")
	}
}

// TestSignOutRouteForgetsTheTokenAndSaysWhatFailed proves the route reaches the
// manager's sign-out rather than merely answering. This server publishes no
// authorization metadata, so revocation cannot happen and the response says so
// — and the token is gone either way, which is the property that matters most.
func TestSignOutRouteForgetsTheTokenAndSaysWhatFailed(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, store := mcpSignInServer(t)
	writeRemoteServerAt(t, configPath, "hosted", silentServer(t), true)

	if err := store.Save("hosted", &mcp.SignInRecord{Token: &oauth2.Token{AccessToken: "stored"}}); err != nil {
		t.Fatalf("Save: %v", err)
	}

	recorder, err := callMCP(t, server, http.MethodPost, "/api/v1/mcp/hosted/signout", nil, map[string]string{"name": "hosted"})
	if err != nil {
		t.Fatalf("signout: %v", err)
	}

	var body responses.MCPServer
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v\n%s", err, recorder.Body.String())
	}
	if body.SignedIn {
		t.Error("the response still reads as signed in")
	}
	// The revocation could not happen — the authorization server does not
	// exist — and the user has to be told, because withdrawing the token is
	// then theirs to do in that service's own account settings.
	if body.Error == "" {
		t.Error("a sign-out that could not revoke must say so; silence reads as a clean withdrawal")
	}

	if _, err := store.Load("hosted"); err == nil {
		t.Error("the token survived the sign-out; it must stop being used from this machine either way")
	}
}

func TestSignOutOfAServerNeverSignedIn(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, _ := mcpSignInServer(t)
	writeRemoteServerAt(t, configPath, "hosted", silentServer(t), true)

	if _, err := callMCP(t, server, http.MethodPost, "/api/v1/mcp/hosted/signout", nil, map[string]string{"name": "hosted"}); err != nil {
		t.Fatalf("signing out of a server that was never signed in is not a failure: %v", err)
	}
}

func serversByName(t *testing.T, server *Server) map[string]responses.MCPServer {
	t.Helper()
	byName := map[string]responses.MCPServer{}
	for _, listed := range listServers(t, server) {
		byName[listed.Name] = listed
	}
	return byName
}

// TestSignInRoutesAreRegistered drives the real mux rather than the test's own
// dispatch. A handler that works but is not routed is unreachable, and the
// dispatch in callMCP would never notice.
func TestSignInRoutesAreRegistered(t *testing.T) {
	configPath, _ := mcpFiles(t)
	server, _ := mcpSignInServer(t)
	writeRemoteServer(t, configPath, "hosted", false)
	handler := server.Handler()

	for _, path := range []string{"/api/v1/mcp/hosted/signin", "/api/v1/mcp/hosted/signout"} {
		recorder := httptest.NewRecorder()
		handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, path, strings.NewReader("")))
		// The request is refused on its merits — the server is not approved —
		// but a 404 or a 405 would mean nothing is listening on that path at
		// all, which is the failure this test exists to catch.
		if recorder.Code == http.StatusNotFound || recorder.Code == http.StatusMethodNotAllowed {
			t.Errorf("POST %s answered %d; the route is not registered", path, recorder.Code)
		}
	}
}

// TestTheListCarriesWhatAConfigurationCosts. The warning channel exists so a
// user can be told what a configuration costs without being refused. The page
// cannot say it unless the API sends it.
func TestTheListCarriesWhatAConfigurationCosts(t *testing.T) {
	configPath, _ := mcpFiles(t)
	cfg, err := mcp.Load(configPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	cfg.Set("leaky", &mcp.ServerSpec{Command: "srv", Args: []string{"--api-key=sk-live-1"}})
	cfg.Set("plain", &mcp.ServerSpec{Command: "srv", Args: []string{"--verbose"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	server, _ := mcpSignInServer(t)
	servers := serversByName(t, server)

	if len(servers["leaky"].Warnings) == 0 {
		t.Error("the page is never told what this configuration costs")
	} else if !strings.Contains(strings.Join(servers["leaky"].Warnings, " "), "process list") {
		t.Errorf("the warning does not say what is exposed: %v", servers["leaky"].Warnings)
	}
	if len(servers["plain"].Warnings) != 0 {
		t.Errorf("an ordinary server was warned about: %v", servers["plain"].Warnings)
	}
	// And it is a note, not a refusal.
	if servers["leaky"].Status == string(mcp.StatusInvalid) {
		t.Error("the warning became a refusal")
	}
}
