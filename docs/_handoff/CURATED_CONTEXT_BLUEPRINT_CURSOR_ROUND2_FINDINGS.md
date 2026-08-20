# Curated Context Blueprint — Cursor Round 2 Findings Handoff

| Field | Value |
|---|---|
| Status | `[HANDOFF]` |
| Author | Fathom (GLM-5.2) |
| Date | 2026-08-18, Australia/Brisbane |
| Reviewer | Cursor agent (gpt-5.3-codex via cursor-agent CLI, --mode ask --print) |
| Document reviewed | `docs/CURATED_CONTEXT_BLUEPRINT.md` (v2, 659 lines) |
| Verdict | BLOCKING — 14 findings |
| Handoff purpose | Each finding explained with correction needed, for a future colleague to fix |

---

## How to use this handoff

Each finding below has: the Cursor finding (verbatim or paraphrased), what is wrong, what needs to change, and which file/section to edit. Fix all 14, then re-trigger the Cursor adversarial review via the host-bridge. Repeat until PASS.

The host-bridge is at `/Users/krypto/GitHub/TECHNE/Tools/host-bridge/`. Create `pending_command.sh` with a `# bridge-protocol: v1` header, touch `_run_now.flag`, poll for `_run_done.flag`, read `_terminal_output.log`.

---

## F1 — LETHE transport mismatch (BLOCKING)

**Cursor finding:** The client targets HTTP `/store`, `/recall`, and `/delete` endpoints (blueprint L359-430), but LETHE exclusively serves JSON-RPC over stdio (`../LETHE/README.md:3-6,37,54-56`). The integration cannot function.

**What is wrong:** The `LetheClient` struct in the blueprint assumes LETHE has an HTTP REST API with `/store`, `/recall`, `/delete` endpoints. LETHE is an MCP server that speaks JSON-RPC over stdio. There is no HTTP API.

**What needs to change:**
- Replace the HTTP-based `LetheClient` with a stdio JSON-RPC client that speaks the MCP protocol
- The client must spawn the LETHE server as a child process (or connect to an existing stdio session) and communicate via JSON-RPC 2.0
- The MCP tool names are `lethe-memory__store`, `lethe-memory__recall`, `lethe-memory__scopes`, etc.
- Store requests go through `lethe-memory__store` with `{scope, content, tags}` 
- Recall requests go through `lethe-memory__recall` with `{query, scope, limit}`
- There is no tag-based filtering in recall — see F3
- There is no delete-by-tag endpoint — see F12
- File: `agent/lethe_client.go` — full rewrite of the transport layer
- File: blueprint §3.2 — full rewrite of the LetheClient interface

---

## F2 — AgentName field doesn't exist in CompactionRequest (BLOCKING)

**Cursor finding:** `req.AgentName` is required at blueprint L121,198,247, while `CompactionRequest` has no such field (`agent/compactor.go:56-71`) and the blueprint mandates no change to that file (blueprint L54).

**What is wrong:** The blueprint uses `req.AgentName` throughout but `CompactionRequest` (defined in `agent/compactor.go`) has no `AgentName` field. The blueprint says `compactor.go` will not be modified.

**What needs to change:**
- Option A: Add `AgentName` field to `CompactionRequest` in `agent/compactor.go` (requires modifying the existing file, contradicting the "no change" claim)
- Option B: Pass `AgentName` through a different mechanism — e.g., a new field on `RunOptions` in `agent/session.go`, or as metadata in the `CompactionRequest.Options` map
- Option C: Create a new `CuratorRequest` struct that wraps `CompactionRequest` and adds `AgentName`
- Recommended: Option B — add `AgentName` to `RunOptions` and pass it through to the Curator separately from the CompactionRequest
- File: blueprint §3.1 and §4.2 — update to show how AgentName flows through

---

## F3 — Recall doesn't support tag filtering (BLOCKING)

**Cursor finding:** Recall attempts tag filtering (blueprint L214-218,244-249), but LETHE recall accepts only `scope`, `query`, and `limit` (`../LETHE/Sources/lethe-memory-mcp/MemoryTools.swift:95-104`). Tags become session-title text (`:574-590`), so agent/chat isolation is impossible.

