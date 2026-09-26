# OQ-3 baseline, revision 3: G1's failure number on the unchanged candidate

Continuation 15, Treadle (Claude Opus 5.5), 27 September 2026, run 08:23:34–08:26:41 AEST. Supersedes `../../proof_cont14/oq3-baseline-r2/` (revision 2) as the baseline the instrument is held to; that figure (66/39) was measured on a ledger without T8's new turn and stays true of that ledger. Instrument revision 3 (`../../proof_cont14/oq3-instrument/`, design in its `DESIGN.md`, repair record in `../oq3-repair-r2/REPAIR.md`), committed before the run; the run's `HEAD` was `34716b26011c4ae116b3f3fb87e08381f7f61ad4`, which also carries the prediction below. Candidate: that commit's source, identical outside `docs/` to Letterlock's `276d92e6` and Thole's G1 design tip `a7c35fd6`; binary SHA-256 `0a882f5598d4c82c7dda0794cb99ca13b0489ff2e0ad1c586c9704b2a1f010f1`, byte-identical to the revision-2 baseline's. Ledger SHA-256 `fc8e6beac4ce6a68b59ce41e3c59325d5c11d1ee36ec51c2ffa421e13b14af23` (27 scenarios, 39 trials). go1.26.5 darwin/arm64, Ash's Mac.

## Result

**Failure number 67/39 = 1.7179. The run is valid, and every auxiliary check passes.** 84 deliveries were observed: 67 incorrect, 17 correct, none missing, none unexpected or unattributed. **This is the figure predicted before the run** (`../oq3-repair-r2/PREDICTION.md`, committed at `34716b26`), in every component the prediction named.

The difference from revision 2 is one delivery: T8's new turn, sent at idle after the held turn and before the idle reload, which requires revision 1 at generation 1 and which the candidate sends with neither carrier nor admission field. Every other trial reads as in revision 2. Of the 67 incorrect deliveries, 65 are every delivery that should carry saved instructions; each lacks the carrier and the admission field (categories `admission-missing` 65, `missing` 65). The other two are the capability-less cases (D11, T10): the candidate, which knows nothing of the capability, sent the turn anyway (`dispatched-despite-required-refusal` 2). T10's session preloaded and went idle as before, so it took the ordinary path; no start-up refusal occurred. All 17 deliveries that should carry no instructions are correct, and the seven pure control trials read zero (D0-C, D4-U, D4-W, D4-Z, T0-X, T4-Y, X3-DE).

| Family | Trials | Observed | Incorrect | Missing | Number |
|---|---|---|---|---|---|
| save/restart/new | 15 | 20 | 20 | 0 | 20/15 = 1.3333 |
| edit/continue | 6 | 15 | 15 | 0 | 15/6 = 2.5000 |
| explicit reload | 5 | 15 | 14 | 0 | 14/5 = 2.8000 |
| disable | 11 | 22 | 6 | 0 | 6/11 = 0.5455 |
| terminal reset | 1 | 2 | 2 | 0 | 2/1 = 2.0000 |
| compaction | 1 | 10 | 10 | 0 | 10/1 = 10.0000 |

**Auxiliary checks.** 16 preloads and 4 compaction summariser requests equal their frozen goldens and carry no instructions. The decoded leak scan found no instruction text in any body, path, query string or header on any path. All 291 requests in run 1 used a method the real daemon registers for their path (none refused 405); every request carried only `Accept`, `Accept-Encoding`, `Content-Length`, `Content-Type` and `User-Agent` headers, so nothing needed redacting. No request used an unknown endpoint.

**Operations.** Every designated G1 surface is absent: 21 desktop saves (HTML fallback with status 200), 21 CLI saves (unknown command), 5 desktop reloads (HTML fallback), 6 terminal `/instructions` commands (unknown command), and the designated store (2 identity checks, 1 event-sequence check). The existing `create-chat` allocation passed; the 13 existing terminal commands did what their steps declare, as the state assertions on the goldens confirm. Required refusals: the desktop reload during a held turn is absent (HTML); the terminal reload was sent while T8's held turn ran, and both that turn's continuation and the ordinary turn after it are scored as deliveries. Every terminal turn, command, start and model switch ended on the chat's own idle proof, not on a quiet screen.

## What this establishes, and what it does not

It establishes, on the declared fixture and through the real entry points, that the candidate delivers saved instructions in none of the 65 cases that require them, ignores the admission capability in both refusal cases, and leaves every no-carrier case as it was; and that every lifecycle step the ledger names happened and reached its declared state. It freezes 102 pre-G1 requests as goldens (`../../proof_cont14/oq3-instrument/goldens/`: 82 deliveries, 16 preloads, 4 summariser requests, with `MANIFEST.sha256`), against which a G1 implementation is measured; the revision-2 goldens are kept as `goldens-superseded-39ff30a1/`.

It does not establish that revision 3 is fit to be the failure-number component of G1's acceptance gate: that is the round-3 review's question. It is that component, not the gate: see DESIGN.md's scope and limits and its six blueprint gaps.

## Controls and repeat

Gate self-test: `run.sh` blocked a failing test, a missing required test and an empty selection. Scorer, runner, auxiliary and harness controls passed (`controls.jsonl`), including every control added in revision 3. The reference arm sent the expected request for all 82 dispatch deliveries through real HTTP and read all 82 as correct; it checks transport and normalisation only. Run 2 compared against the goldens run 1 froze and was identical (`repeat-comparison.txt`). No tracked file changed; `go.mod` kept its hash (`afdaa038…`).

## Files

`run.log`, `environment.json`, `build.log`, `vet.log`, `overlay.json`, `instrument.sha256`, `selftest-*.jsonl`, `controls.jsonl`, `run1/` and `run2/` (ledger, summary, reference summary, per-scenario results with every capture), `repeat-comparison.txt`, `finished.txt`, and `MANIFEST.sha256`.
