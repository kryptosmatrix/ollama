# G1 — Persisted instructions and actual request delivery

| Field | Value |
|---|---|
| Identity | OLLAMA-G1-INSTRUCTIONS; edition 1; one feature, OI-03 + OI-04 |
| Author | Eko, GPT-6 Astra Pro; 21 September 2026 |
| Document class | implementation blueprint |
| Design readiness | [SPEC-DRAFT]; independent review and the explicit open obligations below remain owed |
| Implementation status | Not Yet Implemented |
| Source revision | 17197224933cbdd9875c2277f18767c5f6f134a2 |
| Concept | docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md; SHA256 9396cd684d47d3e6e200c227e75e1a0114d7cd8a20107db528c4bc26871fb241 |
| Tier | 2: persistence yes; process/language crossing yes; estimate above 300 LOC yes |
| Requirement IDs | G1-R-01 through G1-R-18, 18 frozen outcome requirements; first adjudication has not started |
| Permission | PACKS_ALLOWED_FOR_0A_0B_ONLY; no production implementation is authorised by this draft's label |
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
| D6 | Existing no-truncation/no-shift controls are necessary but not a token-budget proof | Treat a byte/rune estimate as exact or silently trim | OQ-1 below blocks promotion until the actual admission contract is closed |

**Open obligations before promotion.** OQ-1: close the full budget-admission contract for supported local/native/cloud configurations without claiming that `truncate=false` counts tokens or that an unverified cloud provider honours it. OQ-2 is resolved prospectively below: G1 permits an unbound ordinary instruction profile; all active non-null colleague bindings are refused until the separately commissioned G2 validator is implemented and qualified. This does not count G2 as delivered. OQ-3: install and execute the full failure-number baseline through the actual HTTP/terminal entry points. The attempted read of the existing TTS test fixture was platform-blocked in continuation-11; no child ran and that specific file was not accessed by another route. This does not make existing system-prompt tests a substitute for the missing baseline. OQ-4: complete independently derived closure/guard/test population and the two-substrate Method16 panel. All are assigned to Eko; no new operator approval is requested. No build pack starts with them open.

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

Canonical digest input: a declared-order Go struct containing schema_version, revision, enabled, text and binding, marshalled by `encoding/json` with no trailing newline; digest field omitted. Binding uses its four fields in the listed order. UTF-8 and line endings are not normalised. Map iteration is not part of this representation. The read path recalculates the hash from actual fields before using text. Encoding `text="line one\nline two — café", enabled=true, revision="2"` and decoding recovers exact bytes; the expected hash is computed from independently supplied fixture bytes, not by calling the implementation's digest function in both arms. Concrete golden hashes are owed before promotion.

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

Preview is a point-in-time result for the explicit `(model, revision, selection_generation, base_digest)` returned alongside it. It returns the exact effective carrier for that tuple through the same construction helpers. The view labels a prospective new-conversation preview as unbound. Changing any tuple member, including a reload generation, makes that preview stale; it is never represented as proof of a later dispatch. It is labelled a preview, not proof of dispatch, model compliance, memory recovery or available context. Actual-request diagnostics separately record revision, generation, source digest, carrier byte count, model and request outcome without storing private instruction text in public logs or repository evidence.

## 6. Context capacity — explicit unresolved boundary

Observed source is not an exact budget oracle. `agent.checkPreflightPromptBudget` uses an estimate and `sanitizeMessagesForEstimate` omits images; `server.chatPrompt` has an image heuristic and, when truncate is false, skips that truncation loop. The native path similarly returns messages unchanged when truncation is disabled. The remote branch forwards the request to another server. Therefore neither a count of the user text nor successful JSON encoding proves the model can consume the mandatory carrier.

The resolved implementation contract has to cover the effective carrier, model template, current task, selected history, tool schemas/pairs, format/thinking overhead, attachments and output allowance, with source-attributable supported context and an explicit unavailable/overflow result. Reusing `ChatRequest.Truncate=false` and `Shift=false` prevents requested history dropping at interfaces that honour them, but is only one part of that contract. The production default is not allowed to claim exactness for the existing estimate. OQ-1 is OPEN; no placeholder capacity constant, assumed cloud tokenizer or no-op budget gate is specified or licensed by this draft.

### 6.1 Measured alternatives and the unresolved decision

