package cmd

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os/exec"
	"path/filepath"
	"runtime"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/ollama/ollama/api"
	"github.com/ollama/ollama/mcp"
)

// modelServer stands in for the Ollama API so agentToolsRegistry can ask a
// model about its capabilities. Only /api/show is exercised.
func modelServer(t *testing.T, capabilities ...string) *api.Client {
	t.Helper()
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/show" {
			http.NotFound(w, r)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]any{"capabilities": capabilities})
	}))
	t.Cleanup(server.Close)

	base, err := url.Parse(server.URL)
	if err != nil {
		t.Fatalf("parse test server url: %v", err)
	}
	return api.NewClient(base, server.Client())
}

// liveMCPManager builds and connects the hand-written MCP server used across
// the MCP tests, approved so it will actually run.
func liveMCPManager(t *testing.T) *mcp.Manager {
	t.Helper()

	name := "rawserver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "github.com/ollama/ollama/mcp/testdata/rawserver")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build rawserver: %v\n%s", err, output)
	}

	cfg := &mcp.Config{}
	cfg.Set("raw", &mcp.ServerSpec{Command: binary})
	stored, _ := cfg.Get("raw")

	approvals := &mcp.Approvals{}
	approvals.Approve(stored, time.Now())

	manager := mcp.NewManager(mcp.Options{ConnectTimeout: 30 * time.Second, Approvals: approvals})
	t.Cleanup(func() { manager.Close() })
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != mcp.StatusConnected {
		t.Fatalf("mcp server status = %q, err = %v", state.Status, state.Err)
	}
	return manager
}

// TestAgentToolsRegistryOffersMCPTools is the activation evidence for the CLI
// entry point: the function the agent TUI actually calls to build its tool
// registry must include the tools of connected MCP servers. Without this, every
// layer beneath it could be perfect and the user would still see nothing.
func TestAgentToolsRegistryOffersMCPTools(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_DISABLE_SHELL", "1")
	t.Setenv("OLLAMA_AGENT_DISABLE_WEBSEARCH", "1")

	client := modelServer(t, "tools")
	manager := liveMCPManager(t)

	registry := agentToolsRegistry(t.Context(), client, "test-model", nil, manager)
	if registry == nil {
		t.Fatal("no registry was built for a tools-capable model")
	}

	names := registry.Names()
	if !slices.Contains(names, "raw__echo") {
		t.Fatalf("the MCP tool is missing from the agent's registry; it has %v", names)
	}

	tool, ok := registry.Get("raw__echo")
	if !ok {
		t.Fatal("raw__echo not retrievable")
	}
	if tool.Schema().Name != "raw__echo" {
		t.Errorf("schema name = %q", tool.Schema().Name)
	}

	// A tool the manager refused must not appear, whatever the server claimed.
	for _, refused := range []string{"raw__bash", "raw__scalar_input", "bash"} {
		if slices.Contains(names, refused) {
			t.Errorf("%q should not be offered to the model", refused)
		}
	}
}

func TestAgentToolsRegistryWithoutMCP(t *testing.T) {
	t.Setenv("OLLAMA_AGENT_DISABLE_SHELL", "1")
	t.Setenv("OLLAMA_AGENT_DISABLE_WEBSEARCH", "1")

	client := modelServer(t, "tools")
	registry := agentToolsRegistry(t.Context(), client, "test-model", nil, nil)
	if registry == nil {
		t.Fatal("a nil manager must not stop the other tools being offered")
	}
	for _, name := range registry.Names() {
		if len(name) > 5 && name[:5] == "raw__" {
			t.Errorf("unexpected MCP tool %q with no manager", name)
		}
	}
}

