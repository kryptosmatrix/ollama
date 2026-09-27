# Generated desktop models — T-09 acceptance evidence

## Scope, authority and candidate

Owner: Eko. Date: 21 September 2026, Australia/Brisbane. Scope: G-01–G-07 in continuation-09 GENERATOR_CONTRACT.md, including normative R2/R3. This is the existing frontend generation/lint maintenance gate, not a causal dependency on cleaning the whole repository before memory design. No formal blueprint promotion or programme-item completion is claimed.

Tested source: `6a3778184d43b23c5cba14e0379922377f615f52`. Prior checkpoint: `8a046447b9e665dc3fccc928243fecd134c8ac98`; prior production baseline: `b880138b610418b0f0cdc5922e89c8c6451c8ba9`. Candidate and permanent consumer tests were committed before mutation runs. The reviewed execution surface is frozen.json (161 source/tool identities plus 36,272 installed frontend dependency entries); recorded cache exclusions are explicit. Go module manifests/toolchain bytes are recorded; no full Go module-cache content census or immutable cloud weights are claimed.

System boundary: actual Go model discovery → pinned generator → annotation/real compiler guard → generated runtime classes; desktop getChat API wrapper → generated models → real Markdown server renderer. Only HTTP transport is simulated. Browser interaction, installed application, live-provider compatibility, real user stores/Keychain, schema-validation and legacy time/byte correctness are not claimed.

Environment: Node v25.8.2, npm 11.11.1, TypeScript 5.8.3, Go go1.26.5 darwin/arm64, macOS 26.6.2 build 25G83, Apple M3 Max, 137438953472 bytes RAM. Full command receipts: environment-and-precloseout.json. Synthetic fixture data and temporary store homes are used. Changing HOME is not presented as Keychain isolation.

## Requirement-to-execution map

| Requirement | Production owner / observable | Positive and rejection evidence |
|---|---|---|
| G-01 — reproducible owning process | app/ui/ui.go directive; codegen/generate.mjs::verifiedGeneration and generate | author-2-generation and judge-2-generation run actual CLI twice/check; author-13-normal-entry-corrected and judge-3-normal-entry execute the normal Go directive and compare output identity |
| G-02 — runtime compatibility | Five Go-owned JSON tags; annotate.mjs::annotate; emission.mjs::compareEmission; actual constructors/getChat/StreamingMarkdownContent | Ten permanent consumer tests, compiler equality on every generation, separate pre-metadata-emission-equivalence.json; nested/map/date/null/malformed and contrasting input assertions |
| G-03 — no explicit-any, no new unrelated lint | Generated AST has no AnyKeyword; actual app typecheck; unchanged lint policy | candidate and full runs; 164/164 frontend cases; full lint comparison proves generated zero and the same normalised outside findings (18 errors/11 warnings), not a clean whole-app lint exit |
| G-04 — fail closed, preserve output | One shared verification path, comparison vs atomic replace only at the final step | Dedicated 22-case Node command exercises real fresh generation/check/no-write/missing-Go/invalid-command and grammar/collision/compiler-refusal paths; record full child output and statuses |
| G-05 — actual consumers and required command inclusion | Default Vitest includes generatedModels.test.tsx; dedicated test:codegen is an explicit required codegen acceptance command; no new CI claim | Current package.json/config, generator README, full 164-case suite and 22 generator cases; real generated classes/renderer, simulated HTTP only |
| G-06 — executed deliberate breaks | Isolated raw generated output, Chat.messages empty-array fault, Message.content constant fault | Clean candidate before and after; each fault compiles and fails its unchanged named assertion. Raw arm also runs real ESLint (99 errors, exit 1). Fault patches/full logs preserved; only regenerable copies retired |
| G-07 — complete retained proof and independent review | Canonical commands, source/config/fixture/tool identities, true exits/discovery/completion | EXECUTION_INDEX.json; per-run records; independent-review-r2 fresh executions; final source checks and independent-review-addendum bounded JUDGE=PASS; all earlier attempts retained |

Exact source symbols, computed line anchors and whole-file SHA-256s: SOURCE_ANCHORS.json. Exact ten requirement-derived test identifiers are in generatedModels.test.tsx and the runner's EXPECTED set. The expected outputs come from literal fixture/contract values, not from calling the production function to compute its own expected answer.

## Frozen execution results

