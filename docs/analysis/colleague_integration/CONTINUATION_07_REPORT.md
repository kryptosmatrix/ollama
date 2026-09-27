# Continuation 07 — native diagnostic repair and frontend type maintenance

Date: 21 September 2026, Australia/Brisbane. Owner: Eko. Worktree: `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`; branch: `eko/chat-read-integrity-20260920`.

Baseline: `281dcaa06c7cd8b299c6045a248dd6b6ac85f954`. Native source commit: `abeb2e5e29d838f7252b271f25c3b40d1bdff41c`. Frontend candidate commit: `acd8bb3de93ea01d43dd8783bbeff036113f9ab9`. All actual source bytes are recorded in `evidence/2026-09-21_cont07/source-commits.json`. A later documentation-only checkpoint does not add tested runtime changes.

## Result and boundary

The three native webview deprecation warnings have a tested, independently executed repair. Frontend lint is reduced from 137 errors to 122; the existing 11 warnings remain. The four frontend production files pass the actual project TypeScript check and emit byte-identical JavaScript under the recorded compiler configuration. The frontend source review is approved, but frontend runtime tests and independent executable acceptance remain blocked. No programme item, main integration, installed application or release is accepted by this report.

The earlier history-preservation repair and its bounded executable-review result remain intact. This continuation did not change the two store functions, their tests, the existing continuation handler or any real conversation data.

## Native declaration repair

Only three spaces were removed from `app/webview/webview.h:1589,1592,1595`, between `operator""` and the `_cls`, `_sel`, `_str` suffixes. Function bodies, signatures and consumers are unchanged. Raw header SHA-256 changed from `8c8b6ac9adc1d51ff579bf692603f34ce61cf8a332064d825f315dc2d6bd31e6` to `60b93d50eefa841c61a37e2c9425107dc720fbb2b112455129776547c2e8eef0`.

The actual `app/webview/webview.cc` translation unit was compiled with Apple Clang 21.0.0, `-std=c++11 -DWEBVIEW_STATIC -DWEBVIEW_COCOA`. Promoting the specific deprecation to an error produced exactly the three expected errors on the original code. The corrected code passed with empty output. A preserved original-source control failed for the same three diagnostics; the clean candidate passed afterwards. This is a diagnostic control, not a behavioural mutation of a new runtime feature.

Ordinary `-O2 -c` builds of the original and corrected translation unit produced identical object bytes: SHA-256 `b6551cda945646f3d49b6dba0336de43d2d385fa3c7b9692978b89896122b09f`. Both objects and complete compiler outputs remain in the evidence directory. This is a measured compatibility result for this toolchain and configuration, not an unrestricted proof about every compiler.

The real `go test -count=1 -json -timeout=90s ./app/cmd/app` command passed all four discovered tests, with zero skipped or unfinished tests. Temporary home/config/token-file paths isolated the fixture. No Keychain test or application GUI was launched. The remaining linker warning is `ignoring duplicate libraries: '-lobjc'`. The installed Go 1.26.5 build source at `src/cmd/go/internal/work/exec.go:2919–2921` adds that flag for Objective-C input. No global toolchain edit or linker-warning suppression was made.

DeepSeek `deepseek-v4.1-flash:cloud` inspected the source, contract and bounded controller, requested fresh compiler/control/object-comparison/app-test execution, and returned `NATIVE_DECLARATION_REVIEW=PASS`. Its recorded tool executions, raw outputs and result are in `independent-native-review/`. Its original-object run used a separate temporary working directory with the same relative source path and compiler flags; the report's abbreviation that only output paths differed is corrected here. Author and independent checks both measured object equality. This is not a full-app or release verdict.

## Frontend type maintenance

The changes are limited to four production files:

| File | Change | Explicit `any` violations removed |
|---|---|---:|
| `app/ui/app/src/components/StreamingMarkdownContent.tsx` | Declare the existing page-stack shape; type React children; infer highlighter token types; remove theme casts | 8 |
| `app/ui/app/src/components/MessageList.tsx` | Derive the browser-result prop from the existing Message component and use the existing ToolCall type | 5 |
| `app/ui/app/src/utils/remarkCitationParser.ts` | Register the existing custom-citation shape in mdast and use discriminant narrowing instead of casts | 2 |
| `app/ui/app/src/lib/highlighter.ts` | Declare the custom-theme instance as Shiki's HighlighterCore | 0 |

The first TypeScript check failed on the already registered custom `one-dark` theme after its old `any` cast was removed. That failure is retained in `typescript-check.stdout`. The fix is at the type owner, not a different theme name or a new cast. A virtual compiler-host experiment first checked the proposed annotation; the actual final project `tsc --build --pretty false` then exited zero with empty output. The explicit owner-scope amendment is retained beside the original maintenance contract.

