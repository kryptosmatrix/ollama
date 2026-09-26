# Review round 3: the OQ-3 failure-number instrument, revision 3 (Ollama fork, G1)

**What you are asked to do.** You are a fresh reviewer: no earlier reviewer's context is shared with you, and the record named below is the only continuity. Rounds 1 and 2 found fourteen and six blocking defects in this instrument, which will become the failure-number component of G1's acceptance gate; all were accepted and repaired. Review revision 3 as a fresh, adversarial test reviewer. Report findings only, each with file:line, a concrete failure scenario and a severity: BLOCKING (a false zero, meaning a wrong G1 implementation that reads 0 in a valid accept run; a false alarm, meaning a correct implementation that fails or makes the run invalid; a claim in the records that the evidence contradicts; or a round-2 repair missing or incomplete) or NON-BLOCKING. Say plainly if you find nothing blocking. Do not rewrite the instrument.

**Lenses.** Declared for this loop (rounds 1 and 2 applied them without naming them): false zero; false alarm; the evidence supports the claims; repair completeness.

## The record (continuity lives here, not in any reviewer)

- Round 1: findings `docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review/reviews/codex/last_message.md`; dispositions `.../oq3-instrument-review/DISPOSITIONS_CODEX.md`.
- Round 2: findings `.../proof_cont14/oq3-instrument-review-r2/reviews/codex/last_message.md`; dispositions `.../oq3-instrument-review-r2/DISPOSITIONS_CODEX.md` (all six accepted; repairs proposed there).
- Settled, not to be re-derived: the blueprint's §7 rule that `ollama instructions set` saves the file's text as an enabled profile (continuation 14), and the admission contract of §6.3 (A1-A13). To reopen a settled ruling, attack its recorded reasoning.

## What to read

- The repair record, `docs/analysis/colleague_integration/proof_cont15/oq3-repair-r2/REPAIR.md`: what was built for each finding, the control that shows it working, the deliberate break that makes that control fail, and the technical decisions with the alternatives they rejected. Overturn any decision you find wrong. `PREDICTION.md` beside it was committed before the baseline ran.
- The repair, measured rather than reported: `git diff 276d92e6 5f1c7365 -- docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` (run.sh is unchanged; `a12e28d9` is whitespace only, so `git show -w a12e28d9` shows no changed line; `5f1c7365` changes one comment and one citation; the goldens trees are measurement output).
- Its blast radius: everything that reads a definition the diff moved, in particular `oq3Capture`, the grouping key and `under` in `score`, `typeLine`, `idle` and `mustIdle`, the `terminal-start` step and its capability-less path, `oq3KnownPaths` and `oq3RouteMethods`, the leak helpers, and every call site of each.
- The instrument as it stands: `docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` (`DESIGN.md`, `run.sh`, `oq3_*_test.go`, `goldens/` with `MANIFEST.sha256`).
- The revision-3 baseline: `docs/analysis/colleague_integration/proof_cont15/oq3-baseline-r3/` (`BASELINE.md`, `run.log`, `run1/summary.json`, `run1/scenarios/*.json`, `repeat-comparison.txt`, `controls.jsonl`, `MANIFEST.sha256`).
- The deliberate-break matrix: `docs/analysis/colleague_integration/proof_cont15/oq3-repair-r2/mutations/` (`mutate.py`, `matrix.json`, one log per mutation).
- The design it measures: `docs/_design/G1_PERSISTED_INSTRUCTIONS.md` (line 18; §4; §5; §6.3 A12; §7; §8; §9).
- Source it drives: `app/ui/`, `app/store/`, `cmd/`, `cmd/tui/chat/` (the idle proof relies on `chat.go`, `render.go`, `input.go`, `approval.go` and `modals.go`), `agent/`, `api/`, `server/routes.go`.

The instrument compiles into package `cmd` only through `go test -overlay` (see `run.sh`); reason from the code and the retained evidence.

## Questions

1. For each of round 2's six findings: is the repair present, complete and correct? Cite the code.
2. Did revision 3 introduce a new false zero, a new false alarm or an unsupported claim? Consider in particular: the idle proof (can its text appear while a turn, compaction, preload or approval prompt is active; can the probe itself change what the chat sends, or make it quit; what happens if a picker is open); the ctrl+u sent before every typed line; the start-up refusal rule (can an unrelated failure pass as a refusal; can a correct refusal fail); the method table against the real route table; wrong-method scoring and golden freezing; the decoded leak scan and header redaction; and T8's new turn (can a correct implementation fail it; does it catch the fault it was added for).
3. Is every figure in `oq3-baseline-r3/BASELINE.md` supported by the evidence? Recompute where you can, and compare it with `PREDICTION.md`.
4. Does the deliberate-break matrix show what `REPAIR.md` says it shows? Would a control still pass if its repair were broken in a way the matrix did not try?
5. Anything else that would make revision 3 unfit to be the failure-number component of G1's acceptance gate.
