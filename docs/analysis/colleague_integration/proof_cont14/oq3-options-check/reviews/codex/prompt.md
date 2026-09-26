You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at commit a7c35fd65e732cd378dc6d2e93ccc43f152efcef. You may open source and test files anywhere under agent/, api/, app/, cmd/, envconfig/, internal/, llm/, server/ and x/, and the two design files docs/_design/G1_PERSISTED_INSTRUCTIONS.md and docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md. Do not open anything else under docs/ (earlier evidence and reviews are withheld on purpose so that your answer is independent) or any file outside the repository. Change nothing and run nothing that writes. Where you verify or refute a fact at source, cite file:line.

---

# Options check: an instrument that counts incorrect saved-instruction deliveries (Ollama fork, G1 OQ-3)

**What you are asked to do.** For each of questions Q1-Q7 in section 5, choose an option, give the strongest objection to every option (including the ones you choose), and say whether a materially better option is missing. Then answer Q8-Q10. Say whether anything in sections 3 and 4 is wrong at source, citing file:line. There is no preferred answer; the options are listed in no order of preference.

## 1. The feature

G1 lets the operator save standing instructions once, in the desktop app or from the terminal. Every new conversation in the terminal agent and the desktop app then sends them to the model inside a leading system message (the "carrier"). An active conversation keeps the revision it started with until an explicit reload. Nothing of G1 is implemented at the commit under review: this is the pre-build baseline. The instrument asked for (OQ-3) must count incorrect deliveries through the real desktop HTTP and terminal entry points, be run on the unchanged candidate to give the baseline number, and be reusable as the final acceptance instrument once G1 is built.

## 2. The contract, quoted verbatim from `docs/_design/G1_PERSISTED_INSTRUCTIONS.md` at commit `a7c35fd65e732cd378dc6d2e93ccc43f152efcef`

You may open this file and the concept `docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md`. Each line below is prefixed with its line number.

```text
18: Ash edits and saves instructions once, restarts, and sees the chosen text influence a genuinely new conversation through its actual outgoing model request. An active conversation keeps its original instruction revision until a deliberate safe reload. The failure number is **incorrect instruction deliveries per controlled conversation**, measured over save/restart/new, edit/continue, explicit reload, disable, terminal reset and compaction cases: any value above zero fails G1. The final instrument is DESIGNATED `app/ui/instructions_test.go::TestInstructionsAcrossProcesses` plus `cmd/tui/chat/instructions_test.go::TestInstructionConversationLifecycle` and the browser journey in §9. The pre-build measured baseline in continuation-11 covers existing system-prompt transport and no-truncation behaviour only; it is NOT this full failure-number measurement. That missing baseline is an explicit promotion obligation, not a zero invented from absence.
```

```text
35: **Open obligations before promotion.** OQ-1 is decided (§6 and `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md`); what remains of it are the verification obligations V1–V7 in §6.5, carried into the packs. OQ-2 is resolved prospectively below: G1 permits an unbound ordinary instruction profile; all active non-null colleague bindings are refused until the separately commissioned G2 validator is implemented and qualified. This does not count G2 as delivered. OQ-3: install and execute the full failure-number baseline through the actual HTTP/terminal entry points. The attempted read of the existing TTS test fixture was platform-blocked in continuation-11; no child ran and that specific file was not accessed by another route. This does not make existing system-prompt tests a substitute for the missing baseline. OQ-4: complete independently derived closure/guard/test population and the two-substrate Method16 panel. All were assigned to Eko and are assigned to Thole from 26 September 2026; no new operator approval is requested. No build pack starts with them open.
```

```text
79: Disabled compatibility means: no additional system text, no change to messages, tool schemas, model options, thinking, format or existing approval controls, and no additional model call. Small instruction metadata writes may still establish a disabled revision binding; that is explicitly not a byte-identical filesystem claim. Existing desktop conversations without a binding select revision zero, even if a new global configuration is enabled. They require explicit reload to adopt it. Desktop conversations bind on the first accepted `Server.chat` POST after acquiring the conversation lease and reading the real transcript status, before the first SetChat or model request. `createChat` only allocates the opaque ID and does not bind a revision. An empty UI draft is therefore not an active conversation; its preview is prospective. Terminal conversations bind on successful normal terminal creation or idle `/new`, before any prompt can be submitted. Neither a settings-editor mount nor an ID allocation is treated as a persisted chat message. Existing transcript content is never copied to the instruction store and is never truncated, reset or migrated by this feature.
```

