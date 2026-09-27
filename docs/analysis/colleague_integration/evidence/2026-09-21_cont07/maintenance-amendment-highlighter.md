# Continuation 07 — custom-theme type correction

This is an explicit extension of the frontend type-only maintenance scope to the existing type owner `app/ui/app/src/lib/highlighter.ts`; no other production file is added to the boundary. Ash's standing pre-approval applies; Eko explained the change and reason before editing. The original maintenance contract and both pre-code review rounds remain unchanged evidence.

The first actual TypeScript run failed because the existing `Awaited<ReturnType<typeof createHighlighter>>` annotation permits bundled theme names, whereas this module registers its own `one-dark` theme. The old consumer hid the mismatch behind `as any`. The exact failed output remains in `typescript-check.stdout`.

Use Shiki's exported `HighlighterCore` interface as the instance annotation. The installed library defines this as `HighlighterGeneric<never, never>` for custom language/theme keys. Keep `createHighlighter`, theme objects, names, colours, language registration, promise resolution and assignments unchanged. Add only the type import and replace the variable type annotation. No runtime import, type assertion, error suppression or dependency upgrade.

Before writing, a virtual TypeScript compiler-host experiment applied these two type changes in memory and checked the complete project. It exited zero with no diagnostics; the exact output is `highlighter-type-experiment.stdout`. That is static evidence, not frontend test execution. After writing, re-run the actual project typecheck, the full ESLint command and identical-JavaScript comparison including this fourth frontend source.

Frontend runtime tests remain NOT RUN: the platform blocked input `eko-cont07-frontend-baseline` before execution. Do not replay that operation through another tool or reviewer harness. Preserve the eleven written regression tests as unexecuted candidates, not accepted tests. Independent native verification does not approve the frontend slice or integration.
