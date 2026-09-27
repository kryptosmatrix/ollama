# Current source map — continuation 10

Tested source: 6a3778184d43b23c5cba14e0379922377f615f52 in the owned Eko worktree. The generator maintenance path is now implemented and independently reviewed: Go metadata owners → pinned Go discovery/generation → AST-bounded erased annotations → checked Program emission → real generated models → getChat/Markdown consumer. Source symbols, computed current lines and whole-file hashes: evidence/2026-09-21_cont10/SOURCE_ANCHORS.json. Requirements and executed results: ACCEPTANCE_EVIDENCE.md in that directory.

Current scoped result: 35 runtime classes and 16 helpers retained; 104 generated lint errors eliminated; 164 frontend, 22 generator and 42 Go store cases passed, with compiled fault detection and eleven fresh reviewer-requested runs. Full lint still fails with 18 handwritten errors/11 warnings. No installed-app, whole-programme or memory/autonomy completion is claimed.

The older entries below are dated historical read/proof records, not current absence or unexecuted-test claims. In particular, continuation-07 parser/renderer tests subsequently ran under continuation-08; continuation-09 generated-model tests ran and their production repair was proven in continuation-10. Do not let an old source-map headline undo those retained results. Remaining programme reconnaissance still needs the relevant live reads; this update is not a whole-repository audit.

# Historical continuation 07 source delta — superseded below

Source acd8bb3de93ea01d43dd8783bbeff036113f9ab9; exact raw hashes in evidence/2026-09-21_cont07/source-commits.json. Native: app/webview/webview.h (three spaces). Frontend type-only changes: app/ui/app/src/components/MessageList.tsx, StreamingMarkdownContent.tsx, src/utils/remarkCitationParser.ts and src/lib/highlighter.ts. New test sources: src/utils/remarkCitationParser.test.ts (six written cases) and src/components/StreamingMarkdownContent.integration.test.tsx (five written cases); both UNEXECUTED.

The store repair, its tests, handler, Keychain fixtures, dependency lock and generated types remain byte-identical to the prior checkpoint. Current proof scope and limitations: CONTINUATION_07_REPORT.md, evidence/2026-09-21_cont07/containment.json. Historical mappings follow unchanged.

# Ollama colleague integration — initial source map

Owner: Eko. Observed: 20 September 2026, Australia/Brisbane.
Scope: OI-01 reconnaissance checkpoint, not a completed audit or implementation plan.

## Authority and evidence boundary