Identical compiler options, file names and source baselines were used to compare JavaScript emission before and after all four edits. All four outputs are byte-identical; hashes and options are in `typescript-emission-final.stdout`. No runtime statement, theme registration, parser payload, rendering branch, generated-type file or ESLint rule was changed. Two obsolete parser `@ts-expect-error` comments were removed after the node acquired its real type; no suppression was added.

Full ESLint still exits one: 122 errors and 11 warnings. All four changed production files and both new test files have zero lint findings. The remaining errors comprise 104 in generated types and 18 elsewhere in handwritten frontend code. The count reduction is not whole-frontend acceptance.

## Tests written but not executed

Six parser cases were added in the normal runner at `src/utils/remarkCitationParser.test.ts`: exact range payload and surrounding text, generic citation payload, adjacent duplicate removal, separated duplicates, code/malformed delimiter preservation, and empty documents.

Five actual-renderer cases were added at `src/components/StreamingMarkdownContent.integration.test.tsx`: contrasting Markdown, real syntax highlighting, correct citation-page selection, absent-page handling, and no model-provided embedded image/iframe resources. Unlike the existing Streamdown-mocked test, these are written to use real Streamdown, Shiki and React rendering. They have not executed and no claim is made that they compile under the test runner or pass.

The platform blocked input `eko-cont07-frontend-baseline` before execution with: `This tool call was blocked by OpenAI because we couldn't determine the safety status of the request.` The blocked payload would have saved baseline copies and invoked `npm test -- --run` for the two new test files. Neither part ran. No alternate execution route, reviewer harness or changed encoding was used to replay it. Later permitted operations were static compilation, emission comparison and lint, not runtime substitutes.

## Independent source review and contested findings

DeepSeek's source-only frontend review initially requested changes. Several objections were disproven: mdast unions registry values rather than requiring map keys to equal discriminants; required data is structurally valid when producers always supply it; and the review's own `style="color:#..."` example contains `color:`. A static compiler-host counterexample showed that removing only the registry declaration restores exact custom-citation assignability and narrowing errors, whereas the unchanged candidate has no diagnostics. This ran the compiler, not the blocked frontend runtime tests.

The reviewer explicitly withdrew those objections and returned `SOURCE_REVIEW=APPROVE`. Both rounds, the counterexample, and their reconciliation remain intact. The pre-existing parser-string/renderer-number annotation mismatch remains disclosed; no runtime payload conversion was smuggled into this type-only maintenance. Source approval is not independent executable acceptance.

## Remaining blockers and next work

The frontend candidate remains `wip` until the two new files, the complete existing frontend suite, relevant real-renderer fault controls and independent executable review can run through a permitted route. Do not bypass the platform block or substitute source review. Preserve all failed attempts.

Remaining lint work is 104 generated-type errors, 18 handwritten errors and 11 warnings. Repair the generation source rather than hand-patch generated output, and handle the handwritten changes in bounded reviewed slices. The Go-generated duplicate-library warning remains visible. The TTS Keychain authorisation failure from continuation 06 has not been retried; root MCP Keychain isolation work and its earlier Bridge-denied edit have not been rerouted. Whole-root, packaged-app and combined integration gates remain open.

Original main source, installed app, credentials and real conversations are unchanged. All 31 commissioned work items remain unaccepted; these dependency repairs do not close their feature contracts. Do not reopen the completed history-preservation authorisation, design or bounded mutation review.

## Custody and evidence

Full evidence lives in `docs/analysis/colleague_integration/evidence/2026-09-21_cont07/` in the original checkout. Curated proof is copied into the owned branch at `docs/analysis/colleague_integration/proof_cont07/`, with a SHA-256 manifest. `source-backup.json` records the actual source push/query result; the final `git-closeout.json` records the documentation tip and remote verification. A successful preservation push is not integration.

The command archive was full on arrival. One completed Eko-owned continuation-02 job, `76f6c3466291456db6a333330d4be1b4`, was retired after checking its preserved command record, file metadata and existing byte-exact custody receipt. Its full command/stream archive remains at the continuation-02 bridge-commands path and its replay tombstone remains in the Bridge. The binary-filtered stdout was not read through an alternate tool; this continuation does not claim it freshly re-hashed those bytes. No other colleague's records or service settings were changed.

Disposition: PARKED. Retain the remote-backed worktree under Eko's ownership. Verify the final source identities, remote tip and explicit pending execution before resuming. Close the current command session before reporting completion of this continuation.
