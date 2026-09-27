# Ollama integration — continuation 02

Date: 20 September 2026, Australia/Brisbane. Owner: Eko.
Source HEAD: `953de98d408a9697fa20a48c23973b9dc8eee921`.
Status: OI-01 reconnaissance advanced; one concrete persistence prerequisite reproduced. No production repair, feature acceptance, commit, push or installation.

## Most important finding: a failed read can replace saved history

The production desktop continuation handler treats `not.Found` as permission to create a new chat under the requested id. `Store.ChatWithOptions` currently labels every database chat-read error as `not.Found`, including a message scan error. `database.saveChat` then deletes the existing messages and reinserts the newly constructed history.

The exact chain is `app/store/database.go:675-713` (header/message load), `app/store/store.go::Store.ChatWithOptions` (all errors relabelled), `app/ui/ui.go:727-787` (new-chat fallback and SetChat), and `app/store/database.go:716-773` (transactional replacement). Mechanically numbered source and whole-file hashes are retained in `evidence/2026-09-20_cont02/mechanical-source-anchors.json`.

A real temporary SQLite store was populated through Store.SetChat with two original messages. The diagnostic then set its persisted stream boolean to invalid text and sent an HTTP POST through the real Server.Handler. The original-message count changed from 2 to 0; the request returned HTTP 200 and made three requests to the loopback inference fixture. Healthy-existing and genuinely absent-chat controls behaved normally. This is a measured data-loss mechanism, not an estimate of how often users encounter it.

Evidence: `evidence/2026-09-20_cont02/history-probe-v2-baseline/observations.json`, `probe-v2-run.json`, `probe-v2-run.stdout`, `probe-v2-run.stderr`, and `testdata/chat_read_probe.go`. The diagnostic program exited 1 because integrity failed; the surrounding capture command exiting 0 is not a test pass.

Scope of proof: real Go HTTP handler, real temporary SQLite, deliberately injected invalid stored value, simulated loopback inference, development-mode authentication bypass. No real user conversation was used or damaged. This diagnostic does not prove authentication, real model behaviour, installed-app execution, preservation of attachments/tool calls, or post-repair correctness.

The first injection, integer 2, did not cause a SQLite boolean-scan error. Its failing integrity assertion was an ineffective fault injection, not evidence of lost messages. Its complete original source/results remain preserved in `chat_read_probe_v1.go.txt`, `history-probe-baseline/`, `probe-run.*` and `probe-v1-disposition.json`. Two source-write attempts containing test authentication fields were refused by the bounded Bridge writer and not applied; the accepted diagnostic was explicitly narrowed to Dev mode, not rerouted through a different mutation tool to evade the refusal.

## Proposed repair boundary

`docs/_design/CHAT_READ_ERROR_INTEGRITY.md` defines the proposed smallest repair: classify only absence of the chat-header row as `not.Found`, and propagate every other database error unchanged through Store.ChatWithOptions. The existing continuation handler already aborts on such errors before SetChat and inference. No schema change, new state owner or UI production edit is required by that proposal.

The six frozen requirements cover absence classification, underlying-error preservation, unchanged saved rows on failure, no inference on failure, valid continuation, and genuine-absence continuation. The draft is not implementation-ready. Permanent default-run tests, authenticated entry-point proof, all-record preservation, reopen/functional-use proof, compiled fault controls, dependency reconciliation and the complete independent panel are still owed.

The shared getter also affects other callers. `Server.getChat:1333-1348` still turns any error into an empty HTTP-200 chat response; `renameChat:1405-1409` still labels errors with misleading text; `deleteChat:1591-1598` distinguishes not.Found correctly. These are explicit reconciliation targets, not silently bundled repairs. The new prerequisite is narrower than all chat-read UX.

## Baseline actually executed

All commands ran on the Mac against this checkout, with a temporary HOME/USERPROFILE and existing Go caches. Dependency fetching was disabled. The installed app and its data were not replaced. Full child arguments, exits, environment overrides and log hashes are in `evidence/2026-09-20_cont02/baseline-results.json`.

| Surface | Actual result |
|---|---|
| `go build -o <owned output> .` | Exit 0; empty stdout/stderr. This is the core Go binary, not a packaged desktop-app build. |
| `go test -count=1 -json -timeout=180s ./agent/... ./cmd/tui/chat/... ./app/store ./app/ui ./app/tools` | Exit 0; 593 passing named test/subtest records, 0 failing records, 1 skip. The count includes parent/subtest records and is not a leaf-test count. |
| `npm test -- --run` | Exit 0; 143 tests in 21 files passed. |
| TypeScript `tsc --build --pretty false` | Exit 0; empty stdout/stderr. |
| `npm run lint` | Exit 1; 137 errors and 11 warnings, all before production edits. |
| Tracked source/build-manifest fingerprints before/after | No changes. |