All eleven reviewer-directed runs completed: candidate, generation, normal-entry, raw-generator, no-messages, fixed-content, full, go-store, gate-self-checks, build, candidate. The default frontend suite has 164 passed/0 failed/0 pending; each clean consumer arm has 10 passed. Node generation has 22 passed including subtests, with zero failed/cancelled/skipped/todo. Go store has 42 started = 42 passed, 17 top-level, zero bad cases. Typechecks and frontend build exit 0.

Normal output retains 35 classes and 16 conversion helpers. Generated TS SHA-256: `e42678b57046bcbfbb50bb9b1d71aab9a917d06a2cb3ca4dd6ac4d49302c26b5`. Raw Go emission SHA-256: `5c243e47f525e822765979531e64081ed52bd591261be2f36d24f5912fd33ab9`. JavaScript is 53014 bytes, SHA-256 `b25ca8e6b74f0aaab8871f17c6b65be7dbde79da63c9f1755ab34de19108d5d5`, equal to the pre-metadata baseline too. Comparison-only compiler overrides are recorded; actual project tsconfig is unchanged.

| Fault | Compile / test exit | Required observed failure | Recovery |
|---|---|---|---|
| raw-generator | 0 / 1 | explicit-any assertion sees 99 sites; actual ESLint exit 1, 99 no-explicit-any findings | Clean rerun passed; fault copy retired |
| no-messages | 0 / 1 | expected two delivered messages but received empty array | Clean rerun passed; fault copy retired |
| fixed-content | 0 / 1 | real renderer output lacks the expected Alpha content | Clean rerun passed; fault copy retired |

The expected-failure arms are not clean tests or waived requirements. Their enclosing control passes only because the specific fault is compiled, active and detected. Additional downstream failures are retained and are not substituted for the named assertion. Compilation/collection failure would fail the control.

Runner calibration rejects a real child exiting 7, a missing required case and an empty selection. The wrong Go selector run exited 0 without execution and was rejected for missing a receipt: author-11-normal-entry stays false. Installed help and dry runs justified correcting the selector, not changing the acceptance condition; author-13-normal-entry-corrected passed. Full failed/aborted attempts and their dispositions remain in entry-selector-correction.json, review-transport-correction.json and REVIEW_RECONCILIATION.md.

## Independent judgement and limits

Requested: deepseek-v4.1-flash:cloud. Actually served: deepseek-v4.1-flash. Observed tag manifest: `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8`. This is one selected separate model, with reviewer-directed real execution, not Eko switching roles. Its original final report and complete-source/raw-log addendum both state JUDGE=PASS for this bounded scope. The addendum closes the incomplete renderer-body/log inspection and corrects factual report slips; see REVIEW_RECONCILIATION.md. No alternate judge was sought.

The first review transport attempt read nothing and ran nothing before rejecting line-zero arguments. The same conversation resumed; all eleven runs occurred before the 26-round cap. The saved complete conversation was then resumed for missing source inspections and final reporting. The later full-source/log addendum is explicitly source/log review of those already fresh, unchanged-candidate executions, not a claim of more executions.

Full lint still exits 1. Existing 18 handwritten errors, 11 lint warnings, dependency/sourcemap/Browserslist/build warnings, duplicate-library warning, root Keychain-test isolation, supervised Keychain proof and combined/package acceptance are not waived. No browser, installed-app, live data, whole-programme or memory/autonomy completion follows. All OI-01–OI-31 remain unaccepted by this slice.

## Craft and reproduction

Go owns public field metadata; the adapter refuses unsupported public types and uses only erased changes. CLI/annotation/compiler checks are separate cohesive modules, with no vendored fork, blanket suppression or runtime interface substitution. Input/output caps, checked external command exits/timeouts and per-invocation scratch ownership are explicit. This is an Eko craft assessment supported by separate source review, not a separate named Craft Judge verdict.

Production reproduction commands from app/ui/app: npm run check:types; npm run test:codegen; npm exec --no -- tsc -b --force; npm test -- --run; npm run lint; npm run build. Run the normal Go generation directive from the repository as recorded above. Capture actual statuses; echoing a status is not the gate. A full lint failure remains a failure, not a reason to bypass a hook.

The archived fault runner is bound to the exact source checkout, HEAD and manifest. It deliberately refuses a changed candidate, including a later HEAD. For a new source revision, create new evidence/freeze and re-review; do not edit old receipts. Documentation-only descendants may cite this evidence only with a separately retained execution-surface equality and explicit source-to-checkpoint link. Historical absolute paths are preserved; the curation manifest maps their copied bytes rather than rewriting receipts.

Fidelity: [FULL] for the bounded G-01–G-07 generated-model maintenance contract only. Main integration and installation remain blocked/unperformed, not implicitly authorised by this label.