```text
175: ## 4. Conversation lifecycle and concurrency
176: 
177: The instruction store owns selections; the original desktop store and terminal model own transcripts. First creation, failed first turn, existing conversation, `/new`, model switch, tool switch, compaction, explicit reload and process restart are distinct events.
178: 
179: A per-conversation process-safe advisory lease is held across a turn or compaction, including pending tools and approval waits. It is keyed by SHA256 of length-prefixed namespace and ID and represented by a lock file under the instruction root. Lock-file contents are not an authority record. OS lock release on process exit handles abandonment; no timestamp is interpreted as proof a worker died. Darwin/Linux and Windows lock implementations are DESIGNATED `lease_unix.go`/`lease_windows.go`; a failed nonblocking acquire returns `conversation_busy` without replacing a running operation. Profile Save uses only a short SQLite transaction and can complete while an existing conversation holds its selected revision.
180: 
181: | Event | Preconditions | Transition and observable |
182: |---|---|---|
183: | New conversation | Host-issued non-empty ID; lease acquired | Transaction binds the current validated revision and generation 1; `opened` event; no inference on failure |
184: | Legacy desktop conversation | Real saved chat exists and no instruction binding exists | Bind revision 0; never silently adopt current text |
185: | Continue existing conversation | Lease acquired; binding and revision decode successfully | Read pinned selection; current global revision is irrelevant |
186: | Save B while A is active | expected current revision matches | Current pointer becomes B; A's selection and outgoing carrier remain A |
187: | Reload | Conversation lease acquired; expected generation and current revision match | One transaction updates selection, increments generation and records `reloaded`; next request reads the new carrier and its explicit revision notice |
188: | Reload while running/compacting | Lease unavailable | `conversation_busy`; old selection and transcript unchanged |
189: | Model/tools change | Existing selection valid | Rebuild only the existing model/skill/tool base; reuse pinned profile and re-evaluate capacity for the actual new request |
190: | Compaction | Same conversation selection and lease | Reattach carrier from selection, not from summarised text; do not store duplicate system messages |
191: | Terminal `/new` | Idle; new selection can be created | Generate UUID and bind before clearing old terminal state; failure leaves old state usable |
192: | Restart | Store readable | New terminal invocation creates a new conversation using current profile; existing desktop ID reads its old selection |
193: 
194: A binding reserved before an initial transcript write may survive a failed first turn. This is intentional: it is an instruction-selection record, not a claim that a conversation message was saved. A retry under the same issued ID uses that reserved revision; starting a genuinely new ID selects the then-current profile. No cross-database atomicity is claimed. Missing or corrupt data after a binding exists blocks the affected request; it does not recreate the chat or select another colleague.
195: 
196: The existing terminal does not provide a normal disk-backed transcript-resume path in the inspected entry. G1 does not invent one or claim terminal transcript restoration. OI-07/OI-08 own that later continuity boundary. Distinct IDs, saved defaults and in-session selection survive the transitions G1 actually supports; reopening desktop transcripts is tested through their existing persistent owner.
```

