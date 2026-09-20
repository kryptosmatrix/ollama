# Scoped Review: app/.gitignore repair

**Change under review.** Baseline line 7 was `*cover*`; candidate replaces it with explicit coverage-artifact patterns: `coverage/`, `*.coverprofile`, `coverage*.out`, `coverage*.html`, `cover.out`, `cover.html` (lines 7–12). All other baseline lines are preserved verbatim (`ollama.syso`, `*.crt`, `*.exe`, `/app/app`, `/app/squirrel`, `ollama`, `.vscode`, `.env`, `.DS_Store`, `.claude`). Candidate rules SHA-256 matches the stated `384f341e…cda908`.

**Fresh verification (I requested it; it executed).** Real `git check-ignore --no-index -v` results:
- Source paths now visible (exit 1, not ignored): `FutureDiscovery.tsx`, `recoverConversation.ts`, `recover_history_test.go`. Baseline controls confirm the old `*cover*` rule *did* ignore all three (exit 0, `app/.gitignore:7:*cover*`). The false-positive is genuinely fixed.
- Coverage outputs still ignored: `coverage/index.html` (rule 7), `coverage.out` (rule 9), `test.coverprofile` (rule 8), `cover.html` (rule 12).
- Credential/other exclusions intact: `.env` (rule 14), `unsafe.crt` (rule 2).

All nine candidate cases and three baseline controls passed. This is real Git, not a simulated matcher.

**Honest limits.** The nine paths are a curated set, not an exhaustive glob proof. `coverage*.out`/`coverage*.html` are broader than the old rule for those suffixes, but they cannot match `Discovery`/`recover` source names, so no source-visibility regression is introduced. I did not enumerate every file in the tree; the claim is bounded to the tested paths plus rule inspection.

**Acceptance boundary (app_result).** `go test -count=1 -json -timeout=180s ./app/...` at head `13139f8d…` exited 1: 300 pass, 1 fail, 0 skip. The single failure is `app/tts` (105.5s) — an actual Keychain-authorisation failure. Per instruction I did not retry it and do not recommend bypassing macOS consent. `source_unchanged: true`; the Keychain scope note confirms the TTS test uses a test-scoped service, not production Ollama services. The suite is **not passing**.

**Lint summary (lint_delta tail).** `main-baseline` run: exit_code 1, 137 errors, 11 warnings. Tail fields: `messages_identical: true`, `affected_source_bytes_identical: true`, scope: "Matching failures are baseline debt, not a passing lint result or a release verdict." Lint is byte-identical to baseline but **remains a failure** — it is not a pass.

**Verdict.** The config change itself is correct and scoped: it restores source visibility while preserving coverage-output and credential exclusions. But the recorded test run fails (Keychain) and lint fails (baseline-identical). I do not approve integration or release of the whole candidate.

GITIGNORE_REVIEW=PASS
INTEGRATION_GATE=BLOCKED