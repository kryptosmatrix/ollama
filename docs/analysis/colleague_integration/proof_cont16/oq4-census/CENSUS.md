# G1 OQ-4 closure census — 2026-10-01

**Feature:** OLLAMA-G1-INSTRUCTIONS (OI-03 + OI-04)
**Blueprint:** `docs/_design/G1_PERSISTED_INSTRUCTIONS.md`
**Candidate/source SHA:** `465b963e35f6afa46f15e6fdec043189a905d1d8`
**Production/test bytes:** unchanged from `04009c432e313eef65172293e75feb038ebe476b` after that candidate's documentation-only lineage.
**Method:** TECHNE Method 16 v2.2.1, Tier 2. This file is evidence for the blueprint amendment; the blueprint itself publishes the frozen reach closure.

## 1. Frozen requirement set

The set is exactly `G1-R-01` … `G1-R-18`. The source document contains eighteen and only eighteen RFC-2119 `MUST` requirement lines. OQ-4 does not reopen the requirement set. Ash's 27 September ruling that classic interactive and one-shot `ollama run` are in G1 changes the implementation closure needed to satisfy G1-R-05/G1-R-06/G1-R-07; it does not add a nineteenth outcome.

## 2. Frozen reach closure

Modes: **VERIFIED** means the production symbol exists at the source SHA. **DESIGNATED** means G1 creates or extends the named site; Pack A must convert every DESIGNATED site to VERIFIED or return `SPEC_INSUFFICIENT`.

