package mcp

import (
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

// buildRawServer compiles testdata/rawserver, the hand-written MCP server that
// depends on no protocol library, and returns the path to the binary.
func buildRawServer(t *testing.T) string {
	t.Helper()

	name := "rawserver"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	binary := filepath.Join(t.TempDir(), name)

	build := exec.Command("go", "build", "-o", binary, "./testdata/rawserver")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build rawserver: %v\n%s", err, output)
	}
	return binary
}

// TestStdioSubprocessEndToEnd is the test a mock cannot pass. It launches a
// real operating-system process, speaks the real stdio transport to it, and
// drives the full path: build the command, start the child, negotiate the
// protocol, list tools, convert schemas, call a tool, read the result, and reap
// the process. Nothing here is injected.
func TestStdioSubprocessEndToEnd(t *testing.T) {
	binary := buildRawServer(t)

	manager := NewManager(Options{ConnectTimeout: 30 * time.Second, CallTimeout: 15 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set("raw", &ServerSpec{Command: binary})
	manager.Connect(t.Context(), cfg)

	state, ok := manager.State("raw")
	if !ok {
		t.Fatal("no state recorded")
	}
	if state.Status != StatusConnected {
		t.Fatalf("status = %q, err = %v", state.Status, state.Err)
	}
	if state.ServerInfo != "rawserver 0.1.0" {
		t.Errorf("ServerInfo = %q, want the peer's own identity read off the wire", state.ServerInfo)
	}

	t.Run("a real tool call reaches a real process", func(t *testing.T) {
		result, err := manager.Call(t.Context(), "raw__echo", map[string]any{"text": "over the wire"})
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		if result.Content != "echo: over the wire" {
			t.Errorf("Content = %q, want the child process's own reply", result.Content)
		}
	})

	t.Run("a tool-level failure is reported without being an error", func(t *testing.T) {
		result, err := manager.Call(t.Context(), "raw__fail", nil)
		if err != nil {
			t.Fatalf("Call: %v", err)
		}
		if !result.IsError {
			t.Error("IsError should be set")
		}
		if !strings.Contains(result.Content, "the tool failed") {
			t.Errorf("Content = %q", result.Content)
		}
	})

	t.Run("an unrepresentable schema is skipped with a reason", func(t *testing.T) {
		// This is the case the SDK's own server refuses to produce, which is
		// why it is proven here against a hand-written peer.
		reason := skipReason(t, state, "scalar_input")
		if !strings.Contains(reason, "only an object of arguments") {
			t.Errorf("reason = %q, want it to explain the schema could not be represented", reason)
		}
		if _, err := manager.Call(t.Context(), "raw__scalar_input", nil); err == nil {
			t.Error("a skipped tool must not be callable")
		}
	})

	t.Run("a tool claiming a first-party name is refused", func(t *testing.T) {
		reason := skipReason(t, state, "bash")
		if !strings.Contains(reason, "reserved") {
			t.Errorf("reason = %q, want it to say the name is reserved", reason)
		}
	})

	t.Run("a hostile description is sanitised on the real path", func(t *testing.T) {
		var found bool
		for _, tool := range state.Tools {
			if tool.Name != "hostile_description" {
				continue
			}
			found = true
			fn, err := tool.Schema()
			if err != nil {
				t.Fatalf("Schema: %v", err)
			}
			for _, forbidden := range []string{"\x1b", "\x00", "\r"} {
				if strings.Contains(fn.Description, forbidden) {
					t.Errorf("control character %q survived the real wire path: %q", forbidden, fn.Description)
				}
			}
			if !strings.Contains(fn.Description, "line one") {
				t.Errorf("legitimate text was lost: %q", fn.Description)
			}
		}
		if !found {
			t.Fatal("hostile_description should be offered, sanitised, not skipped")
		}
	})

	t.Run("the tool list is namespaced", func(t *testing.T) {
		var names []string
		for _, tool := range manager.Tools() {
			names = append(names, tool.QualifiedName())
		}
		for _, name := range names {
			if !strings.HasPrefix(name, "raw__") {
				t.Errorf("tool %q is not namespaced", name)
			}
		}
		if len(names) == 0 {
			t.Fatal("expected at least one usable tool")
		}
	})
}

func skipReason(t *testing.T, state ServerState, tool string) string {
	t.Helper()
	for _, skipped := range state.Skipped {
		if skipped.Name == tool {
			return skipped.Reason
		}
	}
	t.Fatalf("tool %q was not skipped; skipped list is %v", tool, state.Skipped)
	return ""
}

// TestCloseReapsTheChildProcess proves that closing the manager leaves no
// orphaned server process. The child is a real process; after Close it must no
// longer exist.
//
// Honest note on who provides this: the protocol library's stdio transport
// closes stdin, signals, then kills, so today the guarantee is largely its.
// Attempts to falsify an Ollama-side reaping mechanism all failed, and that
// mechanism was removed rather than kept as unfalsifiable decoration. The test
// stays because the property is one Ollama promises its users, and it must keep
// holding if the transport is ever swapped or a second transport is added.
func TestCloseReapsTheChildProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process listing differs on windows; covered on unix")
	}
	binary := buildRawServer(t)

	manager := NewManager(Options{ConnectTimeout: 30 * time.Second, Approvals: allowAll{}})
	cfg := &Config{}
	cfg.Set("raw", &ServerSpec{Command: binary})
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != StatusConnected {
		t.Fatalf("status = %q, err = %v", state.Status, state.Err)
	}
	if !processRunning(t, binary) {
		t.Fatal("the server process should be running while connected")
	}

	if err := manager.Close(); err != nil {
		t.Fatalf("Close: %v", err)
	}

	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		if !processRunning(t, binary) {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Error("the server process outlived Close; a manager that leaks children leaks them for the whole session")
}

// TestFailedHandshakeLeavesNoOrphanProcess covers the harder case: a server
// that starts and then never answers, so no session ever exists for anyone to
// close. The connection must be recorded as failed and the process must not
// survive.
//
// This began as the test meant to justify per-server context cancellation in
// the manager. It did not: with that cancellation removed the test still
// passed, because the protocol library closes the transport when the handshake
// fails. The cancellation was removed and the test kept — a promise to users is
// worth guarding whoever currently keeps it.
func TestFailedHandshakeLeavesNoOrphanProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process listing differs on windows; covered on unix")
	}
	binary := buildRawServer(t)

	manager := NewManager(Options{ConnectTimeout: 2 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set("silent", &ServerSpec{Command: binary, Args: []string{"-silent"}})
	manager.Connect(t.Context(), cfg)

	state, _ := manager.State("silent")
	if state.Status != StatusFailed {
		t.Fatalf("status = %q, want failed: a server that never answers must not read as connected", state.Status)
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !processRunning(t, binary) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Error("the server process outlived a failed handshake; nothing else will ever reap it")
}

// TestConnectTimeoutDoesNotKillTheServer guards the defect that the connect
// timeout, being cancelled on the way out of the dial, must not take the child
// process with it. A server that dies the moment it finishes connecting looks
// exactly like a server that connected.
func TestConnectTimeoutDoesNotKillTheServer(t *testing.T) {
	binary := buildRawServer(t)

	manager := NewManager(Options{ConnectTimeout: 2 * time.Second, CallTimeout: 10 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set("raw", &ServerSpec{Command: binary})
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != StatusConnected {
		t.Fatalf("status = %q, err = %v", state.Status, state.Err)
	}

	// Well past the connect timeout: if the process were tied to it, it is dead
	// by now and this call fails.
	time.Sleep(3 * time.Second)

	result, err := manager.Call(t.Context(), "raw__echo", map[string]any{"text": "still alive"})
	if err != nil {
		t.Fatalf("the server did not survive its own connect timeout: %v", err)
	}
	if result.Content != "echo: still alive" {
		t.Errorf("Content = %q", result.Content)
	}
}

func processRunning(t *testing.T, binary string) bool {
	t.Helper()
	output, err := exec.Command("ps", "-ax", "-o", "command").Output()
	if err != nil {
		t.Fatalf("ps: %v", err)
	}
	return strings.Contains(string(output), binary)
}

// TestMissingCommandFailsClearlyAndIsNotRetried proves a stdio server whose
// command does not exist fails once, with a message naming the command, rather
// than being retried into repeated process spawns.
func TestMissingCommandFailsClearlyAndIsNotRetried(t *testing.T) {
	manager := NewManager(Options{ConnectTimeout: 5 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set("absent", &ServerSpec{Command: filepath.Join(t.TempDir(), "no-such-binary")})

	start := time.Now()
	manager.Connect(t.Context(), cfg)
	elapsed := time.Since(start)

	state, _ := manager.State("absent")
	if state.Status != StatusFailed {
		t.Fatalf("status = %q, want failed", state.Status)
	}
	if !strings.Contains(state.Err.Error(), "not found") {
		t.Errorf("error = %v, want it to say the command was not found", state.Err)
	}
	if elapsed > 2*time.Second {
		t.Errorf("failing took %s; a missing command must not be retried with backoff", elapsed)
	}
}

// TestDisablingAServerStopsItsProcess proves that switching a server off
// actually switches it off. Dropping it from the tool list while leaving the
// process running would be a switch that does not switch anything off, and
// nothing else in the system would ever notice.
func TestDisablingAServerStopsItsProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process listing differs on windows; covered on unix")
	}
	binary := buildRawServer(t)

	manager := NewManager(Options{ConnectTimeout: 30 * time.Second, Approvals: allowAll{}})
	t.Cleanup(func() { manager.Close() })

	cfg := &Config{}
	cfg.Set("raw", &ServerSpec{Command: binary})
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != StatusConnected {
		t.Fatalf("status = %q, err = %v", state.Status, state.Err)
	}
	if !processRunning(t, binary) {
		t.Fatal("the server should be running while connected")
	}

	// The user switches it off and the configuration is applied again, which is
	// exactly what /mcp disable and ollama mcp disable do.
	spec, _ := cfg.Get("raw")
	spec.Disabled = true
	manager.Connect(t.Context(), cfg)

	if state, _ := manager.State("raw"); state.Status != StatusDisabled {
		t.Fatalf("status = %q, want disabled", state.Status)
	}
	if len(manager.Tools()) != 0 {
		t.Error("a disabled server must offer no tools")
	}

	deadline := time.Now().Add(15 * time.Second)
	for time.Now().Before(deadline) {
		if !processRunning(t, binary) {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Error("the server process survived being switched off")
}
