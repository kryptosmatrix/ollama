# Review round 3, repair reader: attack the repair itself (OQ-3 instrument revision 3)

**Your only job is to attack the repair.** Another reviewer is judging the instrument as a whole; you are pointed at the change. A judge reads new text as text, and a repair that overshoots reads as ordinary specification; only a reader pointed at the change sees it as a change.

Read the measured diff, not the author's account of it: `git diff 276d92e6 5f1c7365 -- docs/analysis/colleague_integration/proof_cont14/oq3-instrument/` (`a12e28d9` is whitespace only; the `goldens/` and `goldens-superseded-*/` trees are measurement output, to be read only to check a claim). Then read the author's account, `docs/analysis/colleague_integration/proof_cont15/oq3-repair-r2/REPAIR.md`, and the findings it answers, `docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review-r2/reviews/codex/last_message.md`.

Find: where the account overstates what the code does; where the code does something the account does not mention; where a repair repeats elsewhere the error it corrects (another place that still infers completion from silence, reads raw bytes, ignores a method, or treats an unexplained exit as success); and where a repair broke something that used to work. The source the instrument drives is readable (`cmd/tui/chat/`, `server/routes.go`, `app/ui/`, `agent/`, `api/`).

Report findings only, each with file:line, a concrete failure scenario and a severity: BLOCKING as round 2's packet defines it (`docs/analysis/colleague_integration/proof_cont14/oq3-instrument-review-r2/PACKET.md`) or NON-BLOCKING. Say plainly if you find nothing blocking. Do not rewrite the instrument.
