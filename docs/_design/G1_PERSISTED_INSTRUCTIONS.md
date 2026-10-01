# G1 — Persisted instructions and actual request delivery

| Field | Value |
|---|---|
| Identity | OLLAMA-G1-INSTRUCTIONS; edition 1; one feature, OI-03 + OI-04 |
| Author | Eko, GPT-6 Astra Pro, 21 September 2026 (edition 1 draft); revised by Thole, Claude Opus 5.5, 26 September 2026; OQ-3 by Letterlock/Treadle, Claude Opus 5.5, 26–27 September 2026; OQ-4 operator contract by Abuttal, Claude Opus 5.5, 27 September 2026; OQ-4 closure census reconciled by Eko, GPT-5.6 Sol, 1 October 2026 |
| Document class | implementation blueprint |
| Design readiness | [SPEC-DRAFT]; OQ-1 decided in §6 and amended for cloud models (A13) on Ash's ruling of 26 September 2026; OQ-3 revision 4 measured 67/39 but is not yet fit as the acceptance gate; OQ-4's author-side closure census and operator contract are now incorporated below. Instrument revision 5, its fresh review, and the first Tier-2 Method 16 panel remain owed before any implementation pack may start |
| Implementation status | Not Yet Implemented |
| Source revision | `465b963e35f6afa46f15e6fdec043189a905d1d8`; production and default-test bytes relevant to G1 are unchanged from `04009c432e313eef65172293e75feb038ebe476b`. OQ-4 live anchors were re-derived on 1 October 2026 |
| Concept | docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md; SHA256 9396cd684d47d3e6e200c227e75e1a0114d7cd8a20107db528c4bc26871fb241 |
| Tier | 2: persistence yes; process/language crossing yes; estimate above 300 LOC yes |
| Requirement IDs | G1-R-01 through G1-R-18, 18 frozen outcome requirements; first adjudication has not started |
| Pack permission | PACKS_BLOCKED, derived from [SPEC-DRAFT] under Method 16 §2.1 and R-0; edition 1 authored `PACKS_ALLOWED_FOR_0A_0B_ONLY`, a token the ladder does not define, corrected here. Completing this document and its panel (§10, steps 0A and 0B) is design work, not a licensed pack |
| Scope | Shared saved instructions, immutable conversation selection, editor, safe explicit reload, desktop chat, agent TUI, classic interactive `ollama run`, one-shot/piped `ollama run`, and their admitted `/api/chat` / `/api/generate` request paths |
| Repository line ceiling | UNDECLARED; no 500-line ceiling inferred from generic guidance |

Ash edits and saves instructions once, restarts, and sees the chosen text influence a genuinely new conversation through its actual outgoing model request. An active conversation keeps its original instruction revision until a deliberate safe reload. The failure number is **incorrect instruction deliveries per controlled conversation**, measured over save/restart/new, edit/continue, explicit reload, disable, terminal reset and compaction cases: any value above zero fails G1. The instrument is `docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` (continuation 14; revision 4 from continuation 15): the real desktop handler, the terminal launch function and the built binary's launcher, each in its own process, against a recording daemon, over a frozen ledger of 27 scenarios and 39 trials. Its measured pre-build baseline is 67/39 = 1.7179 on a valid run of revision 4 (`proof_cont15/oq3-baseline-r4/BASELINE.md`; revision 2 read 66/39 on a ledger without T8's added turn), and its accept mode must read 0 against the frozen pre-G1 goldens, as one component of §10's combined gate, once a revision passes review (revision 4 did not: `proof_cont15/oq3-instrument-review-r4/DISPOSITIONS_CODEX.md`). Where it runs in the final suite is an OQ-4 decision: the DESIGNATED `cmd/tui/chat/instructions_test.go` sits below `GenerateAgentTUI`, where the terminal selection is created, so the terminal arm cannot live there; `app/ui/instructions_test.go::TestInstructionsAcrossProcesses` and the browser journey in §9 stay owed.

## 0. Intake, authority and unresolved design obligations

Mode: Create, reconciling the commissioned G1 outcome with existing source. Ash's current Compass continuation authorises ordinary technical decisions, implementation and scoped Git work, conditioned on advance explanation and preservation of recoverable data. That authority does not issue a method stamp or make this draft implemented. The separate history-repair maintenance exception is not used to promote G1.

This blueprint is the custodian of the new instruction profile, revision, conversation binding and reload-event vocabulary. It does not redefine the conversation transcript, model identity, LETHE schemas, tool permission state or goal state. Existing terminal and desktop paths remain separate adapters over one new instruction owner.

| Decision | Selected direction | Alternative rejected | Risk / revision path |
|---|---|---|---|
| D1 | One instruction repository, shared by both adapters | Separate desktop and terminal settings with later synchronisation | Divergent effective instructions; test cross-process edit/consume before acceptance |
| D2 | Immutable revisions plus a current pointer and per-conversation bindings | Read current text on every turn | Active-session drift; test saved B while active A remains A |
| D3 | Instruction bindings are metadata, never a second transcript | Duplicate messages in an instruction or memory database | Divergent archive; instructions store contains no message text or model replies |
| D4 | Explicit reload acquires the same conversation lease as a turn | UI-only disabled button or timestamp heuristic | Reload/turn race; real competing processes are the falsifier |
| D5 | Extend existing request construction, with one shared composition function | Parallel agent engine or replacement system template | Compatibility drift; source-derived carrier golden plus disabled request equality |
| D6 | One opt-in, versioned daemon admission contract for carrier-bearing requests (§6.3); cloud models admitted up to a verified prompt limit and refused above it (A13) | Shift off; keep-all without a reserve; client-side prompt assembly; a version-number gate; cloud allowed with a label; cloud history fitted to a byte bound (ADR 2026-09-26 and its amendment 1) | Reserve floor chosen, estimate gap measured at zero (§6.5 V1); cloud limits are dated evidence (A13); the ADR's falsifiers reopen it |

**Open obligations before promotion.** OQ-1 is decided (§6 and `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md`); its V1–V8 verification obligations remain pack/reality-gate work. OQ-2 is resolved prospectively: G1 permits an unbound ordinary instruction profile and refuses active non-null colleague bindings until G2 exists. OQ-3 revision 4 and its 67/39 baseline remain valid measurements but the instrument itself is not fit as the gate; revision 5 must implement the accepted round-4 repairs and the now-stated OQ-4 semantics, then receive a fresh review. OQ-4's author-side reach/guard/arithmetic/config/boundary census is published in §1.1 and `docs/analysis/colleague_integration/proof_cont16/oq4-census/CENSUS.md` (SHA-256 `7313465c77391dee6aae925d72df0cb9c54bd07de8e05cbad523352db5fcb2d8`), incorporating `ADR_2026-09-27_G1_OQ4_OPERATOR_CONTRACT.md` including Ash's classic-run ruling. The independent Tier-2 Method 16 panel is still owed. No build pack starts before an `EARNED` `[IMPLEMENTATION-READY]` result.

## 1. Frozen requirements

G1-R-01 MUST persist the operator's instruction configuration across process restart under the shared-owner contract in §3.
G1-R-02 MUST assign a distinct immutable revision to each accepted changed configuration under §3.
G1-R-03 MUST retain a conversation's selected revision until the explicit reload transition in §4.
G1-R-04 MUST reject a reload while that conversation has an active turn or compaction under §4.
G1-R-05 MUST compose the selected instruction carrier into the actual desktop and terminal model requests under §5.
G1-R-06 MUST preserve each path's existing model, tool and protocol semantics under the compatibility contract in §2.
G1-R-07 MUST report unavailable or exceeded context capacity without silently discarding mandatory input under §6.
G1-R-08 MUST expose edit, save, scope, revision, preview, disable and reversible reset through the editor contract in §7.
G1-R-09 MUST reject an unresolved active colleague binding under §3’s inactive-binding contract.
G1-R-10 MUST preserve the selected revision through compaction and model/tool changes under §4.
G1-R-11 MUST give terminal new-conversation creation a distinct identity under §4.
G1-R-12 MUST reject invalid or conflicting instruction writes without overwriting the prior accepted revision under §3.
G1-R-13 MUST keep the canonical conversation archive separate from instruction metadata under §2.
G1-R-14 MUST surface instruction recovery failures through the typed failure contract in §8.
G1-R-15 MUST preserve ordinary unconfigured and disabled-mode request behaviour under §2.
G1-R-16 MUST preserve reproducible boundary evidence for the exact candidate under §9.
G1-R-17 MUST demonstrate that the specified deliberate faults fail the behavioural criteria in §9.
G1-R-18 MUST pass the declared combined G1 acceptance surface before OI-03 or OI-04 closes under §10.

### 1.1 Frozen reach closure — OQ-4

Method 16 R-2 reach closure is frozen at source `465b963e35f6afa46f15e6fdec043189a905d1d8` on 1 October 2026. `VERIFIED` means the live symbol exists at that source; `DESIGNATED` means G1 creates or extends it and Pack A must convert it to VERIFIED or return `SPEC_INSUFFICIENT`. The independent derivation, worked arithmetic, write-path guard population, configuration matrix and boundary census are retained in `docs/analysis/colleague_integration/proof_cont16/oq4-census/CENSUS.md` (SHA-256 `7313465c77391dee6aae925d72df0cb9c54bd07de8e05cbad523352db5fcb2d8`).

