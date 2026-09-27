# Eko — Ollama integration checkpoint

20 September 2026, Australia/Brisbane. Active item: OI-01 source/baseline reconnaissance with a concrete OI-02 prerequisite draft. No unattended work is running.

## Resume here

Read `CONTINUATION_02_REPORT.md`, `JUDGE_RECONCILIATION_02.md`, `STATE.json` and `BLOCKERS.md`. The full commission is `../../OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md`; all 31 items remain unaccepted. The original two untracked commissioning/handoff documents remain unchanged.

## What is now proven

The core Go binary builds. The declared affected Go packages passed with 593 named test/subtest pass records and one Windows-only skip; frontend Vitest passed all 143 tests; TypeScript passed. Frontend ESLint is a pre-existing red baseline: 137 errors and 11 warnings. This was not the full root Go suite, the complete app suite, a native/CMake rebuild or a packaged-app build.

A real temporary SQLite + development-mode HTTP probe reproduced a history-loss defect. A malformed persisted boolean causes Store.ChatWithOptions to report not.Found; the normal continuation handler creates a replacement chat, and the two original messages are deleted. Healthy/missing controls behaved normally. No real conversation was used. Authentication, all-table preservation, reopen after repair and post-repair mutation controls remain unproven.

`docs/_design/CHAT_READ_ERROR_INTEGRITY.md` proposes classifying absence at the chat-header query only and otherwise preserving database errors. Current state: SPEC-DRAFT, Tier 2, no build permission. No production code or default-run tests have changed.

## Independent reviewer

Ash selected `deepseek-v4.1-flash:cloud`; use that exact primary judge unless a later decision changes it. Two bounded source-review calls were made, one truncated and one complete. The complete response came from `deepseek-v4.1-flash`, matched the pre/post manifest pin, corroborated the diagnosis/repair and returned BLUEPRINT_JUDGE=FAIL. It had no repository or execution tools. Missing dependencies, incomplete Tier-2 coverage and uncalibrated severity mean this is advisory source review, not a qualified promotion round or executable verification. Do not transmit the author's reconciliation or prior judge arguments as context for the next fresh panel.

## Exact next boundary

Finish the Tier-2 continuation-integrity contract: unchanged six requirements, exact criterion/source/guard closure, source-resolved sentinel, normal authenticated HTTP test, independently observed saved rows, true-missing and healthy controls, reopen functional use, and compiled original-bug/consumer-bypass falsifiers. Supply the commission and required dependency corpus to the full independent panel before promotion. Then implement only the two-file getter/error-classification repair and default-run tests, self-audit, execute the declared proof set, and obtain real independent code/evidence review. Do not silently expand into GET/rename repairs or weaken the red baseline.

Continue the original OI-01 reconnaissance in parallel only on independent read surfaces: existing curated-context designs, terminal history/persistence owners, settings, canonical full-suite isolation and installed-app mapping. The installed resource binary reports eb8c93da6ab20b980449226029a9bf5cf40a6306; checkout HEAD remains 953de98d408a9697fa20a48c23973b9dc8eee921. Nothing was installed.

## Custody

All changes are local and uncommitted. No branch, worktree, commit, push or service/configuration change was made. Raw outputs and the ineffective first fault are preserved in `evidence/2026-09-20_cont02/`; reviewed blueprint/checkpoint originals are under its `checkpoint_before_cont02_closeout/`. Ten completed Bridge jobs have exact local command/output copies. Three task binaries moved unchanged to Git-ignored `build/eko-colleague-integration-20260920/`, with hashes and both paths in `binary-custody.json`.

The archive-capacity blocker from the prior turn is resolved. This continuation forgot no jobs. Check current remaining capacity before starting a long series; do not recreate the full-archive condition through needless probes. Lineage/predecessor and LETHE pointer bookkeeping remain outstanding, not claimed complete.

Next prompt title: **Continue the chat-continuation integrity prerequisite**.
