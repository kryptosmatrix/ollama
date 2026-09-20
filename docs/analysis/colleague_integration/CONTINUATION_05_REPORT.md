# Ollama integration — continuation 05

20 September 2026, Australia/Brisbane. Owner: Eko.

## Result and boundary

The chat-read preservation mechanism is implemented, exercised through the authenticated HTTP handler, and independently reviewed with fresh model-directed executions. The bounded review returned `REPAIR_REVIEW=PASS`. Whole-application acceptance is still open: frontend lint fails, complete app/root testing is not done, and nothing was merged to main or installed.

Tested source candidate: `18ecd2e2bcd8f5593eea4dbdeda8b9dca512514c`.
Branch: `eko/chat-read-integrity-20260920`.
Worktree: `/Users/krypto/GitHub/ollama-eko-chat-read-integrity`.
The source candidate was pushed to the existing origin and its exact remote branch tip was verified. A subsequent documentation-only checkpoint may sit above this tested source commit; it does not change the tested production/test bytes.

All six frozen repair requirements remain intact. This does not accept any of the 31 wider colleague-workspace items or claim an earned implementation-blueprint stamp. Ash's standing authority and the bounded maintenance exception remain recorded in DECISIONS.md; no further permission was requested.

## What the code now demonstrates

`app/store/database.go::getChatWithOptions` identifies absence at the chat-header query only. `app/store/store.go::ChatWithOptions` transports other database failures without converting them to `not.Found`. The unchanged `app/ui/ui.go::chat` then aborts failed-load continuation before saving or contacting inference.

The passing authenticated fault phase recorded:

```text
http=500 model_requests=0 all_records_unchanged=true
```

The test creates a real temporary SQLite database through Store, writes original messages, title, browser state, attachment and tool-call data, then injects a real message-scan failure. Independent raw SQL snapshots compare every row and column of the four affected tables. Separate processes cover seed, fault, recovery and healthy/new continuation. After removing the injected corruption, the original history survives and is used in the next inference request. Inference is an explicit loopback simulation; the store, HTTP router, authentication branch and process separation are real. This is not packaged-app sign-in or live-provider proof.

Focused population: five Store tests plus one HTTP lifecycle test with four named phase subtests, producing ten terminal test/subtest records. The clean runs had ten passes, no failures, no skips and no unfinished records. Counts include the parent and phase subtests; they are not ten independent experiments.

## Executed deliberate-break controls

| Compiled variant | Actual result | Failure the tests observed |
|---|---|---|
| Restore the old Store error relabelling | Go exit 1; expected behavioural failure | Read error becomes absence; original records change; HTTP 200 and inference incorrectly proceed |
| Bypass the HTTP consumer's error abort | Go exit 1; expected behavioural failure | Saved records change and inference is contacted despite the failed load |
| Always return an error | Go exit 1; expected behavioural failure | Healthy reads and genuinely new conversations are refused |
| Return an empty successful Chat | Go exit 1; expected behavioural failure | Original content is missing and malformed data is incorrectly accepted |

Each variant was compiled with a Go overlay, not written over the candidate. No build failure, missing test, skip or timeout was accepted as a successful falsifier. Eko ran all four variants and then reran the clean candidate successfully. The same ten-record population completed in each run. Parser checks also exercised missing terminal results, skips, compiler failure, empty output and malformed evidence; the earlier real missing-assets build failure was correctly rejected by the runner.

## Independent review — what actually ran

The selected `deepseek-v4.1-flash:cloud` received a bounded tool interface. It could read named source/instrument files, request fresh validation cases, and read their raw output. It had no general shell, file-write, credential or deletion tool. The controller follows Ollama's documented tool-calling message exchange; it is a temporary verification instrument, not a delivered Ollama product feature.

DeepSeek requested and received six fresh executions in this order: candidate, old Store relabelling, consumer bypass, always-error, empty-success, candidate again. Its actual tool requests, tool results and API responses are retained locally. It inspected source, the runner and the mutations, read the failing assertions and returned `REPAIR_REVIEW=PASS` for the six-requirement mechanism/test scope. The controller independently checked execution coverage, exit codes, expected outcomes and unchanged source before recording a passing bounded review.

This is independent model-directed execution through Eko's constrained controller on the same Mac, not a separate-machine replication or a reviewer's unrestricted shell. Eko authored the controller and tests; the reviewer inspected them. Full regression, packaged-app and release gates were explicitly outside this bounded verdict.

Requested manifest digest: `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8`.
Controller SHA-256: `349782a8850ce690f0ca4c290d67838fbee41b532202adc19c68edcf6e1b78c4`.
Review report SHA-256: `c1983ecacf04835e8359122101e165484ded6d2ffefe561df467f3ea9e2c6c5d`.
Review result SHA-256: `a7c89cb3f9a650c3f6b6073423fbaa84d601e9701d88ce5c2b7be8ed44656751`.
A cloud manifest identifies the observed tag, not immutable model weights.

