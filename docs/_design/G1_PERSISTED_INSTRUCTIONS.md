# G1 — Persisted instructions and actual request delivery

| Field | Value |
|---|---|
| Identity | OLLAMA-G1-INSTRUCTIONS; edition 1; one feature, OI-03 + OI-04 |
| Author | Eko, GPT-6 Astra Pro, 21 September 2026 (edition 1 draft); revised by Thole, Claude Opus 5.5, 26 September 2026 (continuation 13: header, §0, §5, §6, §7–§11) |
| Document class | implementation blueprint |
| Design readiness | [SPEC-DRAFT]; OQ-1 decided in §6; OQ-3, OQ-4 and the first Method 16 panel remain owed |
| Implementation status | Not Yet Implemented |
| Source revision | 17197224933cbdd9875c2277f18767c5f6f134a2 for §2–§5; 04009c432e313eef65172293e75feb038ebe476b for §6 (no production or test file differs between them) |
| Concept | docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md; SHA256 9396cd684d47d3e6e200c227e75e1a0114d7cd8a20107db528c4bc26871fb241 |
| Tier | 2: persistence yes; process/language crossing yes; estimate above 300 LOC yes |
| Requirement IDs | G1-R-01 through G1-R-18, 18 frozen outcome requirements; first adjudication has not started |
| Pack permission | PACKS_BLOCKED, derived from [SPEC-DRAFT] under Method 16 §2.1 and R-0; edition 1 authored `PACKS_ALLOWED_FOR_0A_0B_ONLY`, a token the ladder does not define, corrected here. Completing this document and its panel (§10, steps 0A and 0B) is design work, not a licensed pack |
| Scope | Shared saved instructions, immutable conversation selection, editor, safe explicit reload and both real request paths |
| Repository line ceiling | UNDECLARED; no 500-line ceiling inferred from generic guidance |

Ash edits and saves instructions once, restarts, and sees the chosen text influence a genuinely new conversation through its actual outgoing model request. An active conversation keeps its original instruction revision until a deliberate safe reload. The failure number is **incorrect instruction deliveries per controlled conversation**, measured over save/restart/new, edit/continue, explicit reload, disable, terminal reset and compaction cases: any value above zero fails G1. The final instrument is DESIGNATED `app/ui/instructions_test.go::TestInstructionsAcrossProcesses` plus `cmd/tui/chat/instructions_test.go::TestInstructionConversationLifecycle` and the browser journey in §9. The pre-build measured baseline in continuation-11 covers existing system-prompt transport and no-truncation behaviour only; it is NOT this full failure-number measurement. That missing baseline is an explicit promotion obligation, not a zero invented from absence.

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
| D6 | One opt-in, versioned daemon admission contract for carrier-bearing requests (§6.3) | Shift off; keep-all without a reserve; client-side prompt assembly; a version-number gate; cloud allowed with a label (ADR 2026-09-26) | Reserve floor provisional until §6.5 V1; the ADR's falsifiers reopen it |

**Open obligations before promotion.** OQ-1 is decided (§6 and `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md`); what remains of it are the verification obligations V1–V7 in §6.5, carried into the packs. OQ-2 is resolved prospectively below: G1 permits an unbound ordinary instruction profile; all active non-null colleague bindings are refused until the separately commissioned G2 validator is implemented and qualified. This does not count G2 as delivered. OQ-3: install and execute the full failure-number baseline through the actual HTTP/terminal entry points. The attempted read of the existing TTS test fixture was platform-blocked in continuation-11; no child ran and that specific file was not accessed by another route. This does not make existing system-prompt tests a substitute for the missing baseline. OQ-4: complete independently derived closure/guard/test population and the two-substrate Method16 panel. All were assigned to Eko and are assigned to Thole from 26 September 2026; no new operator approval is requested. No build pack starts with them open.

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

Contract use in both adapters: every request that carries an enabled carrier also carries the §6.3 A1 admission field, and only after the adapter has confirmed the daemon's `chat.admission.v1` capability (A12). A disabled selection sends no admission field, so its request is unchanged (G1-R-15). A refusal under A6 or A10, or a missing capability, is mapped by §8 and shown to the operator; the adapter never retries without the carrier.