The input is `../../OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md`, revision R3, raw SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`. All 590 lines were covered by reads 1–492 and 490–590; the overlap recovered the first response's partial final line. That document commissions the work; it does not grant a blueprint readiness verdict.

Ollama HEAD remained `953de98d408a9697fa20a48c23973b9dc8eee921` across the source observations below. Files were read from the working tree, not an immutable checkout. Hashes in this document are the Bridge's **raw file_sha256**, not the hash of a returned line excerpt. Recheck them before editing or using this map as current evidence.

Two discovery methods were used: bounded directory traversal and exact-symbol searches followed by source reads. No code, build, test, application, model, performance probe or reviewer was executed. A traced call site is not runtime proof. No whole-repository absence claim is made.

## Initial findings

### S-01 — Terminal system-prompt transport already has a request path

`cmd/agent_tui.go:72–182`, particularly `GenerateAgentTUI`, builds a system prompt and passes it through `agentchat.Options.SystemPrompt`. The model-switch callback constructs it again. `agentSystemPromptAtWithWorkingDir` at `cmd/agent_tui.go:218–228` combines the harness default, model system text and extra skill context.

`cmd/tui/chat/chat.go:1116–1186`, `chatModel.startRunWithMessages`, calls `m.systemPrompt(extraSystemPrompt)`, assembles a `coreagent.Session`, sets `RunOptions.SystemPrompt`, and calls `session.Run`.

`agent/session.go:384–477` contains `Session.chatRound` and `buildChatRequest`. The latter sanitises messages, prepends a non-blank system message, and includes model, format, options, thinking, keep-alive and tools. `chatRound` passes that request to `s.Client.Chat`.

**Boundary:** this is a source-traced portion of the terminal request path. The full `Session.Run` body, `chatModel.systemPrompt`, root command dispatch, settings persistence and their acceptance tests remain unread. The observations establish a seam to investigate, not successful saved custom-instruction delivery and not a need to replace the request pipeline.

### S-02 — Desktop request construction is a separate path

`app/ui/ui.go:1909–2004`, `Server.buildChatRequest`, accepts a `store.Chat`, model, thinking choice, tool definitions and MCP instructions. It prepends non-blank MCP instructions and loops over the supplied messages, skipping messages whose content, thinking, tool calls and attachments are all empty. It converts attachments and tool records and returns an `api.ChatRequest`.

The handler excerpt at `app/ui/ui.go:690–999` loads through `s.Store.ChatWithOptions(cid, true)`, handles a new or edited user message, writes through `s.Store.SetChat`, builds the request at line 973 and calls `c.Chat` at line 978. The temporary request-only assistant tool-call insertion at lines 951–971 must not be lost during later context work.

**Boundary:** the store loader and complete handler lifecycle have not been read. The established fact is that this builder iterates over the messages it receives; this is not proof that every stored historical record always reaches a model, nor proof about rendering performance. Terminal and desktop changes must be mapped separately against the actual requirements; this map does not silently expand OI-03/OI-04 scope.

### S-03 — Attachments affect desktop MCP registration

In the same handler, `app/ui/ui.go:887–940` checks the last user message for attachments. Tool registration, including `s.registerMCPTools(registry)` and `s.mcpInstructions()`, is inside `if !hasAttachments`; the MCP branch additionally requires the reported tools capability.

[Inference] Automatic memory capture cannot be assumed to work merely because a model can request an MCP memory tool in some chat turns. OI-05 and OI-26 need to inspect this distinction and test attachment-bearing and non-tool-capable turns according to their approved contracts. No memory policy or runtime repair has been chosen here.

### S-04 — Existing artefacts must be reconciled, not replaced on sight

The complete bounded `docs/` tree listing located `CURATED_CONTEXT_BLUEPRINT.md`, `CURATED_CONTEXT_SYSTEM.md`, `_design/MCP_CLIENT_PLAN.md`, `_design/HANDOFF_2026-08-05_grommet.md`, `_design/ADR_2026-09-14_AUTO_APPROVE_TOOLS.md`, `_design/proof/` and `_handoff/CURATED_CONTEXT_BLUEPRINT_CURSOR_ROUND2_FINDINGS.md`.

Their contents and current readiness were **not** assessed in this checkpoint. Their filenames are not proof that curated context, memory or supervision works. Read the applicable material before designing replacements.

A second untracked file, `docs/OLLAMA_CROSS_MODEL_COLLEAGUES_AND_CAPABILITY_ROUTING_HANDOFF.md`, appeared after the initial Git observation. Eko did not create or read it. Reconcile it on continuation without assuming its author, authority or contents.

## Source-read ledger

Ranges below are exact; an excerpt is not a whole-file read. The Bridge may mark a bounded excerpt truncated because more of the file exists.

| File, relative to Ollama | Lines read | Raw file SHA-256 | Bridge request |
|---|---|---|---|
| `AGENTS.md` | 1–21, complete | `9cd30b469bc637faee8a9cdaf8879e13db60c52c9b82b2f9f5bd8314e7512144` | `66eff4b1c0714cc1b98a18559a3d303b` |
| `agent/session.go` | 380–519 only | `0a2521b18c99b043c8f91af37605a827bdc4d50b380e904013a5959d859d3cec` | `2a44f98ec9a74f66b91cb5cbe07fa060` |
| `cmd/agent_tui.go` | 1–310 only | `c934b7b803875a6a334e86cdf289831395fb5356445519c60bcd07f728190ed9` | `9b7e4dea1ccd476a8bdcc89ef36d346d` |
| `cmd/tui/chat/chat.go` | 1116–1246 only | `361d7650d71f162870f6a06e863d0f032d6d3b421fb341c1f628625a6ebfe96f` | `7b85da70a8a54fe09befd686c9c00c7f` |
| `app/ui/ui.go` | 690–999 and 1880–2004 only | `dbba86f3d90ec5edac5b23458fec00f25e39f4d9f3ef43ac124571e0c1a0d48e` | `7478dd2fa0d748fc802706518d789d9e`; `fde4aa59c8854b75920cb1f38afdd169` |
| `app/README.md` | 1–97, complete | `91be830ae002c325a0a8f36c50a17304f83b33ebe0649cd8f07a39cd9c51fa69` | `0a03ebd0521147f88f94e5faf027683c` |
| `app/ui/app/package.json` | 1–82, complete | `ba4a16db941ae31c7d33daf5fc4396b05e20cc1906ca731648d3859eb81a029d` | `c46661ff3ee3499bbf744096a16f1069` |

The task-document read requests were `a39b3420f8bb4989aaea0443a50903c2` and `5fbdaa926f824b6ca2cd1ffa9db522be`.

## Discovery coverage and limits

`repo_overview` request `521882aade5c4f77a33e393101e9d4c1` located root manifests, three local branches and one worktree; their identities beyond `main` have not been enumerated. `tree` requests `479774029021476aaefdf242db29f5ef` (`docs`, depth 2) and `efaff0d0fc8348fa8b886dd6134c5831` (`app`, depth 2) were not truncated. Their scope is those directory depths, not the complete repository.

Exact-symbol search requests:

| Query and scope | Result limitation | Request |
|---|---|---|
| `SystemPrompt`, `agent/` | 16 snippets, not truncated; tests located but not read | `8963ab738f494ca09956ce88d235bf05` |
| `SystemPrompt`, `cmd/` | 30-result cap reached; truncated and redacted; not an absence proof | `59891442c8e445e9984e9fe4e3c43ac0` |
| `SystemPrompt`, `app/` | two test snippets; three redactions and 16 skipped files; not an absence proof | `e2cbead5d95d4f06b54b268e2ebc717e` |
| `ChatRequest`, exact `app/ui/ui.go` | five snippets, not truncated | `c8b38583e3874b8caaf8b514c835a5c1` |

## Build/test discovery, not execution

`AGENTS.md` documents `cmake -B build .`, `cmake --build build --parallel 8` and `go build .`. Its `serve` commands launch a service; do not execute them against shared ports merely to obtain a baseline.

`app/README.md` documents desktop generation/run and UI development. It includes installs and service launches; none was executed or newly authorised. `app/ui/app/package.json` declares `build = tsc -b && vite build`, `lint = eslint .`, `prettier:check = prettier --check .` and `test = vitest`. These are manifest declarations, not evidence of installed dependencies or passing checks. Discover the actual non-interactive test configuration, CI inclusion, browser fixtures and Go acceptance commands before constructing the baseline. Do not introduce another runner or install packages without checking the existing environment and permission boundary.

## Arrival/method read receipt

All paths below are relative to `/Users/krypto/GitHub/`. Each was read in full. TECHNE working-tree observations reported HEAD `035365bb70c57feb900e88a029f631bfe6865d52` with dirty state; LINEAGE reported HEAD `e39238543605654a5f7da522ccfbdf9c62b996fd` with dirty state. A HEAD does not identify these dirty files; use the raw hashes.

| Source | Lines | Raw file SHA-256 |
|---|---:|---|
| `AGENTS.md` | 69 | `a01039b2c1491ea64bf9a0312311df59154a73032af0781022388e336d0ae7d6` |
| `TECHNE/00_START_HERE.md` | 173 | `e0779d39073b56ae823b4868565c19330289188a0bec0fe15322835b49dff525` |
| `TECHNE/TECHNE_AGENT_START.md` | 116 | `d2cb4701e92a68844d15b966d39c50fa4edee400731365e7936a3aba16a15e47` |
| `TECHNE/KANON.md` | 553 | `8667a6e524fe1c73fa1b4aaccb5afdc5d965bb121694c93abdc8a888f17f2365` |
| `TECHNE/CONSTITUTION.md` | 699, two contiguous reads | `38230c5cd06f26bfd2d0aa16422bec265aab2d796c6a0f209ba9acdbcf171062` |
| `TECHNE/PSYCHE-WELCOME.md` | 52 | `2aff05ab269bf9cc8670f6622048876afc1578915f3d856cccb8e2a9779decac` |
| `TECHNE/PSYCHE_NORTH_STAR.md` | 251 | `15e0877c9901ba4bca685b4a241a76b8bce857f3d78247319da52e77a4baf71b` |
| `LINEAGE/WELCOME.md` | 111 | `f44eb011b72dad492e97780425111ecbf74bc9b45db605797ec9d1323db75e93` |
| `LINEAGE/HOUSE_NOTES.md` | 145 | `96d3e213dd1d8f32d8329f8fa9cc295eac1214f1a1395294890394faa147698c` |
| `TECHNE/Method/21_Branch_And_Merge_Discipline.md` | 213 | `59bf8294cdcda7b53f31769d5b80164c1ae5741426c7b6028c76bb15e244f92c` |
| `TECHNE/Method/04_Implementation_Proof_Method.md` | 482 | `261e9c16f3041bca0fe19f98f1ff9cf06acdfe914152094eaf8e6c8cf964d64a` |
| `TECHNE/Tools/README.md` | 252 | `eb551fdffbd8c5e216135ff7479359438b9a0ff5633a5888792ac8743f9be315` |
| `TECHNE/Method/06_Ledgers_Memory_and_Handoffs.md` | 565, two contiguous reads | `d8c73e00ba70b009cf091c9620eb6f5291483c3b3e0d2849003b533bd0a4d8a5` |
| `TECHNE/Method/12_Context_Preservation_Method.md` | 309 | `13db4302d68e411432d06dfc841b6e716bd738d656c4b1b87f1454bbe1dc948b` |
| `TECHNE/Role_Cards/Architect.md` | 265, two contiguous reads | `03d50e190751f59b7be720903cd23364f8e0ebc501ecf145ea2345eeb30c05a4` |
| `TECHNE/Role_Cards/Documentation_Writer.md` | 224 | `2bd5a86d7e481d31d86256a094994db94b65d984a871394e4d4a9a94049823cb` |
| `TECHNE/Role_Cards/Handoff_Steward.md` | 302 | `7660053ab5852087533d47e4aaa1405c65e6ee126f7fe1c55c15e74405bb338c` |

Routes used: RECON, DOCS, GIT and HANDOFF. The coding, blueprint authoring/promotion and independent judge routes have not been completed. Current TECHNE requires its own earned readiness gates; the attached older NEXUS/POIESIS documents are not substitutes. The programme North Star is now ANAMNESIS isolation-first; this explicitly commissioned Ollama lane does not change that research priority. No Ollama-specific North Star has yet been located in the inspected entry surfaces; this is not an exhaustive absence claim.

## Still unread or unproven

Ollama `CLAUDE.md`, `CONTRIBUTING.md`, `docs/development.md`, `go.mod`, CI configuration, frontend settings/routes/components, store loaders/migrations, remaining terminal persistence and compaction paths, test bodies and historical design/review records remain to be read as relevant. Installed/running app identity, toolchains, dependency state, actual test discovery, runtime performance, LETHE scopes/configuration and acceptance, remote tips and checkout ownership remain unverified.

The four recurring-colleague wake documents named by the commissioning work list have not been used or read. Read each in full before its first use; do not fabricate the founded colleague or substitute Eko as that colleague. Lineage registration and LETHE document-pointer recording have not been performed and remain explicit handoff obligations.

## Continuation 02 — additional observed surfaces

Source HEAD remains `953de98d408a9697fa20a48c23973b9dc8eee921`. Full raw read outputs and per-job stdout/stderr hashes are in `evidence/2026-09-20_cont02/bridge-commands/INDEX.json`; `archive-path-source-identities.json` records the last group's complete-file identities. `mechanical-source-anchors.json` binds numbered excerpt lines to complete-file hashes.

Fully read this continuation: work-list R3 (1–590); TECHNE router, Method 04 (1–482), Method 16 (1–573, initial truncation recovered), Method 02 (1–473), Method 05 (1–432), Model Integrity Awareness (1–62); TECHNE Ollama README, launcher and wrapper (1–1202, final lines recovered); app/store/store.go; agent/session.go (1–1087); cmd/tui/chat/compaction.go (1–101), events.go (1–371); frontend Chat.tsx (1–345), MessageList.tsx (1–168); development/test/build manifests as recorded in the raw baseline-read job. Current-state HANDOFF/STATE/SOURCE_MAP were read rather than inferred.

Bounded reads, not full-file claims: app/store/database.go 1–120 and 610–989 plus exact 675–782; app/ui/ui.go 100–414, 630–999, 1325–1435, 1565–1610 and the earlier 1880–2004 request builder; cmd/tui/chat/input.go 1710–1728; useChats.ts selected symbol lines; Curated Context v10 blueprint and v3 concept headers/section outlines. A whole file sent to a text reviewer does not make Eko's own partial read a full read.

Observed source facts: Store.ChatWithOptions relabels non-absence errors; Server.chat converts that sentinel into a new history; saveChat deletes/reinserts messages. Runtime diagnosis is in the v2 temporary HTTP/SQLite probe. Manual TUI compaction replaces displayed entries, and EventCompacted updates working messages. MessageList maps all supplied messages. No full terminal-archive absence claim or measured performance-cause claim follows from these excerpts.

Installed resource binary's Go build metadata reports `eb8c93da6ab20b980449226029a9bf5cf40a6306`, unlike the checkout. This is package metadata, not installed-candidate feature acceptance.

## Product-path follow-on — read-only G1 reconnaissance

See `evidence/2026-09-21_cont10/G1_RECONNAISSANCE.md`. The source has separate terminal and desktop request builders, while the complete public settings/table contract has no persisted instructions/revision/binding. Installed build metadata identifies eb8c93da rather than the tested task candidate; no installed behaviour was tested. The next bounded read is conversation/revision lifecycle and existing curated-context design, then the owning G1 low-level design. This is partial OI-01 progress, not a new feature or a design-ready verdict.