| File | Mode | Closure symbols / reason |
|---|---|---|
| `internal/instructions/types.go` | DESIGNATED | `Profile`, `Binding`, `ConversationKey`, `Selection`, `ReloadEvent`, typed `Error` |
| `internal/instructions/store.go` | DESIGNATED | `Open`, `Current`, `Save`, `Begin`, `Reload`; authoritative instruction-state writer |
| `internal/instructions/compose.go` | DESIGNATED | `Compose`; sole carrier composer |
| `internal/instructions/lease_unix.go`; `lease_windows.go` | DESIGNATED | process-safe conversation lease |
| `api/types.go` | VERIFIED + DESIGNATED | `GenerateRequest:62`, `ChatRequest:133`, `ChatResponse:520`; admission request/report types |
| `api/client.go` | VERIFIED + DESIGNATED | `Generate:277`, `Chat:297`, `Version:454`; capability and typed admission-error transport |
| `server/routes.go` | VERIFIED + DESIGNATED | `GenerateHandler:250`, `ChatHandler:2417`, `truncateNativeChatMessages:3010`; local/cloud/remote admission |
| `server/prompt.go` | VERIFIED + DESIGNATED | `chatPrompt:23`; rendered history fitting |
| `server/cloud_admission.go` | DESIGNATED | exact cloud qualification table and `utf8-bound-v1` |
| `server/cloud_proxy.go` | VERIFIED + DESIGNATED | `proxyCloudJSONRequest:158`, `proxyCloudJSONRequestWithPath:165` |
| `llm/server.go` | VERIFIED | `LlamaServer.ContextLength:77`; effective-context producer |
| `llm/llama_server.go` | VERIFIED + DESIGNATED | `completionPromptForRequest:276`, `ContextLength:330`; cut/refusal/keep boundary |
| `agent/session.go` | VERIFIED + DESIGNATED | `Session.chatRound:385`, `buildChatRequest:441`; agent turns/tool continuations |
| `agent/compactor.go` | VERIFIED | `SimpleCompactor.summarize:230`, `estimateCompactionRequestTokens:383`; summariser exclusion + estimate |
| `cmd/agent_tui.go` | VERIFIED + DESIGNATED | `GenerateAgentTUI:73`, `preloadAgentModelIfLocal:583` |
| `cmd/tui/chat/chat.go` | VERIFIED + DESIGNATED | `Run:216`, `resetChat:997`; terminal identity/lifecycle |
| `cmd/tui/chat/input.go` | VERIFIED + DESIGNATED | `submitInput:145`, `handleSystemCommand:424`, `systemPrompt:1719` |
| `cmd/tui/chat/compaction.go` | VERIFIED + DESIGNATED | `startManualCompaction:13` |
| `cmd/tui/chat/debug.go` | VERIFIED + DESIGNATED | `handleSaveCommand:28`, `handlePromptCommand:51`; request preview/export |
| `cmd/tui/chat/render.go` | VERIFIED + DESIGNATED | existing error-entry renderer; capability-refusal notice |
| `cmd/cmd.go` | VERIFIED + DESIGNATED | `loadOrUnloadModel:543`, `RunHandler:736`, `chat:1758`, `generate:1884` |
| `cmd/interactive.go` | VERIFIED + DESIGNATED | `generateInteractive:34`; classic `/clear`, `/load`, `/save`, `/set system` lifecycle |
| `cmd/instructions.go` | DESIGNATED | `ollama instructions set` |
| `app/store/store.go` | VERIFIED | `ChatWithOptions:428`; desktop transcript existence producer |
| `app/store/database.go` | VERIFIED | `getChatWithOptions:677`, `getSettings:1213`; desktop persistence inputs |
| `app/ui/instructions.go` | DESIGNATED | authenticated instruction edit/read/preview/reload handlers |
| `app/ui/ui.go` | VERIFIED + DESIGNATED | `Server.createChat:481`, `Server.chat:668`, `Server.buildChatRequest:1909` |
| `app/ui/app/src/api.ts` | VERIFIED + DESIGNATED | frontend instruction API client |
| `app/ui/app/src/components/Settings.tsx` | VERIFIED + DESIGNATED | `Settings:49`; editor mount |
| `app/ui/app/src/components/InstructionSettings.tsx`; `InstructionStatus.tsx` | DESIGNATED | edit/save/preview and selected-revision/refusal surfaces |
| `app/ui/app/src/components/Chat.tsx` | VERIFIED + DESIGNATED | `Chat:40`; status/reload surface |
| `app/ui/app/src/hooks/useChats.ts` | VERIFIED + DESIGNATED | stream/final-record admission diagnostics |

The closure contains every declared G1 source target plus every live production producer, consumer and write-path guard one hop from the carrier/selection/admission data. Nothing outside this set is credited as implementing G1. The Tier-2 judges re-derive the set and compare it to this table; an omitted sole producer or consumer is BLOCKING.

### 1.2 OQ-4 reachable chains and default-closed emitters

- **Desktop:** real Settings/API save → DESIGNATED repository `Save` → first accepted `Server.chat:668` binds/reads selection → extended `Server.buildChatRequest:1909` preserves model-system + MCP base → `Compose` → `api.Client.Chat:297` → `ChatHandler:2417`. The outgoing request and final admission report are the behaviour-changing observables.
- **Agent TUI:** `GenerateAgentTUI:73` binds → `chat.Run:216` → `systemPrompt:1719` + `Compose` → `Session.chatRound:385` / `buildChatRequest:441` → daemon. Tool continuations follow the same path.
- **Classic interactive `ollama run`:** `RunHandler:736` → `generateInteractive:34` → `chat:1758`; current `/set system` override or model system is the base, then `Compose`.
- **One-shot/piped `ollama run`:** `RunHandler:736` → `generate:1884`; one validated unbound snapshot (generation 0) composes into `GenerateRequest.System`; `Client.Generate:277` → `GenerateHandler:250`.
- **Compaction:** `startManualCompaction:13` supplies the effective prompt only to the estimator; `SimpleCompactor.summarize:230` keeps its fixed summariser prompt and carries no configured instruction carrier. The next real turn re-composes from selection.

Default-closed emitters are explicit: `Profile.enabled=false` opens only by authenticated save/CLI set; a conversation obtains a selection only at real creation or reload; the admission field exists only for an enabled selection after route-capability proof; `chat.admission.v1` and `generate.admission.v1` are advertised only by a daemon implementing the corresponding contract; enabled non-null identity bindings remain closed until the separately judged G2 validator exists.

## 2. Current behaviour, gap and compatibility

All source claims in this section are at the header revision on 21 September 2026. Exact raw file identities and mechanically checked anchor tokens are recorded in `docs/analysis/colleague_integration/evidence/2026-09-21_cont11/G1_SOURCE_ANCHORS.json` in the primary checkout; the accepted source map will be copied with the judgement packet, not guessed from filenames.

| Claim | Verified source anchor and token | G1 change / compatibility |
|---|---|---|
| Terminal builds a system prompt from model/harness/skill context | `cmd/agent_tui.go:112`, `systemPrompt := agentSystemPromptWithWorkingDir` | Append selected configuration through a shared composer; retain that base construction |
| Terminal's normal options literal does not supply ChatID or a storage sink | `cmd/agent_tui.go:114–177`, `agentchat.Run`; the complete literal was read | Allocate a new conversation ID; do not describe its in-memory message list as a disk archive |
| Run initialises its ID from Options.ChatID | `cmd/tui/chat/chat.go:227`, `chatID:` | New normal entry provides an ID; explicit test options remain possible |
| Reset clears transcript-related state, not the conversation ID | `cmd/tui/chat/chat.go:997–1022`, `func (m *chatModel) resetChat` | New-conversation transition obtains a new ID and selection before replacing state |
| Manual compaction consumes the effective system prompt | `cmd/tui/chat/compaction.go:45`, `SystemPrompt: m.systemPrompt` | Preserve the same pinned instruction selection outside the compacted message list |
| Agent requests insert a system message | `agent/session.go:441–467`, `func buildChatRequest` | Use the already-composed carrier; never append a second copy in the agent |
| Desktop settings use SQLite, not the terminal config file | `app/store/database.go:1213`, `func (db *database) getSettings`; `cmd/config/config.go:31`, `func configPath` | New instruction owner avoids full-settings write races or coupling platform-only store into terminal |
| Desktop request assembly is separate | `app/ui/ui.go:1909`, `func (s *Server) buildChatRequest`; call at 973 | Extend this actual builder, including model-system preservation, rather than only the terminal |
| A desktop draft/create-chat call returns an ID without saving a transcript | `app/ui/ui.go:481–493`, `func (s *Server) createChat` | Define binding timing explicitly; a reserved instruction identity is not a saved conversation |
| Desktop read errors now remain errors | `app/store/store.go:428–433`, `return s.db.getChatWithOptions` | Preserve the accepted history guard; never turn an instruction error into NewChat |
| Remote/local server routes prepend model.System only when the first request message is not system | `server/routes.go:2523`, `req.Messages[0].Role != "system"`; same at 2634 | Enabled desktop composition has to include the model-system text explicitly, because adding our system carrier otherwise suppresses it |
| Request API already carries Truncate and Shift | `api/types.go:163`, `Truncate *bool`; `api/types.go:167`, `Shift *bool` | Reuse these carriers; their mere presence is not sufficient for budget qualification |
| No-truncate rendering does not perform capacity admission | `server/prompt.go:35`, `if truncate`; return at 91 | Do not claim full token preflight from a no-truncate flag |
| Existing desktop Settings is the actual settings surface | `app/ui/app/src/components/Settings.tsx:49`, `export default function Settings`; API writes at `src/api.ts:526` | Add a dedicated instructions section/API rather than mixing stale full Settings updates with revision CAS |

Disabled compatibility means: no additional system text, no change to messages, tool schemas, model options, thinking, format or existing approval controls, and no additional model call. Small instruction metadata writes may still establish a disabled revision binding; that is explicitly not a byte-identical filesystem claim. Existing desktop conversations without a binding select revision zero, even if a new global configuration is enabled. They require explicit reload to adopt it. Desktop conversations bind on the first accepted `Server.chat` POST after acquiring the conversation lease and reading the real transcript status, before the first SetChat or model request. `createChat` only allocates the opaque ID and does not bind a revision. An empty UI draft is therefore not an active conversation; its preview is prospective. Terminal conversations bind on successful normal terminal creation or idle `/new`, before any prompt can be submitted. Neither a settings-editor mount nor an ID allocation is treated as a persisted chat message. Existing transcript content is never copied to the instruction store and is never truncated, reset or migrated by this feature.

The existing curated-context v10 document remains a separate unimplemented draft. Its empty-ID-to-`default` proposal (§4.1) is not adopted: it collapses distinct terminal conversations. Its full-history fallback (§7) does not become G1 permission to omit instructions or bypass budget failures. Its useful principles—separate working context from archived content and preserve tool pairs—remain future OI-27 obligations. G1 neither rewrites that document nor treats its old review as current readiness.

## 3. Shared owner, representation and storage design

DESIGNATED package `internal/instructions`, consumed by `cmd/agent_tui.go` and `app/ui/instructions.go`. Existing dependency `github.com/mattn/go-sqlite3` is used; no downloaded package, independent memory engine or model weight is introduced. The target platform qualification for this pack series is Ash's macOS arm64 environment; Windows/Linux compilation and execution status are reported separately, never inferred from macOS tests.

Default storage is `filepath.Join(os.UserHomeDir(), ".ollama", "instructions", "instructions.sqlite")`. Tests inject a root explicitly; process/restart tests set HOME before package initialisation. New directory mode 0700 and database mode 0600. Existing unrelated paths, symlinks and incompatible schemas are refused, not overwritten. This database holds only user-authored instruction revisions and bindings/events, not chat messages, retrieval memories, secrets or tool outputs. Read-only Current on an absent, genuinely uninitialised root returns revision zero disabled. A database missing from an already initialised root is an unavailable-store error, not an invitation to initialise over lost state. The initialisation algorithm is specified in §3.1 below; its process-crash and contention proof remains an OQ-4 verification obligation before promotion.

| NEW TYPE / field | Wire representation / allowed value | Producer → consumer |
|---|---|---|
| `Profile.schema_version` | integer 1 | repository decoder → schema admission |
| `Profile.revision` | decimal string, non-negative signed-64-bit range | committed revision sequence → CAS, bindings, diagnostics |
| `Profile.enabled` | boolean, default false | accepted operator edit → composer |
| `Profile.text` | exact valid UTF-8 string, 0–65536 bytes, NUL refused | editor → store → composer |
| `Profile.binding` | null or inactive `Binding` below | operator binding selection → active-binding rejection and status display |
| `Profile.content_sha256` | 64 lowercase hex characters | canonical profile encoder → decoder validation |
| `Binding.working_name` | non-empty UTF-8 string, maximum 128 bytes when supplied | selected identity → active-binding validator |
| `Binding.founding_reference` | non-empty opaque LINEAGE reference, maximum 1024 UTF-8 bytes | selected identity → active-binding validator |
| `Binding.lethe_scope` | non-empty opaque scope string, maximum 256 UTF-8 bytes | selected identity → active-binding validator |
| `Binding.map_key` | non-empty opaque map key, maximum 1024 UTF-8 bytes | selected identity → active-binding validator |
| `ConversationKey.namespace` | `desktop:<canonical-store-path SHA256>` or `terminal` | host adapter → binding lookup and lock key |
| `ConversationKey.id` | non-empty UTF-8 ID, maximum 128 bytes | existing desktop ID or new terminal UUID → binding lookup |
| `Selection.revision` | revision decimal string | creation/reload transaction → request composer |
| `Selection.generation` | decimal string, starts at 1 | reload transaction → stale-apply rejection |
| `Selection.profile` | complete independently owned decoded Profile from the pinned revision | revision lookup → request composition; never inferred from model replies |
| `ReloadEvent.kind` | `opened` or `reloaded` | binding transaction → next request orientation/diagnostics |
| `ReloadEvent.from_revision`, `to_revision`, `generation` | decimal strings | binding transition → event/recovery validation |
| `Error.code` | §8 closed string set | boundary validation → HTTP/TUI/UI errors |