Seven synthetic calls through the installed Ollama endpoint were retained in continuation-11 `cloud-budget-probe/`. At this service observation the requested cloud tag resolved to `deepseek-v4.1-flash`. A 25-input-token request completed with both requested `num_ctx=4096` and `num_ctx=8`; the latter field is not an enforced local admission boundary for that cloud path. Two zero-output requests (`num_predict=0`) were rejected with provider HTTP 400 and `max_tokens must be positive`. Three actual one-token requests returned input counts 25, 502 and 502, with one output token each; the two 502-input requests had identical input bytes. These are measured records, not a general guarantee of model maxima, complete media accounting or provider no-truncation behaviour.

The public-looking Anthropic CountTokensRequest type is not an exact tokenizer: `anthropic/anthropic.go:1092–1117` explicitly implements a len/4 estimate. The local llama wrapper’s `llamaServerTimings.promptEvalCount` at `llm/llama_server.go:1473–1475` sums cached and newly evaluated input tokens. Neither fact creates an exact cloud preflight API.

A one-token measurement call is a candidate design, not selected production behaviour. It would make an additional real inference request with the full context and would have to discard all generated text and tool calls without execution, preserve the exact request/model identity, enforce a source-attributable output reserve, account for its own latency/cost, and establish that reported counts cover the untruncated requested input. The small probes do not discharge those obligations. An estimate labelled exact, trusting requested num_ctx as enforced, or renaming a one-token inference a zero-cost tokenizer is rejected. The alternative is an actual model-specific tokenization/admission interface with explicit unavailable results on unqualified backends. Selection between these is still OQ-1; neither path is silently wired.

## 7. Editor and terminal interaction

DESIGNATED `app/ui/instructions.go` exposes authenticated `GET /api/v1/instructions`, `PUT /api/v1/instructions`, `GET /api/v1/instructions/revisions`, `GET /api/v1/instructions/revisions/{revision}`, `POST /api/v1/instructions/preview`, and `POST /api/v1/chat/{id}/instructions/reload`. Revision listing returns at most 20 metadata-only entries before an optional decimal `before_revision` cursor; one revision GET returns its full accepted Profile. Restoring from a historical revision sends its content through the same new-revision CAS Save, never mutates history. The PUT envelope is exactly `{ "expected_revision": "<decimal>", "edit": { "enabled": <boolean>, "text": "<UTF-8>", "binding": <null-or-binding> } }`; profile digest/schema/revision are server-authored output and are not accepted input fields. Revision strings have no sign or leading zero except the single digit `0`, and are at most 19 decimal digits with signed-64-bit overflow rejected. GET returns current configuration and revision. PUT requires expected_revision plus Edit; it returns the committed profile only after durable success. Preview takes model and selected scope/revision and returns the shared composer's output plus truthful capacity status. Reload requires expected_generation and expected_target_revision. The first is the conversation generation the operator observed; the second is the exact global revision displayed for adoption. Both are compared in one transaction. A changed target causes `revision_conflict` with `conflict_field="target_revision"`; a changed conversation causes the same code with `conflict_field="generation"`. There is no silent adopt-latest operation: refresh the preview and let the operator choose the newly displayed target. The editor preserves unsaved work on either conflict. Unknown JSON members, invalid UTF-8/NUL, wrong JSON types, more than one top-level JSON value and oversized bodies are refused; the request limit is 524288 bytes. Worst-case JSON escaping costs at most six ASCII bytes per input UTF-8 byte for these bounded strings: text 65536 plus binding 2432 gives 407808 bytes, The independently encoded worst-case significant PUT body is 407970 bytes: 407808 escaped string bytes plus 162 envelope bytes, leaving 116318 bytes below the raw-body cap. This uses U+0001 (valid UTF-8, NUL-free and six-byte escaped), all four binding fields at their maxima, a 19-digit expected revision and enabled=false. Extra whitespace still counts towards the raw-body limit; a padded oversized request is rejected even when its decoded fields would fit. This is a body-allocation bound, not a token-budget measurement. Exact one-object parsing and all decoded field caps still apply; the body cap is not the decoded-text limit. Authentication precedes parsing and state access. Typed errors carry machine codes and human-readable text; no false Saved toast follows a non-2xx reply.