Preview is a point-in-time result for the explicit `(model, revision, selection_generation, base_digest)` returned alongside it. It returns the exact effective carrier for that tuple through the same construction helpers. The view labels a prospective new-conversation preview as unbound. Changing any tuple member, including a reload generation, makes that preview stale; it is never represented as proof of a later dispatch. It is labelled a preview, not proof of dispatch, model compliance, memory recovery or available context. Actual-request diagnostics separately record revision, generation, source digest, carrier byte count, model and request outcome without storing private instruction text in public logs or repository evidence.

## 6. Context capacity and the admission contract

Decided on 26 September 2026 in `docs/_design/ADR_2026-09-26_G1_INSTRUCTION_ADMISSION.md` (commit `756e65503da4b200d5cf9a97c74c268bd7907be4`), after a blind options check on two non-Claude substrates whose claims are dispositioned in `docs/analysis/colleague_integration/proof_cont13/oq1-options-check/`. This section is the contract that G1-R-07 incorporates by reference; G1-R-06, G1-R-14 and G1-R-15 bind the clauses marked. Source anchors below are at `04009c432e313eef65172293e75feb038ebe476b`, verified 26 September 2026; production and test sources are identical to the header revision.

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

Clauses A1–A12 are the contract. A request that carries an enabled carrier is an *admitted request* when it carries A1's field.

- **A1 — wire field.** DESIGNATED `api/types.go` type `ChatAdmission` with `Version int \`json:"version"\`` and `Reserve int \`json:"reserve,omitempty"\``, added to `ChatRequest` (`api/types.go:133-180`) as `Admission *ChatAdmission \`json:"admission,omitempty"\``. Version is 1. Reserve is output room in tokens; zero selects the default. A request without the field is processed exactly as today (G1-R-15).
- **A2 — validation.** The daemon refuses with HTTP 400 and code `invalid_admission` a version other than 1, a negative reserve, or a reserve above half the effective context. An unknown version is refused, never ignored.
- **A3 — reserve.** The effective context C is the runner's context length after the training-context cap (`llm/server.go:111-115`), equal to the native slot size in every measured run. The default reserve R is C/8 clamped to the range 16–4,096 tokens; a supplied reserve replaces it within A2's bounds. These values are chosen, not measured; the floor is provisional until V1.
- **A4 — keep.** The daemon sends a keep of −1 to the runner for every admitted request on both llama-server routes (the chat body's `n_keep`, the completion's `NKeep` at `llm/llama_server.go:1526`), so any context shift keeps the whole prompt.
- **A5 — history fit.** Both history loops (`server/prompt.go` `chatPrompt`; `server/routes.go` `truncateNativeChatMessages`) fit the estimated prompt into C − R instead of C, keep every system message, and drop whole turns from the oldest. A turn begins at a user message. The retained history begins at a user message or is exactly the newest turn, which is every message from the last user message to the end, including assistant tool calls and every tool result after them. No retained history begins with a tool result whose call was dropped (G1-R-06).
- **A6 — refusal.** If the system messages and the newest turn exceed C − R by the daemon's estimate, the daemon refuses before inference with HTTP 400, code `context_exceeded`, and the fields `context_length`, `reserve` and `estimated_prompt_tokens`.
- **A7 — no cut.** For an admitted request the Go runner never takes the cut at `llm/llama_server.go:299-315`; it returns A6's refusal. The native admission check stays the exact backstop and its error is mapped to `context_exceeded` with the native counts (§8).
- **A8 — continuation.** The structured-output continuation (`server/routes.go:2882-2903`) re-renders under A5–A7. The thinking it appends belongs to the newest turn; if the turn no longer fits, the stream ends with A6's error.
- **A9 — diagnostics.** The final response of an admitted request carries `admission` with `version`, `context_length`, `reserve`, `dropped_messages` and `context_exhausted`. `context_exhausted` is true when generation stopped because the context filled — possible only under A11 — and distinguishes that from an output-budget stop, which shares `done_reason: "length"`.
- **A10 — routes refused.** The daemon refuses an admitted request with HTTP 400 and code `context_unavailable`, naming the route, on explicit cloud models (`server/routes.go:2440-2448`), remote-host stubs (`:2495-2585`) and MLX runners (`x/mlxrunner`). Lifting a route needs its own measured qualification; for cloud routes that qualification sends traffic to the provider and is Ash's to authorise.
- **A11 — runners that cannot shift.** Multimodal runners (`server-context.cpp:1215-1217`) and model families launched without shift (`server/sched.go:137-147`) keep the prompt because they never shift. Generation stops when the context fills and A9 reports it.
- **A12 — capability.** `GET` and `HEAD /api/version` (`server/routes.go:1861-1862`) add `"capabilities": ["chat.admission.v1"]` when the daemon implements A1–A11. Both G1 adapters read it through DESIGNATED `api/client.go` `(*Client).Capabilities`, beside `Version` (`:454-464`), once per connection and after reconnect. When it is absent they send no carrier-bearing request and report `context_unavailable` naming the daemon's version (G1-R-14). They never send the carrier outside the contract and never drop the carrier to send anyway.