**What is wrong:** The blueprint assumes LETHE recall can filter by tags. LETHE's `recall` tool accepts only `query`, `scope`, and `limit`. Tags are stored as part of the content metadata but are not queryable filters.

**What needs to change:**
- The tag-based agent/chat isolation strategy does not work with current LETHE
- Option A: Include agent name and chat ID in the query string itself (e.g., `query: "Fathom chat-abc123 ARCHE-Int"`), so semantic search naturally filters by context — this is lossy and unreliable
- Option B: Modify LETHE to add tag-based filtering to the recall tool — this requires changing the LETHE server, not just the Ollama fork
- Option C: Store each agent's memories in a separate scope (e.g., `connected:fathom`, `connected:eko`) — this requires LETHE to support dynamic scopes
- Option D: Use the `surface_relevant` tool instead of `recall`, which accepts a `context` parameter that could include agent/chat identifiers
- Recommended: Option B or D — discuss with Ash which approach fits LETHE's architecture
- File: blueprint §3.1 (retrieveSummaries, searchRelevantChunks) and §3.2 (RecallRequest) — rewrite the retrieval strategy

---

## F4 — Wrong LETHE scope (BLOCKING)

**Cursor finding:** Every operation hardcodes `connected:claude-code` (blueprint L196-218,244-248,493-516), although LETHE identifies Ollama as `connected:ollama` (`../LETHE/Sources/lethe-memory-mcp/main.swift:313-320`). This can access another client's memory or fail consent.

**What is wrong:** The blueprint uses `connected:claude-code` as the scope for all LETHE operations. LETHE has registered Ollama as `connected:ollama`. Using the wrong scope either fails consent or accesses another client's memory.

**What needs to change:**
- Replace all `connected:claude-code` with `connected:ollama` (or the appropriate scope that LETHE has registered for the Ollama client)
- The scope should be configurable, not hardcoded
- Add scope to `CuratorOptions`
- File: blueprint §3.1, §3.2, §4.3 — replace all hardcoded scope references

---

## F5 — TLS facade (BLOCKING)

**Cursor finding:** `TLSClientConfig` does not convert `http://remote-host` to HTTPS (blueprint L389-406), contradicting the TLS guarantee at L572-577. Prefix matching at L410-414 also classifies `http://localhost.evil` as localhost, bypassing authentication and TLS.

**What is wrong:** The TLS enforcement logic has two bugs: (1) it doesn't upgrade http:// to https:// for remote hosts, (2) the localhost prefix check can be bypassed by hostnames like `localhost.evil`.

**What needs to change:**
- Fix the localhost check to use proper hostname parsing (not prefix matching) — check if the hostname is exactly `localhost`, `127.0.0.1`, or `::1`
- For non-localhost URLs, require `https://` scheme and reject `http://`
- File: blueprint §3.2 (LetheClient) — fix `isLocalhost` and `NewLetheClient`

---

## F6 — Storage opt-out ignored (BLOCKING)

**Cursor finding:** `StoreMessagesInLethe` purports to control persistence (blueprint L101-102), but all content is stored unconditionally at L486-527. Disabling storage does nothing.

**What is wrong:** The `StoreMessagesInLethe` option in `CuratorOptions` is declared but never checked in the post-response storage code. All chunks and verbatim copies are stored regardless.

**What needs to change:**
- Wrap all post-response storage calls in `if c.Options.StoreMessagesInLethe { ... }`
- File: blueprint §4.3 (post-response step) — add the conditional check

---

## F7 — Prompt injection elevation (BLOCKING)

**Cursor finding:** Recalled, model-generated memory is inserted as trusted `system` messages (blueprint L264-276). Stored user instructions can therefore gain system-level authority; no listed security test covers this boundary.

**What is wrong:** The curated context packet inserts summaries and synthesized narratives as `system` role messages. If a user has stored malicious instructions in LETHE (tagged as "important"), those instructions would be injected as system messages, gaining system-level authority.