func TestAgentMCPManagerRespectsTheDisableSwitch(t *testing.T) {
	// The configuration must contain a server, and the environment must be
	// isolated from the developer's own ~/.ollama. Without both, this test
	// passes whether or not the switch works — because an absent config also
	// yields a nil manager, and it would then be asserting nothing.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	t.Setenv("OLLAMA_MCP_CONFIG", configPath)
	t.Setenv("OLLAMA_MCP_APPROVALS", filepath.Join(dir, "mcp-approvals.json"))

	cfg := &mcp.Config{}
	cfg.Set("present", &mcp.ServerSpec{Command: "echo", Args: []string{"hello"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	t.Setenv("OLLAMA_AGENT_DISABLE_MCP", "")
	control := agentMCPManager(t.Context())
	if control == nil {
		t.Fatal("with the switch off and a server configured, a manager must be built — otherwise the assertion below proves nothing")
	}
	control.Close()

	t.Setenv("OLLAMA_AGENT_DISABLE_MCP", "1")
	if manager := agentMCPManager(t.Context()); manager != nil {
		manager.Close()
		t.Error("OLLAMA_AGENT_DISABLE_MCP must switch MCP off entirely")
	}
}

// TestAgentMCPManagerConnectsAnApprovedServer is the positive half of the
// ledger check at this entry point. Refusing an unapproved server is not
// evidence that the ledger is consulted — a loader that ignored it entirely
// would also refuse everything, since a manager with no policy denies by
// default. This proves the approval is actually read and acted on.
func TestAgentMCPManagerConnectsAnApprovedServer(t *testing.T) {
	name := "rawserver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "github.com/ollama/ollama/mcp/testdata/rawserver")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build rawserver: %v\n%s", err, output)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	approvalsPath := filepath.Join(dir, "mcp-approvals.json")
	t.Setenv("OLLAMA_MCP_CONFIG", configPath)
	t.Setenv("OLLAMA_MCP_APPROVALS", approvalsPath)
	t.Setenv("OLLAMA_AGENT_DISABLE_MCP", "")

	cfg := &mcp.Config{}
	cfg.Set("raw", &mcp.ServerSpec{Command: binary})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	stored, _ := cfg.Get("raw")

	approvals := &mcp.Approvals{}
	approvals.Approve(stored, time.Now())
	if err := approvals.Save(approvalsPath); err != nil {
		t.Fatalf("save approvals: %v", err)
	}

	manager := agentMCPManager(t.Context())
	if manager == nil {
		t.Fatal("no manager was built")
	}
	t.Cleanup(func() { manager.Close() })

	state, _ := manager.State("raw")
	if state.Status != mcp.StatusConnected {
		t.Fatalf("status = %q, err = %v — an approved server must actually connect", state.Status, state.Err)
	}
	if len(manager.Tools()) == 0 {
		t.Error("an approved, connected server must contribute tools")
	}
}

func TestAgentMCPManagerRefusesUnapprovedServers(t *testing.T) {
	// The whole point of the ledger, exercised through the CLI's own loader:
	// a configured, enabled server that has never been approved is not run.
	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	t.Setenv("OLLAMA_MCP_CONFIG", configPath)
	t.Setenv("OLLAMA_MCP_APPROVALS", filepath.Join(dir, "mcp-approvals.json"))
	t.Setenv("OLLAMA_AGENT_DISABLE_MCP", "")

	cfg := &mcp.Config{}
	cfg.Set("never-approved", &mcp.ServerSpec{Command: "echo", Args: []string{"hello"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}

	manager := agentMCPManager(t.Context())
	if manager == nil {
		t.Fatal("a configured server should still produce a manager, so the user can be told why it is not running")
	}
	t.Cleanup(func() { manager.Close() })

	state, ok := manager.State("never-approved")
	if !ok {
		t.Fatal("no state recorded for the configured server")
	}
	if state.Status != mcp.StatusNeedsApproval {
		t.Errorf("status = %q, want needs-approval", state.Status)
	}
	if len(manager.Tools()) != 0 {
		t.Error("an unapproved server must contribute no tools to the agent")
	}
}

// TestSetAgentMCPEnabledWritesTheConfigAndAppliesIt covers what /mcp enable and
// /mcp disable actually do. The configuration file is the source of truth, so a
// toggle that updated only the running manager would be forgotten at the next
// launch, and one that updated only the file would leave a switched-off server
// still running for the rest of the session.
func TestSetAgentMCPEnabledWritesTheConfigAndAppliesIt(t *testing.T) {
	name := "rawserver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)
	build := exec.Command("go", "build", "-o", binary, "github.com/ollama/ollama/mcp/testdata/rawserver")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build rawserver: %v\n%s", err, output)
	}

	dir := t.TempDir()
	configPath := filepath.Join(dir, "mcp.json")
	approvalsPath := filepath.Join(dir, "mcp-approvals.json")
	t.Setenv("OLLAMA_MCP_CONFIG", configPath)
	t.Setenv("OLLAMA_MCP_APPROVALS", approvalsPath)
	t.Setenv("OLLAMA_AGENT_DISABLE_MCP", "")

	cfg := &mcp.Config{}
	cfg.Set("raw", &mcp.ServerSpec{Command: binary})
	cfg.Set("unapproved", &mcp.ServerSpec{Command: binary, Args: []string{"-silent"}})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save config: %v", err)
	}
	approved, _ := cfg.Get("raw")
	approvals := &mcp.Approvals{}
	approvals.Approve(approved, time.Now())
	if err := approvals.Save(approvalsPath); err != nil {
		t.Fatalf("save approvals: %v", err)
	}

	manager := agentMCPManager(t.Context())
	if manager == nil {
		t.Fatal("no manager")
	}
	t.Cleanup(func() { manager.Close() })
	if len(manager.Tools()) == 0 {
		t.Fatal("the approved server should be offering tools before the toggle")
	}

	t.Run("disable stops offering the tools and is written to the config", func(t *testing.T) {
		if err := setAgentMCPEnabled(t.Context(), manager, "raw", false); err != nil {
			t.Fatalf("disable: %v", err)
		}
		if len(manager.Tools()) != 0 {
			t.Error("a disabled server must stop offering tools immediately")
		}
		reloaded, err := mcp.Load(configPath)
		if err != nil {
			t.Fatalf("reload config: %v", err)
		}
		spec, _ := reloaded.Get("raw")
		if !spec.Disabled {
			t.Error("the change was not written to the config, so it would be forgotten at the next launch")
		}
	})

	t.Run("enable brings it back", func(t *testing.T) {
		if err := setAgentMCPEnabled(t.Context(), manager, "raw", true); err != nil {
			t.Fatalf("enable: %v", err)
		}
		if len(manager.Tools()) == 0 {
			t.Error("re-enabling should reconnect and offer the tools again")
		}
	})

	t.Run("enabling an unapproved server fails and says how to approve it", func(t *testing.T) {
		err := setAgentMCPEnabled(t.Context(), manager, "unapproved", true)
		if err == nil {
			t.Fatal("an unapproved server must not be enabled into use silently")
		}
		if !strings.Contains(err.Error(), "ollama mcp approve unapproved") {
			t.Errorf("error = %v, want it to name the command that would fix it", err)
		}
	})

	t.Run("an unknown server is refused", func(t *testing.T) {
		if err := setAgentMCPEnabled(t.Context(), manager, "absent", true); err == nil {
			t.Fatal("expected an error for an unknown server")
		}
	})
}