| File | Mode | Production symbols in the closure | Why in closure / owner |
|---|---|---|---|
| `internal/instructions/types.go` | DESIGNATED | `Profile`, `Binding`, `ConversationKey`, `Selection`, `ReloadEvent`, `Error` | G1 representation; B1 |
| `internal/instructions/store.go` | DESIGNATED | `Open`, `Current`, `Save`, `Bind`, `Reload`, revision/binding readers | sole instruction-state writer/reader; B1 |
| `internal/instructions/compose.go` | DESIGNATED | `Compose` | sole carrier composer; B1 |
| `internal/instructions/lease_unix.go` | DESIGNATED | conversation lease acquisition/release | busy/reload/turn serialisation; B1 |
| `internal/instructions/lease_windows.go` | DESIGNATED | conversation lease acquisition/release | Windows equivalent; B1 |
| `api/types.go` | VERIFIED + DESIGNATED | `GenerateRequest` line 62; `ChatRequest` line 133; `ChatResponse` line 520; DESIGNATED admission request/report types | request/response boundary; B2a |
| `api/client.go` | VERIFIED + DESIGNATED | `Generate` line 277; `Chat` line 297; `Version` line 454; DESIGNATED capabilities + typed admission errors | both clients and refusal propagation; B2a/B2b |
| `server/routes.go` | VERIFIED + DESIGNATED | `GenerateHandler` line 250; `ChatHandler` line 2417; `truncateNativeChatMessages` line 3010; version route registration | daemon admission, local/cloud/remote routing; B2a |
| `server/prompt.go` | VERIFIED + DESIGNATED | `chatPrompt` line 23 | rendered-history fitting; B2a |
| `server/cloud_admission.go` | DESIGNATED | qualification table, `utf8-bound-v1` prompt bound, request/profile admission | cloud admission; B2a |
| `server/cloud_proxy.go` | VERIFIED + DESIGNATED | `proxyCloudJSONRequest` line 158; `proxyCloudJSONRequestWithPath` line 165 | cloud forwarding/report insertion boundary; B2a |
| `llm/server.go` | VERIFIED | `LlamaServer.ContextLength() int` line 77 | effective local context producer |
| `llm/llama_server.go` | VERIFIED + DESIGNATED | `completionPromptForRequest` line 276; `ContextLength` line 330 | rendered-route cut/refusal and keep-whole enforcement; B2a |
| `agent/session.go` | VERIFIED + DESIGNATED | `Session.chatRound` line 385; `buildChatRequest` line 441 | agent turns/tool continuations; B2b |
| `agent/compactor.go` | VERIFIED | `SimpleCompactor.summarize` line 230; `estimateCompactionRequestTokens` line 383 | summariser intentionally does not carry carrier; estimator reads effective system prompt |
| `cmd/agent_tui.go` | VERIFIED + DESIGNATED | `GenerateAgentTUI` line 73; `preloadAgentModelIfLocal` line 583 | agent-TUI conversation/open/capability path; B2b |
| `cmd/tui/chat/chat.go` | VERIFIED + DESIGNATED | `Run` line 216; `resetChat` line 997 | agent-TUI identity/new conversation lifecycle; B2b |
| `cmd/tui/chat/input.go` | VERIFIED + DESIGNATED | `submitInput` line 145; `handleSystemCommand` line 424; `systemPrompt` line 1719 | effective base, slash controls, turn producer; B2b |
| `cmd/tui/chat/compaction.go` | VERIFIED + DESIGNATED | `startManualCompaction` line 13 | compaction busy gate and effective-prompt estimate; B2b |
| `cmd/tui/chat/debug.go` | VERIFIED + DESIGNATED | `handleSaveCommand` line 28; `handlePromptCommand` line 51 | /prompt and /save consume the composed request; B2b |
| `cmd/tui/chat/render.go` | VERIFIED + DESIGNATED | existing chat entry rendering; D-2 error-entry path | operator-visible refusal report; B3 |
| `cmd/cmd.go` | VERIFIED + DESIGNATED | `loadOrUnloadModel` line 543; `RunHandler` line 736; `chat` line 1758; `generate` line 1884 | classic chat + one-shot/piped `ollama run`; B2b |
| `cmd/interactive.go` | VERIFIED + DESIGNATED | `generateInteractive` line 34 plus /clear,/load,/save,/set-system command path | classic conversation selection and instruction commands; B2b |
| `cmd/instructions.go` | DESIGNATED | `ollama instructions set` command | terminal editor entry; B3 |
| `app/store/store.go` | VERIFIED | `ChatWithOptions` line 428 | real desktop transcript producer/status |
| `app/store/database.go` | VERIFIED | `getChatWithOptions` line 677; `getSettings` line 1213 | real desktop persistence producer; not instruction owner |
| `app/ui/instructions.go` | DESIGNATED | authenticated instructions GET/PUT/revisions/preview/reload handlers | desktop instruction API; B3 |
| `app/ui/ui.go` | VERIFIED + DESIGNATED | `Server.createChat` line 481; `Server.chat` line 668; `Server.buildChatRequest` line 1909 | real desktop turn/request consumer; B2b |
| `app/ui/app/src/api.ts` | VERIFIED + DESIGNATED | existing app API client + DESIGNATED instruction methods | frontend transport; B3 |
| `app/ui/app/src/components/Settings.tsx` | VERIFIED + DESIGNATED | `Settings` line 49 + InstructionSettings mount | desktop editor surface; B3 |
| `app/ui/app/src/components/InstructionSettings.tsx` | DESIGNATED | `InstructionSettings` | save/disable/reset/preview; B3 |
| `app/ui/app/src/components/InstructionStatus.tsx` | DESIGNATED | `InstructionStatus` | selected-revision/reload/refusal display; B3 |
| `app/ui/app/src/components/Chat.tsx` | VERIFIED + DESIGNATED | `Chat` line 40 + status/reload mount | conversation surface; B3 |
| `app/ui/app/src/hooks/useChats.ts` | VERIFIED + DESIGNATED | existing chat stream/event consumer + admission diagnostics | final report/refusal consumer; B3 |

No production path outside this set constructs, mutates, consumes, rejects, truncates or reports the G1 instruction carrier or its selection/admission state in the frozen design. A judge re-derives this set and set-differences it from the published blueprint closure.

## 3. Reachable chains and behaviour-changing readers

