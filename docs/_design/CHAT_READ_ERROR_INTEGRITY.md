# Chat continuation integrity — refuse replacement after a read failure

| Field | Value |
|---|---|
| Document class | Implementation blueprint |
| Design readiness | [SPEC-DRAFT] — source review found readiness gaps; no implementation permission |
| Live implementation status | Not Yet Implemented |
| Programme | OI-01 discovery; first bounded OI-02 prerequisite repair for OI-26/OI-27/OI-29 |
| Author | Eko, 20 September 2026, Australia/Brisbane |
| Source revision | 953de98d408a9697fa20a48c23973b9dc8eee921 |
| Tier proposal | Tier 2 — the declared HTTP observable crosses a process boundary. No persisted schema/wire-format change is proposed. The original aggregate production estimate was 12 lines; itemised estimate and the full Tier-2 independent panel remain owed. |

## Concept intake, intention and boundary

Ash's commissioning work list requires preservation of the canonical complete conversation independently of selected model context and the viewer. A failed read must not become permission to replace an existing conversation. Concrete scenario: a saved message has a value the SQLite loader cannot decode; sending the next prompt must fail without deleting earlier messages. This is not recovery of already lost data, schema redesign, append-only storage, concurrent-writer repair, or delivery of the full memory/export features.

The original commission is `docs/OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md`, SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`, sections OI-01, OI-26, OI-27 and OI-29. Technical repair design is within that commission. This document adds no deployment, credential, service or publication authority. No new operator decision is assumed. The relevant local discovery is the malformed-message probe below; wider OWL/design-corpus reconciliation is not completed and must not be represented as complete by the readiness reviewer.

## Current behaviour, measured gap and compatibility

`app/store/database.go:675-713` queries a chat header, then loads its messages. Only its header query returning `sql.ErrNoRows` means the chat itself is absent. It currently returns untyped `fmt.Errorf("chat not found")` for that case. Other database errors already carry their underlying cause.

`app/store/store.go`, `Store.ChatWithOptions`, currently converts every error from `getChatWithOptions` into `not.Found`. `app/ui/ui.go:727-787` interprets that classification as permission to construct a new chat and call `SetChat`. `database.saveChat:716-782` deletes the existing message rows before reinsertion in one transaction.

Measured baseline: `docs/analysis/colleague_integration/evidence/2026-09-20_cont02/history-probe-v2-baseline/observations.json`. A real temporary SQLite database, the production development-mode HTTP handler, and a loopback inference fixture were used. Setting one persisted boolean column to invalid text produced `not.Found`; the POST returned HTTP 200, contacted inference three times, and reduced the count of the two original messages from 2 to 0. The healthy and genuinely absent-chat controls completed normally. This is not a measurement of defect frequency, authentication, or the installed application. The initial integer-2 fault did not cause a loader failure; its unsuccessful instrument record is retained separately.

Preserve existing successful reads, explicit new-chat creation, the missing-chat error text/classification, caller method signatures, database schema version 17, valid persisted values, and explicit edit/replace behaviour. No feature flag or migration is introduced.

## Ownership, algorithm and rationale

The existing SQLite `database` owns identifying absence of the chat header; `Store.ChatWithOptions` transports the result; `Server.chat` remains the consumer deciding whether to create or abort. No new mutable state or lifecycle is introduced.

At the header-query `sql.ErrNoRows` branch only, `database.getChatWithOptions` returns `fmt.Errorf("%w: chat %s", not.Found, id)` using the existing `app/types/not` sentinel. Other header errors and downstream message/attachment/tool-call errors retain their existing wrapping. `Store.ChatWithOptions` returns the database error unchanged rather than relabelling it. Its existing ensureDB failure propagation remains unchanged. The existing HTTP caller then takes its existing abort branch for a genuine load failure, before SetChat or inference.

This localises absence classification to the query that actually proves absence. Reject inspecting error strings, classifying every nested `sql.ErrNoRows` as absent, returning an empty Chat on error, swallowing the error, or repairing the row implicitly. The cost is one existing sentinel import; no new dependency, file-format field, query, retry, or allocation of state. Successful-path complexity is unchanged.

## Frozen requirements

R-01. Only an absent chat-header row MUST be classified as `not.Found` by `Store.ChatWithOptions`.

R-02. A non-absence chat-load error MUST preserve its underlying error chain through `Store.ChatWithOptions`.

R-03. Continuing an existing chat whose load fails MUST leave its persisted chat, message, attachment and tool-call records unchanged.

R-04. Continuing an existing chat whose load fails MUST return an error before an inference request is issued.

R-05. Valid existing-chat continuation MUST retain the earlier messages.

R-06. Genuinely absent-chat continuation MUST retain the existing create-and-continue behaviour.

## Activation and proof contract

Entry: production `POST /api/v1/chat/{id}` through `Server.Handler`. Trigger: existing chat with a real loader failure. Producer: `database.getChatWithOptions`; transport: `Store.ChatWithOptions`; consumer: `Server.chat`. Observable: HTTP error, no inference, unchanged independently queried records. Negative: a valid or genuinely absent chat still proceeds. The fixture inference endpoint supplies responses only; it is not evidence of real model reasoning.

Planned default-run tests: `app/store/chat_read_errors_test.go` covers valid/missing reads, a closed-database error, and message scan failure through Store; `app/ui/chat_read_errors_test.go` covers the real HTTP continuation path, valid/missing controls, independently queried preserved rows, and reopening after the injected corruption is removed. Fault injection acts only on a test-owned database. Production authentication must be included in the permanent acceptance route or explicitly left as an outstanding proof obligation; the development-mode diagnostic is not its substitute.

The code-changing surfaces are `app/store/database.go` and `app/store/store.go`. No source change to Server.chat is needed. Freeze source/test hashes and run candidate → restored-bug mutation → candidate with the same acceptance tests; retain all exit codes, logs and changed-source identities. A separately compiled consumer-bypass mutation must also fail the preservation/no-inference checks. Do not accept compilation failures as behavioural falsification.

| False implementation | Required rejecting observation |
|---|---|
| zero/no-op getter | valid existing chat and its original messages must be returned |
| constant not.Found | malformed existing chat must fail without replacement |
| empty Chat success | valid original messages must survive continuation |
| identity/pass-through of untyped missing error | genuine absence must remain errors.Is(not.Found) |
| always-abstain error | valid and missing-chat controls must complete |
| always-success read | malformed data must not produce HTTP success or inference |
| unreachable corrected branch | the unchanged restored-bug source must fail the same default acceptance test |
| producer-starved test | the malformed arm must first demonstrate an actual loader error; integer 2 was not such a fault |

## Validation, decomposition and outstanding work

Pack A: verify exact source/test baseline and classify tier/readiness. Pack B: add default-run tests first, implement only the two-file repair, run the behavioural mutation controls. Pack C: independent source/test/evidence review with actual re-execution where required. Packs D/E: update the discovery and requirement-evidence records. Pack F: check containment and the declared affected proof set before accepting this prerequisite; do not promote any of the 31 programme features from this repair alone.

Declared affected proof set: core `go build`; complete Go suites for `./agent/...`, `./cmd/tui/chat/...`, `./app/...`; complete frontend Vitest, TypeScript build and ESLint checks; exact repair tests and two behavioural mutation controls; `git diff --check`; before/after file hashes and Git state. The root `go test ./...` remains additional whole-programme proof, not claimed executed here. Its real login-Keychain tests require isolation/scope reconciliation before running.

The pre-edit bounded baseline passed the core build, 593 passing named Go test/subtest records (including parent/subtest records, not a leaf count) with one platform-inapplicable skip, 143 frontend tests and TypeScript; frontend ESLint failed with 137 errors and 11 warnings. These failures are not waived or suppressed. Their exact relation to the repair and the final acceptance boundary must be adjudicated; this draft does not grant a passing gate over that red baseline.

Open obligations: readiness classification and independent judge; exact default-run acceptance implementation; production-authenticated HTTP proof; complete app Go suite; candidate and mutation execution; full source/error-chain and cross-blueprint reconciliation; independent executable review; baseline lint disposition; Git containment. No requirement is discharged by this design. No code or install is authorised by its unearned readiness label.

## Continuation 02 status correction

The original reviewed bytes are retained under `docs/analysis/colleague_integration/evidence/2026-09-20_cont02/checkpoint_before_cont02_closeout/CHAT_READ_ERROR_INTEGRITY.md`. The six frozen R-01–R-06 requirements are unchanged. This status/title/count correction grants no readiness, changes no runtime source and does not count as a completed repair round. The detailed 15-item intake and evidence/protocol limits are in `docs/analysis/colleague_integration/JUDGE_RECONCILIATION_02.md`. GET-empty-success and rename-error wording are separately tracked; this proposed repair concerns continuing an existing conversation without destructive replacement after a read failure.