```text
198: ## 5. Actual request composition
199: 
200: DESIGNATED `internal/instructions/compose.go::Compose(base string, selection Selection) (string,error)` is a pure function. Disabled selection returns the supplied base byte-for-byte. Enabled selection returns base plus a two-newline separator when base is non-empty, then the literal header `User-configured instructions; revision <revision>; selection <generation>:\n`, then the exact profile text. Profile hash/binding validity is checked before composition. This user-authored text does not alter executor approvals, tool availability or filesystem authority. Recalled prose and arbitrary repository files are not promoted into this carrier by G1.
201: 
202: Terminal: `GenerateAgentTUI` creates/loads the selection through the shared owner and passes it in a new `chat.Options.InstructionSelection` field, never folded into `Options.SystemPrompt`; `chatModel.systemPrompt` first performs its existing built-in-toggle/extra construction, then calls `Compose` with that base and the separate immutable selection. Consequently `/system off` removes only the built-in base, not enabled custom instructions; `startRunWithMessages` and manual compaction use that one effective result; `agent.buildChatRequest` receives the already-composed string. `/system on|off` keeps its existing built-in-prompt meaning; it is not silently redefined as the custom-instructions toggle.
203: 
204: Desktop: the real `Server.chat` derives new-versus-existing from its existing Store lookup, acquires the lease, resolves the selection before saving or invoking inference, then passes the selection and the actual model-system text from `Client.Show` to its extended existing request builder. The DESIGNATED new signature is `buildChatRequest(chat *store.Chat, model string, think any, availableTools api.Tools, mcpInstructions string, selection instructions.Selection, modelSystem string) (*api.ChatRequest, error)`. `Server.chat` supplies `show.System` from its already obtained ShowResponse, rather than a new unproved model-system lookup. Before implementation, all callers/tests of this signature are enumerated and updated; the old source signature is not claimed to already accept these fields. When enabled, model-system text and existing MCP instructions form the base in that order, omitting only empty blocks; the shared composer appends the configured text. The builder adds one effective system message before the conversation. Existing role/tool/attachment conversion remains in that builder. When disabled, its old path remains exactly as before, including the current MCP instruction ordering. Adding a system message without preserving the model's system text fails G1-R-06.
205: 
206: Contract use in both adapters: every request that carries an enabled carrier also carries the §6.3 A1 admission field, and only after the adapter has confirmed the daemon's `chat.admission.v1` capability (A12). That includes every tool continuation: the desktop's pass loop, which begins at `app/ui/ui.go:943` and sends each request at `:978`, and the agent's follow-up requests. A disabled selection sends no admission field, so its request is unchanged (G1-R-15). Both adapters read the final record's `admission` object on every route. The desktop must read it before its early return for records with no content, thinking or tool calls (`app/ui/ui.go:992-994`). Both classify a refusal by its `code` before any message-text rule, because the desktop's rules match text such as "402", which a count can contain (`app/ui/ui.go:598-606`). A refusal under A6, A10 or A13, or a missing capability, is mapped by §8 and shown to the operator. The adapter never retries without the carrier or the field.
207: 
208: Preview is a point-in-time result for the explicit `(model, revision, selection_generation, base_digest)` returned alongside it. It returns the exact effective carrier for that tuple through the same construction helpers. The view labels a prospective new-conversation preview as unbound. Changing any tuple member, including a reload generation, makes that preview stale; it is never represented as proof of a later dispatch. It is labelled a preview, not proof of dispatch, model compliance, memory recovery or available context. Actual-request diagnostics separately record revision, generation, source digest, carrier byte count, model and request outcome without storing private instruction text in public logs or repository evidence.
```

```text
305: Existing conversations display their selected revision and a button `Apply saved instructions` at the conversation header. The button is disabled while the UI knows the conversation is active, and the backend lease remains authoritative. A successful reload displays the new generation/revision; an error leaves the existing selection visible. The editor does not pretend that saving automatically updates open conversations.
```

```text
309: Terminal DESIGNATED `/instructions` displays current selection and scope; `/instructions reload` applies the saved revision only at idle; `/instructions off` records and selects a disabled revision through the same store rather than silently changing only a prompt string. Saved editing is provided by the desktop surface and DESIGNATED `ollama instructions set --file <path> --expected-revision <revision>` for terminal-only use. Input is read as data, never executed as a shell/editor command. Exact CLI registration and validation anchors are part of OQ-4, not assumed present.
```

```text
331: ## 9. Acceptance design, organic producers and deliberate faults
332: 
333: Test storage is isolated; actual production entry/assembly is exercised. External inference may be simulated for transport/store tests, clearly declared. Selected real-model qualification is a separate bounded run; recorded model name/digest, actual served identity, request/response evidence and limits remain attributable. No real user conversation, Keychain or credential is used as a fixture.
334: 
335: | Test ID | Organic path and source-derived assertion | Facade/twin caught |
336: |---|---|---|
337: | I-01 | Real editor/API saves contrasting A and B; fresh process reads the exact last committed bytes | Fixed string, in-memory-only store |
338: | I-02 | Start X at A, save B in another process, send twice in X and once in new Y; capture actual requests | Current-settings-on-every-turn, same default conversation ID |
339: | I-03 | Start a held turn/compaction, attempt real reload, then finish and reload at idle | UI-only gate, always-allow, never-allow |
340: | I-04 | Restart desktop host and reopen X; request uses X's pinned A until explicit reload | NewChat replacement, regenerate from current profile |
341: | I-05 | Normal terminal entry, `/new`, model/tool switches, builtin `/system off` with custom instructions still enabled, and real compaction events | Unused new ID, selection lost on reset, append-only duplicate systems |
```