The Go skip is `agent/tools::TestNormalizeBashWorkingDirWindowsDriveLetter`, inapplicable to this macOS run. Frontend stderr also contains the stale Browserslist database warning and the clipboard test's intentional fallback error. None is silently suppressed or treated as a new regression.

Not run: the full repository `go test ./...`, complete `./app/...`, CMake/native rebuild, frontend Vite production bundle, packaged desktop application or installed-candidate acceptance. Default Darwin MCP tests touch the real login Keychain, so full-root execution needs an explicit safe fixture boundary rather than merely changing HOME. This is not a waiver of whole-programme proof.

## Actual DeepSeek review

Ash explicitly selected `deepseek-v4.1-flash:cloud`. Two bounded generation attempts used that exact requested tag through the existing TECHNE Ollama wrapper; one completed. No other provider/model was substituted and no reviewer had write authority.

The first response exhausted its output limit (`TRUNCATED_RESPONSE`, wrapper exit 5) and was not accepted as a verdict. The second used the identical source/prompt bytes with thinking off and a 16,384-token output allowance. It completed with `done_reason=stop`, wrapper exit 0, served model `deepseek-v4.1-flash`, and final `BLUEPRINT_JUDGE=FAIL`.

Requested-tag manifest before/after: `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8`.
Prompt SHA-256: `4e7215e1cab331398a98a34973a708af2d0ba98c8079c9f8571282be72d9377e` (241,390 bytes).
Completed response SHA-256: `57e6630a39c85086074a88ef38117749782f6c396374da4dab3b2eddebb174e2`.

The judge corroborated the loss mechanism and proposed error-classification repair. It challenged the provisional Tier 1 classification because the observable crosses HTTP, and identified missing design/test/trace material. Tier 2 is the conservative continuation classification. A manifest digest and served-model name are not an immutable cloud-weights identity or a competence benchmark.

This was source-text review of explicitly included material, not an independent agent with repository or execution tools. The commission and some dependent definitions were not supplied; the report lacks complete Tier-2 rule coverage and calibrated B1–B7 exhibits. It is retained as useful adversarial evidence, not a qualified promotion round, independent re-execution or release gate. The detailed disposition of its finite 15-item list is in `JUDGE_RECONCILIATION_02.md`. No PASS was issued by Eko on the author's own design.

## Installed application differs from the checkout

The inspected `/Applications/Ollama.app` reports `0.32.5-47-geb8c93d`. Its resource binary's `go version -m` reports VCS revision `eb8c93da6ab20b980449226029a9bf5cf40a6306`, modified=false. This differs from checkout HEAD `953de98d…`. Source/baseline results above therefore do not prove the installed app has the same behaviour or includes future changes. No architecture/translation-mode conclusion is drawn from a single Go build-info slice.

## Context, archive and viewer source reconnaissance

`agent/session.go` was read in full, including request assembly, tool-result truncation and compaction. `cmd/tui/chat/compaction.go` and `events.go` were read in full. Manual compaction replaces both working messages and displayed entries; automatic EventCompacted replaces working messages. The chat event sink sets a variable named newMessagesPersisted when emitting a UI event; that name does not establish a disk-write acknowledgement. Remaining terminal persistence construction/callers must be traced before asserting a canonical archive exists or is absent.

The complete desktop Chat.tsx and MessageList.tsx paths were read. MessageList maps the whole supplied message array and rescans tool-query history when it changes. No latency/cause claim is made without a measured long-conversation fixture. useChats.ts was inspected by selected symbol lines only. The existing Curated Context v10 blueprint and v3 concept were indexed by headers/sections, not read in full or promoted; their current headers explicitly lack implementation authority.

## Custody and next action

Ten completed Bridge jobs' full command specifications, terminal metadata and exact stdout/stderr have been copied and hash-verified under `evidence/2026-09-20_cont02/bridge-commands/`. No job was forgotten in this continuation. Three task-generated binaries were moved byte-for-byte to the verified Git-ignored `build/eko-colleague-integration-20260920/`; `binary-custody.json` preserves both old and new paths. Original execution records retain their actual historical paths.

Next bounded work: complete and judge the Tier-2 chat-continuation integrity contract with exact public-path tests and source/dependency access, then implement only the two-file error-classification repair and its default-run tests. Preserve the original red probe, run clean/fault/clean controls and a fresh independent code/evidence review before accepting the repair. Broader OI-01 reconciliation and the original 31-item programme remain open. No service, installation, real conversation, global configuration, branch or remote was changed.
