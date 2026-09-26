# OQ-3 baseline, revision 4: G1's failure number on the unchanged candidate

Continuation 15, Treadle (Claude Opus 5.5), 27 September 2026, run 09:16:47–09:19:40 AEST. Supersedes `../oq3-baseline-r3/` (revision 3) as the baseline the instrument is held to. Instrument revision 4 (`../../proof_cont14/oq3-instrument/`, design in its `DESIGN.md`, repair record in `../oq3-repair-r3/REPAIR.md`), committed at `a88ee4f2` before the run; the run's `HEAD` was `7fd9a579b7da493cc4acc532493f4acd7446182f`, which carries the prediction (`../oq3-repair-r3/PREDICTION.md`). Candidate: that commit's source, identical outside `docs/` to Letterlock's `276d92e6` and Thole's G1 design tip `a7c35fd6`; binary SHA-256 `0a882f5598d4c82c7dda0794cb99ca13b0489ff2e0ad1c586c9704b2a1f010f1`, byte-identical to the revision-2 and revision-3 baselines'. Ledger SHA-256 `fc8e6beac4ce6a68b59ce41e3c59325d5c11d1ee36ec51c2ffa421e13b14af23` (27 scenarios, 39 trials; unchanged from revision 3). go1.26.5 darwin/arm64, Ash's Mac.

## Result

**Failure number 67/39 = 1.7179. The run is valid, and every auxiliary check passes.** 84 deliveries were observed: 67 incorrect, 17 correct, none missing, none unexpected or unattributed. **This is the figure predicted before the run, in every component the prediction named**, and every per-trial, per-family, category and operation count is identical to the revision-3 baseline: revision 4 changed what the driver waits for and what the scanner reads, not what the candidate sends. All 102 goldens frozen by this run carry byte-identical requests to the 102 revision 3 froze (`goldens-superseded-01b86567/`).

Of the 67 incorrect deliveries, 65 are every delivery that should carry saved instructions; each lacks the carrier and the admission field (`admission-missing` 65, `missing` 65). The other two are the capability-less cases (D11, T10): the candidate sent the turn anyway (`dispatched-despite-required-refusal` 2). T10's session preloaded and went idle, so it took the ordinary path. All 17 deliveries that should carry no instructions are correct, and the seven pure control trials read zero.

| Family | Trials | Observed | Incorrect | Missing | Number |
|---|---|---|---|---|---|
| save/restart/new | 15 | 20 | 20 | 0 | 20/15 = 1.3333 |
| edit/continue | 6 | 15 | 15 | 0 | 15/6 = 2.5000 |
| explicit reload | 5 | 15 | 14 | 0 | 14/5 = 2.8000 |
| disable | 11 | 22 | 6 | 0 | 6/11 = 0.5455 |
| terminal reset | 1 | 2 | 2 | 0 | 2/1 = 2.0000 |
| compaction | 1 | 10 | 10 | 0 | 10/1 = 10.0000 |

**Auxiliary checks.** 16 preloads and 4 compaction summariser requests equal their frozen goldens and carry no instructions. Every body walked cleanly, with no duplicate member. The scan found no instruction text in any body, path, request target, query, `Host`, header or trailer. All 291 requests in run 1 went to the fake's loopback host, none carried a trailer, and none was refused for its method.

**Operations.** As in revision 3: every designated G1 surface is absent (21 desktop saves, 21 CLI saves, 5 desktop reloads, 6 terminal `/instructions` commands, and the designated store's 3 checks); `create-chat` passed; the 13 existing terminal commands did what their steps declare. T8's refused reload was handled while its turn was held (the marker reached the chat's input box before the release), and both that turn's continuation and the ordinary turn after it are scored. Every preload finished at the daemon before its step's idle proof, and every terminal step ended on that proof. *(Correction, round 4, B1: the barrier checked that each preload's `GET /api/ps` had arrived, not that it had been answered; "finished" overstates it.)*

## What this establishes, and what it does not

It establishes, on the declared fixture and through the real entry points, the same measurement of the unchanged candidate as revision 3, taken by the repaired instrument, and it freezes the 102 goldens (`../../proof_cont14/oq3-instrument/goldens/`, with `MANIFEST.sha256`) that a G1 implementation is measured against. It does not establish that revision 4 is fit to be the failure-number component of G1's acceptance gate: that is the round-4 review's question.

## Controls and repeat

Gate self-test: `run.sh` blocked a failing test, a missing required test and an empty selection. Scorer, runner, auxiliary and harness controls passed (`controls.jsonl`), including every control added in revision 4. The reference arm read all 82 dispatch deliveries as correct. Run 2 compared against the goldens run 1 froze and was identical (`repeat-comparison.txt`). No tracked file changed; `go.mod` kept its hash.

## Files

`run.log`, `environment.json`, `build.log`, `vet.log`, `overlay.json`, `instrument.sha256`, `selftest-*.jsonl`, `controls.jsonl`, `run1/` and `run2/`, `repeat-comparison.txt`, `finished.txt`, and `MANIFEST.sha256`.
