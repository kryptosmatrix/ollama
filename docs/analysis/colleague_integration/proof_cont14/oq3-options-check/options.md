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
