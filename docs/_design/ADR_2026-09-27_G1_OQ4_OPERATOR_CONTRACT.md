# ADR 2026-09-27 — G1 OQ-4, part 1: what the operator sees when G1 refuses, the evidence that checks it, and the composition and placement gaps

| Field | Value |
|---|---|
| Status | Decided by Abuttal under KANON 25 (tier 2, options checked before deciding), except D-11, which is routed to Ash as product scope. D-1's two refinements and D-10's narrowing were not in the options packets and go to a separate falsification pass before the blueprint text lands. Governs G1 blueprint §5, §7, §8, §9 and §10 once written there |
| Decider | Abuttal (Claude Opus 5.5), G1 delivery owner from 27 September 2026 |
| Feature | OLLAMA-G1-INSTRUCTIONS (OI-03 + OI-04), open obligation OQ-4 |
| Candidate | Ollama fork `5899971690b08cecc0e48ed5ab66209b5a5b8286` (production and test sources identical to `04009c432e313eef65172293e75feb038ebe476b`) |
| Evidence | `docs/analysis/colleague_integration/proof_cont16/oq4-options-check/` (packets, Codex answers, `DISPOSITIONS.md`, `PRIOR_LEANINGS.md`) |

## Context

OQ-4 must close the gaps G1's failure-number instrument exposed (`proof_cont14/oq3-instrument/DESIGN.md:58-64`) before the instrument can be repaired against stated text. The fourth review of the instrument found three defects no instrument rule could settle (`proof_cont15/oq3-instrument-review-r4/DISPOSITIONS_CODEX.md`, A1-A3): the terminal's refusal report for a daemon without `chat.admission.v1` has no stated form, so a diagnostic line followed by an unrelated failure passes as a refusal while a correct report in other words invalidates the run; and nothing says whether the failure number or I-14 checks the report, so a silent drop scores as a correct refusal. A12 (blueprint line 253) already requires the report; §7 (line 307) omits it from the notices. Reading the source for the closure census also found two consumers G1's records never mention: the classic `ollama run` paths (`cmd/interactive.go:34`; `cmd/cmd.go:1840-1852`, `:1972-1984`) and the terminal's `/prompt` and `/save` (`cmd/tui/chat/debug.go:28-64`, `:160-222`), which read the effective system prompt and so will show and write the carrier.

## Decisions

**D-1 — the terminal refuses per turn, inside an open chat.** With an enabled selection and a daemon whose `/api/version` lacks `chat.admission.v1`, the agent chat opens as usual (its preload carries no carrier) and shows one notice entry when it opens; every submitted turn, including a skill run, is then refused before any inference request, with one error entry of its own; the chat stays open and idle, so `/instructions off` can select a disabled revision, after which turns are sent without a carrier or an admission member. Refinements beyond the packet, to be attacked by the falsification pass: (a) each adapter reads the capability before every carrier-bearing request, replacing A12's "once per connection and after reconnect", which has no defined meaning over HTTP and would let a daemon replaced mid-session ignore the unknown `admission` member (the daemon decodes with gin's `ShouldBindJSON`, and no handler disallows unknown fields) and receive the carrier outside the contract; (b) a final record of an admitted request that carries no `admission` object is itself a refusal, reported as `context_unavailable` naming the daemon's version.

**D-2 — the terminal's report is a token contract with attribution.** The per-turn report is one error entry, drawn with the chat's existing `err` prefix (`cmd/tui/chat/render.go:487-488`), that appears after its turn is submitted and contains the code `context_unavailable` verbatim and the daemon's version exactly as `api.Client.Version` decodes it; its wording is otherwise free. The notice at open is a separate entry with the same two tokens and cannot stand in for a turn's report.

**D-3 — the desktop refuses on its existing error path.** `Server.chat` resolves the selection before saving, saves the submitted message as today (`app/ui/ui.go:774-787`), sends no inference request for the turn, and streams one `ErrorEvent` (`app/ui/responses/types.go:95-100`) with `code` `context_unavailable` and an `error` text containing the daemon's version; the code is classified before `getError`'s text rules (`app/ui/ui.go:601-606`). Known limit: the frontend holds the error in a query cache, so after a restart the saved message shows no reply and no error, as with every streamed error today.

