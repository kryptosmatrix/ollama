# OQ-3 — the G1 failure-number instrument

Continuation 14, Letterlock (Claude Opus 5.5), 26 September 2026. Design decided after a blind options check on Codex (`../oq3-options-check/`, every claim dispositioned in `DISPOSITIONS_CODEX.md`). This is design evidence for G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`); it changes no production or default test file and runs only through `go test -overlay`.

## What it measures

G1's failure number is "incorrect instruction deliveries per controlled conversation", over save/restart/new, edit/continue, explicit reload, disable, terminal reset and compaction; any value above zero fails G1 (blueprint line 18). The instrument publishes:

    failure number = (bad observed deliveries + missing required deliveries) / controlled conversation trials

with its components (bad observed, missing required, observed, trials), per-category counts (which overlap: one bad request can have several categories but counts once in the numerator), a per-family and per-trial breakdown, operation and required-refusal verdicts, and **a separate validity verdict**. A zero numerator is not acceptance: the run must also be valid and complete, and in acceptance mode every operation must pass on its own evidence.

## Boundary

The capture point is the inference daemon's input: a fake daemon at `OLLAMA_HOST` records every request. Three arms reach it:

- **Desktop:** the real `app/ui.Server.Handler`, assembled as `app/cmd/app/app.go:208-287` does (default store path under a sandbox `HOME`, tool registry, approvals, `Dev` false, a random token), in its own OS process per lifetime, driven over authenticated HTTP. A restart is a SIGTERM, an observed exit, then a new process on the same `HOME`.
- **Terminal lifecycle:** `launchInteractiveModel` (the function the launcher's run-model action calls, `cmd/cmd.go:2242`) → `prepareAgentModel` → `GenerateAgentTUI` → `agentchat.Run`, in its own OS process on a real pseudo-terminal, driven by keystrokes. CLI steps run the real root command (`NewCLI`) in their own processes.
- **Entry:** the built `ollama` binary with no arguments, through the launcher menu and model picker to the agent chat (scenario E1), and its CLI for the save.

Declared simulations (Method 04 T-07): inference is the fake daemon (scripted replies; a tool call to `read` where a turn asks for one; a summary for the compactor; injected failures, held responses and a high prompt count where a step declares them; the `chat.admission.v1` capability advertised and an A9-shaped `admission` object returned when a request carries the field). Declared omissions: the desktop's embedded daemon, MCP manager, updater and webview; the launcher's heartbeat, menu and model resolution on the lifecycle arm (the entry arm covers them). A closed-port proxy blocks every non-loopback HTTP(S) request from the children.

## The ledger

`oq3_scenarios_test.go` freezes 25 scenarios and 36 trials before execution (the run prints the ledger's SHA-256). Every step is declared as a **required dispatch** (with one expectation per model request it must cause, tool continuations included), a **required refusal**, a production **operation** with an expected outcome, or a **harness** action. Designated surfaces absent before G1 are attempted through their designated form (`PUT /api/v1/instructions`, `POST /api/v1/chat/{id}/instructions/reload`, `ollama instructions set --file --expected-revision`, `/instructions reload|off`); their real responses are recorded; later steps run anyway; no expectation is weakened because a save failed. A harness failure makes the run invalid and is never scored as a delivery defect.

Coverage: desktop D0–D10 (unconfigured controls with thinking, an attachment and web-search tools; save/restart/new; edit while continuing; reload and idempotent reload; disable, restart while disabled, restore and reset; a genuine legacy transcript with revision zero and adoption; restart and reopen; binding timing for a pre-allocated ID; a failed first turn, its retry and a new ID; a held turn with a concurrent save from another process and a refused reload; tool and model change). Terminal T0–T9 (unconfigured controls with `/system off|on` and `/tools off`; two tool rounds; `/new`; reload; disable and a new process while disabled; binding timing while idle; manual, repeated, post-reload and automatic compaction; built-in prompt off; a held turn; tools and model switches). Cross-adapter X1–X3. Entry E1.

## The oracle

Every captured request is attributed to the step that caused it (the label the driver set before the operation); summariser requests are recognised by the compactor's own prompt (`agent/compactor.go:31`). An unattributed model request makes the run invalid.

The expected request for each delivery is derived from two inputs only: the request the **pre-G1 candidate** sent for the same step, frozen as a golden by the first baseline run and never regenerated, and the blueprint's composition rule written out here (§5; §6.3 A1). For an expected carrier: the first message has role `system` and is exactly the base, then `\n\n` when the base is non-empty, then `User-configured instructions; revision <r>; selection <g>:\n`, then the profile bytes; the request carries `admission` version 1; everything else equals the golden. The desktop base is the model-system text of the request's model; the terminal base is the golden's own leading system message (the built-in prompt), or nothing after `/system off`. For no carrier: the request equals its golden and carries no `admission`. Normalisation parameterises only the terminal prompt's date line and the sandbox path. No production composition code is called.

A compaction must be shown at the request boundary: its summariser request exists and the next turn's history carries the summary. The summariser's own expectation: the compactor's prompt, no header, no profile text.

## Controls (run by `run.sh` before every measurement)

- **Scorer** (`TestOQ3ScorerControls`): expectations written by hand from the blueprint's words match the transform in all three shapes; a perfect delivery scores correct; eleven planted faults (missing, wrong revision, wrong generation, duplicated, altered text, base lost, misplaced, admission missing, history changed, unexpected carrier, an unnamed difference) each land in their category; a multi-defect request counts once; summariser faults are caught; the arithmetic reconciles; a zero denominator is never printed as a number.
- **Reference arm** (every run): the expected request for every scored delivery is sent through real HTTP to a fresh fake daemon and must score correct; it is kept out of the candidate's numbers.
- **Runner** (`TestOQ3RunnerControls`): empty and unknown selections refused; trial identities unique; a missing trial and a missing scenario caught; a malformed capture, an extra request and a request under an unknown label each invalidate the run; a failing child process invalidates it.
- **Gate self-test** (`OQ3_GATE_SELFTEST=1`): `run.sh` blocks a failing test, a missing required test and an empty selection.
- **Repeat:** the baseline runs twice; the second compares against the goldens the first froze and must give identical per-trial verdicts.
- **Smoke** (`TestOQ3HarnessSmoke`): both processes reach the fake daemon.

Production mutants and degenerate twins (blueprint line 354) belong to the final acceptance of the implementation: a red baseline earns no mutation credit.

## Known limits (Codex Q8 and Q9)

A zero proves only the classified boundary obligations in a valid run. Not covered here, each with its owner (the G1 packs): the desktop's tool pass-loop continuation (needs an MCP server fixture: the app's `startMCPManager` is in package `main`), and the web-search family; tool-output-triggered compaction (`agent/session.go:751`); `/new` failure injection; storage corruption, CAS, capacity and initialisation guards; the browser journey (I-09) and I-10 to I-14; real-model qualification; anything the daemon does after the capture point (template rendering, context shift, the model's compliance). Terminal conversation identity is not visible in a request (`api.ChatRequest` has no conversation id), so it is read from the designated store's `conversations` table, which reports the surface absent until the store exists. Fixture limits: one model family, short texts, no context pressure except the automatic-compaction trigger, and serial schedules apart from the two held turns.

False-alarm guards: refusals are declared, not counted as missing deliveries; revision numbers come from per-scenario stores; idempotent reloads keep their generation; the terminal's date is normalised and the desktop has none; profile sentinels never appear in prompts or fixtures.

## Blueprint gaps this instrument exposed (for OQ-4)

1. The join between desktop model-system text and MCP instructions is unstated (`G1_PERSISTED_INSTRUCTIONS.md:204`); the oracle refuses to guess, so the fixture configures no MCP server.
2. The terminal output of `/instructions reload` and `/instructions off`, for success and for a refusal while a turn runs, is unstated (§7); the instrument records these as unobserved, and acceptance fails them until specified.
3. `ollama instructions set` does not say whether the saved profile is enabled; the instrument assumes it saves an enabled profile.
4. The designated terminal instrument `cmd/tui/chat/instructions_test.go` sits below `GenerateAgentTUI`, where G1 creates the selection, so it cannot carry the restart and new-conversation cases; this instrument lives in package `cmd`.
5. Whether a compaction summariser request should carry the instructions is unstated; the instrument expects today's behaviour (the compactor's own prompt, no carrier).

## Running it

    OQ3_SCRATCH=<disposable dir> OQ3_GATE_SELFTEST=1 bash run.sh baseline <evidence dir>   # pre-G1, once: freezes goldens/
    OQ3_SCRATCH=<disposable dir> bash run.sh accept <evidence dir>                           # after G1: number must be 0