### 6.4 Supported-configuration matrix

| Route | Carrier protection | Overflow outcome | Reply room | G1 status | Evidence |
|---|---|---|---|---|---|
| llama-server native chat, local GGUF, text | A4, A5 | A6 before inference; native check backstop | at least R; shifting beyond it keeps the prompt | supported, measured on qwen3:8b | run1–run3b native |
| llama-server rendered (Go template, renderer or parser) | A4, A5, A7 | A6 / A7 | as above | supported, measured on qwen3:8b | run1–run3b rendered |
| llama-server with media | never shifts (A11) | native check; A6 with the existing 768-token image estimate | stops when full; A9 reports it | supported by construction, not measured (V5) | source |
| Model families launched without shift | never shifts (A11) | as the native route | stops when full; A9 | supported by construction, not measured | source |
| MLX | — | — | — | refused (A10); no MLX model is available here to qualify it | `x/mlxrunner/pipeline.go:47-57` |
| Explicit cloud models | — | — | — | refused (A10) until qualified with Ash's authorisation | source |
| Remote-host stubs | — | — | — | refused (A10) until qualified | source |
| Any daemon without the capability | — | — | — | refused by the client (A12) | — |

### 6.5 Verification obligations carried into the packs

- **V1** — measure the gap between the daemon's estimate and the runner's own count (text, tools, thinking, and templates that insert dates) on at least three local models; set the reserve floor to at least 4 plus the measured maximum plus 1, or reopen the native reserve.
- **V2** — streaming requests on both routes under the contract.
- **V3** — tool conversations: newest-turn retention with its whole tool group, and whole-turn dropping.
- **V4** — parallel slots: C against the native per-sequence size.
- **V5** — media: the 768-token estimate against actual projector tokens, and A6's behaviour.
- **V6** — the structured-output continuation (A8).
- **V7** — real-model qualification of carrier delivery (I-10) on each adapter's default local model.

## 7. Editor and terminal interaction

DESIGNATED `app/ui/instructions.go` exposes authenticated `GET /api/v1/instructions`, `PUT /api/v1/instructions`, `GET /api/v1/instructions/revisions`, `GET /api/v1/instructions/revisions/{revision}`, `POST /api/v1/instructions/preview`, and `POST /api/v1/chat/{id}/instructions/reload`. Revision listing returns at most 20 metadata-only entries before an optional decimal `before_revision` cursor; one revision GET returns its full accepted Profile. Restoring from a historical revision sends its content through the same new-revision CAS Save, never mutates history. The PUT envelope is exactly `{ "expected_revision": "<decimal>", "edit": { "enabled": <boolean>, "text": "<UTF-8>", "binding": <null-or-binding> } }`; profile digest/schema/revision are server-authored output and are not accepted input fields. Revision strings have no sign or leading zero except the single digit `0`, and are at most 19 decimal digits with signed-64-bit overflow rejected. GET returns current configuration and revision. PUT requires expected_revision plus Edit; it returns the committed profile only after durable success. Preview takes model and selected scope/revision and returns the shared composer's output plus truthful capacity status. Reload requires expected_generation and expected_target_revision. The first is the conversation generation the operator observed; the second is the exact global revision displayed for adoption. Both are compared in one transaction. A changed target causes `revision_conflict` with `conflict_field="target_revision"`; a changed conversation causes the same code with `conflict_field="generation"`. There is no silent adopt-latest operation: refresh the preview and let the operator choose the newly displayed target. The editor preserves unsaved work on either conflict. Unknown JSON members, invalid UTF-8/NUL, wrong JSON types, more than one top-level JSON value and oversized bodies are refused; the request limit is 524288 bytes. Worst-case JSON escaping costs at most six ASCII bytes per input UTF-8 byte for these bounded strings: text 65536 plus binding 2432 gives 407808 bytes, The independently encoded worst-case significant PUT body is 407970 bytes: 407808 escaped string bytes plus 162 envelope bytes, leaving 116318 bytes below the raw-body cap. This uses U+0001 (valid UTF-8, NUL-free and six-byte escaped), all four binding fields at their maxima, a 19-digit expected revision and enabled=false. Extra whitespace still counts towards the raw-body limit; a padded oversized request is rejected even when its decoded fields would fit. This is a body-allocation bound, not a token-budget measurement. Exact one-object parsing and all decoded field caps still apply; the body cap is not the decoded-text limit. Authentication precedes parsing and state access. Typed errors carry machine codes and human-readable text; no false Saved toast follows a non-2xx reply.