```text
348: | I-12 | All required tests are discovered, finish, and reconcile; a child exit 7, empty selection and missing case fail the gate | Vacuous pass and shell-exit proxy |
```

```text
354: The eight standard degenerate twins are constant output, identity/no-op, always-success, always-reject, never-write, never-read, all-zero/empty selection and dropped configured input. I-01/I-02 detect constant/empty/dropped input; I-06 detects no-op/always-enabled; I-07 detects always-success; I-01/I-03 detect always-reject; I-04 detects never-write/never-read. A disconnected final composer, stale-binding loader and failed CAS are additional specific mutants. Mutations occur only in isolated copies after candidate/tests are committed. Each mutant compiles first and then fails its intended assertion; clean before and after runs pass. A compile failure or missing browser binary is an infrastructure failure, not detection credit.
355: 
356: Existing baseline, executed this continuation: `go test -count=1 -json -timeout=120s -run '^(TestSessionAddsSystemPromptOnlyToRequest|TestChatSystemCommandControlsBuiltInSystemPrompt)$' ./agent ./cmd/tui/chat` passed two named tests; `go test -count=1 -json -timeout=120s -run '^TestChatPrompt$/^no_truncate_with_limit_exceeded$' ./server` passed its parent and named subtest. They prove existing request construction and retained over-limit prompt behaviour, not G1 persistence, context admission or a passing new-feature failure number.
357: 
358: Continuation-12 recovered the previously timed-out entry baseline without changing its test: the authenticated existing-settings control passed; the instructions PUT returned HTTP 200 with the application HTML rather than a saved Profile and failed its JSON/value assertion. Both named tests completed, process exit 1. A separately built real CLI passed ordinary --help and rejected instructions --help with unknown-command/exit 1. These are actual pre-feature entry failures, not a save/restart success or a census of every lifecycle scenario. Commands, logs, isolated environment and exact test/binary hashes are retained in continuation-12 http-baseline-original/ and cli-baseline/.
359: 
360: DESIGNATED default Go test files are `internal/instructions/store_test.go`, `lease_test.go`, `compose_test.go`, `app/ui/instructions_test.go`, `cmd/tui/chat/instructions_test.go`, and `cmd/instructions_test.go`; named test functions and requirement assignments are completed before promotion. Default frontend tests cover API/state transitions without a browser dependency; real Playwright acceptance uses the existing installed Playwright package, actual frontend routes and isolated real backend. The browser suite is mandatory G1 acceptance, never substituted with SSR.
```

## 3. What exists at this commit (the unchanged candidate), with source anchors

Verify any of these at source; cite file:line if you find one wrong.

