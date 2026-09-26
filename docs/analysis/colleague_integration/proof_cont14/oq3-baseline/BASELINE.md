# OQ-3 baseline: G1's failure number on the unchanged candidate

Continuation 14, Letterlock (Claude Opus 5.5), 26 September 2026, 23:00:29–23:03:04 AEST. Instrument: `../oq3-instrument/` (design in its `DESIGN.md`), committed at `1869da26cdf87c9511e71e3eb98507e05d888b79` before the run. Candidate: that commit's source, which differs from Thole's G1 design tip `a7c35fd6` only under `docs/`; binary SHA-256 `0a882f5598d4c82c7dda0794cb99ca13b0489ff2e0ad1c586c9704b2a1f010f1`, byte-identical to the binary Eko built for the continuation-12 CLI baseline. Ledger SHA-256 `71982d99af9798ff8f62fe0f185adf298727010779df3da7f25fa94acb1390af` (25 scenarios, 36 trials). go1.26.5 darwin/arm64, Ash's Mac.

## Result

**Failure number 61/36 = 1.6944** (bad observed deliveries + missing required deliveries, over controlled conversation trials). **The run is valid.** 78 deliveries were observed, 61 incorrect and 17 correct; none was missing and no request was unexpected or unattributed.

Every one of the 61 incorrect deliveries has the same two defects: the carrier is absent and the admission field is absent. They are exactly the deliveries whose expectation is a carrier: nothing of G1 exists, so no request carries saved instructions. Every one of the 17 deliveries whose expectation is no carrier is correct. They are the unconfigured controls (D0, T0), the conversations started or reselected after a disable or reset (D4-Z, D4-W, D4-U, D4-X after its reload, T4, X3), and the legacy transcript before adoption (D5-L). All seven pure control trials read zero: D0-C, T0-X, D4-Z, D4-W, D4-U, T4-Y, X3-DE.

| Family | Trials | Observed | Incorrect | Missing | Number |
|---|---|---|---|---|---|
| save/restart/new | 12 | 17 | 17 | 0 | 17/12 = 1.4167 |
| edit/continue | 6 | 15 | 15 | 0 | 15/6 = 2.5000 |
| explicit reload | 5 | 13 | 12 | 0 | 12/5 = 2.4000 |
| disable | 11 | 22 | 6 | 0 | 6/11 = 0.5455 |
| terminal reset | 1 | 1 | 1 | 0 | 1/1 = 1.0000 |
| compaction | 1 | 10 | 10 | 0 | 10/1 = 10.0000 |

Per-trial counts are in `run1/summary.json` (`per_trial`); every delivery's verdict, the request it was scored on and its golden are in `run1/scenarios/<id>.json`.

**Operations.** Every designated G1 surface is absent: 20 desktop saves (`PUT /api/v1/instructions` returns the application's HTML with status 200), 19 CLI saves (`ollama instructions set` is an unknown command), 5 desktop reloads (HTML fallback), 5 terminal `/instructions` commands (unknown command), and the designated instruction store (2 identity checks). The existing `create-chat` allocation passed. Of the two required refusals, the desktop reload during a held turn is absent (HTML), and the terminal reload during a held turn is **unobserved**: while a turn runs the screen only animates, and the blueprint names no terminal output for it. The four compaction summariser requests met their own expectation.

## What this establishes, and what it does not

It establishes, on the declared fixture and through the real entry points, that the candidate delivers no saved instructions in any of the 61 required cases, that it leaves every no-carrier case as it was, and that every lifecycle step the ledger names actually happened: four compactions (each shown by the next request's history), two tool continuations with approvals, two model switches, the built-in prompt toggled, a failed first turn, two held turns with a save from another process during each, restarts of both hosts, a legacy transcript and the built binary's launcher path. It freezes the 78 pre-G1 requests as goldens (`../oq3-instrument/goldens/`), against which a G1 implementation is measured.

It does not establish anything about the model's compliance, the daemon's handling after the capture point, the desktop's tool pass-loop continuation, the web-search family, tool-output-triggered compaction, `/new` failure injection, storage guards, the browser journey, or real models. Those, and five blueprint gaps the instrument exposed, are in `../oq3-instrument/DESIGN.md`.

## Controls and repeat

Gate self-test: `run.sh` blocked a failing test, a missing required test and an empty selection; for the last two Go's own exit status was 0, and only the reconciliation of its JSON test events caught them. Scorer and runner controls passed (`controls.jsonl`). The reference arm sent the expected request for all 78 deliveries through real HTTP and read all 78 as correct. Run 2 compared against the goldens run 1 froze, and its per-trial verdicts, counts, categories and operation verdicts were identical (`repeat-comparison.txt`). No tracked file changed during the run, and `go.mod` kept its hash.

## Files

`run.log` (the gate's own log), `environment.json`, `build.log`, `vet.log`, `overlay.json`, `instrument.sha256`, `selftest-*.jsonl` and `controls.jsonl` (Go test event streams), `run1/` and `run2/` (ledger, summary, reference summary, per-scenario results with every capture, the reference arm), `repeat-comparison.txt`, `finished.txt`, and `MANIFEST.sha256` over all of them.
