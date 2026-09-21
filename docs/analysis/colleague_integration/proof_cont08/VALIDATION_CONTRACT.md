# Continuation 08 — frontend verification and test-environment repair

21 September 2026, Brisbane. Eko. Ash renewed full implementation authority in the current conversation, subject to recoverability and explanation before necessary changes. This is continuation of the existing frontend maintenance boundary, not a new feature blueprint or a relaxation of acceptance.

Baseline: d559da25a4db89fa9369d8be4106100803e82a58, owned worktree /Users/krypto/GitHub/ollama-eko-chat-read-integrity. Main remains 953de98d408a9697fa20a48c23973b9dc8eee921. Prior frontend source changes are committed at acd8bb3de93ea01d43dd8783bbeff036113f9ab9. The original runtime/frontend test attempt in continuation 07 was not executed; its record remains intact. A fresh, plainly named request through the same Bridge input tool in continuation 08 was admitted. No security setting or credential permission was changed to obtain execution.

## Findings and current repair boundary

The first actual focused npm run completed with exit 1: six parser tests passed, but the renderer module failed to load before its five tests were collected. The exact failure was Node rejecting katex/dist/katex.min.css. No test was skipped or weakened. npm stdout/stderr and its real exit status are retained as frontend-focused.*.

Repair the normal Vitest configuration so Vite processes Streamdown and its CSS import rather than delegating that package directly to Node. Set test.server.deps.inline to ["streamdown"]. Do not replace Streamdown, Shiki, remark or the production renderer with mocks. This proves rendered markup under the Node test environment; it does not prove browser layout, CSS appearance or a packaged app launch.

Primary reference: Vitest v3 config, https://v3.vitest.dev/config/#server-deps-inline. The v3 environment guide contains an inconsistent reference to external instead of inline; the explicit config definition and installed behaviour are the verification targets, not that guide's typo.

## Test expectation correction, after the first collected run

With Streamdown inlined, all eleven cases were collected: ten passed and one failed. The failing assertion required `<strong>important</strong>`. Inspection of the installed Streamdown implementation shows its MarkdownStrong component emits a `span` with `data-streamdown="strong"` and the `font-semibold` class, preserving the child text. The original test was incorrect about the existing library's HTML contract. Correct it to require that exact labelled-span role around the input word; allow attributes on `em`. This changes no production rendering. The failed run remains in frontend-focused-r2.*; the amended positive run and full suite are retained under runs/author-clean and runs/author-full. No-output fault detection is required to protect the input-dependent formatting assertion.

## Required proof

- All eleven previously committed parser/real-renderer tests must be discovered and finish. Retain every attempt and investigate real failures; no pass on zero assertions or failed file collection.
- Run the existing complete frontend suite and actual TypeScript/build/lint commands. Record remaining baseline lint and other warnings as failures/debt, not a passing whole-programme gate.
- Demonstrate unchanged assertions reject isolated faults: a no-output renderer, a disconnected citation plugin, and a wrong upstream citation-page selection. Use a disposable source copy or test-only module transformation; never modify the installed app or live user data. Compilation or setup failure is not a successful behavioural control.
- A clean rerun follows fault controls. Independent DeepSeek review must inspect the instruments, request fresh executions, and distinguish fault-detection from clean success.
- Any discovered production defect expands the maintenance contract explicitly before its fix. Preserve unrelated behaviour and the existing history-preservation repair.

## Exclusions

No Keychain access, no retry of the refused credential fixture edit, no application installation/launch, no model training, no global toolchain modification, no main merge while required gates are incomplete. Generated-type and remaining handwritten lint fixes are separate follow-on scopes unless explicitly added with inspected source and appropriate proof. Keep the exact user work list and all 31 items unchanged.