**What needs to change:**
- Option A: Insert summaries and narratives as `user` role messages instead of `system` — they are context, not system instructions
- Option B: Add a sanitisation step that strips system-level instructions from recalled content before injection
- Option C: Add a boundary marker (e.g., `<memory_context>...</memory_context>`) and instruct the model to treat content within as context, not instructions
- Recommended: Option A + C — use `user` role with boundary markers
- File: blueprint §3.1 (constructCuratedMessages) — change role from `system` to `user` with markers
- Add a test: `TestCuratorDoesNotElevateUserInstructions` — store a user instruction tagged "important", verify it does not appear as a system message

---

## F8 — Curator unreachable in production (BLOCKING)

**Cursor finding:** The settings and toggle are specified at blueprint L547-570, but the actual entry point always constructs `SimpleCompactor` (`cmd/agent_tui.go:153-156`). No affected configuration or command file supplies an activation path, while `TestCuratorToggle` could still pass in isolation.

**What is wrong:** The blueprint describes how to toggle between Compactor and Curator via settings, but the actual TUI entry point (`cmd/agent_tui.go`) always constructs a `SimpleCompactor`. There is no code path that reads the settings and constructs a `Curator` instead. The toggle test passes in isolation but the Curator is never activated in production.

**What needs to change:**
- Add `cmd/agent_tui.go` to the affected files list (blueprint §2.2)
- Modify the TUI entry point to read the `context_management.mode` setting and construct either `SimpleCompactor` or `Curator`
- File: blueprint §2.2 — add `cmd/agent_tui.go` to modified files
- File: blueprint §4.2 — add the actual construction logic

---

## F9 — Recent-history construction loses protocol state (BLOCKING)

**Cursor finding:** `recentStart` is forced to at least 1 (blueprint L279-288), dropping the first user message when no system message occupies index zero. Ollama carries its system prompt separately (`agent/session.go:34-43,443-448`). Arbitrary message-count slicing can also orphan tool calls or results.

**What is wrong:** The `constructCuratedMessages` function slices messages by index, assuming index 0 is a system message. But Ollama carries its system prompt separately from the message list. The slicing can drop the first user message and can orphan tool calls (a tool call without its preceding tool result, or vice versa).

**What needs to change:**
- Don't slice by raw index. Instead, keep the last N complete conversation turns (user + assistant + tool calls + tool results as a unit)
- Don't assume index 0 is system — check the role
- Preserve tool call/result pairs — never split them
- File: blueprint §3.1 (constructCuratedMessages) — rewrite the slicing logic to preserve conversation turn boundaries

---

## F10 — Agent switching is a locked no-op (BLOCKING)

**Cursor finding:** `SwitchAgent` saves, recovers, and changes nothing (blueprint L293-305,537-542). Post-response writes at L492-527 do not acquire the mutex, contradicting the non-interleaving guarantee at L591-593. Race tests can pass simply because the tested method has no effect.

**What is wrong:** The `SwitchAgent` method acquires the mutex but doesn't actually change any state — it saves and recovers but doesn't update the current agent or swap conversation history. Additionally, the post-response storage writes (§4.3) don't acquire the mutex, so they can interleave with agent switching.

**What needs to change:**
- `SwitchAgent` must actually change the current agent name, swap the conversation history, and update the ChatID
- Post-response writes must acquire the mutex (or use a separate storage mutex)
- The agent switching test must verify that the conversation history actually changed, not just that the method didn't panic
- File: blueprint §3.1 (SwitchAgent) — implement actual state change
- File: blueprint §4.3 (post-response step) — wrap in mutex
- File: blueprint §7.3 — add `TestAgentSwitchChangesHistory` that verifies history actually changes

---

## F11 — Write failures are silently lossy (BLOCKING)

**Cursor finding:** The ledger claims all store errors are returned (blueprint L23), but post-response writes only log failures (L491-526). The promised retry at L499 has no retry mechanism.

**What is wrong:** The post-response storage code catches errors and logs them but doesn't return them or retry. The blueprint's correction ledger says "all store errors are returned" but the actual code doesn't do this. The "retry with backoff" comment at L499 has no actual retry implementation.