G1 binding admission: `binding=null` supports ordinary user-authored instructions. A supplied binding is retained only in a disabled profile when all four bounded strings are non-empty, valid UTF-8 and NUL-free; this stores a proposed reference, not verified identity. Every attempt to enable a non-null binding returns `binding_unresolved`, including plausible-looking names and references. There is no always-success identity validator. The UI labels retained references `Inactive — continuity setup required`. G2 owns the later separately judged change that resolves real LINEAGE/LETHE records before enabling them. The G1 profile editor does not fabricate or preselect Eko.

Canonical digest input: a declared-order Go struct containing schema_version, revision, enabled, text and binding, marshalled by `encoding/json` with no trailing newline; digest field omitted. Binding uses its four fields in the listed order. UTF-8 and line endings are not normalised. Map iteration is not part of this representation. The read path recalculates the hash from actual fields before using text. Encoding `text="line one\nline two — café", enabled=true, revision="2"` and decoding recovers exact bytes; the expected hash is computed from independently supplied fixture bytes, not by calling the implementation's digest function in both arms. Three unbound-profile reference vectors are now fixed in continuation-12 profile-goldens/manifest.json: revision-zero is 76 bytes / SHA256 157826516a6fd8d118dc84113f5d5635604ba0fd871d98412816611627e43b36; the revision-2 Unicode example is 103 bytes / bf623f85cbc81809c618dbe0fa0767901ea22449804e2dc392079d22d27ac984; revision-3 CRLF/HTML/U+2028 escaping is 111 bytes / d11cedadb67f8a65b05ee10f15fe6d7342218530dd2b93e13ff7f4accfa4d104. Independently authored literal UTF-8 bytes and Python SHA-256 were checked against Go encoding/json and crypto/sha256; changed-text and line-ending controls also ran. This is reference-serialization proof only. Inactive non-null binding vectors and the actual production write/read/next-request consumer still require their own coverage; these fixtures do not discharge G1 runtime acceptance.

Repository API declarations, DESIGNATED in `internal/instructions/store.go`:
`Open(root string) (*Store,error)`; `(*Store).Current(context.Context) (Profile,error)`; `(*Store).Save(context.Context, expectedRevision string, next Edit) (Profile,error)`; `(*Store).Begin(context.Context, ConversationKey, CreationKind) (*Lease,error)`; `(*Store).Reload(context.Context, ConversationKey, expectedGeneration string, expectedTargetRevision string) (Selection,error)`; `(*Lease).Close() error`. `Edit` carries enabled, text and binding only; clients cannot author a revision or digest. `CreationKind` is `NewConversation` or `LegacyConversation`, supplied by the host's actual transcript lookup, not by an untrusted request field. `Lease.Selection` is immutable for the lifetime of the lease. Its Profile is a deep value copy, including a copied nullable Binding; it does not point to the mutable current-profile object. Historical revision rows themselves are immutable. Carrying the full pinned Profile is deliberate: text, enabled state, binding and digest participate in request validation, so removing the inactive binding would invalidate the canonical digest or conceal a selected-but-unresolved record. A concurrent global Save cannot mutate this copy. Maximum resident text per active conversation is 65536 bytes, not 1024 historical profiles; historical revisions stay in SQLite.

Tables: `metadata` has one schema/current-revision row; `revisions` has immutable sequence/enabled/text/binding/digest rows; `conversations` has the unique `(namespace,id)` selection; `events` has an ordered generation-keyed transition record. All foreign keys are enforced. Save and reload use immediate SQLite transactions and compare expected values inside the transaction. A stale editor does not win by arriving last. A no-change Save returns the same revision. Reset is a new disabled empty revision, not deletion; previous accepted revisions remain recoverable. Restoring an earlier value creates a new current revision referencing the chosen content; immutable history is never edited in place.

Declared capacity: 1024 non-zero revisions, 65536 text bytes per revision, 20000 conversation bindings, 256 reload transitions per conversation, plus the initial opened event (257 retained events maximum). Revision zero is the fixed disabled empty profile and does not consume the 1024 count. Exceeding any cap yields `capacity_exceeded` before transaction commit; no eviction, truncation or partial-success count. These are design choices, not measured existing workload. The consequences are visible and testing uses smaller injected capacities with identical transitions. At the per-conversation reload cap the old selection and full transcript remain usable; only another reload is refused. Starting a new conversation selects the current profile while preserving the capped conversation unchanged. At the global revision or binding cap the last accepted profiles and conversations remain readable; writes that need a new row fail with capacity_exceeded. These are declared hard operational limits, not a retention/eviction policy. Capacity expansion or a separately reviewed preservation/export operation is a named follow-on; no automatic garbage collection or silent rollover is licensed. Operational adequacy of the global caps remains an OQ-4 review obligation, not a hidden promise of unlimited history.

### 3.1 Atomic initialisation and exact persistence layout

DESIGNATED `Store.initialise` serialises cooperating creators with a process-safe parent lock at `<root-parent>/.instructions-initialise.lock`; this uses the same platform locking primitive as §4, not a PID/time-based stale-lock guess. The lock is held only for initialisation. Current on a genuinely absent root is still a non-writing disabled read. The first state-writing operation, including binding a disabled new conversation, initialises the store before taking the conversation lease.

Under the initialisation lock, recheck the final root. If it exists, validate its layout and database; missing members, unexpected layout or malformed schema return a store error, never a fresh empty store. If absent, create a private sibling staging directory on the same filesystem, create its 0600 database, and in one transaction create the tables below plus revision zero and metadata. Revision zero is `(revision=0, enabled=0, text="", binding_json=NULL)` with the computed canonical digest. Commit, checkpoint and close SQLite, create a 0600 `layout.json` containing exactly `{"format":"ollama-instructions","version":1}`, sync the files and staging directory, then publish by renaming the staging directory to the absent final root while retaining the creator lock. Sync the parent and only then report initialisation complete. A final root that appeared under a non-cooperating writer is refused rather than overwritten. Publication never replaces an existing root.

A crash before publication can leave an owned staging directory, but not a published half-database. It is not selected as current state and is not deleted automatically. A crash after publication has either a verifiable database or a visible store error, never fabricated disabled success. Directory durability is qualified on the target platform; unsupported sync semantics are reported rather than silently claimed. The initialisation lock file carries no history and is not deleted on release. Opening this new store does not migrate the existing chat database or terminal config.

The following is the new database schema, not an assertion that these tables already exist:

```sql
CREATE TABLE revisions (
 revision INTEGER PRIMARY KEY CHECK (revision >= 0),
 enabled INTEGER NOT NULL CHECK (enabled IN (0,1)),
 text TEXT NOT NULL,
 binding_json TEXT,
 content_sha256 TEXT NOT NULL CHECK (length(content_sha256)=64)
);
CREATE TABLE metadata (
 id INTEGER PRIMARY KEY CHECK (id=1),
 schema_version INTEGER NOT NULL CHECK (schema_version=1),
 current_revision INTEGER NOT NULL REFERENCES revisions(revision)
);
CREATE TABLE conversations (
 namespace TEXT NOT NULL,
 conversation_id TEXT NOT NULL,
 revision INTEGER NOT NULL REFERENCES revisions(revision),
 generation INTEGER NOT NULL CHECK (generation >= 1),
 PRIMARY KEY(namespace,conversation_id)
);
CREATE TABLE events (
 namespace TEXT NOT NULL,
 conversation_id TEXT NOT NULL,
 generation INTEGER NOT NULL CHECK (generation >= 1),
 kind TEXT NOT NULL CHECK (kind IN ('opened','reloaded')),
 from_revision INTEGER REFERENCES revisions(revision),
 to_revision INTEGER NOT NULL REFERENCES revisions(revision),
 PRIMARY KEY(namespace,conversation_id,generation),
 FOREIGN KEY(namespace,conversation_id) REFERENCES conversations(namespace,conversation_id)
);
```

Foreign keys are enabled on every connection. Write operations use an immediate transaction, a bounded busy timeout of 5000 milliseconds, and the caller's context cancellation; an exhausted wait is a visible store error. Durable writes use SQLite FULL synchronous semantics. Readers reconstruct and validate the Profile from actual fields and its referenced revision before releasing it to the caller. Stored numbers cross the API as canonical decimal strings, never JavaScript Numbers. There is no cascade-delete operation in this schema.

`Save` compares parsed expected revision to metadata.current_revision. It validates Edit and capacity before insert. If enabled/text/binding are identical, return the existing profile without changing the sequence. Otherwise allocate exactly current_revision+1 after checking overflow, insert the fully hashed immutable profile, update the current pointer and commit. Revisions are not removed or re-used. A failure at any stage rolls the transaction back.

`Begin` first takes the conversation OS lease. In a short transaction it reads an existing binding, or inserts a new binding at generation 1 using current_revision for NewConversation and zero for LegacyConversation; it inserts exactly one opened event with from_revision=NULL. The decoded Selection is returned only after successful commit. If validation/commit fails, release the lease and return an error. No global SQLite transaction remains open while the model or a tool is running.

`Reload` takes the same conversation lease, compares the observed generation and exact expected target revision inside its transaction, and validates the target Profile. If the selected revision already equals the target, it is an idempotent no-op with the same generation and no new event. Otherwise reject if 256 prior reload transitions exist; increment generation exactly once, update the binding, insert the reloaded event with the previous and target revisions, and commit before returning the new selection. Selecting revision zero or a disabled target is allowed; selecting an unresolved active binding is not. There are at most 257 events and generation at most 257 under the declared cap.

HTTP GET and successful PUT return a Profile directly (not a Settings envelope). Revision-list output is `{ "revisions": [ { "revision": "...", "enabled": false, "text_bytes": 0, "content_sha256": "...", "has_binding": false } ], "next_before_revision": <decimal-or-null> }`, sorted by revision descending; the cursor is exclusive. A single-revision GET returns the decoded full Profile. Reload returns Selection. Missing requested revision is invalid_input, not a fabricated empty profile. HTTP failures return `{ "error": { "code": "...", "message": "...", "conflict_field": <"generation"|"target_revision"|null>, "upstream_status": <integer-or-null> } }`. These diagnostics describe local failures; safe upstream details remain distinguishable from local status codes.

Boundary examples: initial current=0, Save A with expected=0 yields revision1; repeating the identical edit with expected=1 keeps1; changed Save B with expected=1 yields2; stale expected1 while current2 refuses. New X at revision1 has generation1/opened; reload to2 yields generation2; another reload to2 keeps2; 256 actual revision changes end at generation257 and the next change refuses without a write. Empty disabled revision0 is meaningful to Compose's baseline consumer; it is not evidence of enabled instructions.

