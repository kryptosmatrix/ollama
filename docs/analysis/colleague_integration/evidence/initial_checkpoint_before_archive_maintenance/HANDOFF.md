# Eko — Ollama integration checkpoint

Date: 20 September 2026, Australia/Brisbane.
Controller: Eko. Active item: OI-01. Disposition: PARKED — execution prerequisite missing.

## Resume here

The live checkpoint is at `/Users/krypto/GitHub/ollama/docs/analysis/colleague_integration/`.
Read `STATE.json`, `BLOCKERS.md` and `SOURCE_MAP.md` there before continuing. This handoff is an initial reconnaissance record, not delivery of a runtime feature.

Ash commissioned Eko to implement the complete saved work list in active, human-prompted sessions. There is no unattended ChatGPT task or scheduler running. The eventual recurring Ollama colleague has not been founded; Eko is the delivery controller, not a substitute founding identity.

## Immediate blocker

The Eko Write Bridge can read files, inspect local Git state and create report files, but command admission returns `COMMAND_ARCHIVE_FULL`.

Two separate attempts were refused: `eko-ollama-20260920-arrival-01`, a read-only document probe, and `eko-ollama-20260920-execution-recheck-01`, which attempted only `/usr/bin/true`. Neither returned a job identifier or a child exit code. The complete error responses are retained in `evidence/2026-09-20_eko_execution_blocker.json` and `evidence/2026-09-20_eko_execution_recheck.json`.

No logs have been deleted, jobs forgotten, service settings changed or processes stopped. Existing job histories require preservation-first maintenance with appropriate approval. The exact archive count and retention condition have not been diagnosed; do not assume deleting one record will fix it.

## What this session established

The commissioning document was read in full: `docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md`, R3, 590 lines, raw SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`. All 31 identifiers are retained in `REQUIREMENTS.md`. No item is accepted.

Current source observations were made at Ollama HEAD `953de98d408a9697fa20a48c23973b9dc8eee921`. Directory traversal and exact-symbol searches found existing code and earlier design material. Limited source reads traced terminal system-prompt construction through `GenerateAgentTUI`, `chatModel.startRunWithMessages`, `RunOptions.SystemPrompt`, `buildChatRequest` and the `Client.Chat` call site. Unread intermediate bodies and entry points are explicitly listed in `SOURCE_MAP.md`; this is not an executed end-to-end proof.

The desktop uses a separate builder in `app/ui/ui.go`. That builder iterates over the messages it receives, apart from its empty-message guard; the handler excerpt loads and writes through the store. Its MCP-registration branch is skipped for the checked attachment-bearing user turn. These are source observations requiring further tests, not claims that the desired memory/context features are complete or that a performance cause has been measured.

The live TECHNE router and relevant arrival, reconnaissance, documentation, Git and handoff sources were read. `SOURCE_MAP.md` records exact read ranges and raw hashes, including partial reads. Existing curated-context designs, frontend settings, store internals, acceptance-test bodies and installed-app identity still need investigation.

## What did not happen

No runtime source or tests were changed. No command job was accepted, so no build, test, benchmark, fault control, application launch or model invocation ran. No blueprint or pack was promoted, no independent colleague was launched, and no completion or release gate passed. Eko's own role changes are not independent review.

No branch or worktree was created, switched or removed. No commit, push, merge, fetch or installation occurred. Lineage registration, predecessor observation and LETHE document-pointer recording remain outstanding; none is represented as completed.

## Files and custody

New programme records are `STATE.json`, `REQUIREMENTS.md`, `DECISIONS.md`, `SOURCE_MAP.md`, `BLOCKERS.md`, this `HANDOFF.md`, and the two evidence JSON files. They are local, uncommitted and unpushed. File-write receipts prove creation, not remote backup or runtime correctness.

The initial checkout was `main`, with no staged, unstaged or conflicted tracked files and one untracked commissioning document. A subsequent pre-write check found a second untracked document, `docs/OLLAMA_CROSS_MODEL_COLLEAGUES_AND_CAPABILITY_ROUTING_HANDOFF.md`. Eko did not create, read or change it. Preserve both original documents and establish checkout ownership before any Git mutation. Locally recorded `origin/main` equality was not a fresh remote check.

No task-owned branch needs cleanup. Do not delete the three pre-existing local branches or alter the existing worktree under the guise of closing this task. `STATE.json` carries the latest recorded containment check; if that field is null, the check has not yet been recorded.

## Exact continuation boundary

1. Resolve command capacity through authorised, preservation-first Bridge maintenance. Recheck with a no-op command and verify the returned job's real terminal exit status; do not infer execution from a request acknowledgement.
2. Recheck the task document's raw hash and Git state. Read the newly arrived handoff and identify any concurrent owner. Resume OI-01 rather than jumping to runtime implementation.
3. Complete the relevant source/test map: `cmd/tui/chat/input.go` system-prompt composition, remaining `agent/session.go` run lifecycle, terminal config/history/compaction, frontend settings, `app/store` loaders and migrations, and the existing context/MCP designs. Resolve lineage and LETHE bookkeeping through their actual normal paths.
4. Discover and inspect the canonical build, non-interactive test and system-acceptance commands from `docs/development.md`, manifests and CI. The inspected frontend manifest declares build, lint, formatting checks and Vitest; no installed version or passing result was established. Capture the complete host baseline without changing or replacing the running application or using live user data for tests.
5. Use the commissioned design authority to define the next bounded slice, then obtain actual independent review and the live TECHNE readiness gates before code. G1 couples OI-03/OI-04; do not claim the editor complete without saved-value restart and actual request-delivery proof. OI-28 and OI-29 retain their own independent dependency paths; do not wait on OI-27 merely for numerical order.
6. Preserve the clean/broken/clean fault controls, full execution output, candidate identity and independent review required by TECHNE. No self-authored report substitutes for those results.

Next prompt title: **Continue OI-01 after restoring Bridge execution**.
Purpose: finish current-state and baseline reconnaissance, reconcile existing work, and establish the first independently reviewed implementation boundary.

PACK_PREREQUISITE_MISSING
