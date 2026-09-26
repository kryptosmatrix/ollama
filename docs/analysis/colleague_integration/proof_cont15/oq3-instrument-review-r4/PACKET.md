# Review round 4: the OQ-3 failure-number instrument, revision 4 (Ollama fork, G1)

**What you are asked to do.** You are a fresh reviewer: no earlier reviewer's context is shared with you, and the record named below is the only continuity. Rounds 1, 2 and 3 found fourteen, six and eight blocking defects in this instrument, which will become the failure-number component of G1's acceptance gate; all were accepted and repaired. Review revision 4 as a fresh, adversarial test reviewer. Report findings only, each with file:line, a concrete failure scenario and a severity: BLOCKING (a false zero, meaning a wrong G1 implementation that reads 0 in a valid accept run; a false alarm, meaning a correct implementation that fails or makes the run invalid; a claim in the records that the evidence contradicts; or a round-3 repair missing or incomplete) or NON-BLOCKING. Say plainly if you find nothing blocking. Do not rewrite the instrument.

**Lenses.** Declared for this loop: false zero; false alarm; the evidence supports the claims; repair completeness.

**Threat model, stated so findings can be weighed.** The instrument is a gate against G1 implementations written in good faith that are wrong, not against an adversary crafting requests to evade it. A finding that needs a deliberately evasive encoding is still reported, and marked as such; the declared limits (base64 and other non-JSON encodings are not decoded) stand unless you show an honest implementation path that they miss.

## The record (continuity lives here, not in any reviewer)

- Rounds 1 and 2: `docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review/` and `.../oq3-instrument-review-r2/` (each `reviews/codex/last_message.md` and `DISPOSITIONS_CODEX.md`).
- Round 3: `docs/analysis/colleague_integration/proof_cont15/oq3-instrument-review-r3/` (`reviews/codex-judge/last_message.md`, `reviews/codex-repair/last_message.md`, `DISPOSITIONS_CODEX.md`: eight defects, all accepted).
- Settled, not to be re-derived: the blueprint's §7 rule that `ollama instructions set` saves the file's text as an enabled profile, and the admission contract of §6.3 (A1-A13). To reopen a settled ruling, attack its recorded reasoning.

## What to read

- The repair record, `docs/analysis/colleague_integration/proof_cont15/oq3-repair-r3/REPAIR.md`, and `PREDICTION.md` beside it (committed before the baseline ran, with a marked correction of one fact in its header).
- The repair, measured rather than reported: `git diff 5eb77c90 HEAD -- docs/analysis/colleague_integration/proof_cont14/oq3-instrument/*.go docs/analysis/colleague_integration/proof_cont14/oq3-instrument/DESIGN.md` (revision 3 at the round-3 freeze to revision 4; `run.sh` is unchanged; the goldens trees are measurement output).
- Its blast radius: `oq3JSONScan`, `oq3StringLeaks`, `record`, `oq3BodyScan`, `oq3EnvelopeLeaks`, the body scans in `score`, `preloadDone` and `mustPreload` and every step that calls them, `terminalStartWithoutCapability`, the refused path of `terminalTurn`, the open-session path of `terminal-stop`, the marker in the required-refusal branch of `terminalSlash`, `oq3RefusalReported`, and every call site of each.
- The in-place corrections of two revision-3 records: `git show ffe0423d`.
- The revision-4 baseline: `docs/analysis/colleague_integration/proof_cont15/oq3-baseline-r4/` (`BASELINE.md`, `run.log`, `run1/summary.json`, `run1/scenarios/*.json`, `repeat-comparison.txt`, `controls.jsonl`, `MANIFEST.sha256`).
- The deliberate-break matrix: `docs/analysis/colleague_integration/proof_cont15/oq3-repair-r3/mutations/`.
- The design it measures: `docs/_design/G1_PERSISTED_INSTRUCTIONS.md` (line 18; §4; §5; §6.3 A12; §7; §8; §9).
- Source it drives: `app/ui/`, `app/store/`, `cmd/` (the preload function is `cmd/agent_tui.go` `preloadAgentModelIfLocal`), `cmd/tui/chat/` (`chat.go`, `render.go`, `input.go`, `approval.go`, `modals.go`), `cmd/tui/selector.go`, `agent/`, `api/`, `server/routes.go`.

The instrument compiles into package `cmd` only through `go test -overlay` (see `run.sh`); reason from the code and the retained evidence.

## Questions

1. For each of round 3's eight defects: is the repair present, complete and correct? Cite the code.
2. Did revision 4 introduce a new false zero, a new false alarm or an unsupported claim? Consider in particular: the preload barrier's dependence on the preload function's order of requests (can a correct G1 preload differently and be failed; can a request still arrive under the wrong label); the marker typed after T8's refused command (can it change what the chat sends, reach the wrong handler, or be read too early); the refusal vocabulary and the open-session path (can an unrelated failure still pass; can a correct refusal fail; can a request after the refusal escape scoring); duplicate members as defects (can a correct request trip it); the token walk and the envelope scan.
3. Is every figure in `oq3-baseline-r4/BASELINE.md` supported by the evidence? Recompute where you can, and compare it with `PREDICTION.md`.
4. Does the deliberate-break matrix show what `REPAIR.md` says it shows, including the three declared survivors?
5. Anything else that would make revision 4 unfit to be the failure-number component of G1's acceptance gate.