**D-4 — the refusal report is a separate mandatory verdict, not part of the numerator.** The failure number keeps its meaning (bad observed deliveries plus missing required deliveries, over controlled conversation trials). T10 and D11 each carry a refusal-report verdict that must pass for acceptance: T10 checks the notice and the turn's own error entry (D-2), D11 the streamed event (D-3). Any inference request under a refused turn's label is an incorrect delivery, whether or not it carries the carrier: sending with the carrier stripped is what A12 forbids. Trial validity is established independently of the report: the fake daemon's version response without the capability, the enabled selection committed before the turn, the turn delivered, and the capture complete. A harness failure invalidates the run; a missing or malformed report fails the verdict and the gate, and is never read as a harness failure. The report verdict is published beside the number. I-14 keeps the adapters' mapping of every other daemon error; I-09 shows the desktop's event reaching the screen.

**D-5 — `/instructions` output names values; failures name codes.** `/instructions` writes a slash entry naming the scope, the conversation's selected revision, its selection generation and whether that selection is enabled, and the saved current revision as a separate value. `/instructions reload` names the resulting selection, or says the selection is unchanged when the target is already selected. `/instructions off` names the disabled revision it committed and the resulting selection. A failure is an error entry naming the operation and its §8 code and announces no success. The instrument checks displayed values against the store's committed state as well as the requests that follow.

**D-6 — I-05 proves the terminal's runtime identity; the failure number does not.** I-05 drives normal creation and `/new` and asserts that the identity passed to an actual run (`coreagent.RunOptions.ChatID`, which also reaches `CompactionRequest`, `agent/session.go:811-814`) equals the committed conversation key; two conversations at one revision have distinct non-empty identities; an injected binding failure leaves the old identity and state usable. The decisive mutant updates the binding and leaves the runtime identity unchanged. The instrument's coverage statement says it cannot observe this.

**D-7 — the desktop's enabled base.** Model-system text, then MCP instructions; a block is omitted iff `strings.TrimSpace(block) == ""`; retained blocks are joined by exactly `\n\n` and otherwise keep their bytes, including leading and trailing newlines. Disabled mode keeps today's path. The instrument gains an MCP-bearing desktop case.

**D-8 — the compaction summariser carries no carrier.** Its request keeps its fixed system prompt and archive; no profile text is added from the selection (archived messages may still quote instructions the operator typed); the conversation's effective system prompt still enters the compactor's size estimate (`agent/compactor.go:383-389`); the carrier returns on the next request from the selection.

**D-9 — the failure-number instrument's final place.** Committed tests in package `cmd`: `cmd/instructions_delivery_darwin_test.go` with `TestG1InstructionDelivery` and `TestG1InstructionDeliveryControls`, discovered by default on darwin, with committed frozen goldens, scratch under `t.TempDir()`, and responsibility for building the entry binary; run in the combined gate as `go test -count=1 -json -timeout=30m -run '^TestG1InstructionDelivery(Controls)?$' ./cmd`, whose discovery and completion I-12 reconciles. `cmd/tui/chat/instructions_test.go` keeps the chat-model tests. Measured: the revision-4 failure-number test ran 27.84 s and 27.07 s and its runner controls 94.33 s on this Mac.

**D-10 — `/prompt` and `/save` stay faithful to what G1 adds.** Both show and write the composed carrier (they read the effective system prompt) and, when the selection is enabled, the admission member the dispatch would carry, attached by the same helper that attaches it to the real request; the view is labelled a preview. `/save` of a request carrying an enabled carrier writes mode 0600 and warns in one line, outside the file, that it contains saved instructions. Disabled and unconfigured `/prompt` and `/save` are unchanged (G1-R-15). Delegated default for Ash to veto: a faithful export, warned and private, rather than a redacted one.

**D-11 — the classic `ollama run` paths: routed to Ash.** Recommendation: exclude them from G1 and have the classic interactive chat show a one-line notice when saved instructions are enabled, so the boundary is visible; inclusion of the classic chat stays open as a separately scoped extension.

## Alternatives considered