## 4. Conversation lifecycle and concurrency

The instruction store owns selections; the original desktop store and terminal model own transcripts. First creation, failed first turn, existing conversation, `/new`, model switch, tool switch, compaction, explicit reload and process restart are distinct events.

A per-conversation process-safe advisory lease is held across a turn or compaction, including pending tools and approval waits. It is keyed by SHA256 of length-prefixed namespace and ID and represented by a lock file under the instruction root. Lock-file contents are not an authority record. OS lock release on process exit handles abandonment; no timestamp is interpreted as proof a worker died. Darwin/Linux and Windows lock implementations are DESIGNATED `lease_unix.go`/`lease_windows.go`; a failed nonblocking acquire returns `conversation_busy` without replacing a running operation. Profile Save uses only a short SQLite transaction and can complete while an existing conversation holds its selected revision.

| Event | Preconditions | Transition and observable |
|---|---|---|
| New conversation | Host-issued non-empty ID; lease acquired | Transaction binds the current validated revision and generation 1; `opened` event; no inference on failure |
| Legacy desktop conversation | Real saved chat exists and no instruction binding exists | Bind revision 0; never silently adopt current text |
| Continue existing conversation | Lease acquired; binding and revision decode successfully | Read pinned selection; current global revision is irrelevant |
| Save B while A is active | expected current revision matches | Current pointer becomes B; A's selection and outgoing carrier remain A |
| Reload | Conversation lease acquired; expected generation and current revision match | One transaction updates selection, increments generation and records `reloaded`; next request reads the new carrier and its explicit revision notice |
| Reload while running/compacting | Lease unavailable | `conversation_busy`; old selection and transcript unchanged |
| Model/tools change | Existing selection valid | Rebuild only the existing model/skill/tool base; reuse pinned profile and re-evaluate capacity for the actual new request |
| Compaction | Same conversation selection and lease | Reattach carrier from selection, not from summarised text; do not store duplicate system messages |
| Terminal `/new` | Idle; new selection can be created | Generate UUID and bind before clearing old terminal state; failure leaves old state usable |
| Restart | Store readable | New terminal invocation creates a new conversation using current profile; existing desktop ID reads its old selection |

A binding reserved before an initial transcript write may survive a failed first turn. This is intentional: it is an instruction-selection record, not a claim that a conversation message was saved. A retry under the same issued ID uses that reserved revision; starting a genuinely new ID selects the then-current profile. No cross-database atomicity is claimed. Missing or corrupt data after a binding exists blocks the affected request; it does not recreate the chat or select another colleague.

The existing terminal does not provide a normal disk-backed transcript-resume path in the inspected entry. G1 does not invent one or claim terminal transcript restoration. OI-07/OI-08 own that later continuity boundary. Distinct IDs, saved defaults and in-session selection survive the transitions G1 actually supports; reopening desktop transcripts is tested through their existing persistent owner.

## 5. Actual request composition

DESIGNATED `internal/instructions/compose.go::Compose(base string, selection Selection) (string,error)` is a pure function. Disabled selection returns the supplied base byte-for-byte. Enabled selection returns base plus a two-newline separator when base is non-empty, then the literal header `User-configured instructions; revision <revision>; selection <generation>:\n`, then the exact profile text. Profile hash/binding validity is checked before composition. This user-authored text does not alter executor approvals, tool availability or filesystem authority. Recalled prose and arbitrary repository files are not promoted into this carrier by G1.

Terminal: `GenerateAgentTUI` creates/loads the selection through the shared owner and passes it in a new `chat.Options.InstructionSelection` field, never folded into `Options.SystemPrompt`; `chatModel.systemPrompt` first performs its existing built-in-toggle/extra construction, then calls `Compose` with that base and the separate immutable selection. Consequently `/system off` removes only the built-in base, not enabled custom instructions; `startRunWithMessages` and manual compaction use that one effective result; `agent.buildChatRequest` receives the already-composed string. `/system on|off` keeps its existing built-in-prompt meaning; it is not silently redefined as the custom-instructions toggle.

Desktop: the real `Server.chat` derives new-versus-existing from its existing Store lookup, acquires the lease, resolves the selection before saving or invoking inference, then passes the selection and the actual model-system text from `Client.Show` to its extended existing request builder. The DESIGNATED new signature is `buildChatRequest(chat *store.Chat, model string, think any, availableTools api.Tools, mcpInstructions string, selection instructions.Selection, modelSystem string) (*api.ChatRequest, error)`. `Server.chat` supplies `show.System` from its already obtained ShowResponse, rather than a new unproved model-system lookup. Before implementation, all callers/tests of this signature are enumerated and updated; the old source signature is not claimed to already accept these fields. When enabled, model-system text and existing MCP instructions form the base in that order, omitting only empty blocks; the shared composer appends the configured text. The builder adds one effective system message before the conversation. Existing role/tool/attachment conversion remains in that builder. When disabled, its old path remains exactly as before, including the current MCP instruction ordering. Adding a system message without preserving the model's system text fails G1-R-06.

Contract use in both adapters: every request that carries an enabled carrier also carries the §6.3 A1 admission field, and only after the adapter has confirmed the daemon's `chat.admission.v1` capability (A12). That includes every tool continuation: the desktop's pass loop, which begins at `app/ui/ui.go:943` and sends each request at `:978`, and the agent's follow-up requests. A disabled selection sends no admission field, so its request is unchanged (G1-R-15). Both adapters read the final record's `admission` object on every route. The desktop must read it before its early return for records with no content, thinking or tool calls (`app/ui/ui.go:992-994`). Both classify a refusal by its `code` before any message-text rule, because the desktop's rules match text such as "402", which a count can contain (`app/ui/ui.go:598-606`). A refusal under A6, A10 or A13, or a missing capability, is mapped by §8 and shown to the operator. The adapter never retries without the carrier or the field.

Preview is a point-in-time result for the explicit `(model, revision, selection_generation, base_digest)` returned alongside it. It returns the exact effective carrier for that tuple through the same construction helpers. The view labels a prospective new-conversation preview as unbound. Changing any tuple member, including a reload generation, makes that preview stale; it is never represented as proof of a later dispatch. It is labelled a preview, not proof of dispatch, model compliance, memory recovery or available context. Actual-request diagnostics separately record revision, generation, source digest, carrier byte count, model and request outcome without storing private instruction text in public logs or repository evidence.

### 5.1 OQ-4 adapter and composition decisions

These clauses incorporate `ADR_2026-09-27_G1_OQ4_OPERATOR_CONTRACT.md`, including Ash's D-11 ruling.

1. **Capability custody.** Every carrier-bearing request checks the relevant route capability immediately before dispatch. Agent TUI, desktop and classic chat require `chat.admission.v1`; one-shot/piped generation requires `generate.admission.v1`. A missing capability refuses that turn/invocation and sends no carrier-bearing inference request. A successful response missing its final admission report is also refused/unverified; no adapter retries by stripping the carrier.
2. **Agent-TUI refusal.** A capability-less daemon does not close the TUI. The chat shows one open-time notice and, for each submitted turn, one attributable error entry containing `context_unavailable` and the exact daemon version. The turn sends no inference request and the chat returns idle so `/instructions off` remains usable. The open-time notice cannot substitute for the turn report.
3. **Desktop refusal.** `Server.chat` preserves today's submitted-message save, sends no inference request, and streams the existing error event with code `context_unavailable` plus daemon version. The machine code is classified before text heuristics.
4. **Refusal evidence.** Refusal-report correctness is a separate mandatory verdict beside the OQ-3 failure number. Sending *any* inference request under a turn that should have been refused is an incorrect delivery, even if the carrier was stripped first.
5. **Desktop base.** When enabled, base = model-system text then MCP instructions, omitting a block only when `strings.TrimSpace(block)==""`, joining retained blocks with exactly two newlines, otherwise preserving their bytes. `Compose` appends the configured carrier. Disabled mode retains today's builder path.
6. **Compaction.** The summariser request itself carries no configured carrier. The conversation's effective prompt participates in token estimation only; the next real turn re-composes the pinned selection.
7. **Runtime identity.** I-05 proves the terminal runtime `ChatID` equals the committed `ConversationKey`; two conversations at one revision have distinct non-empty IDs. A mutant that updates only the binding but leaves runtime identity unchanged must fail.
8. **`/prompt` and `/save`.** In the agent TUI these remain faithful previews of what dispatch would send: enabled selection shows/writes the composed carrier and admission member. Enabled `/save` writes mode 0600 and emits an external warning that saved instructions are present. Disabled/unconfigured behaviour is unchanged.
9. **Classic interactive `ollama run`.** Session open binds a new terminal conversation. The latest explicit `/set system` value, when supplied, is the base; otherwise model-system text is the base. With saved instructions enabled, superseded `/set system` values are not retained as separate mandatory system-history messages. `/clear` and `/load` each start and bind a new conversation before replacing old state; `/load` preserves the explicit base override. It gains `/instructions`, reload and off commands. A refused turn stays visible to the operator but is not inserted into the history of the next request.
10. **One-shot/piped `ollama run`.** One invocation takes one validated unbound snapshot of the current Profile with selection generation 0; it writes no binding, lease or `opened` event. `GenerateRequest.System = Compose(modelSystem, snapshot)`; `Prompt` remains the user's prompt. Standard output stays the model answer; admission notices/refusals are standard error. Missing final report exits non-zero and labels any streamed answer unverified. Cloud one-shot carrier delivery remains refused until a separate generation profile is qualified with Ash's authorisation.
11. **Operator-visible `/instructions`.** Status names scope, selected revision, selection generation, enabled state and saved-current revision separately. Reload/off name the resulting selection or failure code. No success wording follows a failed operation.

## 6. Context capacity and the admission contract