DESIGNATED `InstructionSettings.tsx` is mounted in the real Settings component adjacent to SpeechSettings. It displays scope `Desktop and terminal — new conversations`, current revision, enabled state, editable text, byte limit, explicit Save, preview, Disable and Reset. Save is not an every-keystroke write. Dirty edits are kept on a stale-revision conflict; no automatic last-writer-wins retry. Reset makes a new disabled empty revision with a route to select a previous retained value. The existing general Settings reset does not destroy instruction history. The draft editor never activates an unresolved identity.

Existing conversations display their selected revision and a button `Apply saved instructions` at the conversation header. The button is disabled while the UI knows the conversation is active, and the backend lease remains authoritative. A successful reload displays the new generation/revision; an error leaves the existing selection visible. The editor does not pretend that saving automatically updates open conversations.

Terminal DESIGNATED `/instructions` displays current selection and scope; `/instructions reload` applies the saved revision only at idle; `/instructions off` records and selects a disabled revision through the same store rather than silently changing only a prompt string. Saved editing is provided by the desktop surface and DESIGNATED `ollama instructions set --file <path> --expected-revision <revision>` for terminal-only use. Input is read as data, never executed as a shell/editor command. Exact CLI registration and validation anchors are part of OQ-4, not assumed present.

## 8. Failure representation, guards and cannot-represent scope

Closed error codes: `invalid_input`, `revision_conflict`, `conversation_busy`, `store_unavailable`, `corrupt_record`, `unsupported_schema`, `binding_unresolved`, `capacity_exceeded`, `context_unavailable`, `context_exceeded`. The following is the local instructions API mapping, not a claim about an upstream provider's status codes. When a provider error is translated, the diagnostic retains its upstream status and a safe reason without relabelling the upstream response itself. HTTP mapping: invalid_input 400; conflict/busy 409; binding_unresolved 422; capacity_exceeded 507; context unavailable/exceeded 422; store/corruption/schema 503. Errors stop the affected change/dispatch and preserve the old record. They do not require another operator approval and do not terminate independent conversations.

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
| Model budget | Exact full carrier, supported context and allowance | context_unavailable/context_exceeded | Silent mandatory-input loss; OQ-1 needs exact algorithm and real backend proof |

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

The eight standard degenerate twins are constant output, identity/no-op, always-success, always-reject, never-write, never-read, all-zero/empty selection and dropped configured input. I-01/I-02 detect constant/empty/dropped input; I-06 detects no-op/always-enabled; I-07 detects always-success; I-01/I-03 detect always-reject; I-04 detects never-write/never-read. A disconnected final composer, stale-binding loader and failed CAS are additional specific mutants. Mutations occur only in isolated copies after candidate/tests are committed. Each mutant compiles first and then fails its intended assertion; clean before and after runs pass. A compile failure or missing browser binary is an infrastructure failure, not detection credit.

Existing baseline, executed this continuation: `go test -count=1 -json -timeout=120s -run '^(TestSessionAddsSystemPromptOnlyToRequest|TestChatSystemCommandControlsBuiltInSystemPrompt)$' ./agent ./cmd/tui/chat` passed two named tests; `go test -count=1 -json -timeout=120s -run '^TestChatPrompt$/^no_truncate_with_limit_exceeded$' ./server` passed its parent and named subtest. They prove existing request construction and retained over-limit prompt behaviour, not G1 persistence, context admission or a passing new-feature failure number.

DESIGNATED default Go test files are `internal/instructions/store_test.go`, `lease_test.go`, `compose_test.go`, `app/ui/instructions_test.go`, `cmd/tui/chat/instructions_test.go`, and `cmd/instructions_test.go`; named test functions and requirement assignments are completed before promotion. Default frontend tests cover API/state transitions without a browser dependency; real Playwright acceptance uses the existing installed Playwright package, actual frontend routes and isolated real backend. The browser suite is mandatory G1 acceptance, never substituted with SSR.

## 10. Component packs, full-system proof and closure

One feature, one blueprint. Component packs are not independently accepted OI items. The ordering is:
0A resolve this draft and source/guard populations → 0B independent Tier-2 panel → A verified build handoff → B1 shared owner/representation/lease → C1 independent proof → B2 both host request/lifecycle adapters → C2 proof → B3 actual editor/CLI/reload UI → C3 proof → D/E documentation verification → F combined G1 reality gate. B1's public store acceptance is not advertised as a delivered instructions feature. No external memory engine, autonomous scheduler or estate-wide cleanup is included.