### 3.1 Desktop save → new conversation → actual chat request
`InstructionSettings` → frontend API → DESIGNATED `app/ui/instructions.go` → `internal/instructions.Repository.Save` → durable SQLite revision/current pointer. On a real `Server.chat` turn (`app/ui/ui.go:668`), `ChatWithOptions` supplies existing/new transcript state, the repository supplies the pinned selection, the extended `Server.buildChatRequest` (`:1909`) builds the existing model/MCP base and `Compose` appends the carrier. `api.Client.Chat` (`api/client.go:297`) sends to `Server.ChatHandler` (`server/routes.go:2417`). Admission changes history fitting/refusal, then the response admission object changes the operator notice. Observable: captured outgoing request carries the selected revision text; save/restart/new, edit/continue and reload trials change or preserve the carrier exactly as specified.

### 3.2 Agent TUI
`GenerateAgentTUI` (`cmd/agent_tui.go:73`) creates the terminal conversation and selection → `chat.Run` (`cmd/tui/chat/chat.go:216`) → `chatModel.systemPrompt` (`cmd/tui/chat/input.go:1719`) builds the existing base then `Compose` → `Session.chatRound` (`agent/session.go:385`) → `buildChatRequest` (`:441`) → `Client.Chat` → daemon. Tool continuations repeat the same route. Observable: request capture and D-2 per-turn refusal entry. `/system off` changes only the built-in base; enabled custom instructions remain.

### 3.3 Classic interactive `ollama run`
`RunHandler` (`cmd/cmd.go:736`) → `generateInteractive` (`cmd/interactive.go:34`) binds one conversation. Classic turns reach `chat` (`cmd/cmd.go:1758`). The latest explicit `/set system` text is the base; otherwise the model system text is the base; `Compose` adds the selected carrier. `/clear` and `/load` create a new selection before replacing old state. Observable: captured /api/chat request and attributable refusal line; a refused turn is not placed in the next request history.

### 3.4 One-shot and piped `ollama run`
Non-interactive `RunHandler` → `generate` (`cmd/cmd.go:1884`). One immutable validated unbound snapshot (generation 0) is composed with the model system text into `GenerateRequest.System`; the prompt remains `Prompt`. `Client.Generate` (`api/client.go:277`) → `GenerateHandler` (`server/routes.go:250`). `generate.admission.v1` is required before a carrier-bearing request. Observable: stdout answer unchanged; admission/refusal diagnostics are stderr; no binding/opened event is written.

### 3.5 Compaction
`startManualCompaction` (`cmd/tui/chat/compaction.go:13`) passes the effective system prompt only to the compaction estimate. `SimpleCompactor.summarize` (`agent/compactor.go:230`) continues to send its fixed summariser system prompt and archive, not the carrier. After compaction, the next real turn re-composes from the pinned selection. Observable: summariser request golden remains carrier-free while the next model request still carries the pinned revision.

## 4. Default-closed emitters

| Gate/default | Production emitter/opener | Behaviour-changing reader | Observable |
|---|---|---|---|
| `Profile.enabled=false` | authenticated PUT / `ollama instructions set` commits enabled profile | `Compose` | carrier absent → present |
| no conversation binding | first accepted desktop turn; normal agent/classic conversation open; explicit reload | selection lookup | selected revision/generation |
| one-shot generation 0 | one-shot invocation reads current validated profile | `Compose` | request uses snapshot but binding count remains unchanged |
| no admission field | adapter attaches v1 admission only for enabled selection after capability check | ChatHandler/GenerateHandler | admitted/refused request and report |
| capability absent | daemon advertises `chat.admission.v1` / `generate.admission.v1` only when implemented | each adapter checks before every carrier-bearing request | no inference on absence |
| non-null identity binding inactive | G2 validator in later blueprint; **not G1** | G1 save/enable validator | `binding_unresolved` |

## 5. Worked arithmetic/extreme cells

All arithmetic is integer division unless stated.