**What needs to change:**
- Implement an actual retry mechanism (e.g., 3 retries with exponential backoff)
- Return errors from post-response storage so the session knows storage failed
- Log the error AND return it, not just log it
- Add a test that verifies errors are propagated, not silently logged
- File: blueprint §4.3 (post-response step) — implement retry and error propagation
- File: blueprint §7.3 — add `TestStoreFailurePropagated` that verifies errors reach the caller

---

## F12 — Summary deduplication cannot work (BLOCKING)

**Cursor finding:** The blueprint repeatedly calls append-only `Store` (blueprint L186-225) while requiring `TestCuratorDoesNotDuplicateSummaries` at L627. LETHE creates fresh session and exchange UUIDs for every store (`../LETHE/Sources/lethe-memory-mcp/MemoryTools.swift:583-605`); no update/delete mechanism is specified.

**What is wrong:** LETHE is append-only. Every `Store` call creates a new memory with fresh UUIDs. The blueprint calls `Store` for summaries every `SummaryInterval` messages, which creates duplicate summaries. The test `TestCuratorDoesNotDuplicateSummaries` cannot pass because there is no way to update or delete old summaries.

**What needs to change:**
- Option A: Use the `forget` tool to delete old summaries before storing new ones — this requires knowing the memory IDs of old summaries
- Option B: Accept that summaries accumulate and handle deduplication at retrieval time (retrieve all, keep the most recent)
- Option C: Modify LETHE to support update-in-place for tagged memories
- Recommended: Option B — retrieve all summaries for a type, keep the one with the highest score or most recent timestamp
- File: blueprint §3.1 (retrieveSummaries) — change to retrieve all and select the most recent
- File: blueprint §7.3 — update `TestCuratorDoesNotDuplicateSummaries` to verify that retrieval returns only the latest summary

---

## F13 — Zero-value configuration panics (BLOCKING)

**Cursor finding:** `SummaryInterval` has only a comment declaring its default (blueprint L88-106); no constructor or validation applies it. The modulo at L180-182 panics when the Go zero value is used.

**What is wrong:** `CuratorOptions` has default values in comments but no constructor function. If a user creates a `Curator` with `CuratorOptions{}` (zero value), `SummaryInterval` is 0, and `len(req.Messages) % 0` panics with division by zero.

**What needs to change:**
- Add a `DefaultCuratorOptions()` constructor function that sets all defaults
- Add validation in the Curator constructor that checks for zero values and applies defaults
- Guard the modulo: `if c.Options.SummaryInterval > 0 && len(req.Messages) % c.Options.SummaryInterval == 0`
- File: blueprint §3.1 — add `DefaultCuratorOptions()` and validation

---

## F14 — Wrong context option key (BLOCKING)

**Cursor finding:** The blueprint reads `context_window_tokens` (blueprint L151-169), while Ollama's runtime contract uses `num_ctx` (`agent/compactor.go:213-220`). The advertised float64 test can pass while the real context override remains ignored.

**What is wrong:** The blueprint reads `context_window_tokens` from the options map to determine the context window size. Ollama uses `num_ctx` as the option key. The blueprint's key is wrong, so the configured context window size is never read in production.

**What needs to change:**
- Change the key from `context_window_tokens` to `num_ctx`
- Also handle the float64 type properly (this was already fixed in the blueprint but the key was wrong)
- File: blueprint §3.1 (ContextWindowTokens) — change key to `num_ctx`

---

## F15 — Unbounded AGORA scope creep (BLOCKING)

**Cursor finding:** The blueprint adds an AGORA Workspace UI (blueprint L581-585,653-654) while declaring no UI or AGORA files affected (L38-55) and claiming AGORA remains unchanged (L642). No implementation boundary exists for this work.

**What is wrong:** The blueprint describes a dual interface (Ollama + AGORA workspace) with agent switching, UI animation, and AGORA integration, but lists no UI files or AGORA files as affected. The "Compatibility" section claims AGORA remains unchanged, which contradicts the dual interface design.

