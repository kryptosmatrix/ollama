# OQ-3 baseline, revision 2: G1's failure number on the unchanged candidate

Continuation 14, Letterlock (Claude Opus 5.5), 26 September 2026, finished 23:40:56 AEST. Supersedes `../oq3-baseline/` (revision 1), whose instrument a Codex review found unfit (`../oq3-instrument-review/DISPOSITIONS_CODEX.md`). Instrument revision 2 (`../oq3-instrument/`, design in its `DESIGN.md`), committed at `e23aeb5d71ee7e6babc03c6137c13b1d2928d3a4` before the run. Candidate: that commit's source, identical outside `docs/` to Thole's G1 design tip `a7c35fd6`; binary SHA-256 `0a882f5598d4c82c7dda0794cb99ca13b0489ff2e0ad1c586c9704b2a1f010f1`, byte-identical to Eko's continuation-12 build. Ledger SHA-256 `639b9e1a7e94c2f018ec5cdf8db6ffbb9e7237c09c695d92d0f3d3c184913698` (27 scenarios, 39 trials). go1.26.5 darwin/arm64, Ash's Mac.

## Result

**Failure number 66/39 = 1.6923. The run is valid, and every auxiliary check passes.** 83 deliveries were observed: 66 incorrect, 17 correct, none missing, none unexpected or unattributed.

Of the 66 incorrect deliveries, 64 are every delivery that should carry saved instructions; each lacks the carrier and the admission field. The other two are the capability-less cases (D11, T10): G1 requires that no carrier-bearing request reach a daemon without `chat.admission.v1`, and the candidate, which knows nothing of the capability, sent the turn anyway. All 17 deliveries that should carry no instructions are correct: the unconfigured controls, the conversations started or reselected after a disable or reset, and the legacy transcript before adoption. The seven pure control trials read zero (D0-C, T0-X, D4-Z, D4-W, D4-U, T4-Y, X3-DE).

| Family | Trials | Observed | Incorrect | Missing | Number |
|---|---|---|---|---|---|
| save/restart/new | 15 | 20 | 20 | 0 | 20/15 = 1.3333 |
| edit/continue | 6 | 15 | 15 | 0 | 15/6 = 2.5000 |
| explicit reload | 5 | 14 | 13 | 0 | 13/5 = 2.6000 |
| disable | 11 | 22 | 6 | 0 | 6/11 = 0.5455 |
| terminal reset | 1 | 2 | 2 | 0 | 2/1 = 2.0000 |
| compaction | 1 | 10 | 10 | 0 | 10/1 = 10.0000 |

**Auxiliary checks.** 16 preloads and 4 compaction summariser requests equal their frozen goldens and carry no instructions; no other request on any path carried instruction text; no request used an unknown endpoint.

**Operations.** Every designated G1 surface is absent: 21 desktop saves (HTML fallback with status 200), 21 CLI saves (unknown command), 5 desktop reloads (HTML fallback), 6 terminal `/instructions` commands (unknown command), and the designated store (2 identity checks, 1 event-sequence check). The existing `create-chat` allocation passed; the 13 existing terminal commands (`/system`, `/tools`, `/new`, `/compact`, `/model`) did what their steps declare, as the state assertions on the goldens confirm. Required refusals: the desktop reload during a held turn is absent (HTML); the terminal reload was sent while T8's held turn ran, and that turn's tool continuation is scored as a delivery.

## What this establishes, and what it does not

It establishes, on the declared fixture and through the real entry points, that the candidate delivers saved instructions in none of the 64 cases that require them, ignores the admission capability in both refusal cases, and leaves every no-carrier case as it was; and that every lifecycle step the ledger names happened and reached its declared state: four compactions, three tool continuations with approvals (two in T1, one in T8's held turn), the tools toggle and two model switches, the built-in prompt toggled, a failed first turn, two held turns with a save from another process during each, restarts of both hosts, a legacy transcript, and the built binary's launcher path. It freezes 101 pre-G1 requests as goldens (`../oq3-instrument/goldens/`: 81 deliveries, 16 preloads, 4 summariser requests), against which a G1 implementation is measured.

It is the failure-number component of G1's combined gate, not the gate: see DESIGN.md's scope and limits, and its five blueprint gaps.

## Controls and repeat

Gate self-test: `run.sh` blocked a failing test, a missing required test and an empty selection (the last two with Go's exit status 0, caught only by reconciling its JSON test events). Scorer, runner, auxiliary and harness controls passed (`controls.jsonl`). The reference arm sent the expected request for all 81 dispatch deliveries through real HTTP and read all 81 as correct; it checks transport and normalisation only. Run 2 compared against the goldens run 1 froze and was identical (`repeat-comparison.txt`). No tracked file changed; `go.mod` kept its hash.

## Files

`run.log`, `environment.json`, `build.log`, `vet.log`, `overlay.json`, `instrument.sha256`, `selftest-*.jsonl`, `controls.jsonl`, `run1/` and `run2/` (ledger, summary, reference summary, per-scenario results with every capture), `repeat-comparison.txt`, `finished.txt`, and `MANIFEST.sha256`.
