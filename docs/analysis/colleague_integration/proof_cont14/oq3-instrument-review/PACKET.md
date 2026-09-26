# Review: the OQ-3 failure-number instrument and its pre-G1 baseline (Ollama fork, G1)

**What you are asked to do.** Review the built instrument and its first measurement as an adversarial test reviewer. It will become the acceptance gate for G1, a feature not yet implemented, so the question that matters most is whether a wrong G1 implementation could pass it. Report findings only, each with file:line, a concrete failure scenario, and a severity: BLOCKING (a false zero, a false alarm on a correct implementation, or a claim in the reports that the evidence contradicts) or NON-BLOCKING. Say plainly if you find nothing blocking. Do not rewrite the instrument.

## What to read

All paths are in the repository at the current HEAD. Read what you need; nothing under `docs/` is withheld except as stated.

- The instrument: `docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` — `DESIGN.md` (its declared rules, limits and gaps), `run.sh` (the gate), `oq3_harness_test.go` (helper processes, pseudo-terminal, fake daemon, drivers), `oq3_scenarios_test.go` (the frozen ledger), `oq3_runner_test.go` (execution, attribution, scoring, summary), `oq3_oracle_test.go` (normalisation, the expected request, the scorer), `oq3_main_test.go` (the top-level test, validity, the reference arm), `oq3_controls_test.go` (scorer and runner controls), `oq3_smoke_test.go`, and `goldens/` (78 frozen pre-G1 requests).
- The baseline: `docs/analysis/colleague_integration/proof_cont14/oq3-baseline/` — `BASELINE.md`, `run.log`, `run1/summary.json`, `run1/scenarios/*.json`, `repeat-comparison.txt`.
- The design it measures: `docs/_design/G1_PERSISTED_INSTRUCTIONS.md` (the failure number at line 18; lifecycle §4; composition §5; the admission contract §6.3; acceptance §9).
- The options check that shaped it, if useful: `docs/analysis/colleague_integration/proof_cont14/oq3-options-check/DISPOSITIONS_CODEX.md`.
- Source it drives: `app/ui/`, `app/store/`, `cmd/`, `cmd/tui/chat/`, `agent/`, `api/`.

The instrument compiles into package `cmd` only through `go test -overlay` (see `run.sh`). You cannot run it in a read-only sandbox; reason from the code and the retained evidence.

## Questions

1. **False zero.** Find every way a G1 implementation that delivers instructions wrongly could still score zero incorrect deliveries in a valid acceptance run (`run.sh accept`). Consider attribution by label, the summariser classification, golden loading and normalisation, the transform, the scorer's comparison, the completeness and validity checks, and what acceptance mode requires of operations and refusals.
2. **False alarm.** Find every way a correct G1 implementation (one that follows the blueprint's text) would score above zero or make the run invalid. Consider revision and generation numbering in each scenario, the expected base on each adapter, the admission field and its capability, the terminal commands' outputs, the CLI's semantics, and timing in the drivers.
3. **Claims.** Is every figure and statement in `BASELINE.md` and `DESIGN.md` supported by the code and the retained evidence? Recompute the counts from `run1/summary.json` and the scenario files where you can.
4. **The reference arm and the controls.** Do they show what `DESIGN.md` says they show, or is any of them circular or vacuous?
5. **Anything else** that would make this instrument unfit to be G1's acceptance gate.