| Quantity | Producer / allowed input | Minimum worked cell | Maximum/result-changing worked cell | Survivorship / consequence |
|---|---|---|---|---|
| default local reserve R | runtime C from `llm.LlamaServer.ContextLength`; G1 admits local carrier only with C ≥ 32 | C=32: C/8=4; clamp lower→R=16; C−R=16 | C=32768: C/8=4096→R=4096; every C>32768 remains R=4096 | C<32 is refused `context_unavailable` before carrier dispatch; supplied reserve must be 0..C/2 |
| local fit | rendered/native prompt estimate P and C−R | P=C−R survives intact | P=C−R+1 refuses | no partial newest turn; A6 reports C,R,P |
| revision decimal | edit CAS expected revision, signed-64 domain | `0` parses to 0 | `9223372036854775807` parses; next revision attempt refuses capacity/overflow | leading zero/sign/20 digits reject `invalid_input` |
| revision retention | store cap 1024 | count=0 save creates first retained revision | count=1024 next retention-requiring save refuses before write | `capacity_exceeded`; old current remains |
| bindings | store cap 20000 | count=0 first bound conversation allowed | count=20000 new binding refuses; one-shot gen-0 snapshot does not consume cap | `capacity_exceeded` |
| reload events per conversation | cap 256 | generation=1; first real reload produces generation=2 | 256 retained reloads: next reload refuses before mutation | old selection remains |
| instruction text | UTF-8 byte producer in editor/CLI | 0 bytes allowed only disabled; enabled whitespace-only rejects | 65536 bytes accepted when valid UTF-8/NUL-free | max+1 rejects before write |
| raw PUT body | HTTP reader cap | smallest valid edit object survives | 524288 bytes accepted by raw cap; 524289 rejects before parse | decoded field caps still apply; whitespace counts |
| cloud bound B | `utf8-bound-v1` over rendered request | base 16 plus actual fields/allowances | B=V admitted; B=V+1 refuses before forwarding | no history dropping; report `prompt_bound` |
| cloud 80% notice | final admission report | bound < floor(0.8V): no notice | bound ≥ ceil(0.8V): room-left notice | notice is diagnostic only, never admission evidence |

The reserve contract is clarified here: **C<32 is unsupported for an enabled local carrier** because no reserve can satisfy the default floor 16 and A2's at-most-half-context invariant simultaneously. This is a design correction, not current behaviour.

## 6. Write-path guards

Pre-code workloads for new instruction-store quantities are zero because no G1 store/endpoint exists. Existing request-path workloads use the measured continuation-13 values.

