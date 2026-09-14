//go:build windows || darwin

package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/ollama/ollama/mcp"
)

// mcpPaths points the manager at files belonging to this test rather than the
// developer's own, which it would otherwise read — and connect.
func mcpPaths(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	t.Setenv(mcp.ConfigPathEnv, filepath.Join(dir, "mcp.json"))
	t.Setenv(mcp.ApprovalsPathEnv, filepath.Join(dir, "mcp-approvals.json"))
	t.Setenv("OLLAMA_MCP_TOKENS", filepath.Join(dir, "mcp-tokens.json"))
	return dir
}

// TestManagerExistsWithAnEmptyConfiguration is the defect this test was written
// for: the app used to return no manager at all when nothing was configured at
// launch. A server added afterwards from the MCP Servers page then had nothing
// to connect it — approving it reached nothing, the page had no state to
// report, and it sat saying "restart Ollama to connect this server". Adding
// servers while the app runs is what that page is for.
func TestManagerExistsWithAnEmptyConfiguration(t *testing.T) {
	mcpPaths(t)

	manager := startMCPManager(t.Context())
	if manager == nil {
		t.Fatal("no manager was built, so a server added while the app runs could never connect")
	}
	t.Cleanup(func() { manager.Close() })

	if states := manager.States(); len(states) != 0 {
		t.Errorf("an empty configuration produced states: %+v", states)
	}
}

// A server added after launch connects through the manager that now exists,
// without a restart.
func TestAServerAddedAfterLaunchCanBeConnected(t *testing.T) {
	dir := mcpPaths(t)

	manager := startMCPManager(t.Context())
	if manager == nil {
		t.Fatal("no manager")
	}
	t.Cleanup(func() { manager.Close() })

	// What the MCP Servers page does: write the configuration, approve it, and
	// bring the running manager into line.
	configPath := filepath.Join(dir, "mcp.json")
	cfg, err := mcp.Load(configPath)
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	// A command that does not exist: the point is that the manager takes the
	// server on and reports on it, not that this particular server runs.
	cfg.Set("added-later", &mcp.ServerSpec{Command: "definitely-not-a-real-command"})
	if err := cfg.Save(configPath); err != nil {
		t.Fatalf("save: %v", err)
	}

	manager.Connect(t.Context(), cfg)

	state, ok := manager.State("added-later")
	if !ok {
		t.Fatal("the manager has no state for a server added after launch")
	}
	if state.Status == "" {
		t.Errorf("state = %+v", state)
	}
}

func TestManagerIsRefusedWhenMCPIsSwitchedOff(t *testing.T) {
	mcpPaths(t)
	t.Setenv("OLLAMA_DISABLE_MCP", "1")

	if manager := startMCPManager(t.Context()); manager != nil {
		defer manager.Close()
		t.Error("OLLAMA_DISABLE_MCP was ignored")
	}
}

func TestManagerConnectsWhatIsAlreadyConfigured(t *testing.T) {
	dir := mcpPaths(t)
	configPath := filepath.Join(dir, "mcp.json")
	if err := os.WriteFile(configPath, []byte(`{"mcpServers":{"at-launch":{"command":"definitely-not-a-real-command"}}}`), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	manager := startMCPManager(t.Context())
	if manager == nil {
		t.Fatal("no manager")
	}
	t.Cleanup(func() { manager.Close() })

	if _, ok := manager.State("at-launch"); !ok {
		t.Error("a server configured at launch has no state")
	}
}
