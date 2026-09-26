# OQ-3 — the G1 failure-number instrument (revision 2)

Continuation 14, Letterlock (Claude Opus 5.5), 26 September 2026. Design decided after a blind options check on Codex (`../oq3-options-check/DISPOSITIONS_CODEX.md`); revision 2 repairs the fourteen blocking findings of a Codex review of revision 1 (`../oq3-instrument-review/DISPOSITIONS_CODEX.md`), which superseded the first baseline. Design evidence for G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`); it changes no production or default test file and runs only through `go test -overlay`.

## What it measures

G1's failure number is "incorrect instruction deliveries per controlled conversation", over save/restart/new, edit/continue, explicit reload, disable, terminal reset and compaction; any value above zero fails G1 (blueprint line 18). The instrument publishes

    failure number = (bad observed deliveries + missing required deliveries) / controlled conversation trials

with its components, per-category counts (overlapping; one bad request counts once in the numerator), per-family and per-trial breakdowns, operation and refusal verdicts, **auxiliary checks** (summariser requests, preloads, leaks) and **a separate validity verdict**. A zero numerator is not acceptance: the run must be valid, every auxiliary check must pass, and every operation must pass on its own evidence.

## Boundary

The capture point is the inference daemon's input: a fake daemon at `OLLAMA_HOST` records every request on every path. Three arms reach it, each in its own OS process: the real `app/ui.Server.Handler`, assembled as `app/cmd/app/app.go:208-287` does (restart = SIGTERM, observed exit, new process on the same `HOME`); `launchInteractiveModel` → `GenerateAgentTUI` → `agentchat.Run` on a real pseudo-terminal (the function the launcher's run-model action calls, `cmd/cmd.go:2242`), with CLI steps through the real root command; and the built `ollama` binary with no arguments, through the launcher menu and model picker (E1).

Declared simulations: inference is the fake daemon (scripted replies; `read` tool calls where a turn asks; a summary for the compactor; injected failures, held responses and a high prompt count where declared; `chat.admission.v1` advertised, except in D11 and T10; an A9-shaped `admission` object returned to a request that carries the field). Declared omissions: the desktop's embedded daemon, MCP manager, updater and webview; on the lifecycle arm, the launcher's heartbeat, menu and model resolution (the entry arm covers them). A closed-port proxy blocks every non-loopback HTTP(S) request from the children.

## The ledger

`oq3_scenarios_test.go` freezes 27 scenarios and 39 trials (the run prints the ledger's SHA-256). Every step is a **required dispatch** (one expectation per model request it must cause, tool continuations included), a **required no-dispatch** (A12: no carrier-bearing request may be sent to a daemon without the capability), a **required refusal**, a production **operation** with an expected outcome, or a **harness** action. Designated surfaces absent before G1 are attempted in their designated form and their real responses recorded; later steps run anyway; no expectation is weakened because a save failed. A harness failure makes the run invalid; the steps it stops (and a held turn whose completion failed) are recorded as not measured and never enter the number.

State-changing steps assert their effect on the frozen pre-G1 request: the tool count after the `/tools` toggle, the model after a switch, the presence of the system message after `/system off|on`. A scenario that does not reach its declared state is invalid.

Coverage: desktop D0–D11 (unconfigured controls; save/restart/new; edit while continuing; reload and idempotent reload; disable, restart while disabled, restore and reset; a legacy transcript with revision zero and adoption; restart and reopen; binding timing for a pre-allocated ID; a failed first turn, its retry and a new ID; a held turn with a concurrent save from another process and a refused reload; web-search tools and a model change; a capability-less daemon). Terminal T0–T10 (unconfigured controls; two tool rounds; `/new`, then a reload inside the new conversation; reload; disable and a new process; binding timing while idle, then `/new` to show the save took effect; manual, repeated, post-reload and automatic compaction; built-in prompt off; a held tool turn whose continuation must keep revision 1 while a reload is attempted; the tools toggle and a model switch; a capability-less daemon). Cross-adapter X1–X3. Entry E1.

## The oracle

Every request is attributed to the step that caused it (the label set before the operation; a held turn's label stays active until its await completes). The expected request for each delivery comes from the frozen **pre-G1** request for the same step and the blueprint's composition rule, written out independently (§5; §6.3 A1): for a carrier, the first message is the base, `\n\n` when the base is non-empty, `User-configured instructions; revision <r>; selection <g>:\n` and the profile bytes, and the request carries `admission` version 1, optionally with a reserve within A2's bounds; for no carrier, the request equals its golden and carries no `admission`. The desktop base is the model-system text of the request's model; the terminal base is the golden's own leading system message, or nothing after `/system off`. Normalisation parameterises only the terminal prompt's date and both spellings of the sandbox path. No production composition code is called.

Auxiliary traffic has its own expectations: a compaction summariser request only under a step that declares it, in exactly the declared number, equal to its frozen golden and carrying no header or profile text; a preload equal to its frozen golden; no profile sentinel or carrier header in any other request on any path. An inference path the ledger does not know makes the run invalid.

A turn is complete when the daemon has fully written every response under its label and the terminal has then gone quiet; a quiet that times out is a harness failure. A terminal `/instructions` command reads "absent" on an unknown-command error and "no error seen" otherwise; the blueprint names no terminal output, so success and refusal are judged by the deliveries that follow (T8's held continuation for the refusal). Saves pass only when the response (desktop) or exit status (CLI) agrees with the designated store's committed revision, enabled flag and text. Terminal identity is read from the store's event sequence: two conversations opened under distinct identities, and T2's reload recorded under the second.

## Controls (run by `run.sh` before every measurement)

- **Scorer** (`TestOQ3ScorerControls`): expectations written by hand from the blueprint's words match the transform in all three shapes; a perfect delivery scores correct; twelve planted faults land in their categories and a multi-defect request counts once; a permitted reserve is accepted and four invalid admission objects are caught; both spellings of the sandbox path normalise; summariser faults are caught; the arithmetic reconciles.
- **Reference arm** (every run): the expected request for every scored delivery goes through real HTTP to a fresh fake daemon and must score correct. This shows HTTP and normalisation round-trip only; it cannot check the expectations themselves, which the hand-written scorer controls do.
- **Runner** (`TestOQ3RunnerControls`): empty and unknown selections refused; trial identities unique; a missing trial and a missing scenario caught; a malformed capture, an extra request and an unknown label invalidate the run; an undeclared summariser and instruction text in a metadata request are auxiliary failures; an unknown inference endpoint invalidates the run; a failing child invalidates it without changing the number.
- **Gate self-test** (`OQ3_GATE_SELFTEST=1`): `run.sh` blocks a failing test, a missing required test and an empty selection; the overlay is regenerated, checked and hashed before any test.
- **Repeat:** the baseline runs twice; the second compares against the goldens the first froze and must give identical per-trial verdicts.

## Scope and known limits

`run.sh accept` is the **failure-number component** of G1-R-18's combined gate (§10): necessary, not sufficient. It does not replace I-09 (browser journey), I-10 (real models), I-13 (the daemon contract behind the capture point), the rest of I-14 (adapter behaviour for each daemon error), storage corruption, CAS, capacity and initialisation guards, production mutants and degenerate twins, the desktop's tool pass-loop continuation (needs an MCP server fixture; `startMCPManager` is in package `main`), the web-search family, tool-output-triggered compaction or `/new` failure injection. Fixture limits: one model family, short texts, no context pressure except the automatic-compaction trigger, serial schedules apart from the two held turns.

## Blueprint gaps this instrument exposed

1. The join between desktop model-system text and MCP instructions is unstated (`G1_PERSISTED_INSTRUCTIONS.md:204`); the oracle refuses to guess, so the fixture configures no MCP server. Open for OQ-4.
2. The terminal output of `/instructions reload|off` is unstated (§7). Handled by delivery evidence; still worth stating for the operator.
3. Whether `ollama instructions set` enables the profile: resolved in the blueprint (§7) in continuation 14 — `set` saves the file's text as an enabled profile.
4. The designated terminal instrument `cmd/tui/chat/instructions_test.go` sits below `GenerateAgentTUI`, where G1 creates the selection. Open for OQ-4: this instrument belongs in package `cmd`.
5. Whether a compaction summariser should carry the instructions is unstated; the instrument expects today's behaviour. Open for OQ-4.

## Running it

    OQ3_SCRATCH=<disposable dir> OQ3_GATE_SELFTEST=1 bash run.sh baseline <evidence dir>   # pre-G1, once: freezes goldens/
    OQ3_SCRATCH=<disposable dir> bash run.sh accept <evidence dir>                           # after G1