**What needs to change:**
- Option A: Remove the AGORA dual interface from this blueprint entirely. Make it a separate future blueprint. This blueprint should only cover the Curated Context System (Curator + LETHE + chunking + synthesis).
- Option B: Add all affected UI files and AGORA files to §2.2, and update the compatibility section to acknowledge changes
- Recommended: Option A — the AGORA dual interface is a separate feature that should be its own blueprint. This keeps the blueprint focused and bounded.
- File: blueprint §5 (Dual interface) — remove or mark as future scope
- File: blueprint §2.2 — remove AGORA references from affected files
- File: blueprint §8 (Compatibility) — remove the "AGORA unchanged" claim since it's now out of scope

---

## Summary of fixes needed

| Finding | Severity | Fix complexity | Files affected |
|---|---|---|---|
| F1 — LETHE transport | Blocking | High (full transport rewrite) | lethe_client.go, blueprint §3.2 |
| F2 — AgentName field | Blocking | Medium (new field or wrapper) | session.go, blueprint §3.1, §4.2 |
| F3 — No tag filtering | Blocking | High (requires LETHE change or alternative strategy) | lethe_client.go, blueprint §3.1 |
| F4 — Wrong scope | Blocking | Low (string replacement) | All scope references |
| F5 — TLS facade | Blocking | Low (fix hostname check) | lethe_client.go, blueprint §3.2 |
| F6 — Storage opt-out | Blocking | Low (add conditional) | blueprint §4.3 |
| F7 — Prompt injection | Blocking | Medium (change role + add markers) | blueprint §3.1, §7.3 |
| F8 — No activation path | Blocking | Medium (add TUI wiring) | cmd/agent_tui.go, blueprint §2.2, §4.2 |
| F9 — History slicing | Blocking | Medium (turn-based slicing) | blueprint §3.1 |
| F10 — Agent switch no-op | Blocking | Medium (implement state change + mutex) | blueprint §3.1, §4.3, §7.3 |
| F11 — Silent write failures | Blocking | Medium (retry + error propagation) | blueprint §4.3, §7.3 |
| F12 — Summary deduplication | Blocking | Medium (retrieval-side dedup) | blueprint §3.1, §7.3 |
| F13 — Zero-value panic | Blocking | Low (add constructor + guard) | blueprint §3.1 |
| F14 — Wrong option key | Blocking | Low (change key name) | blueprint §3.1 |
| F15 — AGORA scope creep | Blocking | Low (remove from this blueprint) | blueprint §5, §2.2, §8 |

**Total: 14 blocking findings. 4 low complexity, 6 medium, 4 high.**

---

## Next steps for the colleague

1. Fix all 14 findings using the corrections above
2. Update the blueprint to v3 with a correction ledger (like the ARCHE-Int process)
3. Re-trigger the Cursor adversarial review via host-bridge:
   - Create `pending_command.sh` at `/Users/krypto/GitHub/TECHNE/Tools/host-bridge/`
   - Content: `#!/bin/bash` + `# bridge-protocol: v1` + `cd /Users/krypto/GitHub/ollama` + cursor-agent review command
   - Touch `_run_now.flag`
   - Poll for `_run_done.flag`
   - Read `_terminal_output.log`
4. If BLOCKING: fix findings, re-review
5. If PASS: blueprint is ready for implementation

---

## Cursor review command template

```bash
#!/bin/bash
# bridge-protocol: v1
cd /Users/krypto/GitHub/ollama
/Users/krypto/.local/bin/cursor-agent --mode ask --print "You are an adversarial blueprint judge. Review the implementation blueprint at docs/CURATED_CONTEXT_BLUEPRINT.md in this repository. Look for: 1) Facades that could pass tests, 2) Missing error handling, 3) Race conditions, 4) Security issues with the LETHE client, 5) Incorrect Go types or interfaces, 6) Scope creep, 7) Transport mismatches with LETHE (JSON-RPC over stdio, not HTTP), 8) Wrong scope identifiers. Return BLOCKING findings only with specific line references. If none, return PASS." 2>&1
echo "exit=$?"
```

---

*End of document.*