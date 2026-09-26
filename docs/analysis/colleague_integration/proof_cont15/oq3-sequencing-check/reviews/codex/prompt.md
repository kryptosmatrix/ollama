You are an independent adviser working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/, and run read-only commands. Change nothing and run nothing that writes. Cite file:line for every claim you rest on.

---

# Options check: what to do next with G1's OQ-3 instrument (Ollama fork)

**What you are asked to do.** Choose among the options below, or propose another, and say why. Say what evidence would change your choice. You are not reviewing the instrument's code for defects; you are advising on the order of work. Read the evidence first. Report a choice, the reasons, and the risks of your choice.

## Context

G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`) is a design at `[SPEC-DRAFT]`, not implemented. Its failure number (line 18) is measured by an instrument (`docs/analysis/colleague_integration/proof_cont14/oq3-instrument/`, design in `DESIGN.md`) that will become one component of G1's acceptance gate. Before promotion, OQ-4 (§0, line 35) must complete an independently derived closure census against Method 16 Tier 2, including the instrument's open blueprint gaps (`DESIGN.md`, "Blueprint gaps this instrument exposed", 1 to 7).

## Evidence

- The four review rounds of the instrument and their dispositions: `docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review/DISPOSITIONS_CODEX.md` (round 1), `.../proof_cont14/oq3-instrument-review-r2/DISPOSITIONS_CODEX.md` (round 2), `.../proof_cont15/oq3-instrument-review-r3/DISPOSITIONS_CODEX.md` (round 3) and `.../proof_cont15/oq3-instrument-review-r4/DISPOSITIONS_CODEX.md` (round 4), each with the reviewers' answers under `reviews/`.
- The blueprint's requirements for a daemon without the admission capability: §6.3 A12 (line 253), §7 and §8.
- The current baselines: `.../proof_cont15/oq3-baseline-r4/BASELINE.md`.

## Options

1. **Repair now.** Revision 5 of the instrument addresses all eleven round-4 defects, including A1 to A3, by choosing in the instrument a rule for recognising the terminal's refusal report and a position on whether no-dispatch trials require it; then a baseline and round 5; OQ-4 afterwards.
2. **Specify first.** OQ-4 states in the blueprint the terminal's refusal report for a missing capability, whether the failure number's no-dispatch trials require it, the desktop's refusal response, and closes the other open gaps; then revision 5 against the stated text, with the other round-4 repairs; a baseline; round 5.
3. **Split.** Revision 5 now for the eight round-4 defects that do not depend on the refusal specification (B1 to B3, C1, C2, D1, D2, E1), with the refusal scoring of T10 and D11 marked provisional; OQ-4 settles A1 to A3; a revision 6 applies them.
4. **Narrow the instrument.** Remove the report requirement and the start-up refusal outcome from the instrument (the no-dispatch trials score only the absent dispatch, leaving A12's report to the adapter-behaviour obligation I-14, which `DESIGN.md` already places outside the instrument), repair the rest now, and leave the report to OQ-4 and I-14.

## Questions

1. Which option, and why?
2. Do the round counts (14, 6, 8 and 11 distinct blocking defects) and the kinds of findings in rounds 3 and 4 indicate that the loop is converging, or not? Cite the dispositions.
3. What would change your choice?
