# Continuation 12 — failed-turn recovery and G1 design progress

Eko; 21 September 2026, Australia/Brisbane. Work is scoped to recovery and the instructions editor/request-delivery feature, not general cleanup. G1 remains SPEC-DRAFT and Not Yet Implemented. No OI programme item is accepted.

## What survived

Fresh native Git reads verified the owned branch eko/chat-read-integrity-20260920 at baedaf58a711606f343c9c83e13bb68a0169528f, clean and matching the actual remote branch. Main and remote main remain 953de98d408a9697fa20a48c23973b9dc8eee921. The entire difference from continuation10 is documentation: the two G1 design files and the proof_cont11 evidence. Production code remains the previously tested 6a3778184d43b23c5cba14e0379922377f615f52 surface.

The failed turn's prior shell is terminal cancelled/exit-15 with retained output, not an active successful worker. Its report, corrected design, source consultation and synthetic probes survived. Its timed-out HTTP baseline did not retain useful partial child output. The new run below closes that missing entry result without rewriting the old record. ChatGPT's internal Thinking failed cause is unknown; the local records do not diagnose the platform. No claim is made that unsaved internal work was recovered or that detached processes are absent.

One completed diagnostics-only Eko Bridge record was retired after lossless preservation to free a command slot. Its full 8373-byte stdout was reconstructed and verified against SHA256 b3e32c431774d4f4d12804eef031c92c6adc9b27ac8f2a8011f2807bd3627fd9; the original request/command is retained, and the Bridge retains its replay tombstone. No service configuration or security setting changed.

## What actually ran

| Surface | Result | Evidence under evidence/2026-09-21_cont12/ |
|---|---|---|
| Unchanged recovered HTTP diagnostic through real authenticated Server.Handler and synthetic store | Both tests completed: existing settings control passed; instructions PUT failed its saved-Profile assertion because it returned HTML200, decoded=false. Process exit1, no timeout/skips/unfinished tests | http-baseline-original/result.json, stdout.jsonl, key-events.json |
| Real candidate CLI build and entry | go build succeeded with empty stdout/stderr; ordinary --help succeeded; instructions --help returned unknown command/exit1 | cli-baseline/result.json plus three command log pairs |
| Independently authored canonical-profile vectors | Three literal JSON/Python SHA256 pairs match Go encoding/json and crypto/sha256; changed-text and line-ending checks pass. Five named test/subtest records, zero failures | profile-goldens/manifest.json, canonical_profile_test.go, result.json, stdout.jsonl |
| Native backend version-only probe | Timed out after 20 seconds, before the planned help/model-specific probe. No runtime qualification obtained | native-counter-probe/blocked-diagnostic.json; Bridge stderr0–1315 |
| Follow-up native diagnostic | OpenAI refused before execution; no reroute, no protection changes | native-counter-probe/blocked-diagnostic.json |

The golden test is a reference-serialization check for null-binding profiles, not the still-unimplemented storage/request consumer. The two red entry results establish missing real endpoints, not a successful persistence/restart feature or the whole declared lifecycle failure metric. No broad frontend/root/app/Keychain suite was rerun here.

## Product design progress

The selected pinned native dependency b10091 resolves to b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45. Server/common files used below match their pinned Git blobs; three unrelated existing build patches elsewhere were left alone. Source inspection identified the non-inferencing /v1/chat/completions/input_tokens path (tools/server/server-context.cpp:4703–4705,5295–5343). It uses the existing chat/template parser and native token-counting path. Its existence is source evidence, not successful native execution.

More importantly, the native backend already checks actual task tokens against the selected slot before prompt evaluation (server-context.cpp:3068–3150). Slot context can be capped to the model training context (:1250–1255); Go ContextLength returns requested options.NumCtx and cannot substitute for that actual slot. Existing input-size checks differ between splittable and non-splittable tasks and must remain intact. They do not reserve a finite output allowance.

The design now rejects auxiliary one-token inference as the production measurement mechanism and selects extension of that existing actual-token/slot guard. A count-only preview remains a point-in-time preview: clock-dependent native template inputs mean an equal later JSON request is not proof of equal prepared tokens. This avoids a duplicate prompt assembler or new inference engine. The exact finite output policy, versioned ingress/capability and backend coverage remain assigned design decisions; they are not invented as working code. Cloud/media/MLX obligations are not silently removed.

Three concrete null-binding digest vectors were added, and the executed missing-entry results are recorded in the owning blueprint. All 18 frozen MUST outcome lines are byte-identical. The amended file is 59,128 bytes under the existing 59,560-byte cap; the estimate is not padded. SPEC-DRAFT and formal promotion rounds 0 remain.

## Independent challenge and errors

The selected DeepSeek model was consulted three times in the same continued source/log conversation, served as deepseek-v4.1-flash. These were not three independent judges, formal promotion rounds, or reviewer-directed executions. It challenged the design and confirmed the existing slot guard as a supported extension site after the additional source was supplied. Complete reports and dispositions are in the three consultation directories and REVIEW_RECONCILIATION.md.

Eko corrected an unnecessarily early before-queue admission requirement to before-prompt-evaluation at the actual slot. Reviewer claims about header-only strictness, public tokenizer flags, preview identity and media/cache reuse were checked rather than accepted as authority. The media cache-reuse restriction does not imply media lacks a slot; runtime media qualification still remains owed.

Two instrument failures remain explicit. The version probe used capture-after-return, so its timeout did not leave the promised partial child-output files. Subsequent execution diagnostics were blocked, and that limit is not bypassed. The closeout aggregator also initially expected the wrong error string/capitalisation from the already-retained HTTP test; it failed before changing control files. It was corrected against the exact G1_SAVE_BOUNDARY_MISSING raw event, without changing or rerunning the diagnostic assertion. See closeout-parser-correction.json.

## Remaining bounded work

Finish OQ-1's finite output allowance/representation, versioned capability or ingress that old daemons cannot silently ignore, and explicit local/rendered/native/media/MLX/remote outcomes. Use the discovered selected-slot guard; do not start another search for whether one exists or reopen the rejected one-token mechanism. Then finish the G1 lifecycle/failure-number population, inactive-binding golden coverage, concrete CLI/test/guard closure and independent two-substrate design gates before runtime implementation. Native runtime proof remains blocked/unverified; it is not a reason to restart all design or reroute the refused diagnostic.

The broader 31-item inventory and prior release obligations remain:18 handwritten lint errors/11 warnings (historical, not freshly rerun), dependency/build warnings, native duplicate-library warning, root Keychain-test isolation, supervised real-Keychain acceptance, combined/package acceptance, main integration and installation. No real conversation, live store, Keychain or installed application was changed. No LETHE write or new colleague wake is claimed.

## Reproduction and custody

All command arrays, full logs, exits and file hashes are in this continuation evidence directory. precloseout-containment.json proves no tracked production/default-test/dependency delta from the tested source. The candidate CLI binary stays locally in evidence and is excluded from the Git proof package. A curated lossless package, final commit/remote verification and shell terminal receipt are separate closeout records; no pending action is represented here as already complete.
