# Current checkpoint — continuation 09

21 September 2026, Brisbane. Eko owns `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`, branch `eko/chat-read-integrity-20260920`. Latest remote-verified checkpoint `8a046447b9e665dc3fccc928243fecd134c8ac98`; test-only WIP `f1d12c7820309cbda29b651a409db7850a079d1b`; unchanged production baseline `b880138b610418b0f0cdc5922e89c8c6451c8ba9`. Main remains `953de98d408a9697fa20a48c23973b9dc8eee921`. No runtime, dependency, lint-policy, compiler-config or installation change.

## What actually completed

Fresh pinned Go generation reproduced the committed generated file exactly. The actual project tsc passed and the established frontend suite passed 154/154 BEFORE a new test file was added. Full lint still reports 122 errors and 11 warnings. The generator has 35 classes and 16 repeated conversion helpers; constructors are actual consumers, not disposable type declarations.

The bounded generation repair now has a source-reviewed contract at `evidence/2026-09-21_cont09/GENERATOR_CONTRACT.md`, including R2/R3. DeepSeek's third source-only design consultation returned APPROVE; it is not runtime acceptance or formal blueprint promotion. Public JSON annotations belong in the two owning Go files; the adapter must reject public any rather than infer it. Normal generation/check share one guarded pipeline. Read the final contract, not the superseded prototype's public-type inference.

Compiler experiments proved generated-module byte equivalence. Eko prematurely claimed a Program comparison pass before reading a TS5096 failure, corrected the record explicitly, and executed a succeeding rerun with documented comparison-only options. Both attempts are retained. No result is attributed to a real browser/Vite run.

## Exact unfinished boundary

`app/ui/app/src/utils/generatedModels.test.tsx` contains ten source-declared cases. They have NOT RUN. Input `eko-cont09-consumer-baseline` was platform-blocked before execution; no alternative route was attempted. The former 154-test result is not acceptance of this WIP candidate. The generated no-any assertion is expected to fail on the old generator, but even that red baseline is not yet measured.

First obtain permitted execution of the normal new test command, from `app/ui/app`:

```sh
npm test -- --run src/utils/generatedModels.test.tsx --reporter=json
```

Do not bypass the recorded refusal. Inspect actual collection and failures. Then implement the already-reviewed R3 contract through the normal generation path; no pipeline or Go ts_type changes have been applied yet. Commit candidate and permanent tests before isolated controls, run required generator/consumer/full frontend/Go store-UI/build checks, and obtain independent executable review. Do not restart the resolved design debate or count source-only review as T-10.

Remaining inventory: 104 generated and 18 handwritten lint errors, 11 lint warnings, dependency/build warnings, native duplicate-library warning, root Keychain isolation, supervised real-Keychain proof and combined/package acceptance. Prior history/native/renderer proof retains its recorded scope. All 31 OI rows remain unaccepted. Standing authority is already granted; do not request blanket approval again.

## Durable evidence and prior state

`CONTINUATION_09_REPORT.md` explains the boundary. Full local evidence is `evidence/2026-09-21_cont09/`; selected exact evidence is remote-backed in `proof_cont09/` on the task branch. `git-closeout.json` records the verified remote. The complete prior handoff/state/decisions/blockers/index are byte-preserved in `evidence/2026-09-21_cont09/checkpoint_before_cont09/`. Older continuation reports remain unchanged; retrieve them rather than recopy the history into this handoff.
