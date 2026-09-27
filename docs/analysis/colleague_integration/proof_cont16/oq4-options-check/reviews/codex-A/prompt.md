You are an independent adviser working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/, and run read-only commands. Change nothing and run nothing that writes. Cite file:line for every claim you rest on.

---

# Options check A: what the operator sees when G1 refuses, and which evidence checks it (Ollama fork)

**What you are asked to do.** For each question below, choose among the options or propose another, and say why, citing file:line for every claim you rest on. Say what evidence would change your choice and what risk your choice carries. Where you think a question is not an engineering choice at all (for example, it decides what the product does for its user rather than how), say so and why. You are advising on design; you are not reviewing code for defects. The options are listed in no order of preference.

## Context

G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`) is a design at `[SPEC-DRAFT]`, not implemented: the operator's saved instructions (the "carrier") are composed into the system message of every new conversation in two adapters, the desktop app (`app/ui`) and the terminal agent chat (`cmd/agent_tui.go`, `cmd/tui/chat/`). A carrier-bearing request carries an opt-in admission contract (§6.3, A1-A13). A12 (line 253) says both adapters read the daemon's capability `chat.admission.v1` "once per connection and after reconnect", and "when it is absent they send no carrier-bearing request and report `context_unavailable` naming the daemon's version (G1-R-14). They never send the carrier outside the contract and never drop the carrier to send anyway." §7 (line 307) lists the notices for A6, A9, A10 and A13 but not for a missing capability; §8 (line 315) maps a missing capability to `context_unavailable` "naming the route, the feature or the daemon's version". A disabled selection sends no admission field and its request is unchanged (§5, line 206).

G1's failure number (line 18) is "incorrect instruction deliveries per controlled conversation", measured by an instrument (`docs/analysis/colleague_integration/proof_cont14/oq3-instrument/`, design in `DESIGN.md`) that is one component of G1's combined acceptance gate (§10). Its scenarios T10 (terminal) and D11 (desktop) run against a daemon without the capability and are scored as a "required refusal" with no carrier-bearing request allowed (`oq3_scenarios_test.go:333-341`). The instrument's DESIGN.md lists seven blueprint gaps (lines 58-64); this packet covers gaps 7, 2 and 6. I-14 (blueprint line 350) is the acceptance test of adapter behaviour for each daemon error, outside the instrument (`DESIGN.md:52`).

The fourth review of the instrument (`proof_cont15/oq3-instrument-review-r4/`, its reviewers' answers under `reviews/`) found three defects that the blueprint text must settle first, A1-A3 in its `DISPOSITIONS_CODEX.md`: a diagnostic line naming the capability and the version, followed by an unrelated failure, passes as a refusal (A1); a correct refusal worded outside the instrument's accepted vocabulary makes the run invalid (A2); and the no-request paths score a refusal as correct without any report, so a terminal that silently drops the prompt or a desktop that answers an unrelated error reads as refused as required (A3). Those dispositions and `proof_cont15/oq3-sequencing-check/DISPOSITION.md` are earlier colleagues' records, not decisions on the questions below.

Existing surfaces, all in the unchanged candidate:

- Terminal start-up failure: `GenerateAgentTUI` (`cmd/agent_tui.go:73`) returns an error to `launchInteractiveModel` (`cmd/cmd.go:2130`), which wraps it as "error running agent: %w" (`:2152-2156`); the launcher loop prints `Error: <err>` to standard error and returns to its menu (`cmd/cmd.go:2180-2188`). `GenerateAgentTUI` has one production caller.
- Terminal turn failure: a failed run appends a transcript entry with role "error" holding the error's text and sets the status to "error"; the chat stays open (`cmd/tui/chat/chat.go:407-411`).
- Terminal slash-command output: `/system` writes a "slash" entry and sets a status (`cmd/tui/chat/input.go:424-440`; `newSlashEntry`, `cmd/tui/chat/render.go:1816`); a usage error is an "error" entry.
- Desktop turn failure: the user's message is saved first (`app/ui/ui.go:785`, `:806`), the handler streams one `responses.ErrorEvent` `{eventName:"error", error, code}` and returns (`app/ui/ui.go:1294-1299`, `getError` at `:588-624`); the frontend keeps it per chat and renders it (`app/ui/app/src/hooks/useChats.ts:643-646`, `components/ErrorMessage.tsx`). Codes today include `cloud_unauthorized` and `usage_limit_upgrade`, the latter chosen by the text "402" (`app/ui/ui.go:601-606`).
- The daemon's version: `GET /api/version` returns `{"version": ...}` (`server/routes.go:1861-1862`); the client reads it at `api/client.go:454-464`.

## Questions

**A-1. When and where does the terminal refuse?** The operator has an enabled saved profile and starts the terminal agent chat against a daemon that lacks `chat.admission.v1`.
1. Refuse at start-up, before the chat opens: `GenerateAgentTUI` returns a typed error; the operator sees the launcher's `Error:` line and is back at the menu. No conversation exists, so `/instructions off` (§7, line 309) cannot be typed in this terminal.
2. Open the chat normally (its preload carries no carrier) and refuse each submitted turn before any request, with the chat's existing "error" entry; the chat stays usable, so `/instructions off` can be typed and later turns sent without the carrier.
3. As 2, and also show a one-line notice when the chat opens, before any turn is typed.
4. Open the chat in a restricted state in which only slash commands are accepted until the capability appears or the operator disables the instructions.

**A-2. What is the normative form of the terminal's report?**
1. A verbatim sentence fixed by the blueprint.
2. A token contract: one report (one entry, or one line) that contains the code `context_unavailable` verbatim and the daemon's version string exactly as `/api/version` returns it; the wording is otherwise free.
3. A fixed prefix: the report begins `context_unavailable:` followed by free text that must contain the daemon's version.
4. No text contract: the adapter produces a typed error with the code and the version as fields, checked in-process by I-14; the terminal's text is not normative.

**A-3. What is the desktop's refusal response?**
1. The existing error path: the user's message is saved, one `ErrorEvent` with `code` `context_unavailable` and an `error` text naming the daemon's version is streamed, and no request reaches the daemon; the frontend shows it as the chat's error.
2. Refuse before anything is saved, with an HTTP error status and the §3 error envelope (`{"error":{"code":"context_unavailable",...}}`, line 171) instead of a streamed event.
3. As 1, but the user's message is not saved.

**A-4. Which evidence checks the refusal?** Three things can be checked on T10 and D11: that no carrier-bearing request reached the daemon (dispatch evidence), that the report appeared in the form A-2 and A-3 state (report evidence), and that the trial reached the state it was meant to test (trial validity).
1. The failure number scores dispatch only; the report is checked only by I-14; a trial with no dispatch is scored correct whether or not a report appeared (the combined gate still needs I-14).
2. The failure number's no-dispatch trial requires both: no carrier-bearing request, and the report in the stated form. A missing or malformed report counts in the numerator, as a missing required delivery does. I-14 covers the other codes and the client's mapping.
3. The report is a trial-validity condition: without it the run is invalid (which fails the gate) rather than counted in the numerator.
4. Divide by adapter: the desktop's report is in the HTTP response the instrument already reads, so the failure number checks it there; the terminal's report is left to I-14.

**A-5. What does the terminal show for `/instructions`, `/instructions reload` and `/instructions off`?** (§7, line 309; gap 2.) Today the instrument judges these only by the deliveries that follow them.
1. State each command's output as a "slash" entry carrying named values (for example the scope, the selected revision and generation, and whether the profile is enabled), and each failure as an "error" entry carrying its §8 code; the wording is otherwise free.
2. State only the failure codes; success is judged by its effects (the store's committed state and the requests that follow).
3. Fix every line verbatim.

**A-6. Where is the terminal's conversation identity checked?** (Gap 6.) §4 (line 191) requires `/new` to generate a new identity and bind it before the old state is cleared. The identity the chat's run uses, `m.chatID`, goes to `coreagent.RunOptions.ChatID` (`cmd/tui/chat/chat.go:1157-1158`) and from there only into in-process agent events (`agent/events.go:65`, `:98-151`); `api.ChatRequest` carries no conversation identity, and the carrier's header (`revision <r>; selection <g>`, §5 line 200) is the same for two new conversations at one revision. §5 (line 208) says actual-request diagnostics record revision, generation, source digest, carrier byte count, model and request outcome, without naming where they are recorded. The instrument reads the store's event sequence (`DESIGN.md:37`).
1. Add the conversation key to the actual-request diagnostics, and name the diagnostics' sink, so an external instrument can see which identity each request used.
2. Allocate it to a white-box acceptance test (I-05): after creation and after `/new`, the chat's run identity equals the bound conversation key, with a mutant that leaves the identity unchanged; state that the failure number cannot observe it.
3. Carry the conversation identity in the request (for example a header the daemon ignores).
4. Treat it as not load-bearing: the store's event sequence already shows which conversation each open and reload was recorded under, and the run identity has no other G1-observable effect.

## For every answer

Choose, give reasons with citations, name what would change your choice, and name the risk. If two questions constrain each other (A-1 with A-2 and A-4, for instance), say how.
