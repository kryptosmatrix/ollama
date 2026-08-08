## Findings

### HIGH — Removed servers are never cleaned up

**File:** `mcp/manager.go`, lines 218–239

`Connect` iterates only `cfg.Names()` and handles each server in the new config. It never iterates `m.clients` or `m.states` to find servers that were present in a previous config but are absent from the new one. When `Connect` is called again after a configuration change (as the doc says it is), a server removed from the config is left in `m.clients` and `m.states` with its existing `StatusConnected` state.

**Scenario:** User configures server "foo", calls `Connect` — "foo" connects. User removes "foo" from the config and calls `Connect` again. `Connect` never calls `closeSession("foo")`, so "foo"'s subprocess keeps running, `Tools()` still returns its tools, `Schemas()` still offers them to the model, and `Call("foo__bar", ...)` still works. A server the user removed is still live and its tools are still offered.

---

### HIGH — Data race on `ServerState` fields between `Call` and `Close`

**File:** `mcp/manager.go`, lines 520–536 vs 568–573

`Call` obtains a `*ServerState` pointer under `RLock` (line 522), releases the lock (line 525), then reads `state.Status` (line 530) and iterates `state.Tools` (line 536) without holding any lock. `Close` writes `state.Status` and `state.Tools = nil` under the write lock (lines 569–571). These are unsynchronised concurrent accesses to the same memory.

**Scenario:** `Close` is called while `Call` is executing. `Call` read the `state` pointer before `Close` acquired the write lock. `Close` sets `state.Status = StatusDisabled` and `state.Tools = nil` while `Call` is reading those same fields. Because Go string and slice assignments are multi-word and non-atomic, `Call` can observe a torn slice header (a nil pointer with a non-zero length), causing a panic in `slices.ContainsFunc`. The race detector will also flag this.

---

### HIGH — Concurrent `Connect` calls can resurrect a disabled server

**File:** `mcp/manager.go`, lines 241–249 and 386–403

`Connect` launches goroutines that call `dial`, then blocks on `wg.Wait()`. If a second `Connect` is called while the first is still in flight (the Manager is documented as safe for concurrent use), the second call's `closeSession(name)` removes the server from `m.clients` and sets its state to `StatusDisabled`, but cannot cancel the first call's in-flight `dial` goroutine. When that goroutine completes, `dial` stores a new session into `m.clients[name]` and overwrites `m.states[name]` with `StatusConnected`.

**Scenario:** `Connect` #1 starts with "foo" enabled; its `dial` goroutine is waiting on a slow server. `Connect` #2 starts with "foo" disabled; it calls `closeSession("foo")` and sets state to `StatusDisabled`, then returns. `Connect` #1's goroutine finishes `dial`, stores the session, and sets state to `StatusConnected`. "foo" is now connected and its tools are offered, even though the user disabled it.

---

### MEDIUM — `dial` closes the previous session while holding the write lock

**File:** `mcp/manager.go`, lines 392–397

Inside the `m.mu.Lock()` critical section, `dial` calls `previous.Close()` and `previous()` (the release function). `closeSession` (lines 257–273) performs the same operations *outside* the lock, which is the correct pattern. `session.Close()` on a stdio server waits for the process to exit; if the process is unresponsive, this blocks the lock indefinitely, preventing all other callers (`Call`, `States`, `Close`, other `dial` goroutines) from proceeding.

**Scenario:** Server "foo" is being reconnected (config changed, still enabled). `dial` acquires `m.mu`, finds a previous session, and calls `previous.Close()`. The old server process is hung and doesn't respond to the shutdown signal. `m.mu` is held for the entire wait. Meanwhile, `Call` for server "bar" blocks on `m.mu.RLock()` and times out, even though "bar" is unaffected.

---

### MEDIUM — Stdio process leak on failed dial

**File:** `mcp/server.go`, line 174; `mcp/manager.go`, lines 360–363

For stdio, `newTransport` returns `nothingToRelease` (a no-op) as the release function. If `client.Connect` starts the subprocess but then fails (e.g., the server sends a malformed handshake), `dial` returns the error and the `defer` calls `release()` — which does nothing. The process was started with `exec.CommandContext(m.lifetime, ...)`, so it is only killed when `m.lifetime` is cancelled, which happens only in `Close`.

**Scenario:** A stdio MCP server starts, prints garbage on stdout, and causes `client.Connect` to fail. `dial` returns an error, `connectOne` sets state to `StatusFailed`, but the server process is still running. It remains running until the manager is closed. If the user reconfigures and the server is retried, each failed attempt can leave another orphaned process.

---

### LOW — `Connect` does not check `m.closed`

**File:** `mcp/manager.go`, line 206

`Connect` does not check `m.closed` before proceeding. If called after `Close`, it sets states (e.g., `StatusConnecting`) and launches goroutines. The goroutines reach `dial`, detect `m.closed` under the lock, close their sessions, and return errors. `connectOne` then overwrites the states with `StatusFailed`. This leaves a closed manager with `StatusFailed` entries and wastes resources launching goroutines that can never succeed.

**Scenario:** `Close` is called during shutdown. A concurrent or subsequent `Connect` call proceeds, sets states to `StatusConnecting`, launches goroutines that immediately fail, and leaves states as `StatusFailed` with "manager is closed" errors — confusing for a caller that inspects `States()` after closing.

---

**No defect found** in: `headerTransport.RoundTrip` (correctly clones the request and falls back to `DefaultTransport`), `signInRequiredTransport.RoundTrip` (correctly closes the body before returning an error), `childEnv` (correctly overlays declared variables on the inherited base), `toolsDigest` (correct hash over the tool set), or the `Connect` closure capture (Go 1.22+ per-iteration loop variable semantics apply, confirmed by `range attempts` on line 292).