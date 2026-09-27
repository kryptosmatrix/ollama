# Ollama integration — continuation 06

Date: 20 September 2026, Australia/Brisbane. Owner: Eko.
Disposition: PARKED for integration; source-visibility repair independently verified, complete default app run finished with one Keychain-authorisation failure.

## Delivered change

Commit `85c69f45e7c0fa731dcb18058c24c2b0a6d03d12`, branch `eko/chat-read-integrity-20260920`, changes only `app/.gitignore` in `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`.

The old `*cover*` pattern hid source files containing `Discovery`, `discover` or `recover`. Six affected files had already been recovered and tracked in continuation 05. This continuation fixes the rule itself: `coverage/`, `*.coverprofile`, `coverage*.out`, `coverage*.html`, `cover.out`, and `cover.html` replace that broad substring pattern. All other exclusions are unchanged.

Nine real Git checks cover three future source paths that must remain visible, four coverage outputs that must remain ignored, and the existing `.env` and `.crt` exclusions. Three negative controls against the unchanged main checkout show the old rule hiding each source path. The paths are inputs to `git check-ignore --no-index -v`; no dummy source files were created. This is a bounded set of examples plus rule inspection, not an exhaustive filename proof.

Rules SHA-256: `384f341ecedfd8242d7249573dc4215009925a915219148bb4d219d009cda908`.
Evidence: `evidence/2026-09-20_cont06/ignore-rules-proof.json`, `ignore-negative-controls.json`, and `ignore-commit.json`.

## Independent verification

One independent colleague configuration was used: `deepseek-v4.1-flash:cloud`, observed served name `deepseek-v4.1-flash`. The expected local manifest remains `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8`; this is an observed cloud-tag identity, not immutable weights provenance.

DeepSeek inspected the rules and bounded controller, requested fresh Git execution of all nine candidate cases and all three old-rule controls, and reviewed the recorded app/lint failures. It returned `GITIGNORE_REVIEW=PASS` for the configuration correction and explicitly retained `INTEGRATION_GATE=BLOCKED`. This review did not rerun the complete app suite or approve the application release. The controller exposed only named evidence reads and those fixed Git checks, not arbitrary shell, edits or Keychain access.

The review's incidental sentence describing `coverage*.out` and `coverage*.html` as broader than the old `*cover*` pattern is incorrect: those patterns are subsets of the former substring match. The raw report is preserved unchanged; its executed checks and scoped verdict do not rely on that sentence. No additional review pass or broader approval is invented.

Evidence: `evidence/2026-09-20_cont06/ignore-review/`, `review-wrapper-result.json`, and `review_ignore.py`. Report SHA-256: `400b355e6d830af33f9ccdebfa890369ea5c7ec1e1b4acadeb9e3ffddc83a803`. Controller SHA-256: `ea4fe9e43d26364d7a7bba52c40093d98bd5bb67ad6af0c040fd1c9d8fca7927`.

## Complete default app test run — actual failure retained

Executed `/opt/homebrew/bin/go test -count=1 -json -timeout=180s ./app/...` from the owned worktree. The source revision was `13139f8d36ec5f447c67ad634c8ad65b713a49db` plus the then-uncommitted ignore-rule correction, subsequently committed as `85c69f45`. No runtime or test source changed during the run.

The process completed with exit 1 after approximately 140.6 seconds: 300 passing named test/subtest records, one failing record, zero skipped tests and zero unfinished records. These include parent/subtest records and are not a leaf-only count. Seventeen package terminals reconciled: nine passing, one failing, and seven packages with no test files. The seven package-level `skip` events are not skipped test cases.

The sole failing test was `app/tts::TestKeychainRoundTripUsesNamedServiceNotMCP`. At `secrets_darwin_test.go:27`, macOS returned: `add the TTS keychain item: The authorization was canceled by the user.` That is the operating system's reported reason; it does not establish who dismissed a prompt. The test was not retried, suppressed, replaced with a fake, or converted to a pass.

The existing test uses a service name containing the current nanosecond timestamp and test name, with a synthetic value and no fallback. It does not target the production `Ollama TTS` or `Ollama MCP` services. It invokes its existing cleanup; no separate post-run Keychain inventory was performed, so this report makes no claim of independently verified cleanup. The run used isolated HOME, USERPROFILE, LOCALAPPDATA, XDG and explicit MCP/TTS file paths. HOME alone is not claimed to isolate Keychain access. Default updater tests use temporary app bundles; the separately tagged live updater test was not enabled. No installed application or real conversation database was targeted.