DESIGNATED `InstructionSettings.tsx` is mounted in the real Settings component adjacent to SpeechSettings. It displays scope `Desktop and terminal — new conversations`, current revision, enabled state, editable text, byte limit, explicit Save, preview, Disable and Reset. Save is not an every-keystroke write. Dirty edits are kept on a stale-revision conflict; no automatic last-writer-wins retry. Reset makes a new disabled empty revision with a route to select a previous retained value. The existing general Settings reset does not destroy instruction history. The draft editor never activates an unresolved identity.

Existing conversations display their selected revision and a button `Apply saved instructions` at the conversation header. The button is disabled while the UI knows the conversation is active, and the backend lease remains authoritative. A successful reload displays the new generation/revision; an error leaves the existing selection visible. The editor does not pretend that saving automatically updates open conversations.

When an admitted reply ends with A9's `context_exhausted`, or a request is refused under A6 or A10, the conversation view and the terminal show a one-line notice naming the cause and its counts; a reply that ended at its output budget is labelled as that, not as a full context. Preview reports capacity from A3's reserve and the route's §6.4 status, so a refused route is visible before a conversation starts.

Terminal DESIGNATED `/instructions` displays current selection and scope; `/instructions reload` applies the saved revision only at idle; `/instructions off` records and selects a disabled revision through the same store rather than silently changing only a prompt string. Saved editing is provided by the desktop surface and DESIGNATED `ollama instructions set --file <path> --expected-revision <revision>` for terminal-only use. Input is read as data, never executed as a shell/editor command. Exact CLI registration and validation anchors are part of OQ-4, not assumed present.

## 8. Failure representation, guards and cannot-represent scope

Closed error codes: `invalid_input`, `revision_conflict`, `conversation_busy`, `store_unavailable`, `corrupt_record`, `unsupported_schema`, `binding_unresolved`, `capacity_exceeded`, `context_unavailable`, `context_exceeded`. The following is the local instructions API mapping, not a claim about an upstream provider's status codes. When a provider error is translated, the diagnostic retains its upstream status and a safe reason without relabelling the upstream response itself. HTTP mapping: invalid_input 400; conflict/busy 409; binding_unresolved 422; capacity_exceeded 507; context unavailable/exceeded 422; store/corruption/schema 503. Errors stop the affected change/dispatch and preserve the old record. They do not require another operator approval and do not terminate independent conversations.

Daemon errors map as follows. `context_exceeded` from A6; llama-server's `exceed_context_size_error`, parsed from the JSON nested inside the daemon's `error` string for its `n_prompt_tokens` and `n_ctx`; the Go runner's 400 "the prompt is longer than the context length…"; and MLX's 400 "input length (N tokens) exceeds the model's maximum context length" all become `context_exceeded`, carrying whichever counts the source supplies. `context_unavailable` from A10, `invalid_admission` from A2, and a missing `chat.admission.v1` capability all become `context_unavailable`, naming the route or the daemon's version. An unrecognised daemon error is passed through with its status and is never reported as success.

