# Ollama integration — continuation 04

Date: 20 September 2026, Australia/Brisbane. Owner: Eko (GPT-6 Astra Pro).
Status: source-reviewed maintenance candidate and permanent tests committed locally; compilation, runtime acceptance, independent executable review and remote backup are not complete.

## Actual result

Candidate commit: `927fcb5dd706ffd741612a4ab124646aa22a60cb`.
Branch: `eko/chat-read-integrity-20260920`.
Worktree: `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`.
Parent: `953de98d408a9697fa20a48c23973b9dc8eee921`.
The candidate worktree was verified clean after the commit. The ordinary `/Users/krypto/GitHub/ollama` checkout remains separate on its previous main revision; no candidate was installed or merged.

The commit contains exactly four files: two changed production files and two added test files. It is explicitly labelled `wip:` and its message states that execution is blocked and integration is not accepted. The source implementation is no longer merely proposed; its runtime correctness is still unverified.

## Production change

`app/store/database.go::getChatWithOptions` adds the existing `app/types/not` import and emits the wrapped `not.Found` sentinel only when the chat-header QueryRow/Scan returns `sql.ErrNoRows`. All other existing error wrappers remain unchanged.

`app/store/store.go::ChatWithOptions` retains its ensureDB guard and directly returns `s.db.getChatWithOptions(id, loadAttachmentData)`. It no longer rewrites every database failure into `not.Found`. The remaining use of not.Found in the rename path means the Store's existing import remains necessary.

The existing `app/ui/ui.go::chat` is unchanged. Its error branch already aborts any non-not.Found error before SetChat and before inference. The expected effect of the candidate is therefore refusal to overwrite an existing conversation after a read error, while preserving genuinely new-chat behaviour. This is a source-derived expectation, not a passing runtime result.

No database schema, public signature, feature flag, authentication policy, retry, concurrency protocol, installed binary or live conversation was changed. GET-empty-success, rename wording, and silently ignored JSON fields remain separately scoped issues, not claimed repaired.

## Permanent tests added

`app/store/chat_read_errors_test.go` contains five top-level tests:

- TestChatReadMissingHeaderIsNotFound: the missing-header sentinel in both attachment-loading modes.
- TestChatReadValidEmptyAndPopulated: a valid empty chat is not absence; populated chats preserve exact body/role/title/ID values.
- TestChatReadRejectsMalformedMessage: a real header and independently observed invalid message boolean reach Scan, produce an error, and are not classified as absence.
- TestChatReadInitialisationFailureIsNotAbsence: an actual filesystem initialisation failure preserves its native path-error cause.
- TestChatReadPreservesDatabaseErrorCause: a real closed-database error preserves errors.Is identity instead of becoming not.Found.

`app/ui/chat_read_errors_test.go` contains TestChatReadIntegrityAcrossProcesses, with four process-separated phases: seed, fault, recover and positive. It uses the existing authenticated Server fixture, real HTTP router and SQLite store; inference alone is explicitly simulated. It adds a no-cookie refusal, exact all-record preservation, live service-side request-counter checks, exact four-body recovery assertions, proof that original history reaches the simulated inference request, and a genuinely absent-chat control with the requested ID.

These are normal package test files, not optional diagnostic overlays. Their discovery, compilation and execution have NOT been demonstrated in this continuation. The historical diagnostics they extend did compile and expose the old bug in continuation 03; that evidence cannot be promoted into a result for these changed files.

## Independent pre-code review

The selected `deepseek-v4.1-flash:cloud` was invoked twice through the existing TECHNE wrapper, with expected manifest digest `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8` and expected served model `deepseek-v4.1-flash`. The manifest is not an immutable weights identity.

Round 1 prompt: PRECODE_REVIEW_04_PROMPT.md, raw SHA256 `63a26b5a6a6d37f3abd1f6105ca04aa0e9bd2e900b79f3e1907f6cdb3fb1b43c`.
Response: `/Users/krypto/GitHub/TECHNE/Tools/ollama/_runs/2026-09-20T11-16-06Z_eko-chat-read-precode-cont04_ly5419p0/response.txt`, raw SHA256 `6f9e87f9d3f46a6c24c03231058e960d766692a302275df6ba5834db962a95ae`. Result REVISE. Eight findings were checked individually. Some were useful precision requests, while others misread the supplied algorithm or asked for an already specified Store-rewrapping fault control.

Round 2 prompt: PRECODE_REVIEW_04_R2_PROMPT.md, raw SHA256 `6505333b5d3d8c786496a7510a8c07cb5e111a8fc7bb47fcd7c69a869c3e564e`. It preserves each finding's disposition, supplies exact resulting production methods and test assertions, and corrects the hypothetical missing-child ErrNoRows claim against the real Query/Next/Scan/Err source.
Response: `/Users/krypto/GitHub/TECHNE/Tools/ollama/_runs/2026-09-20T11-19-41Z_eko-chat-read-precode-cont04-r2_74j29e2d/response.txt`, raw SHA256 `dc6d487e9eafc7dbfcb2bb1d24559abc127fbd8e3dfdc4f2beaf0b51c127b800`. Result APPROVE, explicitly design-only. Both wrapper executions returned 0. This closes the pre-code technical review, not code acceptance, formal blueprint promotion or T-10 independent execution.