| Pack | DESIGNATED changed/new production files | Current → projected LOC / added estimate |
|---|---|---|
| B1 | internal/instructions/types.go; store.go; compose.go; lease_unix.go; lease_windows.go | 0→140; 0→310; 0→70; 0→55; 0→60; subtotal 635 |
| B2 | app/ui/instructions.go; app/ui/ui.go; cmd/agent_tui.go; cmd/tui/chat/chat.go; cmd/tui/chat/input.go; cmd/tui/chat/compaction.go; agent/session.go | 0→200; 2004→2054; 852→892; 1246→1286; 1728→1778; 101→111; 1087→1107; subtotal 410 |
| B3 | app/ui/app/src/components/InstructionSettings.tsx; InstructionStatus.tsx; app/ui/app/src/api.ts; Settings.tsx; Chat.tsx; cmd/instructions.go; cmd/cmd.go | 0→210; 0→70; 879→924; 650→654; 345→355; 0→100; 2623→2628; subtotal 444 |

Total estimated new production LOC 1489; byte ceiling 59560. Tests, generated output, this document and evidence do not inflate that estimate. Context-admission work is NOT hidden in the estimate: OQ-1 has to choose its exact existing or newly owned surfaces, revise this pre-adjudication estimate honestly, and obtain review before freezing the first panel. No code is allocated to an unselected mechanism.

Declared combined proof set, before G1 closure: all tests in `./internal/instructions`, `./agent`, `./cmd/tui/chat`, `./cmd`, `./app/store`, `./app/ui`, `./server`, `./llm`; full frontend Vitest; generator reproducibility/check tests; TypeScript project check; full ESLint with exact baseline comparison; frontend production build; root Go build; the actual browser save/restart/dispatch journey; the real chosen-model carrier/capacity qualification; and all deliberate-break controls above. This is not a waiver of previously recorded root/app/Keychain/package release obligations. Keychain-sensitive suites require proven isolation or a supervised OS-authorised run, not another blind invocation. A known-red baseline remains a blocker to the affected combined gate, while unrelated design work may proceed.

Only G1's real joint save→restart→request and reload/disabled/failure results can close OI-03/OI-04. No installed-app claim follows from HTTP tests or a pushed branch. Main integration and installation retain their separately recorded release gates.

## 11. Requirement mapping and handoff

| Requirements | Representation / owning contract | Acceptance | Build / review |
|---|---|---|---|
| G1-R-01,02,12 | Profile, Edit, revisions/current CAS; §3 | I-01,I-07,I-08 | B1/C1/F |
| G1-R-03,04,10,11 | ConversationKey, Selection, Lease, ReloadEvent; §4 | I-02,I-03,I-04,I-05,I-11 | B1+B2/C1+C2/F |
| G1-R-05,06,15 | Compose and real request adapters; §§2,5 | I-05,I-06,I-10 | B2/C2/F |
| G1-R-07 | Full-capacity admission, unresolved §6 | I-10 | OQ-1 blocks pack allocation and promotion |
| G1-R-08 | Actual editor/preview/CLI/reload controls; §7 | I-01,I-03,I-09 | B3/C3/F |
| G1-R-09 | Null/unbound enabled or non-null inactive binding admission; §3 | I-07 | B1/C1/F; activating bound colleagues remains G2 |
| G1-R-13 | Separate instruction/transcript ownership; §2 | I-04,I-06,I-11 | B1+B2/C1+C2/F |
| G1-R-14 | Typed errors; §8 | I-03,I-07,I-08 | B1+B2+B3/C1+C2+C3/F |
| G1-R-16,17,18 | Frozen candidate, complete proof records and joint acceptance; §§9,10 | I-01..I-12 | C1+C2+C3+E/F |

Next work is bounded: resolve OQ-1's real capacity/transport contract; execute the G1 failure baseline; complete exact declared sites/guard closure and independent source tracing; then the ordinary two-substrate panel. The chosen store and lifecycle design is a concrete proposal, not permission to fill those gaps silently during code. All 31 programme requirements remain in the canonical inventory; no feature is removed or accepted here.

BLUEPRINT_CREATED=PARTIAL
PACKS_ALLOWED_FOR_0A_0B_ONLY
