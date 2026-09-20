The candidate's fault phase shows `http=500 model_requests=0 all_records_unchanged=true` — the R-03/R-04 assertions executed and passed non-vacuously. I have all evidence needed.

# REPAIR_REVIEW — candidate 18ecd2e2bcd8f5593eea4dbdeda8b9dca512514c

## Scope and method
I inspected the two production functions, both test sources, the proof runner, the controller, the mutant manifest, and all four mutant sources. I then executed six fresh validations (candidate, rewrap, consumer-bypass, always-error, empty-success, candidate rerun) and read the raw failing assertions. All six ran to completion with `source_unchanged_during_run=true`, no build failures, no skips, no timeouts.

## Production mechanism
- `Store.ChatWithOptions` (app/store/store.go:428-434) calls `ensureDB`, then returns `s.db.getChatWithOptions(id, loadAttachmentData)` directly — no re-wrapping. This is the repair: the old method collapsed every error into `not.Found`.
- `database.getChatWithOptions` (app/store/database.go:677-716) returns `fmt.Errorf("%w: chat %s", not.Found, id)` **only** when `err == sql.ErrNoRows` (line 695-696); all other header-scan errors become `fmt.Errorf("query chat: %w", err)` (line 698), and message-load errors become `fmt.Errorf("get messages: %w", err)` (line 711). The `%w` chain is preserved.
- Consumer `Server.chat` (app/ui/ui.go:728-734) aborts with `return err` unless `errors.Is(err, not.Found)`, in which case it creates a fresh chat. This is the R-04/R-06 gate.

## Requirement findings
- **R-01** — Only an absent header row is `not.Found`. `database.go:695-696` gates the sentinel on `sql.ErrNoRows`; the malformed-message path (line 948) and query path (line 698) never produce it. Verified by `TestChatReadRejectsMalformedMessage` (store test:90-92) and the fault phase (ui test:226-228).
- **R-02** — Underlying chain preserved. `TestChatReadPreservesDatabaseErrorCause` asserts `errors.Is(err, cause)` against a real closed-DB `Ping` error (store test:139-140); `TestChatReadInitialisationFailureIsNotAbsence` asserts `errors.As(err, &cause)` for an `*os.PathError` (store test:110). Both passed on candidate.
- **R-03** — Failed-load continuation leaves records unchanged. The fault phase snapshots all four tables via raw SQL and compares before/after (ui test:229-244); candidate logged `all_records_unchanged=true`. The snapshot helper fails on empty tables (ui test:364-366), so equality is non-vacuous.
- **R-04** — Error before inference. Fault phase asserts status 500, a non-empty error body, and `hits.Load()==0` (ui test:231-240); candidate logged `http=500 model_requests=0`.
- **R-05** — Valid continuation retains earlier messages. Recover phase asserts 4 messages with exact contents and that restored history reaches the model request (ui test:266-294). Passed.
- **R-06** — Genuinely absent continuation creates and continues. Positive phase asserts `errors.Is(err, not.Found)` for a new id, then two successful continuations yielding 4 messages (ui test:296-317). Passed.

## Mutation results (all requested and executed)
| Case | Exit | Behavioural failure observed |
|---|---|---|
| rewrap | 1 | `R-01: scan failure became absence`; `R-02: database error identity lost`; fault `http=200 model_requests=3 all_records_unchanged=false` |
| consumer-bypass | 1 | fault `R-04: status=200 ... want 500`; `inference reached 3 times`; `persisted records changed` |
| always-error | 1 | `valid read failed: deliberate always-error getter`; `R-06: new-chat control was not genuinely absent` |
| empty-success | 1 | `R-01: malformed message returned success`; `R-05: valid chat changed`; `R-06 ... not genuinely absent` |

Each mutant is a genuine, targeted edit of the exact production function (rewrap store.go:433-436; consumer-bypass ui.go:729-731; always-error store.go:433; empty-success store.go:433). Failures are behavioural, not instrument failures: the instrument reported healthy (no build failures, no missing/unfinished tests, no skips) in every run, and the failing assertions name the specific requirement.

## Instrument audit
The proof runner uses Go overlays (never replacing working files), a fresh `HOME`/`LOCALAPPDATA`, `GOPROXY=off`, `GOTOOLCHAIN=local`, and `-count=1`. It records source digests before/after and rejects unhealthy instruments. The store fixture (`setupTestStore`, store_test.go:270-287) uses a real temp SQLite DB; the UI fixture uses the real authenticated router (`Dev:false`, token cookie) and a real `httptest` model server. Inference is simulated; SQLite, the router, and process separation are real.

## Limitations
- This is a bounded mechanism/test review only. Whole-programme lint, full regression, packaged-app, and release gates are out of scope and not approved.
- Cloud tag provenance is observed, not immutable model-weight identity.
- The six recovered Discovery files are pre-existing checkout content, not a new feature; I did not treat them as evidence of a redesign.
- I did not run the full `go test ./...` suite; only `-run=^TestChatRead` was exercised.

## Verdict
All six requirements are implemented in the inspected source, all six required cases executed, the candidate passed twice (10/10), and all four mutants failed behaviourally with requirement-specific assertions. No code defect and no unexecuted required case was found.

REPAIR_REVIEW=PASS