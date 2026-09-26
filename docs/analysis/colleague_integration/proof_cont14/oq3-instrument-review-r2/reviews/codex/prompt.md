You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/. Change nothing and run nothing that writes. Cite file:line for every finding.

---

# Review round 2: the repaired OQ-3 failure-number instrument (Ollama fork, G1)

**What you are asked to do.** Round 1 of this review found fourteen blocking defects in the instrument that will become G1's failure-number gate; all were accepted and repaired. Review revision 2 as a fresh, adversarial test reviewer. Report findings only, each with file:line, a concrete failure scenario and a severity: BLOCKING (a false zero, a false alarm on a correct implementation, a claim in the reports that the evidence contradicts, or a round-1 repair that is missing or incomplete) or NON-BLOCKING. Say plainly if you find nothing blocking. Do not rewrite the instrument.

## What to read

- Round 1: `docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review/reviews/codex/last_message.md` (the findings) and `.../oq3-instrument-review/DISPOSITIONS_CODEX.md` (what was decided for each).
- The repair: `git diff 1869da26 e23aeb5d -- docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` (revision 1 to revision 2), and the files themselves in `docs/analysis/colleague_integration/proof_cont14/oq3-instrument/`: `DESIGN.md`, `run.sh`, `oq3_*_test.go`, `goldens/`.
- The revision-2 baseline: `docs/analysis/colleague_integration/proof_cont14/oq3-baseline-r2/` (`BASELINE.md`, `run.log`, `run1/summary.json`, `run1/scenarios/*.json`, `repeat-comparison.txt`).
- The design it measures: `docs/_design/G1_PERSISTED_INSTRUCTIONS.md` (line 18; §4; §5; §6.3; §7; §9).
- Source it drives: `app/ui/`, `app/store/`, `cmd/`, `cmd/tui/chat/`, `agent/`, `api/`.

The instrument compiles into package `cmd` only through `go test -overlay` (see `run.sh`); reason from the code and the retained evidence.

## Questions

1. For each of round 1's fourteen blocking findings: is the repair present, complete and correct? Cite the code.
2. Did the repair introduce a new false zero, a new false alarm, or a new unsupported claim? Consider in particular the held-turn label handling, the completion rule, the summariser and preload goldens, the leak scan, the state assertions, the store checks, the capability-less scenarios, and the acceptance-mode verdict allowances.
3. Is every figure in `oq3-baseline-r2/BASELINE.md` supported by the evidence? Recompute where you can.
4. Anything else that would make this instrument unfit to be the failure-number component of G1's acceptance gate.