1. Desktop. `ui.Server` fields `app/ui/ui.go:100-137`; `Handler` `:212-359` registers `POST /api/v1/create-chat` (`createChat` `:481-493`, allocates an ID only) and `POST /api/v1/chat/{id}` (`chat` `:668-`; the literal id `new` creates a chat, `:686-695`). There are no `/api/v1/instructions` routes; `GET/PUT/POST/PATCH/DELETE /` fall through to `appHandler` (`:352-356`). The chat handler calls `Show` on the daemon (`:801`, `:870`), builds the request with `buildChatRequest` (`:973`, defined `:1909-2004`: MCP instructions become a leading system message when non-empty, `:1914-1916`; no model-system text is added) and sends it to `envconfig.Host()` (`inferenceClient`, `:400-402`). The tool pass loop is `:943-`. The desktop app assembles the server at `app/cmd/app/app.go:208-287` (`store.Store{}` with its default path, a tool registry, approvals, an MCP manager, an updater).
2. Terminal. The agent chat is reached only through the launcher: `runInteractiveTUI` (`cmd/cmd.go:2161-2199`) → `runLauncherAction` (`:2223-`) → `launchInteractiveModel` (`:2130-2158`, which calls `prepareAgentModel` then `GenerateAgentTUI`). `GenerateAgentTUI` (`cmd/agent_tui.go:73-`) builds the system prompt (`:112`; `agentSystemPromptWithWorkingDir` `:214-253` includes the current date and the working directory) and calls `agentchat.Run` (`cmd/tui/chat/chat.go:216-267`), which runs a Bubble Tea program on the process's own stdin and stdout (`:256`). Slash commands: `cmd/tui/chat/input.go:54-67` and `:153-190`; `/new` is `resetChat` (`chat.go:997-1022`, which does not change `chatID`); `/compact` is `startManualCompaction` (`compaction.go:13-69`); `/system on|off` is `handleSystemCommand` (`input.go:424-`), and the prompt is `systemPrompt` (`input.go:1719-1728`). The agent request is built at `agent/session.go:441-467`. The compaction summariser sends its own fixed system prompt, not the conversation's (`agent/compactor.go:230-250`; `CompactionRequest.SystemPrompt` is used only for an estimate at `:385-387`).
3. `ollama instructions` is an unknown command at this commit (measured earlier: exit 1). `PUT /api/v1/instructions` with the designated envelope returned HTTP 200 and the application's HTML (measured earlier).
4. Existing test seams: `cmd/tui/chat/test_helpers_test.go` (`chatCaptureClient` records `*api.ChatRequest`), used by `cmd/tui/chat/input_test.go:972-` to drive a `chatModel` constructed directly, below `GenerateAgentTUI`.

## 4. The instrument as proposed so far (not in question unless you object)

- Capture boundary: a fake daemon (an `httptest` server) at `OLLAMA_HOST` that records every request body sent to `/api/chat` and answers `/api/version`, `/api/show` (with a model-system text carrying a random sentinel), `/api/tags` and whatever else the entry points call, streaming a short assistant reply. This is a declared simulation of the external inference service.
- Isolation: a fresh `HOME` per run; no Keychain, credentials or real conversations.
- Instruction texts A and B contrast, and each carries unique random sentinels at its start, middle and end.
- Every step that uses a designated surface absent from this commit is attempted through that surface (the designated `PUT /api/v1/instructions` envelope; `ollama instructions set --file <path> --expected-revision <revision>`; `/instructions reload` and `/instructions off` in the terminal; `POST /api/v1/chat/{id}/instructions/reload`), and the step's actual outcome is recorded.
- The report gives the numerator, the denominator, a per-scenario and per-category breakdown (carrier missing, wrong revision or generation, duplicated, text altered, carrier present when none is expected, base prompt lost, request never sent) and every raw captured request.
- Proposed scenario matrix. Each conversation identity is one controlled conversation. "Expect A" means the carrier for revision 1 (text A) and selection generation 1 unless stated; "no carrier" means the request must equal the request the unchanged candidate sends for the same input.
  - D1 save/restart/new: desktop process P1 saves A; P1 exits; P2 starts on the same HOME; new chat X, two turns, expect A on both.
  - D2 edit/continue: P1 saves A; new chat X turn 1 (A); P1 saves B; X turns 2 and 3 expect A; new chat Y turn 1 expects B (revision 2).
  - D3 explicit reload: X at A; save B; reload X to revision 2 at generation 1; next X turn expects B at generation 2.
  - D4 disable: save A then save disabled; new chat Z turn 1 expects no carrier.
  - D5 legacy: chat L created and answered before any instruction is saved; save A; L turn 2 expects no carrier (revision 0) until an explicit reload.
  - D6 restart and reopen: P1 saves A; X turn 1 (A); saves B; P1 exits; P2: X turn 2 expects A (pinned); new chat Y expects B.
  - T0 unconfigured control: terminal process with no instruction configured; two turns expect no carrier.
  - T1 save/restart/new: CLI process sets A; a new terminal process; turn 1 expects A; turn 2 triggers a tool call answered by the fake daemon, and the continuation request also expects A.
  - T2 edit/continue and terminal reset: terminal P1 at A turn 1; CLI sets B; P1 turn 2 expects A; `/new`; turn 3 is conversation Y and expects B.
  - T3 explicit reload: P1 at A; CLI sets B; `/instructions reload`; next turn expects B at generation 2.
  - T4 disable: P1 at A turn 1; `/instructions off`; next turn expects no carrier.
  - T6 compaction: P1 at A, four turns, `/compact`, next turn expects A exactly once and no duplicated system message.
  - T7 built-in prompt off: P1 at A; `/system off`; next turn expects the carrier alone (the built-in base removed, the custom instructions kept), per §5.