Decided on 26 September 2026 in `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md` (commit `756e65503da4b200d5cf9a97c74c268bd7907be4`), after a blind options check on two non-Claude substrates whose claims are dispositioned in `docs/analysis/colleague_integration/proof_cont13/oq1-options-check/`. It was amended the same day for explicit cloud models (A13; the ADR's amendment 1) on Ash's ruling, after a second blind check on Codex, dispositioned in `proof_cont13/cloud-admission-check/`, and the bounded cloud check in `proof_cont13/cloud-qualification/`. This section is the contract that G1-R-07 incorporates by reference; G1-R-06, G1-R-14 and G1-R-15 bind the clauses marked. Source anchors below are at `04009c432e313eef65172293e75feb038ebe476b`, verified 26 September 2026; production and test sources are identical to the header revision.

### 6.1 Measured behaviour of the unchanged candidate

Measured on the operator's Mac with the candidate daemon, the pinned native llama-server b10091 (`b4d6c7d8`), qwen3:8b (Q4_K_M), `num_ctx` 512, one slot, `think` false, non-streaming, text only; every probe used a fresh daemon in an isolated sandbox. Results, logs and harnesses: `proof_cont13/oq1-measurements/`.

| Condition | Observed on both routes unless stated | Probe |
|---|---|---|
| Default settings, reply longer than the context | Each shift logs `n_keep = 4, n_left = 507, n_discard = 253`; the carrier (tokens 16–66) is discarded; HTTP 200; after a packed history the reply degenerates | run1 `shift_default_long`; run3b `longreply_default` |
| Default settings, oversized prompt, rendered route | HTTP 200 after `truncating input prompt limit=258 prompt=4289 keep=4`; carrier lost | run1 rendered `oversize_default` |
| Oversized prompt, native route | HTTP 400 carrying llama-server's `exceed_context_size_error` JSON, with `n_prompt_tokens` and `n_ctx`, nested inside Ollama's `error` string | run1 native `oversize_*` |
| Shift off, short conversation | Stops at the context limit, `done_reason: "length"` at 406 / 402 tokens (512 minus the prompt) | run1 `shift_false_long` |
| Shift off, history packed to 494 / 498 tokens | Reply cut mid-sentence after 18 / 14 tokens | run3b `longreply_shift_false` |
| Shift off, oversized prompt, rendered route | HTTP 400 "the prompt is longer than the context length currently available to the model…" | run1 rendered `oversize_shift_false` |
| Changing shift for one model | A new runner launch on every change; load 0.91–1.00 s with the model file in page cache | run2 `reload_sequence` |
| Keep the whole prompt (`num_keep` −1) | Shifts keep the full prompt (106 / 110; 494 / 498 tokens) and discard generated text only; the reply finishes and follows the carrier | run2 `keep_all_long`; run3b `longreply_keep_all` |
| Keep the whole prompt, prompt at 510 of 512 | Keep clamped to 508; 118 one-token shifts consume the prompt's tail; the tail instruction is lost; output degenerates | run3b `clampedge_keep_all` |
| Keep the whole prompt, oversized prompt, rendered route | HTTP 200 after `limit=511 prompt=4289 keep=511 new=511`: the prompt's end is dropped | run2 rendered `keep_all_oversize` |
| Runner capacity | Slot `n_ctx` read from the runner's `/props` equals the requested 512 in every run | all runs |

Mechanisms, verified at source: the rendered-route cut at `llm/llama_server.go:299-315` (`nKeep := req.Options.NumKeep`), refused instead at `:292-297` when the runner has no context shift; the per-request keep sent at `:2115` (`"n_keep": req.Options.NumKeep`), default 4 at `api/types.go:1102`; the native shift at `server-context.cpp:2868-2885`, whose keep is clamped by `n_keep = std::min(slot.n_ctx - 4, n_keep);` at `:2874`; the native stop at the limit when shift is off at `:1858-1863`; the native admission check at `:3134-3142`; the reload on a shift mismatch at `server/sched.go:1417` (`if runner.contextShift != contextShift {`). Neither client reads `done_reason` today: no reference in `app/ui/*.go`, the desktop frontend source, `cmd/tui` or `agent`; `app/ui/app/src/hooks/useChats.ts:598` tests only `event.done`.

### 6.2 Superseded directions

Edition 1 of this section (commit `04009c432e313eef65172293e75feb038ebe476b`, file SHA-256 `3d2f2db8ff514119f63f302b939455f6ed81b9257c5082e29f1cd9534e2b9241`, §6.1–§6.2) rejected an auxiliary one-token inference and proposed extending the native slot guard to reserve output. That extension is not selected: a daemon-side reserve keeps prompts out of the clamp zone while the native guard stays the exact backstop, so the fork carries no llama.cpp patch. The ADR names the condition that reopens it: the measured estimate gap (6.5, V1) approaching the reserve floor. Edition 1's continuation-12 findings — the count-only endpoint, the tokenizer flag mismatch and time-dependent templates — stand in that edition and in `evidence/2026-09-21_cont12/`.

### 6.3 The admission contract, version 1

Clauses A1–A13 are the contract. A request that carries an enabled carrier is an *admitted request* when it carries A1's field. Every refusal the contract defines has the body `{"error": <message>, "code": <code>, "admission": {<the clause's fields>}}`. DESIGNATED `api/client.go` error decoding (`:221-237`, which today keeps only `error` and `signin_url`) keeps `code` and `admission` in `api.StatusError`, so both adapters classify refusals by code, never by message text (G1-R-14).

- **A1 — wire field.** DESIGNATED `api/types.go` type `RequestAdmission` with `Version int \`json:"version"\`` and `Reserve int \`json:"reserve,omitempty"\``. It is added to both `ChatRequest` (`api/types.go:133`) and `GenerateRequest` (`api/types.go:62`) as `Admission *RequestAdmission \`json:"admission,omitempty"\``. Version is 1. Reserve is output room in tokens; zero selects the route default. A request without the field is processed exactly as today (G1-R-15). The shared wire type does not make the two route capabilities interchangeable.
- **A2 — validation.** The daemon refuses with HTTP 400 and code `invalid_admission` a version other than 1, a negative reserve, or a reserve above half the effective local context. An unknown version is refused, never ignored. `/api/chat` accepts the field only under `chat.admission.v1`. `/api/generate` accepts it only under `generate.admission.v1` and C-4's chat-like constraints: raw mode, a non-empty request template override, suffix mode or legacy `context` with admission are refused before inference. OpenAI- and Anthropic-compatible inference routes refuse any top-level `admission` member before explicit-cloud passthrough. Prompt-less load/unload requests carry no admission field.
- **A3 — reserve.** The effective context C is the runner's context length after the training-context cap (`llm/server.go:77` interface; live llama-server `ContextLength` at `llm/llama_server.go:330`), equal to the native slot size in every measured run. Enabled local carrier admission requires C ≥ 32; smaller contexts are refused `context_unavailable` because the floor below cannot also satisfy A2's at-most-half-context invariant. Otherwise default reserve R is C/8 clamped to 16–4,096 tokens; a supplied reserve replaces it within A2's bounds. Worked extremes: C=32 gives R=16 and C−R=16; C=32,768 gives R=4,096; above that the upper clamp remains 4,096. V1 measured no estimate/evaluated-token gap in 24 cases across three models and both routes; templates outside that measurement remain covered by margin, not by evidence.
- **A4 — keep.** The daemon sends a keep of −1 to the runner for every admitted request on both llama-server routes (the chat body's `n_keep`, the completion's `NKeep` at `llm/llama_server.go:1526`), so any context shift keeps the whole prompt.
- **A5 — history fit.** Both history loops (`server/prompt.go` `chatPrompt`; `server/routes.go` `truncateNativeChatMessages`) fit the estimated prompt into C − R instead of C, keep every system message, and drop whole turns from the oldest. A turn begins at a user message. The retained history begins at a user message or is exactly the newest turn, which is every message from the last user message to the end, including assistant tool calls and every tool result after them. No retained history begins with a tool result whose call was dropped (G1-R-06).
- **A6 — refusal.** If the system messages and the newest turn exceed C − R by the daemon's estimate, the daemon refuses before inference with HTTP 400, code `context_exceeded`, and the fields `context_length`, `reserve` and `estimated_prompt_tokens`.
- **A7 — no cut.** For an admitted request the Go runner never takes the cut at `llm/llama_server.go:299-315`; it returns A6's refusal. The native admission check stays the exact backstop and its error is mapped to `context_exceeded` with the native counts (§8).
- **A8 — continuation.** The structured-output continuation (`server/routes.go:2882-2903`) re-renders under A5–A7. The thinking it appends belongs to the newest turn; if the turn no longer fits, the stream ends with A6's error.
- **A9 — diagnostics.** The final response of an admitted request carries `admission` with `version`, `route` (`"local"`), `context_length`, `reserve`, `dropped_messages` and `context_exhausted`. `context_exhausted` is true when generation stopped because the context filled — possible only under A11 — and distinguishes that from an output-budget stop, which shares `done_reason: "length"`. DESIGNATED `api/types.go` adds the object to `ChatResponse` (`:520-549`) so the Go client decodes it. A13(f) defines the cloud form.
- **A10 — routes refused.** The daemon refuses an admitted request with HTTP 400 and code `context_unavailable`, naming the route and the reason. This applies on remote-host stubs (`server/routes.go:2495-2585`), on MLX runners (`x/mlxrunner`), and on every explicit cloud request (`:2440-2448`) that A13 does not admit. Lifting a route needs its own measured qualification. For cloud routes that qualification sends traffic to the provider and is Ash's to authorise, as he did for A13.
- **A11 — runners that cannot shift.** Multimodal runners (`server-context.cpp:1215-1217`) and model families launched without shift (`server/sched.go:137-147`) keep the prompt because they never shift. Generation stops when the context fills and A9 reports it.
- **A12 — capabilities and report custody.** `GET` and `HEAD /api/version` advertise `chat.admission.v1` only when `/api/chat` implements its full contract and `generate.admission.v1` only when `/api/generate` implements C-4. Every G1 adapter reads the route capability and daemon version through DESIGNATED `api/client.go` immediately before **every** carrier-bearing request; HTTP has no durable "same connection" semantic strong enough to protect against a daemon replacement. When the required capability is absent, the adapter sends no carrier-bearing inference request and reports `context_unavailable` naming the daemon version. A successful carrier-bearing response whose final record lacks the route's admission report is also `context_unavailable`: the answer is labelled unverified, never silently accepted. The adapter never retries by stripping the carrier or field.

- **A13 — explicit cloud models, up to a verified prompt limit.** Ash ruled on 26 September 2026 that carrier-bearing requests should work on the qualified cloud models up to the size verified for each, and that larger requests are refused clearly (DECISIONS.md `ash-cloud-admit-verified-size-2026-09-26`). On the explicit cloud route, after A1–A2, the daemon applies (a)–(g).
  - (a) *Qualification.* The daemon admits only a request that matches an entry of DESIGNATED `server/cloud_admission.go`'s table. Entries are keyed by provider origin (the host of `cloudProxyBaseURL`, `server/cloud_proxy.go:39`), exact forwarded model name (`modelRef.Base`) and request profile. Each entry records the verified prompt limit V, which is the largest passing probe's provider count, together with counting method `utf8-bound-v1`, the date and the evidence (§6.4). Any other model, origin or profile is refused under A10 with reason `not_qualified`. The table changes only through a new recorded check. It is dated evidence, not continuous certification. A model the provider retires (HTTP 410) leaves the table at the next change; until then its errors pass through.
  - (b) *Profile.* Qualified requests may carry messages of any role with text, thinking, tool calls and tool results, and tool definitions. `think` may be true, false or omitted, and the request may stream or not. Images, `format`, the options `num_ctx` and `num_keep`, and the fields `truncate` and `shift` are refused under A10 when supplied, naming the feature. A non-zero `reserve` is refused under A2, because the daemon cannot reserve room at the provider.
  - (c) *Prompt bound.* B starts at 16. For each message it adds 16, plus, for each of role, content, thinking, tool name and tool-call id, the larger of the field's UTF-8 length and its NFKC-normalised UTF-8 length. A tool result's content counts at its ASCII-escaped JSON string length when that is larger. Each tool call adds 16 plus the ASCII-escaped compact JSON length of the call. When tool definitions are present, B adds their ASCII-escaped compact JSON length plus a tool allowance of 304. The premise is that the provider's tokenizer emits at least one byte of rendered text per ordinary token, and that the allowances cover template text. Every one of the 23 counted qualification probes had a provider count at or below its bound; runs 1 and 2 were checked under a simpler bound, which this one never undercuts. B is reported as a bound, never as a count. On the probes of 11,000 tokens and more it ran 2.1 to 2.8 times the provider's count; the fixed allowances make the ratio larger on short requests.
  - (d) *Refusal above the limit.* If B exceeds V, the daemon refuses before forwarding, with HTTP 400, code `context_exceeded`, `route` `"cloud"`, `limit` V, `limit_kind` `"verified_prompt"` and `prompt_bound` B. It drops no message. This route drops no history today, and fitting history to a bound two to three times the real count would discard history the model could use (G1-R-06).
  - (e) *Forwarding.* The daemon removes the admission field and forwards the request, otherwise unchanged, through the existing proxy (`server/cloud_proxy.go:165-173`).
  - (f) *Report.* On a 2xx response the daemon inserts one `admission` member into the final JSON object, without re-encoding the rest of it. When streaming, that is the record with `done` true; otherwise it is the only object. The member holds `version`, `route` `"cloud"`, `limit`, `limit_kind`, `prompt_bound`, `counting` `"utf8-bound-v1"` and `dropped_messages` 0. Every other record, and every error, passes through byte for byte. The daemon reports no `context_length`, `reserve` or `context_exhausted`, because it cannot observe the provider's context or retention.
  - (g) *Generation.* Retention during generation is not observed on this route, and no guarantee covers it. On 26 September 2026 the models advertised a context of 1,048,576 tokens, 944,905 more than the largest limit. That figure is provider metadata, served through a cache that can be stale (`server/model_show_cache.go:177-186`). The daemon imposes no output limit. A `length` stop is labelled as reaching a length limit, without saying which limit.