| Site / mode | Guard / bound | Current live/shipped value | Breach behaviour and observable | Forecloses | Breach proof |
|---|---|---:|---|---|---|
| DESIGNATED `Repository.Save` | text ≤65536, valid UTF-8, no NUL; enabled nonblank | 0 G1 profiles | `invalid_input`, no write | unbounded/private binary instruction payloads | 65537 + NUL + blank-enabled |
| DESIGNATED API parser | raw body ≤524288, one object, unknown members refused | endpoint absent | HTTP error/code, no state access/write | ambiguous/multi-object edits | 524289/padded body |
| DESIGNATED `Repository.Save` | expected revision CAS | 0 G1 revisions | `revision_conflict`; old current remains | silent last-writer-wins | two processes same expected revision |
| DESIGNATED store | retained revisions ≤1024 | 0 | `capacity_exceeded` pre-write | unbounded history | small injected cap + organic writes |
| DESIGNATED store | bindings ≤20000 | 0 | `capacity_exceeded` pre-bind | unbounded binding growth | small injected cap + organic opens |
| DESIGNATED store | reload events ≤256/conversation | 0 | `capacity_exceeded`; selection unchanged | unbounded event growth | small injected cap + organic reloads |
| DESIGNATED lease | active turn/compaction owns conversation | existing TUI has `running/compacting` state, no G1 lease | `conversation_busy` | mid-turn revision mutation | held real turn/compaction |
| DESIGNATED decoder | schema=1 + digest exact | no G1 rows | `unsupported_schema`/`corrupt_record` | silent state reinterpretation | raw corrupt/schema fixtures |
| DESIGNATED binding admission | enabled requires binding=null | no G1 bindings | `binding_unresolved` | fake founded identity | plausible non-null binding |
| `server/prompt.go:chatPrompt` VERIFIED, amended | today drops oldest messages when >C; G1 fit target C−R whole turns | measured packed prompts 494/498 at C=512 | admitted path reports dropped count; newest/system overflow refuses | silent carrier/newest-turn loss | I-13 whole-turn/tool-group cases |
| `server/routes.go:truncateNativeChatMessages` VERIFIED, amended | same local fit | measured C=512 | same | silent native-route history loss | I-13 native cases |
| `llm/llama_server.go:completionPromptForRequest` line 276 VERIFIED, amended | current rendered cut; G1 admitted requests keep whole prompt/refuse cut | measured 4289-token prompt cut to 258/511 | `context_exceeded` before inference | silent tail/carrier loss | clean→cut mutant→clean |
| native llama-server guard (external pinned dependency) | prompt ≤ selected slot; context shift clamp | C=512 measured | existing loud error remains backstop | over-capacity evaluation | continuation-13 native oversize |
| DESIGNATED local admission | C≥32; reserve 0..C/2; estimate ≤C−R | measured C=512 | `invalid_admission`, `context_unavailable` or `context_exceeded` with counts | silent mandatory-input loss | I-13 |
| `GenerateHandler` line 250 + DESIGNATED generation admission | raw/template/suffix/legacy-context carrier combinations refused | current feature absent | `invalid_admission`; no inference | carrier outside chat-like semantics | C-4 cases |
| DESIGNATED capabilities | required route capability before **every** carrier request | current daemon advertises none | client `context_unavailable`; no inference | old daemon silently ignoring unknown field | T10/D11 + classic cases |
| DESIGNATED admitted final report | required on successful carrier request | current daemon has none | missing report → `context_unavailable`, result unverified | false success with no retention evidence | report-drop mutant |
| DESIGNATED cloud table | exact origin+model+profile and B≤V | measured V 99680/103671/25722 | `not_qualified` or `context_exceeded`; nothing forwarded | unmeasured cloud carrier delivery | I-13 synthetic provider; live qualification only with Ash authority |
| existing auth boundary + DESIGNATED instruction handlers | authenticated Server.Handler | existing product auth | existing auth rejection before instruction parse | unauthorised instruction read/write | real token/no-token test |

Generic pre-existing model-existence, model-capability, think, format and provider-auth checks remain authoritative and can reject the complete request, but they do not transform, truncate, persist or selectively drop the G1 carrier. Pack A records their unchanged presence; G1 does not claim to replace their errors with instruction errors.

## 7. Shipping configuration × trigger matrix

| Shipping configuration | Trigger that arms G1 | Required proof / re-trigger |
|---|---|---|
| Desktop app | saved profile enabled; first accepted chat binds; explicit Apply reloads | Playwright real settings→restart→new-chat; failed store/capability turn remains retryable after store/daemon restoration without changing the profile |
| Agent TUI | normal open or /new binds; /instructions reload/off | I-05 + failure-number terminal arm; capability is re-read on each carrier-bearing turn |
| Classic interactive `ollama run` | session open binds; /clear and /load start a new conversation; /instructions reload/off | classic instrument scenarios; refused turn excluded from later request history |
| One-shot/piped `ollama run` | process invocation snapshots current profile as generation 0 | one-shot instrument cases; no durable binding/opened event |
| Local daemon /api/chat | v1 admission present | I-13 native + rendered; missing field is legacy path unchanged |
| Local daemon /api/generate | generation admission v1 present | C-4 conformance; promptless load/unload unchanged |
| Explicit cloud via qualified /api/chat | exact A13 table/profile, within V | synthetic provider tests by default; live provider requalification needs Ash |
| Headless daemon without G1-aware client | no profile injection occurs | ordinary requests remain unchanged; daemon only enforces admission when field is supplied |

## 8. Boundary vocabulary totality