One independent colleague/model was used, across two source-only calls. No supplementary Codex invocation was repeated.

## Blocked execution — not test failures

The following calls were refused by the platform with “This tool call was blocked by OpenAI because we couldn't determine the safety status of the request.” They have no claimed local execution or passing result:

1. `run_command` request `eko-ollama-cont04-20260920-preflight`: a combined Python/shell preflight. Source inspection continued through the bounded read/Git tools. The blocked shell payload was not rerun through another route.
2. `send_command_input` ID `eko-cont04-red-tests-run`: formatting the two new tests and executing the permanent red baseline runner. Neither gofmt nor that Go-test invocation is claimed to have run. No candidate build, deliberate-break control, regression suite or runner self-test ran later as a substitute.
3. `send_command_input` ID `eko-cont04-backup-candidate-branch`: the candidate branch push and subsequent remote-tip query. The push was not queued/executed, and no remote backup is claimed. The destination had been read as the existing `kryptosmatrix/ollama` origin, not guessed.

These are platform execution restrictions, not a request for further operator approval and not evidence that the code failed. No alternative harness, encoding or route was used to repeat the refused build/test or push actions. Runtime acceptance remains blocked.

The source and tests were created using permitted bounded writes and an authorised test-file-copy operation in the isolated worktree. A separate Git checkpoint command was accepted and completed with exit 0. Commit success is not compilation or runtime proof.

## Candidate identities

| File | Raw SHA256 |
|---|---|
| app/store/database.go | 6905b4e6146f76e0aed66f97e17a39245d27a0182f0051d06642389de042b551 |
| app/store/store.go | 71097b8bc6664c0c9eb87bedfab5c0b79acc2768df20e0dd0319af2fbd5c0862 |
| app/store/chat_read_errors_test.go | 78adbad3f024feb30e912722026aa9bb8f6ea1f96b08fc241e5ab0c121b57c99 |
| app/ui/chat_read_errors_test.go | 299356bec304683503fe4410f19f2f0496307ff64848ebaf0f13fd854421440a |

Production diff was retrieved with bounded git_diff request `3350d95a6e7345af9553ee5f7f951f3a`; no omitted files or truncation. The clean committed worktree and full commit ID were verified by repo_overview request `d89d0e038e694c7ba128c2e35ef0d68f`. Commit summary: four files, 516 inserted lines and seven deleted lines; most additions are tests, not production code.

A retained proof runner exists at `evidence/2026-09-20_cont04/run_proof.py`, raw SHA256 `8a1f06496c6e25e43fe7a0e46cc00c971f1d0df3c965ced2210ad90c31b67e0f`. It is NOT executed or independently validated. It is designed to create a fresh home, retain complete logs/child exits/source hashes, require the ten named Go parent/subtest records, reject skipped/missing/uncompleted tests and distinguish an expected behavioural failure from a build failure. Its own fail-closed controls remain owed.

## Preservation and operational custody

To restore one command slot, the already-backed-up completed continuation-02 baseline job `39cbb17656ee455cb92d43a50e580bbc` was retired after its stored command/metadata/output hashes and live terminal status were rechecked. Its complete records remain under continuation-02's bridge-commands directory.

The first new review command `2199fec77ef5473a8a551cb76d25978f` was then retired after its complete invocation, status and output were preserved and read back in COMMAND_CUSTODY_04_REVIEW1.json (raw SHA256 `fcf29eb34e31d462a5789bd26016b8946f3867de7d78feac0170ac2bc86d01c8`). The generated review artefacts remain in TECHNE's wrapper archive. No other colleague's job, active job, tombstone, global configuration or service limit was removed.

Subsequent permitted operations used one explicit shell session, Bridge job `4638c679a84e4c08881a66f37af853b0`, with separately identified input acknowledgements and child-exit sentinels. The shell's eventual exit status is session-lifecycle evidence only, not test evidence. Its command/input/output records are retained in the Bridge archive.

## Next boundary and closeout

Disposition: PARKED — Eko owns the local candidate branch and worktree. The local commit is recoverable through Git, but has no verified remote copy. Retain both branch and worktree while proof and remote backup are outstanding; no cleanup of unique data is justified.

Do not repeat the authorisation question or the completed design review. Do not count the original diagnostic red results as candidate passes. Once execution is permitted, verify candidate identity, run the new permanent tests with formatting and source/runner checks, execute the original-bug and consumer-bypass plus constant/error controls in isolated copies/compilation overlays, re-run the clean candidate, run the full declared affected suite safely, obtain actual independent executable review and preserve/verify the remote branch. Land on main only after the applicable gates pass.

The six frozen requirements remain unchanged; all 31 wider programme items remain unaccepted. Existing frontend lint failures and the root Darwin Keychain-isolation problem are still outstanding earlier findings, not newly rerun results. No installed-app, live-provider or complete-application proof is claimed.
