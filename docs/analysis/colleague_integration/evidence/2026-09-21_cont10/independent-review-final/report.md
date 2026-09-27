The raw-lint stdout is a separate file not addressable by ordinal (only tests.stdout is). The runner's `raw_lint_rejected` requires exit 1 AND every message ruleId `@typescript-eslint/no-explicit-any` severity 2 — a strict check that would fail if any other rule fired. That is recorded as true with 99 findings. I have enough.

## Review report — bounded generator maintenance contract

**Scope and method.** I read the frozen contract (R2/R3), self-audit, generator (`generate.mjs`), annotator (`annotate.mjs`), emission guard, Go metadata owners (`store.go`, `responses/types.go`), generated target, both test suites, `api.ts` `getChat`, the real `StreamingMarkdownContent`, `validate_codegen.py`, the reused `validate_frontend.py` transport, `tsconfig.app.json`, `vitest.config.ts`, and the selector-correction record. I then ran all eleven required arms fresh at head `6a3778184d43b23c5cba14e0379922377f615f52`.

**Positive evidence (all reproduced).**
- **candidate (first and last):** 10/10 named consumer assertions pass, `tsc -b --force` exit 0, source/deps unchanged. Both runs identical in outcome.
- **generation:** `test:codegen` 22 tests / 22 pass / 0 fail / 0 skipped; `check:types` exit 0. Matches the required 22 including subtests.
- **normal-entry:** the real `//go:generate node ./app/ui` directive ran via `go generate -run '^//go:generate node '`, exit 0, receipt `mode=write`, `classes=35`, `helpers=16`, `emission.identical=true`, two upstream Go runs exit 0, output hash equals the committed target.
- **full:** 164/164 pass, typecheck exit 0; lint exits 1 with exactly 18 errors / 11 warnings, `generated_findings=[]`, and the non-codegen message set byte-equal to the recorded baseline (`bounded_delta_ok=true`).
- **go-store:** 42 started = 42 passed, 17 top-level, 0 bad — subcases reconcile.
- **gate-self-checks:** clean positive true; a real child exiting 7 blocks; missing-case and empty-selection block.
- **build:** exit 0, existing dependency/sourcemap warnings retained (not waived).

**Negative controls (each compiled, then failed its named assertion for the intended reason).**
- **raw-generator:** raw upstream output compiles (tsc exit 0) but fails `ships generated declarations without explicit-any syntax` (99 any sites), and the real ESLint rule rejects it (exit 1, 99 findings, all `@typescript-eslint/no-explicit-any` severity 2). Fault bytes unchanged; isolated copy retired.
- **no-messages:** compiles; fails `materialises nested chat models through the real API consumer` (`expected [] to have a length of 2`).
- **fixed-content:** compiles; fails `delivers distinct decoded content to the real Markdown renderer` (renderer output lacks "Alpha").

**Source-level checks.** The annotator is a pure AST transform: it rejects public `AnyKeyword` (including inside generics/overrides), refuses `JSONValue` name collision, requires the exact constructor/helper templates, rejects overlapping ranges, and re-walks the result for any residual `any`. Public types come from Go `ts_type` tags — I confirmed `ToolFunction.Result any ... ts_type:"JSONValue"` at `store.go:97` and the five R2 annotations in `responses/types.go`; the generated target carries `result?: JSONValue` and no `any`. Emission compares raw vs refined JavaScript under parsed project options with only the documented comparison overrides (`noEmit=false`, `allowImportingTsExtensions=false`), rejecting diagnostics/emitSkipped/missing output. `--check` and write share one verification path; the target is untouched until the final atomic action. `getChat` really constructs `ChatResponse`; the renderer is the real Streamdown/Shiki component (HTTP transport simulated only, as declared).

**Selector correction.** The failed `^node ` run is preserved as `expectation_met:false` with an empty receipt; only the proof selector changed to `^//go:generate node `. No production byte or positive assertion changed. This is a legitimate instrument fix, not a rewritten result.

**Boundaries (not waived).** Full lint still exits 1 (18 handwritten errors, 11 warnings); Browserslist/sourcemap/build warnings retained; root/Keychain/package/main-integration/installation, browser interaction, privacy isolation and memory/autonomy are outside this bounded result. This is not a JSON validator and does not bless legacy time/byte/raw-JSON quirks.

**Residual observations (non-blocking).** The raw-lint stdout file is not independently addressable through my read tool, so I rely on the runner's strict all-messages-are-no-explicit-any check plus the recorded 99 findings; the vitest JSON confirms the assertion failure independently. No genuine defect or missing mandatory evidence found within scope.

JUDGE=PASS