## Fresh-checkout defect recovered during testing

The first focused attempt passed all five Store tests but could not compile the UI because the fresh worktree lacked built frontend assets. Installing exactly the lockfile dependencies offline succeeded. Building then exposed source files that existed in the ordinary checkout but were absent from Git: `app/.gitignore` uses `*cover*`, which also matches Discovery/discover filenames.

Six existing files were read, copied byte-for-byte, checked against their original SHA-256 hashes and committed without redesign:

- `app/ui/app/src/components/MCPLocalDiscovery.tsx` and its `.test.tsx` file.
- `app/ui/app/src/utils/mcpDiscovery.ts` and its `.test.ts` file.
- `app/ui/mcp_discover.go` and `mcp_discover_test.go`.

The tracked frontend and router already referenced these inputs. Their originals were retained unchanged. Recovery commits are `1916ee940c1f479f5ce60da8555976f03b8a8401`, `ea3ba51e93fddf048a516e0ad9d9e680dab83254` and `18ecd2e2bcd8f5593eea4dbdeda8b9dca512514c`, above the original repair commit `927fcb5dd706ffd741612a4ab124646aa22a60cb`.

After recovery, the actual frontend build and focused Go tests succeeded. No dummy assets or replacement discovery implementation were introduced. The six files are now tracked, so the existing ignore rule no longer hides them from this branch. The broad ignore rule itself remains a follow-up risk for future files; it was not silently rewritten here.

## Broader verification by Eko

| Check | Result |
|---|---|
| gofmt comparison of the four original repair files | No formatting difference |
| Core `go build` | Exit 0; stdout and stderr empty |
| `go test -count=1 -json -timeout=180s ./agent/... ./cmd/tui/chat/... ./app/store ./app/ui ./app/tools` | Exit 0; 603 passing named test/subtest records, one existing Windows-only skip, no unfinished records |
| Frontend `npm test -- --run` | 143 tests across 21 files passed |
| TypeScript build check | Exit 0 |
| Actual frontend `npm run build` after source recovery | Exit 0 |
| Frontend ESLint | Exit 1; 137 errors and 11 warnings, matching the earlier baseline totals |
| Source diff check and containment | Passed before documentation closeout; original main tracked source and all six recovered originals unchanged |

The inherited skip is `TestNormalizeBashWorkingDirWindowsDriveLetter` in `agent/tools`, not a new repair-test skip. Frontend tests also emit a stale Browserslist warning and the expected clipboard-refusal diagnostic. No warning-free frontend or whole-programme claim is made.

The regression batch deliberately retained its non-zero aggregate exit because lint failed. Equal historical totals explain a baseline; they do not waive a required gate. The independent reviewer did not rerun this broader regression batch.

## Evidence locations

Full local evidence root: `/Users/krypto/GitHub/ollama/docs/analysis/colleague_integration/evidence/2026-09-20_cont05/`.

The unchanged earlier proof runner writes its new runs under `evidence/2026-09-20_cont04/`: `cont05-candidate-01`, `cont05-candidate-03`, `cont05-candidate-04`, the five `cont05-author-*` runs, and the six `cont05-judge-review01-*` runs. There is no executed candidate-02 test run: its prerequisite frontend build failed first. Setup failures remain separate from behavioural results.

Key records: `author-matrix.json`, `regression-results.json`, `parser-self-checks.json`, the three source-recovery receipts, `final-containment.json`, and `judge/review01/{report.md,result.json}`. Curated, safely shareable proof receipts are copied into the candidate branch at `docs/analysis/colleague_integration/proof_cont05/`; full model exchanges and raw logs remain in the local evidence directories.

Ollama protocol reference: https://docs.ollama.com/capabilities/tool-calling and https://docs.ollama.com/api/chat, consulted on 20 September 2026. These document the transport, not evidence that this particular review ran; the retained execution records establish that.

## Remaining gate and custody

Disposition: PARKED for integration; bounded mechanism review passed.

Complete `./app/...` and root `go test ./...` remain unexecuted. The app has a direct Keychain-test surface and the root suite has the earlier unresolved real-Keychain isolation issue. Those checks were not run against the real login Keychain and were not silently converted into passing coverage. Packaged-app startup, installed-candidate acceptance and independent broader-regression review also remain open.

The installed Ollama application and real conversation data were not replaced or used by the new synthetic proofs. Main was not merged. The remote-backed branch and its worktree are retained under Eko's ownership while the remaining gates are resolved. Do not delete unique work or ask Ash for the already-granted maintenance permission again.

Operational record: the first command admission failed because the Bridge archive was full. One previously backed-up, completed Eko review record (`f9dee0186502441d9b5ebbeedf6b7884`) was retired after its command/output provenance and terminal state were rechecked. Its archive remains intact. A broad ignored-source census was platform-blocked and was not executed; the specific missing inputs above were located from compiler errors and bounded source reads. The direct Keychain-test source read was denied and not repeated through another route. All active validation work was confined to this session.