### 6.4 Supported-configuration matrix

| Route | Carrier protection | Overflow outcome | Reply room | G1 status | Evidence |
|---|---|---|---|---|---|
| llama-server native chat, local GGUF, text | A4, A5 | A6 before inference; native check backstop | at least R; shifting beyond it keeps the prompt | supported, measured on qwen3:8b | run1–run3b native |
| llama-server rendered (Go template, renderer or parser) | A4, A5, A7 | A6 / A7 | as above | supported, measured on qwen3:8b | run1–run3b rendered |
| llama-server with media | never shifts (A11) | native check; A6 with the existing 768-token image estimate | stops when full; A9 reports it | supported by construction, not measured (V5) | source |
| Model families launched without shift | never shifts (A11) | as the native route | stops when full; A9 | supported by construction, not measured | source |
| MLX | — | — | — | refused (A10); no MLX model is available here to qualify it | `x/mlxrunner/pipeline.go:47-57` |
| Explicit cloud models in A13's table | A13(a)–(d): within V by the prompt bound; the provider holds the prompt | A13(d): refused above V by the prompt bound; nothing dropped | the provider's; not observed (A13(g)) | supported up to V (table below), on Ash's ruling of 26 September 2026 | `proof_cont13/cloud-qualification/` |
| Other explicit cloud models, and cloud requests outside A13's profile | — | — | — | refused (A10: `not_qualified`, or the feature named) | — |
| Remote-host stubs | — | — | — | refused (A10) until qualified | source |
| Any daemon without the capability | — | — | — | refused by the client (A12) | — |

A13's table as qualified on 26 September 2026, through Ash's installed daemon and within his authorisation. The check made 23 counted synthetic requests to the live models plus one refused request, and used 453,965 prompt tokens.

| Forwarded model | V (tokens) | Size evidence | Client-profile evidence | Text admitted in practice |
|---|---|---|---|---|
| `deepseek-v4.1-flash` | 99,680 | plain text, run 1 Q3 | run 3 T1 at 11,001 tokens | about 99 KB, less any tool definitions |
| `glm-5.2` | 103,671 | plain text, run 1 Q3 | run 3 T1 at 13,266 tokens | about 103 KB, less any tool definitions |
| `glm-5.3` | 25,722 | run 3 T1, in the client profile | the same probe | about 25 KB, less any tool definitions |

Every probe returned the carrier's random code and the other expected values exactly; run 3 also checked a middle value. Run 3 used the client profile: streaming, `think` true, false and omitted, tool definitions, and a tool call with its result. Between each model's T1 size and V, the client profile is inferred from the two measurements, not measured. Middle delivery above run 3's sizes is unmeasured. The bound ran 2.1 to 2.8 times the provider's count on these number-heavy probes, and would run higher on ordinary prose. A request is therefore refused while its real count is still well below V: at roughly half of V for text like the probes, and nearer a quarter for English prose, at about four bytes a token. `deepseek-v4-flash` returned HTTP 410 (retired 25 September 2026) and is not listed.

### 6.5 Verification obligations carried into the packs

- **V1 — measured, 26 September 2026.** Three local models (qwen3:8b on both routes, gpt-oss:20b through the harmony renderer, gemma4:26b through its renderer) and six prompt shapes (plain, multi-turn, tools, tool calls, Unicode, thinking): in all 24 cases the daemon's current estimate — the runner's `/tokenize` with Go's flags, whose runner defaults are `add_special` false and `parse_special` true (`server-context.cpp:4875-4876`) — equalled the tokens actually evaluated (`prompt_eval_count`, cache plus new). This corrects edition 1's reading that the flags differ in a way that undercounts. Not covered: media (V5) and native templates that insert the date, which could differ by a few tokens across midnight. Evidence: `proof_cont13/v1-estimate-gap/`.
- **V2** — streaming requests on both local routes under the contract; on cloud, amendment 2 checked streaming at T1's sizes.
- **V3** — tool conversations: newest-turn retention with its whole tool group, and whole-turn dropping.
- **V4** — parallel slots: C against the native per-sequence size.
- **V5** — media: the 768-token estimate against actual projector tokens, and A6's behaviour.
- **V6** — the structured-output continuation (A8).
- **V7** — real-model qualification of carrier delivery (I-10) on each adapter's default local model.
- **V8** — A13's table: a model retired or changed by its provider leaves the table, and enters or returns only through a new recorded check. The reality gate's cloud arm (§9) spends provider tokens and needs Ash's authorisation.

## 7. Editor and terminal interaction

DESIGNATED `app/ui/instructions.go` exposes authenticated `GET /api/v1/instructions`, `PUT /api/v1/instructions`, `GET /api/v1/instructions/revisions`, `GET /api/v1/instructions/revisions/{revision}`, `POST /api/v1/instructions/preview`, and `POST /api/v1/chat/{id}/instructions/reload`. Revision listing returns at most 20 metadata-only entries before an optional decimal `before_revision` cursor; one revision GET returns its full accepted Profile. Restoring from a historical revision sends its content through the same new-revision CAS Save, never mutates history. The PUT envelope is exactly `{ "expected_revision": "<decimal>", "edit": { "enabled": <boolean>, "text": "<UTF-8>", "binding": <null-or-binding> } }`; profile digest/schema/revision are server-authored output and are not accepted input fields. Revision strings have no sign or leading zero except the single digit `0`, and are at most 19 decimal digits with signed-64-bit overflow rejected. GET returns current configuration and revision. PUT requires expected_revision plus Edit; it returns the committed profile only after durable success. Preview takes model and selected scope/revision and returns the shared composer's output plus truthful capacity status. Reload requires expected_generation and expected_target_revision. The first is the conversation generation the operator observed; the second is the exact global revision displayed for adoption. Both are compared in one transaction. A changed target causes `revision_conflict` with `conflict_field="target_revision"`; a changed conversation causes the same code with `conflict_field="generation"`. There is no silent adopt-latest operation: refresh the preview and let the operator choose the newly displayed target. The editor preserves unsaved work on either conflict. Unknown JSON members, invalid UTF-8/NUL, wrong JSON types, more than one top-level JSON value and oversized bodies are refused; the request limit is 524288 bytes. Worst-case JSON escaping costs at most six ASCII bytes per input UTF-8 byte for these bounded strings: text 65536 plus binding 2432 gives 407808 bytes, The independently encoded worst-case significant PUT body is 407970 bytes: 407808 escaped string bytes plus 162 envelope bytes, leaving 116318 bytes below the raw-body cap. This uses U+0001 (valid UTF-8, NUL-free and six-byte escaped), all four binding fields at their maxima, a 19-digit expected revision and enabled=false. Extra whitespace still counts towards the raw-body limit; a padded oversized request is rejected even when its decoded fields would fit. This is a body-allocation bound, not a token-budget measurement. Exact one-object parsing and all decoded field caps still apply; the body cap is not the decoded-text limit. Authentication precedes parsing and state access. Typed errors carry machine codes and human-readable text; no false Saved toast follows a non-2xx reply.

DESIGNATED `InstructionSettings.tsx` is mounted in the real Settings component adjacent to SpeechSettings. It displays scope `Desktop and terminal — new conversations`, current revision, enabled state, editable text, byte limit, explicit Save, preview, Disable and Reset. Save is not an every-keystroke write. Dirty edits are kept on a stale-revision conflict; no automatic last-writer-wins retry. Reset makes a new disabled empty revision with a route to select a previous retained value. The existing general Settings reset does not destroy instruction history. The draft editor never activates an unresolved identity.

Existing conversations display their selected revision and a button `Apply saved instructions` at the conversation header. The button is disabled while the UI knows the conversation is active, and the backend lease remains authoritative. A successful reload displays the new generation/revision; an error leaves the existing selection visible. The editor does not pretend that saving automatically updates open conversations.

When an admitted reply ends with A9's `context_exhausted`, or a request is refused under A6, A10 or A13, the conversation view and the terminal show a one-line notice naming the cause and its counts. A reply that ended at its output budget is labelled as that, not as a full context. An A13 refusal names the model and its verified limit, and offers a new conversation. Once an admitted cloud response reports a prompt bound at or above 80% of its limit, the conversation shows the room left, so the limit is visible before it is reached. Preview reports capacity from A3's reserve, or from A13's limit on a cloud model, and from the route's §6.4 status, so a refused route is visible before a conversation starts.

Terminal DESIGNATED `/instructions` displays scope, selected revision, selection generation, selected enabled state and the saved-current revision as separate values. `/instructions reload` applies the saved revision only at idle and names the resulting selection, or says it is unchanged when already selected; `/instructions off` commits and selects a disabled revision and names it. Every failure is an error entry naming the operation and its §8 code, with no success wording. Saved editing is provided by the desktop surface and DESIGNATED `ollama instructions set --file <path> --expected-revision <revision>`; `set` reads the file strictly as data and saves its text as an enabled profile. In the agent TUI, a capability-less daemon produces the open-time notice plus an attributable per-turn `context_unavailable` entry containing the exact daemon version; the chat remains idle and open. Classic interactive `ollama run` exposes the same status/reload/off semantics as plain output lines, while one-shot/piped runs expose refusals/notices only on stderr. Agent-TUI `/prompt` and enabled `/save` include the carrier and admission field the real dispatch would carry; enabled `/save` is mode 0600 and warns outside the file. Classic `/save` never includes the carrier.

## 8. Failure representation, guards and cannot-represent scope

Closed error codes: `invalid_input`, `revision_conflict`, `conversation_busy`, `store_unavailable`, `corrupt_record`, `unsupported_schema`, `binding_unresolved`, `capacity_exceeded`, `context_unavailable`, `context_exceeded`. The following is the local instructions API mapping, not a claim about an upstream provider's status codes. When a provider error is translated, the diagnostic retains its upstream status and a safe reason without relabelling the upstream response itself. HTTP mapping: invalid_input 400; conflict/busy 409; binding_unresolved 422; capacity_exceeded 507; context unavailable/exceeded 422; store/corruption/schema 503. Errors stop the affected change/dispatch and preserve the old record. They do not require another operator approval and do not terminate independent conversations.