| Boundary | Allowed values | Unknown/invalid behaviour |
|---|---|---|
| instruction `Error.code` | `invalid_input`, `revision_conflict`, `conversation_busy`, `store_unavailable`, `corrupt_record`, `unsupported_schema`, `binding_unresolved`, `capacity_exceeded`, `context_unavailable`, `context_exceeded` | decoder refuses/propagates; never success |
| daemon admission refusal `code` | `invalid_admission`, `context_unavailable`, `context_exceeded` plus existing provider errors passed through unchanged | unrecognised daemon/provider error passes through as error with original status |
| `ChatAdmission.version` / generation admission version | exactly 1 | HTTP 400 `invalid_admission` |
| capability | `chat.admission.v1`, `generate.admission.v1` | absent/other capability does not authorise carrier dispatch |
| admission report route | `local` or `cloud` | client marks request unverified / `context_unavailable` |
| cloud `limit_kind` | `verified_prompt` | client marks report invalid/unverified |
| `ReloadEvent.kind` | `opened`, `reloaded` | persisted unknown value → `corrupt_record` |
| selection generation | 0 only for ephemeral one-shot; bound conversations start 1 and increment on reload | other impossible/overflow values refused |
| profile schema | 1 | `unsupported_schema` |

## 9. Requirement ↔ acceptance closure

| Requirement | Acceptance IDs |
|---|---|
| G1-R-01 | I-01, I-04 |
| G1-R-02 | I-01, I-07, I-08 |
| G1-R-03 | I-02, I-04, I-05 |
| G1-R-04 | I-03 |
| G1-R-05 | I-02, I-05, I-09, I-10, I-13, I-14 |
| G1-R-06 | I-05, I-06, I-10, I-13 |
| G1-R-07 | I-10, I-13, I-14 |
| G1-R-08 | I-01, I-03, I-09 |
| G1-R-09 | I-07 |
| G1-R-10 | I-04, I-05 |
| G1-R-11 | I-05 |
| G1-R-12 | I-01, I-07 |
| G1-R-13 | I-04, I-06, I-11 |
| G1-R-14 | I-03, I-07, I-08, I-14 |
| G1-R-15 | I-06, I-13, I-14 |
| G1-R-16 | I-12 + combined gate |
| G1-R-17 | deliberate-break controls attached to I-01..I-14 |
| G1-R-18 | I-01..I-14 + failure-number/report verdict + combined gate |

Reverse closure is enforced by adding a `Requirement IDs` and `Mode` column to every I-01..I-14 row in the blueprint. Published pre-code check:

```sh
python3 docs/analysis/colleague_integration/proof_cont16/oq4-census/check_requirement_acceptance.py
```

Expected: `G1_REQUIREMENT_ACCEPTANCE_BIJECTION=PASS requirements=18 acceptance=14`.

## 10. Acceptance modes

At least I-01, I-02, I-05, I-09, I-10, I-13 and I-14 are **ORGANIC**: they enter through the shipping editor/CLI/chat/daemon entry points and do not inject instruction state after entry. I-03 is ORGANIC for the held-turn/compaction state. I-04 is ORGANIC across process restart. I-06 is ORGANIC disabled/unconfigured parity. I-07 and I-08 include **INJECTED** corruption/limit arms alongside organic happy arms. I-11 is ORGANIC store round-trip. I-12 is a gate/instrument criterion. Deliberate faults/mutants are **INJECTED controls**, never positive activation evidence.

## 11. OQ-4 disposition

The author-side OQ-4 closure census is complete enough to amend the blueprint and to repair the OQ-3 instrument against stated text. It does **not** itself earn a Method-16 stamp. Before implementation:

1. amend the blueprint with the frozen reach set, D-1…D-11/C-1…C-7 decisions, worked cells, closure guards, config matrix, reverse requirement mapping and boundary table;
2. repair the failure-number instrument to revision 5 and run it against a fresh pre-G1 baseline/goldens;
3. run the required fresh Tier-2 two-substrate Method-16 panel on the same `document_sha` and `source_sha`;
4. only an `EARNED` outcome enables implementation packs.