| Guard | Workload/bound | Rejection / observable | Forecloses / test |
|---|---|---|---|
| Authentication | Existing Server.Handler authentication, not Dev mode | Existing auth rejection | Unauthorised profile read/write; real token/no-token test |
| Text size | Operator-controlled UTF-8, 0..65536 bytes | invalid_input before commit | Larger instruction values; exact/max+1 fixture |
| Active empty text | Enabled with whitespace-only text | invalid_input | Misleading empty enabled profile; disabled empty remains valid |
| Revision CAS | Real concurrent editor snapshots | revision_conflict | Silent overwritten edits; two-process same-expected race |
| Capacity | 1024 revisions; 20000 bindings; 256 reloads | capacity_exceeded | Unbounded retained metadata; small-cap organic boundary fixture |
| Lease | Active turn/compaction including tools | conversation_busy | Mid-turn configuration change; competing process fixture |
| Schema/digest | Stored actual bytes and declared schema | unsupported_schema/corrupt_record | Fabricated success from unreadable state; raw corruption/restart |
| Binding | G1 permits enabled only when binding is null | binding_unresolved | Activating a persona before G2; both plausible and placeholder-like non-null bindings fail while an unbound enabled profile succeeds |
| Model budget | §6.3 A3 reserve, A5 fit, A6–A7 refusal, A10 route refusal, A12 capability | context_exceeded / context_unavailable | Silent loss of the carrier or the newest turn; I-13, I-14, I-10 |

This table is a design guard list, not yet the complete independently re-derived R-8 closure census. The latter is an explicit promotion obligation. The new vocabulary cannot represent a recovered LETHE memory, a verified founding identity solely from a name, a transcript page, a completed autonomous goal, or a tool-authorisation grant. These five author examples do not substitute for Method16's five independent judge-supplied statements. Any proposed foreclosure intersecting a G1 requirement remains blocking, not cured by disclosure.

## 9. Acceptance design, organic producers and deliberate faults

Test storage is isolated; actual production entry/assembly is exercised. External inference may be simulated for transport/store tests, clearly declared. Selected real-model qualification is a separate bounded run; recorded model name/digest, actual served identity, request/response evidence and limits remain attributable. No real user conversation, Keychain or credential is used as a fixture.

| Test ID | Organic path and source-derived assertion | Facade/twin caught |
|---|---|---|
| I-01 | Real editor/API saves contrasting A and B; fresh process reads the exact last committed bytes | Fixed string, in-memory-only store |
| I-02 | Start X at A, save B in another process, send twice in X and once in new Y; capture actual requests | Current-settings-on-every-turn, same default conversation ID |
| I-03 | Start a held turn/compaction, attempt real reload, then finish and reload at idle | UI-only gate, always-allow, never-allow |
| I-04 | Restart desktop host and reopen X; request uses X's pinned A until explicit reload | NewChat replacement, regenerate from current profile |
| I-05 | Normal terminal entry, `/new`, model/tool switches, builtin `/system off` with custom instructions still enabled, and real compaction events | Unused new ID, selection lost on reset, append-only duplicate systems |
| I-06 | Disabled/unconfigured requests compared with frozen pre-feature request goldens including tools/model options | Always-enabled, never-enabled, constant receipt |
| I-07 | Invalid text, unknown schema, corrupt digest, wrong binding, write denial and stale CAS | Always-success, swallowed read errors, schema-valid wrong identity |
| I-08 | Limits exercised at zero, exact maximum and maximum+1, with live persisted counts | Prefix truncation, cap after write, unsigned wrap |
| I-09 | Real Settings route and real top-level chat route in Playwright; save/restart/reload and visible errors | Story-only component, local echo, inaccessible input |
| I-10 | Qualified model receives complete expected carrier with preserved model-system text/tools; oversized mandatory input fails visibly | Editor-only wiring, lower-layer truncation, model-template replacement |
| I-11 | Snapshot/reload event roundtrip through actual store and next request diagnostics | Hash-only record with no consumer |
| I-12 | All required tests are discovered, finish, and reconcile; a child exit 7, empty selection and missing case fail the gate | Vacuous pass and shell-exit proxy |
| I-13 | Daemon contract through the real `/api/chat` handler with the runner replaced by the existing test seam `mockRunner` (`server/routes_generate_test.go:52`), on the native and rendered routes: the runner receives keep −1; history is fitted to C − R by whole turns with no orphaned tool result; A6 refuses with counts before any runner call; A7 refuses instead of cutting; A10 refuses cloud, remote-host and MLX; A9 fields are present and `context_exhausted` distinguishes a full context from a budget stop; a request without the field produces a runner request identical to its pre-feature golden | Keep left at 4, reserve ignored, a cut instead of a refusal, a cloud route allowed, a disabled request altered |
| I-14 | Both adapters against a daemon lacking the capability, one advertising it, and one returning each A2/A6/A10 error and the native nested error: no carrier-bearing request reaches the capability-less daemon; a disabled selection sends no admission field; each error maps as §8 states and appears in the conversation notice | Capability ignored, carrier dropped to send anyway, error swallowed, silent retry |