Daemon errors map as follows. `context_exceeded` comes from A6/A13(d), the native `exceed_context_size_error` (with its nested counts), the rendered runner's prompt-too-long 400, and equivalent qualified-backend overflow. `context_unavailable` comes from A10, a missing required `chat.admission.v1` or `generate.admission.v1` capability, C<32 local context, or a successful carrier-bearing response whose final record lacks the required admission report. `invalid_admission` remains the daemon's wire-contract rejection and is mapped to a typed local failure without erasing the original code. On cloud, provider 401/402/410/429/5xx errors pass through with their original status rather than being relabelled as admission refusals. An unrecognised daemon/provider error stays an error. No adapter may turn any of these failures into success or retry by silently removing the carrier.

The table below is the complete **G1-specific** write-path guard set across the frozen reach closure. Generic pre-existing model-existence, model-capability, thinking/format and provider-auth checks remain authoritative but do not transform, truncate, persist or selectively drop G1 carrier/selection/admission state; Pack A records their continued presence rather than relabelling them as G1 guards. New-store live workload is zero because the feature does not yet exist.

| Site / guard | Bound / current workload | Breach behaviour / observable | Forecloses / over-limit fixture |
|---|---|---|---|
| Existing authenticated Server.Handler | authenticated caller; existing product workload | existing auth rejection before parse/state access | unauthorised instruction read/write; real token/no-token |
| DESIGNATED profile validator | text 0..65536 bytes, UTF-8, NUL-free; enabled text nonblank; current G1 profiles 0 | `invalid_input`, no write | binary/unbounded/empty-active carrier; max+1/NUL/blank |
| DESIGNATED HTTP instruction parser | raw body ≤524288, exactly one object, unknown members refused; endpoint currently absent | typed HTTP failure before state write | ambiguous/multi-object edits; 524289/padded body |
| DESIGNATED `Repository.Save` CAS | expected revision = current; revisions currently 0 | `revision_conflict`; old current retained | last-writer-wins; two-process same expected |
| DESIGNATED revision retention | ≤1024 nonzero revisions; current 0 | `capacity_exceeded` before insert | unbounded history; injected-small-cap organic writes |
| DESIGNATED binding retention | ≤20000 conversation bindings; current 0 | `capacity_exceeded` before bind | unbounded binding growth; small-cap organic opens |
| DESIGNATED reload retention | ≤256 actual reloads/conversation; current 0 | `capacity_exceeded`; selection unchanged | unbounded event growth; small-cap organic reloads |
| DESIGNATED lease | one active turn/compaction per conversation | `conversation_busy` | mid-turn revision mutation; held real turn/compaction |
| DESIGNATED schema/digest decoder | schema exactly 1; digest recomputed from actual fields | `unsupported_schema` / `corrupt_record` | fabricated state recovery; raw corrupt/restart |
| DESIGNATED binding admission | enabled profile requires `binding=null` in G1 | `binding_unresolved` | fake founded identity; plausible non-null binding |
| DESIGNATED local admission | C≥32; reserve 0..C/2; prompt estimate ≤C−R | `context_unavailable`, `invalid_admission` or `context_exceeded` with counts | silent mandatory-input loss; I-13 min/max/overflow |
| `server/prompt.go:chatPrompt` amended | admitted history fits C−R by whole turns; current code drops oldest messages at C | dropped count in final admission report; newest/system overflow refuses | orphaned tool result / silent carrier drop; I-13 tool-group cases |
| `server/routes.go:truncateNativeChatMessages` amended | same C−R rule | same | native-route silent loss; I-13 native cases |
| `llm/llama_server.go:completionPromptForRequest:276` amended | admitted request keeps whole prompt; current measured 4289-token cut to 258/511 at C=512 | A6 refusal before cut/inference | rendered silent tail loss; clean→cut mutant→clean |
| DESIGNATED generate admission | carrier with raw, explicit template, suffix or legacy context is invalid | `invalid_admission`; no inference | carrier under unqualified generate semantics; C-4 cases |
| DESIGNATED route capability check | correct capability before every carrier dispatch; current daemon advertises neither G1 capability | `context_unavailable`; no inference | old daemon silently ignores field; T10/D11/classic cases |
| DESIGNATED final admission report | final report required after every successful carrier dispatch; current response has none | `context_unavailable`, answer marked unverified | false success with no retention evidence; report-drop mutant |
| DESIGNATED cloud qualification | exact origin+model+profile and B≤V; measured V 99680/103671/25722 | `not_qualified` or `context_exceeded`; nothing forwarded | unmeasured cloud carrier use; synthetic-provider tests, live requalification only with Ash authority |

The new vocabulary cannot represent a recovered LETHE memory, a verified founding identity solely from a name, a transcript page, a completed autonomous goal, or a tool-authorisation grant. These are author examples only; Method 16 R-7.2 still requires each independent judge to supply its own five cannot-represent statements. Any foreclosure intersecting a frozen G1 requirement is BLOCKING.

### 8.1 OQ-4 worked arithmetic, total boundary mappings and shipping triggers

| Quantity | Producer-attainable minimum | Producer-attainable maximum / threshold | Exact result and survivorship |
|---|---|---|---|
| Local reserve `R = clamp(C/8,16,4096)` | C=32 | C=32,768 and above | C=32 → 32/8=4 → lower clamp R=16 → fit budget 16. C=32,768 → 4096 → R=4096. C<32 refuses before carrier dispatch because R=16 would exceed C/2. |
| Local fit `P ≤ C−R` | P=0 | exact boundary and +1 | P=C−R survives intact; P=C−R+1 refuses `context_exceeded`; no newest-turn partial cut. |
| Revision arithmetic | current 0 | signed-64 max 9,223,372,036,854,775,807 | valid changed save uses current+1; overflow refuses before insert. Signs, leading zeroes except `0`, and >19 digits are `invalid_input`. |
| Reload generation | bound conversation begins 1 | after 256 actual reloads generation=257 | idempotent same-target reload does not increment; next actual change refuses before mutation. |
| Raw PUT escape bound | significant body can be minimal valid edit | 524,288 raw bytes | independently derived worst significant body 407,970 bytes; 524,289 or padded over-limit body rejects before parse. |
| Cloud prompt bound B | starts 16 plus actual field costs | V per exact table entry | B=V admits; B=V+1 refuses before forwarding. No whole-history fitting to B. |
| Cloud room warning | B < 0.8V | B ≥ 0.8V | below threshold no room notice; at/above threshold show remaining verified-bound room. Diagnostic only, never admission evidence. |

Boundary decoders are total: Profile schema accepts only `1`; `ReloadEvent.kind` only `opened|reloaded`; bound selection generations are 1..257 while ephemeral one-shot alone uses 0; request admission version is exactly 1; final admission route is `local|cloud`; cloud `limit_kind` is exactly `verified_prompt`; G1-local error codes are the closed set at the start of §8; daemon admission codes are `invalid_admission|context_unavailable|context_exceeded` while provider errors remain provider errors. Unknown persisted values refuse as `corrupt_record`/schema error; unknown wire admission values never map to success.

| Shipping configuration | Trigger that arms G1 | Required recovery / proof |
|---|---|---|
| Desktop app | saved profile enabled; first accepted chat binds; Apply performs reload | real Settings→restart→new-chat browser journey; restoring store/daemon allows a later turn without rewriting profile |
| Agent TUI | normal open or `/new`; reload/off | I-05 + OQ-3 terminal arm; capability re-read before every carrier turn |
| Classic interactive `ollama run` | session open; `/clear` or `/load`; reload/off | classic OQ-3 scenarios; refused turn excluded from later history |
| One-shot/piped `ollama run` | invocation snapshots current profile as generation 0 | no binding/event; stdout parity and stderr diagnostics |
| Local `/api/chat` | `chat.admission.v1` field | I-13 native/rendered; legacy no-field request unchanged |
| Local `/api/generate` | `generate.admission.v1` field | C-4 conformance; promptless load/unload unchanged |
| Qualified explicit cloud chat | exact A13 origin+model+profile and B≤V | synthetic-provider default proof; live requalification only with Ash authorisation |
| Daemon without G1-aware client | no field supplied | existing requests unchanged |

## 9. Acceptance design, organic producers and deliberate faults

Test storage is isolated; actual production entry/assembly is exercised. External inference may be simulated for transport/store tests, clearly declared. Selected real-model qualification is a separate bounded run; recorded model name/digest, actual served identity, request/response evidence and limits remain attributable. No real user conversation, Keychain or credential is used as a fixture.

| Test ID | Requirement IDs | Mode | Production path / source-derived assertion | Facade/twin caught |
|---|---|---|---|---|
| I-01 | G1-R-01, G1-R-02, G1-R-08, G1-R-12 | ORGANIC | Real editor/API saves contrasting A and B; fresh process reads exact committed bytes and current revision | Fixed string, in-memory-only store |
| I-02 | G1-R-03, G1-R-05 | ORGANIC | Start X at A, save B elsewhere, send twice in X and once in new Y; capture actual requests | Current-settings-on-every-turn, same default conversation ID |
| I-03 | G1-R-04, G1-R-08, G1-R-14 | ORGANIC | Hold a real turn/compaction, attempt reload, then finish and reload idle | UI-only gate, always-allow, never-allow |
| I-04 | G1-R-01, G1-R-03, G1-R-10, G1-R-13 | ORGANIC | Restart desktop host and reopen X; X stays on A until explicit reload | NewChat replacement, regenerate from current profile |
| I-05 | G1-R-03, G1-R-05, G1-R-06, G1-R-10, G1-R-11 | ORGANIC | Agent TUI normal entry and `/new`; classic-chat open/`/clear`/`/load`; model/tool switches; built-in `/system off`; compaction. Assert runtime ChatID equals committed ConversationKey and distinct conversations have distinct IDs | Unused new ID, binding-only identity, selection lost on reset, duplicate systems |
| I-06 | G1-R-06, G1-R-13, G1-R-15 | ORGANIC | Disabled/unconfigured desktop, agent, classic and one-shot requests equal frozen pre-feature request goldens | Always-enabled, no-op flag, constant receipt |
| I-07 | G1-R-02, G1-R-09, G1-R-12, G1-R-14 | INJECTED | Invalid text, unknown schema, corrupt digest, non-null active binding, write denial and stale CAS through real boundaries | Always-success, swallowed read error, schema-valid wrong identity |
| I-08 | G1-R-02, G1-R-12, G1-R-14 | INJECTED | Zero/exact-max/max+1 for text/body/revision/binding/reload capacities; smaller injected capacities preserve identical transitions | Prefix truncation, cap-after-write, signed overflow |
| I-09 | G1-R-05, G1-R-08, G1-R-14 | ORGANIC | Real Settings route and top-level desktop chat in Playwright: save, restart, new conversation, reload and visible refusal/error | Story-only component, local echo, inaccessible input |
| I-10 | G1-R-05, G1-R-06, G1-R-07 | ORGANIC | Qualified local model receives complete carrier through real adapter with model-system/tools preserved; oversized mandatory input refuses visibly; classic and one-shot local arms included | Editor-only wiring, lower-layer truncation, model-template replacement |
| I-11 | G1-R-03, G1-R-13 | ORGANIC | Open/reload event roundtrip through actual store and next-request diagnostics | Hash-only event with no consumer |
| I-12 | G1-R-16, G1-R-17, G1-R-18 | INJECTED | Gate reconciles discovered tests and true child exits; exit 7, empty selection and missing case fail | Vacuous pass, shell-exit proxy |
| I-13 | G1-R-05, G1-R-06, G1-R-07, G1-R-15, G1-R-17 | INJECTED | Real `/api/chat` and `/api/generate` handlers with existing runner seams: route-specific capability, C≥32, keep −1, C−R whole-turn fitting, A6/A7 refusal, reports, structured continuation, remote/MLX refusal, qualified-chat cloud table/profile/bound, and C-4 generate restrictions. Generate cloud remains `not_qualified`; compatibility routes reject admission. Requests without admission equal pre-feature goldens | Keep 4, reserve ignored, silent cut, unqualified route admitted, field forwarded, report re-encoded/dropped, disabled request altered |
| I-14 | G1-R-05, G1-R-07, G1-R-14, G1-R-15, G1-R-17 | INJECTED | Desktop, agent TUI, classic chat and one-shot adapters against daemons with missing/present route capabilities and each admission error. No carrier reaches a capability-less route; every tool continuation checks again; missing final report is failure; refusal report tokens/attribution and 80% cloud notice are asserted | Capability ignored, carrier stripped-and-retried, code/count lost, silent refusal, report skipped, unprotected continuation |