The app's command/bootstrap-manager, server, updater, storage, tools and UI packages passed. This is broader package execution, not packaged-app launch or installed-candidate acceptance. The full root `go test ./...` was not run.

Full output SHA-256: `97bdc2fa82d522a89b8d15b53f9d9c8d462b1f9708dda615a24e8ecdd4a68f3a`.
Evidence: `evidence/2026-09-20_cont06/app-suite.stdout.jsonl`, `app-suite.stderr`, `app-suite-result.json`.

## Newly observed native warnings

The wider app compilation surfaced three deprecated literal-operator spacing warnings at `app/webview/webview.h:1589,1592,1595`, and a duplicate `-lobjc` linker warning for `app/cmd/app.test`. Those source files were not changed by this repair. Their presence is recorded in `native-warnings.json`; none was suppressed or repaired in this continuation. A warning-free native app build is not claimed.

## Lint comparison — identical debt, not a pass

Both the original main frontend and candidate frontend freshly ran `npm run lint -- --format json`. Each returned exit 1 with 137 errors and 11 warnings. Exact relative-file/line/column/rule/message records and the raw bytes of every file carrying a finding were compared; both comparisons matched. Equal totals alone were not used as the no-new-lint evidence.

All 137 errors are `@typescript-eslint/no-explicit-any`: 104 in the generated `codegen/gotypes.gen.ts` and 33 in handwritten frontend code. Nine warnings concern hook dependencies and two concern component-only refresh exports. The generated file is produced by the existing `tscriptify` directive in `app/ui/ui.go`; manually patching generated output or disabling the lint rule would not be a durable correction. No lint source, generator, ESLint configuration or suppression changed. The exact comparison is in `lint-delta.json`, with both full outputs retained alongside it.

These are now characterised baseline failures. The comparison neither repairs them nor discharges the declared failing integration gate.

## Root Keychain test isolation remains blocked

Source inspection found that `mcp/tokenstore_darwin_test.go:25-38` reuses a service derived only from the test name and clears it before and after each run. Concurrent executions of the same test can therefore share a service. The cleanup at lines 50-58 treats every failed deletion command as absence, and lines 353-355 of `TestAnExplicitTokensPathIsHonoured` read a production Keychain entry while assuming it does not exist.

A bounded edit proposed per-invocation UUID services, independent absence checking, fail-closed cleanup and a file-store reopen assertion instead of the production credential read. The Bridge rejected the entire edit with `SECRET_CONTENT_DENIED`, request `d9b6769e0c6d4bbd9327a91f7161d401`, `application_state=not_applied`. No part was applied, and the rejected change was not retried through shell, another agent or a changed access policy.

The unchanged test-file SHA-256 was rechecked as `e64778cde71f8a13b35f7d2db1a12b4c95b475275392358f32cd7ee2da73a5dc`. Root MCP Keychain tests were not run. This is a tooling/isolation limitation, not missing standing operator approval.

## Custody and next boundary

The original main checkout remains at `953de98d408a9697fa20a48c23973b9dc8eee921` with no tracked source or staged changes. The owned candidate is locally committed and clean at `85c69f45` before this report is copied into its documentation checkpoint. The closeout receipt records the later documentation commit and fresh remote verification; this report does not guess that hash before the operation.

One completed Eko-owned command record, `0356010d95a44e55832f4fa2cefa7f2e`, was retired only after its complete archived command and output identities and live terminal status were verified. The retained bytes remain in continuation 02's `bridge-commands` archive, and the Bridge tombstone prevents replay. No service limit, other colleague's job or unrelated file was removed. The current shell session is `b261d7be2bf44a19a2797fce7a232177`; its eventual termination is recorded separately and is not a test verdict.

All 31 programme items remain unaccepted. The prior history-preservation mechanism PASS remains valid for its recorded source/test boundary; it is not revoked by an unrelated environment failure, nor promoted into release approval. No merge, installation or cleanup of the retained worktree occurred.

Next work: correct the generator/handwritten frontend lint debt in separately reviewed bounded changes; resolve the root credential-test edit through a permitted access path without bypassing its refusal; arrange a supervised real-Keychain test run only when macOS authorisation is available; address the native warnings; then obtain the remaining combined-candidate and release proof. Do not repeat the completed design review or request Ash's engineering pre-approval again. Do not blindly rerun the denied credential edit or cancelled OS interaction.