Contract controls, each run as clean → one fault → clean against unchanged I-13/I-14 assertions: send keep 4 instead of −1; fit to C instead of C − R; take the rendered cut instead of refusing; allow the cloud route; skip the capability check; drop tool-group integrity. I-10's real-model arm reuses the continuation-13 harness with the contract enabled: a packed history with a long reply and a prompt at the clamp edge must show no shift whose keep is below the prompt length, the carrier's sentinel in the reply, and the clamp-edge prompt refused under A6.

The eight standard degenerate twins are constant output, identity/no-op, always-success, always-reject, never-write, never-read, all-zero/empty selection and dropped configured input. I-01/I-02 detect constant/empty/dropped input; I-06 detects no-op/always-enabled; I-07 detects always-success; I-01/I-03 detect always-reject; I-04 detects never-write/never-read. A disconnected final composer, stale-binding loader and failed CAS are additional specific mutants. Mutations occur only in isolated copies after candidate/tests are committed. Each mutant compiles first and then fails its intended assertion; clean before and after runs pass. A compile failure or missing browser binary is an infrastructure failure, not detection credit.

Existing baseline, executed this continuation: `go test -count=1 -json -timeout=120s -run '^(TestSessionAddsSystemPromptOnlyToRequest|TestChatSystemCommandControlsBuiltInSystemPrompt)$' ./agent ./cmd/tui/chat` passed two named tests; `go test -count=1 -json -timeout=120s -run '^TestChatPrompt$/^no_truncate_with_limit_exceeded$' ./server` passed its parent and named subtest. They prove existing request construction and retained over-limit prompt behaviour, not G1 persistence, context admission or a passing new-feature failure number.

Continuation-12 recovered the previously timed-out entry baseline without changing its test: the authenticated existing-settings control passed; the instructions PUT returned HTTP 200 with the application HTML rather than a saved Profile and failed its JSON/value assertion. Both named tests completed, process exit 1. A separately built real CLI passed ordinary --help and rejected instructions --help with unknown-command/exit 1. These are actual pre-feature entry failures, not a save/restart success or a census of every lifecycle scenario. Commands, logs, isolated environment and exact test/binary hashes are retained in continuation-12 http-baseline-original/ and cli-baseline/.

DESIGNATED default Go test files are `internal/instructions/store_test.go`, `lease_test.go`, `compose_test.go`, `app/ui/instructions_test.go`, `cmd/tui/chat/instructions_test.go`, and `cmd/instructions_test.go`; named test functions and requirement assignments are completed before promotion. Default frontend tests cover API/state transitions without a browser dependency; real Playwright acceptance uses the existing installed Playwright package, actual frontend routes and isolated real backend. The browser suite is mandatory G1 acceptance, never substituted with SSR.

## 10. Component packs, full-system proof and closure

One feature, one blueprint. Component packs are not independently accepted OI items. The ordering is:
0A resolve this draft and source/guard populations → 0B independent Tier-2 panel → A verified build handoff → B1 shared owner/representation/lease → C1 independent proof → B2a daemon admission contract (§6.3) → C2a proof → B2b both host request/lifecycle adapters → C2b proof → B3 actual editor/CLI/reload UI → C3 proof → D/E documentation verification → F combined G1 reality gate. B2a does not depend on B1 and may be built in parallel with it under separate file custody. B1's public store acceptance is not advertised as a delivered instructions feature. No external memory engine, autonomous scheduler or estate-wide cleanup is included.

