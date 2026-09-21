# Continuation 08 — real frontend verification completed

21 September 2026, Australia/Brisbane. Owner: Eko (GPT-6 Astra Pro). Primary independent reviewer: the requested `deepseek-v4.1-flash:cloud`, served as `deepseek-v4.1-flash`.

## Outcome and boundaries

The previously unexecuted frontend maintenance tests now run through the normal Vitest entry point. All eleven new parser/real-renderer tests pass. The complete frontend suite passes **154 tests**, with no failed, pending, skipped or todo assertions. The production frontend build succeeds, and TypeScript checks pass. Three compiled broken implementations fail the intended behavioural assertions; a clean rerun passes afterwards. DeepSeek independently requested six fresh execution batches and returned `FRONTEND_RUNTIME_REVIEW=PASS` for this bounded Node-rendering/type-regression scope.

This is not a whole-programme, browser-layout, packaged-app or release pass. Full lint still exits 1 with **122 errors and 11 warnings**. Dependency/build warnings remain visible. No new colleague feature was enabled, no Keychain access was attempted, and nothing was merged into main or installed. The original 31-item work list is unchanged and none of those programme items is newly declared accepted.

## Candidate identity and actual diff

Parent checkpoint: `d559da25a4db89fa9369d8be4106100803e82a58`.

New source/test-configuration commit: **`8977b03cc4224a8226f1fac93647246b8edfa16a`**. Its bytes were compared to the independently reviewed working-tree snapshot before and after commit. This source commit was pushed and freshly verified on `origin/eko/chat-read-integrity-20260920`. A later documentation-only checkpoint is recorded in `evidence/2026-09-21_cont08/git-closeout.json`.

Owned worktree: `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`.

Only two repository files changed in the source commit (9 insertions, 2 deletions):

| File | Change and reason |
|---|---|
| `app/ui/app/vitest.config.ts:19–23` | Add `server.deps.inline: ["streamdown"]` under the normal test configuration. The real Streamdown module imports KaTeX CSS; externalising it directly to Node prevented collection. Vite now handles that import without substituting a mocked renderer. |
| `app/ui/app/src/components/StreamingMarkdownContent.integration.test.tsx:24–29` | Correct the new test's incorrect bare-`strong` HTML expectation. Installed Streamdown 1.4.0 emits a span labelled `data-streamdown="strong"`; the assertion still couples formatting to the exact input word. Permit attributes on the emphasis tag. |

No production rendering function, highlighter theme, citation algorithm, generated type, lint rule, dependency lockfile or chat-history preservation code changed this continuation. The four type-only production files from continuation 07 retain their previously recorded bytes. The source checkpoint receipt names the exact changed-file set and the review result.

## Failed attempts retained, not discarded

| Attempt | Actual result | Disposition |
|---|---|---|
| `frontend-focused.*` | npm exit 1; six parser cases passed; renderer file failed collection because Node rejected `katex.min.css`. | Test-environment defect, not renderer acceptance and not a platform-blocked execution. |
| `frontend-focused-r2.*` | All eleven cases collected; ten passed; one failed on the bare-`strong` expectation. | New test was inconsistent with the installed renderer's implementation. Production behaviour was not changed to satisfy it. |
| `runs/author-no-output/` | Fault injector rejected an incorrect indentation anchor before compiling or running the mutant. | Instrument failure. No negative proof credited. Original harness retained as `validate_frontend_v1.py.txt`; correction recorded separately. |
| Later author and independent runs | Clean, full-suite and intended-failure results below. | Every attempt has separate output and receipts; earlier failures were not overwritten. |

The continuation-07 blocked invocation remains a historical non-execution. The new continuation-08 invocation was admitted through the same Bridge input mechanism after renewed operator instruction; no security setting, credential policy or alternate tool route was changed to obtain execution.

## Requirement-to-evidence map

| Required observable | Real path / assertion | Retained proof |
|---|---|---|
| The actual renderer and parser tests load and finish | Normal `npm test -- --run` via `vitest.config.ts`, real Streamdown/Shiki/remark | `runs/author-clean`, `runs/author-control-clean`, independent candidate arms |
| Output depends on supplied Markdown and code | `StreamingMarkdownContent.integration.test.tsx`: distinct Alpha/Bravo content, labelled bold span, emphasis, code text and syntax-colour output | Eleven-case clean runs; no-output mutation fails all five renderer assertions |
| Citation data reaches the correct downstream page | Renderer receives page stack; parser supplies cursor; renderer selects `pageStack[cursor]` | Correct-page test; disconnected-plugin and fixed-page mutants fail |
| Missing pages do not manufacture links; embedded model-provided resources stay excluded | Missing citation page, image and raw-HTML assertions through real rendering | Clean runs; no-output/disconnected controls demonstrate meaningful positive assertions |
| Existing frontend behaviour remains covered | Entire normal frontend suite; forced TypeScript rebuild in final control and judge arms | `runs/author-full` and `runs/judge-5-full`: 154 assertions passed, zero skips/pending/todo |
| Broken implementations are detected behaviourally | Isolated source copies, unchanged acceptance assertions, TypeScript compile first | Three mutants compile with exit 0, then test exit 1 for the required named failure |
| Clean implementation still works after controls | Original source fingerprint compared before/after; final clean run | `runs/author-final` and `runs/judge-6-candidate` |