## 5. Questions

**Q1 — The number.** The blueprint defines it as "incorrect instruction deliveries per controlled conversation", failing above zero.
- (a) Count every outgoing model request that belongs to a controlled conversation as one delivery; the number is incorrect deliveries divided by controlled conversations, reported with both counts.
- (b) Score each controlled conversation 0 or 1 (1 if any of its deliveries is incorrect); the number is failing conversations divided by conversations.
- (c) Incorrect deliveries divided by total deliveries (a per-request rate).

**Q2 — Which requests are deliveries.** Candidates: user-turn requests; tool-continuation requests; the compaction summariser's request; model preload or warm-up requests; `/api/show`; any title or background generation.
- (a) Turn requests and tool continuations only; the summariser's request is recorded and checked separately (its system message must be the compactor's own and must not carry the carrier twice or leak it into history), and preload, show and background requests are recorded but not counted.
- (b) Every `/api/chat` request the host sends, including the summariser's, must carry the carrier.
- (c) Turn requests only.

**Q3 — The terminal boundary.**
- (a) Drive the built `ollama` binary in a pseudo-terminal through the launcher menu to the agent chat, with the fake daemon answering the launcher's calls.
- (b) A test compiled into package `cmd` (added by `go test -overlay`, so the default suite is untouched) that runs `launchInteractiveModel` in re-executed test-binary processes whose stdin and stdout are the slave side of a real pseudo-terminal, one OS process per terminal session; the CLI steps run as separate processes.
- (c) The designated location `cmd/tui/chat/instructions_test.go`: construct `chatModel` directly and drive `handleSubmit` and `startRun` with `chatCaptureClient`.

**Q4 — Desktop restart.**
- (a) Each desktop lifetime is a separate OS process (the test binary re-executed) serving the real `Handler` on the same HOME; the driver and the fake daemon live in the parent.
- (b) A new `ui.Server` value in the same process on the same database file.

**Q5 — Steps whose production surface does not exist yet.**
- (a) Attempt the step, record its outcome, continue the scenario, and score the deliveries that actually reach the daemon.
- (b) Stop the scenario at the first absent surface and score the conversation as failed without deliveries.
- (c) As (a), and additionally score every expected delivery that never reached the daemon as incorrect ("request never sent").

**Q6 — How exact the oracle is.**
- (a) The exact expected system message string. For the desktop this needs the join between the model-system text and MCP instructions, which the blueprint does not state; the fixture could configure no MCP server so that the base is the model-system text alone.
- (b) A structural predicate: exactly one message carries the carrier header; its revision and generation are the expected ones; the exact profile text follows the header and ends that system message; the base that precedes it is the expected base (terminal: equal to the system message the unchanged candidate sends for the same model, working directory and date; desktop: equal to the model-system text returned by `/api/show`); no other message contains any profile sentinel.
- (c) Sentinel presence only.

**Q7 — Validating the instrument itself.** Choose every control you think is required, and add any missing:
- (a) Scorer controls on synthetic captured sequences: a perfect sequence scores zero; each failure category, planted alone, is detected in its own category.
- (b) A reference double that sends correct carriers through the real HTTP transport to the fake daemon, to show the capture path reads a correct delivery as correct.
- (c) Harness faults: the fake daemon drops system messages, or the driver sends a step twice; the affected scenarios must change their verdict.
- (d) Runner controls: a failing child process, a missing required scenario and an empty scenario selection each block the run.
- (e) Repeat the baseline and require identical verdicts.

**Q8 — Blind spots.** Name every way this instrument could read zero while deliveries are wrong once G1 is implemented.

**Q9 — False alarms.** Name every way it could read above zero while deliveries are right.

**Q10 — Coverage.** Is the scenario matrix complete for the six named cases and for the blueprint's lifecycle table, and for "every tool continuation" on both adapters (the desktop has tools here only through MCP or web search)? Name what is missing and whether it belongs in this baseline or in the final instrument.