| Decision | Rejected | Why |
|---|---|---|
| D-1 | Refuse at start-up (`GenerateAgentTUI` returns an error to the launcher) | A terminal-only operator could not reach `/instructions off`; the launcher's `Error:` line then returns to the menu (`cmd/cmd.go:2152-2156`, `:2180-2188`) |
| D-1 | Restricted slash-only chat | Skill slash commands start runs (`cmd/tui/chat/chat.go:1098-1113`), so the restriction is not a boundary |
| D-2 | Verbatim sentence; fixed prefix; no text contract | A sentence fixes wording no requirement needs; a prefix constrains rendering without adding attribution; a typed error alone leaves the operator's report unchecked, which I-14 names as a fault ("report skipped", line 350) |
| D-3 | HTTP error before saving; not saving the message | Both diverge from every other streamed failure the desktop reports, and the latter loses what the operator typed |
| D-4 | Report only in I-14; a missing report counted in the numerator; report as a validity condition; split by adapter | The first scores a silent drop correct inside the failure number; the second changes what the number means (KANON 6); the third mistakes a product failure for an experiment failure; the fourth covers the adapters unequally |
| D-5 | Codes only; verbatim lines | Delivery evidence cannot show what the operator was told; verbatim lines fix wording no requirement needs |
| D-6 | Conversation key in diagnostics or in the request; not load-bearing | No operational need for external correlation, and a wire field nothing else uses; I-05 already names "unused new ID" as a fault, so dropping the check weakens a requirement |
| D-7 | Separate MCP system message; single `\n`; labelled blocks | The first contradicts §5's single effective system message and moves MCP after the carrier; a single newline differs from the composer's own separator for no reason; labels duplicate the MCP preamble (`app/ui/mcp.go:521`) |
| D-8 | Always; per profile | A second behavioural contract on an internal transform, whose fixed task (preserve goals, decisions, files) response preferences can contradict |
| D-9 | Overlay under `docs/`; a separate tagged package | The overlay is not in the default suite; a tag is justified only by runtime or isolation costs, which the measurement does not show |
| D-10 | Redacted preview; no admission member; one shared builder for preview and dispatch | Redaction breaks §5's exact-preview promise; omitting the member hides a dispatch-relevant field; a shared builder would also change the disabled preview, a pre-existing divergence outside G1 |

## Checks made before deciding (KANON 25.3)

Two blind options packets, frozen at `ef6d6ee9`, went to Codex `gpt-6-astra` in parallel. Codex chose as I leaned on nine of eleven questions, changed my decision on the report's allocation (D-4), and recommended including the classic interactive chat (D-11) while classifying that question as product scope, as I do. Every source claim in its answers held when read. Claim-by-claim record: `proof_cont16/oq4-options-check/DISPOSITIONS.md`.

## Consequences

- Blueprint: §5 gains the capability-read rule, the missing-report rule and D-7's join; §7 gains the terminal's notice, per-turn refusal and `/instructions` outputs, the desktop's refusal response, and `/prompt` and `/save`; §8 maps a missing `admission` report to `context_unavailable`; §9 moves the instrument (D-9), gives T10 and D11 their report verdicts (D-4), and gives I-05 its identity assertion (D-6); §10's estimate grows for the capability read, the notice entries and the `/save` mode, to be itemised with the census.
- Instrument revision 5 (owed): A1-A3 are answered by D-1, D-2 and D-4; B1-B3, C1, C2, D1, D2, E1 and F stand as proposed; an MCP-bearing case (D-7) and a classic-path decision (D-11) extend its ledger.

## Strongest case against

D-1 keeps an operator in a chat that refuses every turn until they act; a start-up refusal would at least never look ready. The notice at open, naming the cause and the recovery command, is the mitigation; if operators still type into a chat that cannot answer, the start-up refusal with a launcher-level disable is the alternative to revisit.

## Falsifiers — any one reopens this decision

- A carrier-bearing request reaches a daemon that does not advertise `chat.admission.v1`, in any adapter, in any trial.
- A refused turn whose report is missing, or whose report lacks the code or the version, passes T10's or D11's report verdict.
- A report that is attributed to its turn and contains the code and the version fails T10's verdict because of its other wording.
- An inference request sent with the carrier stripped, under a refused turn, is scored correct.
- I-05 passes against the mutant that updates the binding and leaves the runtime identity unchanged.

## Reconsider when

An operator ruling makes a refused session close before it opens; a real need to correlate requests with terminal conversations outside the process appears; the instrument's port shows default discovery impractical; or Ash rules D-11 towards inclusion, which extends G1's terminal adapters and this record.