The positive path is real Node server-side rendering. It is not a browser interaction, a CSS visual test, a running Ollama application or live-provider compatibility proof. Existing unrelated mock-based tests remain part of the broader suite; no mock replaces the real renderer/parser/highlighter in these new integration cases.

## Author and independent controls

| Arm | Compile | Behavioural result | Author + judge agreement |
|---|---|---|---|
| Clean candidate | exit 0 | 11 passed, zero failures | Yes, including final clean reruns |
| No output (`{content}` replaced with empty string) | exit 0 | 5 renderer assertions fail; 6 parser assertions pass | Yes |
| Citation plugin disconnected | exit 0 | Exactly 2 citation assertions fail; 9 others pass | Yes |
| Page index fixed at zero | exit 0 | Correct-upstream-page assertion fails; 10 others pass | Yes |
| Entire frontend suite | exit 0 | 154 passed, zero failures/skips/pending/todo | Yes |

These are representative fault controls, not an exhaustive proof against every conceivable defect. Mutant compilation failure, missing collection and timeout do not count as a successful control. The original working source is not overwritten by fault injection.

DeepSeek inspected the source and instruments and requested, in order: candidate, no-output, no-citations, wrong-page, full, candidate. The retained review controller confirms all required cases actually ran, all expectations were met, and source bytes remained unchanged. The review report hash is `15fa062d0d667ed3af1075304d45ccd99c674021045b03812b6ae5969df73bd7`.

This is a separate model directing fresh executions on the same Mac through an allowlisted controller, not an independent machine or an immutable-weights attestation. Requested model tag and observed local manifest are retained separately.

## Regressions, versions and unresolved gates

Actual production build: `npm run build` (`tsc -b && vite build`), exit 0. Full raw stdout/stderr and hashes remain under `regression/`. Build success is not a warning-free claim. The bounded text reader classified the build stderr file as binary; its recorded non-empty hash and original bytes remain intact. No warning suppression or alternative extraction was used to turn that into a clean result.

Actual lint: `npm run lint -- --format=json`, exit 1, **122 errors / 11 warnings**. The errors remain 104 in generated classes and 18 in handwritten source. Both changed files have no lint findings. Exact per-file population is in `regression/summary.json`; generator/hook follow-on notes are in `LINT_FOLLOW_ON.md`.

Observed dependencies: Streamdown 1.4.0, Shiki 3.14.0, React/React DOM 19.1.0, Vitest 3.2.4, Vite 6.3.5, TypeScript 5.8.3, remark 15.0.1, KaTeX 0.16.22, rehype-harden 1.1.5, Node v25.8.2, npm 11.11.1. Package metadata hashes are recorded; no dependency was upgraded.

Still open: generated/handwritten lint and hook warnings; stale Browserslist data; missing dependency sourcemap; previously observed Go duplicate-library warning; root Keychain-test isolation; supervised real-Keychain acceptance; combined-candidate review and packaged-app proof. The completed frontend review is not a waiver of those obligations.

## Recoverability, evidence and next action

The full local evidence root is `ollama/docs/analysis/colleague_integration/evidence/2026-09-21_cont08/`. Curated reports, raw test outputs, source identities and instruments are copied into the task branch's `docs/analysis/colleague_integration/proof_cont08/`. Mutant source copies, build artefacts and full local execution material are retained locally, not uploaded as product source.

One completed task-only Bridge record was preserved (original request, terminal metadata and complete output), verified and retired to restore command capacity. The archive operation changed no repository source or service configuration; its custody receipt is retained.

Main remains `953de98d408a9697fa20a48c23973b9dc8eee921`; the owned task branch is retained under Eko's responsibility. Disposition: **PARKED**, pending the named remaining integration gates. Source and proof have recovery custody; no irreversible user-data deletion occurred.

Next implementation boundary: inspect and repair the pinned generated-type pipeline and remaining handwritten lint in bounded, tested slices. Do not switch generated classes to interfaces: `new Message(...)` is used by the real chat-stream consumer. Do not restart the already completed history-preservation, source-visibility, native-declaration or frontend-rendering proof as though it were missing. No further blanket engineering approval is required.