Contract controls, each run as clean → one fault → clean against unchanged I-13/I-14 assertions: send keep 4 instead of −1; fit to C instead of C − R; take the rendered cut instead of refusing; admit a cloud request above its limit; admit an unlisted cloud model; forward the admission field; count bytes without NFKC or JSON escaping; re-encode the final record instead of inserting; skip the capability check; drop tool-group integrity. I-10's real-model arm reuses the continuation-13 harness with the contract enabled: a packed history with a long reply and a prompt at the clamp edge must show no shift whose keep is below the prompt length, the carrier's sentinel in the reply, and the clamp-edge prompt refused under A6. Its cloud arm sends carrier-bearing requests through each adapter to each model in A13's table, near its limit and just above it. That spends provider tokens, so it runs only with Ash's authorisation when the reality gate is reached.

The eight standard degenerate twins are constant output, identity/no-op, always-success, always-reject, never-write, never-read, all-zero/empty selection and dropped configured input. I-01/I-02 detect constant/empty/dropped input; I-06 detects no-op/always-enabled; I-07 detects always-success; I-01/I-03 detect always-reject; I-04 detects never-write/never-read. A disconnected final composer, stale-binding loader and failed CAS are additional specific mutants. Mutations occur only in isolated copies after candidate/tests are committed. Each mutant compiles first and then fails its intended assertion; clean before and after runs pass. A compile failure or missing browser binary is an infrastructure failure, not detection credit.

Existing baseline, executed this continuation: `go test -count=1 -json -timeout=120s -run '^(TestSessionAddsSystemPromptOnlyToRequest|TestChatSystemCommandControlsBuiltInSystemPrompt)$' ./agent ./cmd/tui/chat` passed two named tests; `go test -count=1 -json -timeout=120s -run '^TestChatPrompt$/^no_truncate_with_limit_exceeded$' ./server` passed its parent and named subtest. They prove existing request construction and retained over-limit prompt behaviour, not G1 persistence, context admission or a passing new-feature failure number.

Continuation-12 recovered the previously timed-out entry baseline without changing its test: the authenticated existing-settings control passed; the instructions PUT returned HTTP 200 with the application HTML rather than a saved Profile and failed its JSON/value assertion. Both named tests completed, process exit 1. A separately built real CLI passed ordinary --help and rejected instructions --help with unknown-command/exit 1. These are actual pre-feature entry failures, not a save/restart success or a census of every lifecycle scenario. Commands, logs, isolated environment and exact test/binary hashes are retained in continuation-12 http-baseline-original/ and cli-baseline/. Continuation 14 measured the full failure number through the actual entry points: 66/39 on the unchanged candidate, on a valid run, with every no-carrier case correct (`proof_cont14/oq3-baseline-r2/`).

DESIGNATED default Go test files are `internal/instructions/store_test.go`, `lease_test.go`, `compose_test.go`, `app/ui/instructions_test.go`, `cmd/tui/chat/instructions_test.go`, `cmd/instructions_test.go`, and Darwin integration gate `cmd/instructions_delivery_darwin_test.go` containing `TestG1InstructionDelivery` and `TestG1InstructionDeliveryControls`. The latter owns the process-separated desktop, agent-TUI, classic-chat and one-shot failure-number/report scenarios and is discovered by the normal Darwin `./cmd` suite; its combined-gate invocation is `go test -count=1 -json -timeout=30m -run '^TestG1InstructionDelivery(Controls)?$' ./cmd`. Default frontend tests cover API/state transitions without a browser dependency; real Playwright acceptance uses the existing installed Playwright package, actual frontend routes and isolated real backend. The browser suite is mandatory G1 acceptance, never substituted with SSR.

## 10. Component packs, full-system proof and closure

One feature, one blueprint. Component packs are not independently accepted OI items. The ordering is:
0A resolve this draft and source/guard populations → 0B independent Tier-2 panel → A verified build handoff → B1 shared owner/representation/lease → C1 independent proof → B2a daemon admission contract (§6.3) → C2a proof → B2b both host request/lifecycle adapters → C2b proof → B3 actual editor/CLI/reload UI → C3 proof → D/E documentation verification → F combined G1 reality gate. B2a does not depend on B1 and may be built in parallel with it under separate file custody. B1's public store acceptance is not advertised as a delivered instructions feature. No external memory engine, autonomous scheduler or estate-wide cleanup is included.

| Pack | DESIGNATED changed/new production files | Current → projected LOC / added estimate |
|---|---|---|
| B1 | internal/instructions/types.go; store.go; compose.go; lease_unix.go; lease_windows.go | 0→140; 0→310; 0→70; 0→55; 0→60; subtotal 635 |
| B2a | api/types.go; api/client.go; server/routes.go; server/prompt.go; server/cloud_admission.go; server/cloud_proxy.go; llm/server.go; llm/llama_server.go | 1348→1418; 512→562; 3118→3348; 156→201; 0→230; 568→598; 292→304; 2804→2834; subtotal 697 |
| B2b | app/ui/instructions.go; app/ui/ui.go; cmd/agent_tui.go; cmd/tui/chat/chat.go; cmd/tui/chat/input.go; cmd/tui/chat/compaction.go; cmd/tui/chat/debug.go; agent/session.go; cmd/cmd.go; cmd/interactive.go | 0→220; 2004→2124; 852→942; 1246→1306; 1728→1818; 101→121; 565→600; 1087→1122; 2623→2703; 735→820; subtotal 835 |
| B3 | app/ui/app/src/components/InstructionSettings.tsx; InstructionStatus.tsx; app/ui/app/src/api.ts; Settings.tsx; Chat.tsx; app/ui/app/src/hooks/useChats.ts; cmd/tui/chat/render.go; cmd/instructions.go | 0→230; 0→90; 879→934; 650→660; 345→400; 760→795; 2157→2192; 0→120; subtotal 630 |

Total estimated new production LOC is **2797** (B1 635, B2a 697, B2b 835, B3 630), giving a Method-16 R-3.2 document-byte ceiling of **111,880 bytes**. The increase from the earlier 2181-line estimate is not padding: Ash's 27 September D-11 ruling added classic interactive and one-shot `ollama run`, which adds `cmd/cmd.go`, `cmd/interactive.go`, generate-route admission and their diagnostics/tests to the production scope; OQ-4 also made `cmd/tui/chat/debug.go` and per-request capability/report custody explicit. Tests, generated output, this document and evidence do not inflate the estimate. The repository declares no numeric file ceiling in `AGENTS.md`, so R-3.4's over-ceiling arm is inapplicable while its current/projected-line obligation remains satisfied above. Pack-series closeout records estimate versus actual.

Declared combined proof set, before G1 closure: all tests in `./internal/instructions`, `./agent`, `./cmd/tui/chat`, `./cmd`, `./app/store`, `./app/ui`, `./server`, `./llm`; full frontend Vitest; generator reproducibility/check tests; TypeScript project check; full ESLint with exact baseline comparison; frontend production build; root Go build; the actual browser save/restart/dispatch journey; the real chosen-model carrier/capacity qualification; and all deliberate-break controls above. This is not a waiver of previously recorded root/app/Keychain/package release obligations. Keychain-sensitive suites require proven isolation or a supervised OS-authorised run, not another blind invocation. A known-red baseline remains a blocker to the affected combined gate, while unrelated design work may proceed.

Only G1's real joint save→restart→request and reload/disabled/failure results can close OI-03/OI-04. No installed-app claim follows from HTTP tests or a pushed branch. Main integration and installation retain their separately recorded release gates.

## 11. Requirement mapping and handoff

| Requirements | Representation / owning contract | Acceptance | Build / review |
|---|---|---|---|
| G1-R-01,02,12 | Profile, Edit, revisions/current CAS; §3 | I-01,I-07,I-08 | B1/C1/F |
| G1-R-03,04,10,11 | ConversationKey, Selection, Lease, ReloadEvent; §4 | I-02,I-03,I-04,I-05,I-11 | B1+B2/C1+C2/F |
| G1-R-05,06,15 | Compose and real request adapters; §§2,5 | I-05,I-06,I-10 | B2/C2/F |
| G1-R-07 | Admission contract A1–A13 (§6.3) and matrix (§6.4) | I-10, I-13, I-14 | B2a+B2b/C2a+C2b/F |
| G1-R-08 | Actual editor/preview/CLI/reload controls; §7 | I-01,I-03,I-09 | B3/C3/F |
| G1-R-09 | Null/unbound enabled or non-null inactive binding admission; §3 | I-07 | B1/C1/F; activating bound colleagues remains G2 |
| G1-R-13 | Separate instruction/transcript ownership; §2 | I-04,I-06,I-11 | B1+B2/C1+C2/F |
| G1-R-14 | Typed errors and operator-visible refusal/report contract; §8 | I-03,I-07,I-08,I-14 | B1+B2+B3/C1+C2+C3/F |
| G1-R-16 | Exact candidate, instruments and reproducible boundary evidence; §§9,10 | I-12 plus all owning evidence | C1+C2+C3+E/F |
| G1-R-17 | Deliberate faults for each load-bearing behaviour; §9 | I-01..I-14 controls | C1+C2+C3/F |
| G1-R-18 | Joint save→restart→request/reload/disabled/failure acceptance; §§9,10 | I-01..I-14 + failure-number/report verdict | F |

OQ-4's author-side census is now incorporated; the exact supporting population and R-14 checker live in `proof_cont16/oq4-census/`. The next work is therefore **not more design discovery**: repair the OQ-3 instrument to revision 5 against this stated contract, move old goldens aside, pre-register and run a fresh baseline, obtain the fresh revision-5 review, then dispatch the first two-substrate Tier-2 Method 16 panel over one frozen document SHA and this source SHA. Only an `EARNED` `[IMPLEMENTATION-READY]` result unlocks Pack A/B. All 31 programme requirements remain in the canonical inventory; no feature is removed or accepted here.

BLUEPRINT_CREATED=PARTIAL
PACK_PERMISSION=PACKS_BLOCKED (derived from [SPEC-DRAFT])