| Pack | DESIGNATED changed/new production files | Current → projected LOC / added estimate |
|---|---|---|
| B1 | internal/instructions/types.go; store.go; compose.go; lease_unix.go; lease_windows.go | 0→140; 0→310; 0→70; 0→55; 0→60; subtotal 635 |
| B2a | api/types.go; api/client.go; server/routes.go; server/prompt.go; llm/server.go; llm/llama_server.go | 1348→1383; 512→537; 3118→3248; 156→191; 292→300; 2804→2822; subtotal 251 |
| B2b | app/ui/instructions.go; app/ui/ui.go; cmd/agent_tui.go; cmd/tui/chat/chat.go; cmd/tui/chat/input.go; cmd/tui/chat/compaction.go; agent/session.go | 0→200; 2004→2094; 852→922; 1246→1286; 1728→1778; 101→111; 1087→1107; subtotal 480 |
| B3 | app/ui/app/src/components/InstructionSettings.tsx; InstructionStatus.tsx; app/ui/app/src/api.ts; Settings.tsx; Chat.tsx; app/ui/app/src/hooks/useChats.ts; cmd/tui/chat/render.go; cmd/instructions.go; cmd/cmd.go | 0→210; 0→70; 879→924; 650→654; 345→375; 760→775; 2157→2172; 0→100; 2623→2628; subtotal 494 |

Total estimated new production LOC 1860 (B1 635, B2a 251, B2b 480, B3 494); byte ceiling 74,400 under R-3.2. Tests, generated output, this document and evidence do not inflate that estimate. The continuation-13 revision allocates the admission contract to B2a and its adapter and notice work to B2b and B3; the repository declares no line ceiling (R-3.4a, `AGENTS.md`), and current line counts were read on 26 September 2026.

Declared combined proof set, before G1 closure: all tests in `./internal/instructions`, `./agent`, `./cmd/tui/chat`, `./cmd`, `./app/store`, `./app/ui`, `./server`, `./llm`; full frontend Vitest; generator reproducibility/check tests; TypeScript project check; full ESLint with exact baseline comparison; frontend production build; root Go build; the actual browser save/restart/dispatch journey; the real chosen-model carrier/capacity qualification; and all deliberate-break controls above. This is not a waiver of previously recorded root/app/Keychain/package release obligations. Keychain-sensitive suites require proven isolation or a supervised OS-authorised run, not another blind invocation. A known-red baseline remains a blocker to the affected combined gate, while unrelated design work may proceed.

Only G1's real joint save→restart→request and reload/disabled/failure results can close OI-03/OI-04. No installed-app claim follows from HTTP tests or a pushed branch. Main integration and installation retain their separately recorded release gates.

## 11. Requirement mapping and handoff

| Requirements | Representation / owning contract | Acceptance | Build / review |
|---|---|---|---|
| G1-R-01,02,12 | Profile, Edit, revisions/current CAS; §3 | I-01,I-07,I-08 | B1/C1/F |
| G1-R-03,04,10,11 | ConversationKey, Selection, Lease, ReloadEvent; §4 | I-02,I-03,I-04,I-05,I-11 | B1+B2/C1+C2/F |
| G1-R-05,06,15 | Compose and real request adapters; §§2,5 | I-05,I-06,I-10 | B2/C2/F |
| G1-R-07 | Admission contract A1–A12 (§6.3) and matrix (§6.4) | I-10, I-13, I-14 | B2a+B2b/C2a+C2b/F |
| G1-R-08 | Actual editor/preview/CLI/reload controls; §7 | I-01,I-03,I-09 | B3/C3/F |
| G1-R-09 | Null/unbound enabled or non-null inactive binding admission; §3 | I-07 | B1/C1/F; activating bound colleagues remains G2 |
| G1-R-13 | Separate instruction/transcript ownership; §2 | I-04,I-06,I-11 | B1+B2/C1+C2/F |
| G1-R-14 | Typed errors; §8 | I-03,I-07,I-08 | B1+B2+B3/C1+C2+C3/F |
| G1-R-16,17,18 | Frozen candidate, complete proof records and joint acceptance; §§9,10 | I-01..I-12 | C1+C2+C3+E/F |

Next work is bounded: measure the §6.5 V1 estimate gap; execute the G1 failure baseline (OQ-3); complete exact declared sites/guard closure and independent source tracing (OQ-4); then the ordinary two-substrate panel with its lenses declared first. The chosen store and lifecycle design is a concrete proposal, not permission to fill those gaps silently during code. All 31 programme requirements remain in the canonical inventory; no feature is removed or accepted here.

BLUEPRINT_CREATED=PARTIAL
PACK_PERMISSION=PACKS_BLOCKED (derived from [SPEC-DRAFT])
