# Options check C: how G1 reaches the classic `ollama run` paths (Ollama fork)

**What you are asked to do.** For each question below, choose among the options or propose another, and say why, citing file:line for every claim you rest on. Say what evidence would change your choice and what risk your choice carries. Where you think a question is not an engineering choice (it decides what the product does for its user rather than how), say so and why. You are advising on design; you are not reviewing code for defects. The options are listed in no order of preference.

## Context

G1 (`docs/_design/G1_PERSISTED_INSTRUCTIONS.md`) is a design at `[SPEC-DRAFT]`, not implemented. The operator's saved instructions (the "carrier") are pinned per conversation (§3-§4: a binding row and an `opened` event per conversation, reload by explicit command, lease per conversation) and composed into the request's system message by `Compose(base, selection)` (§5, line 200). A carrier-bearing request carries an opt-in admission contract (§6.3, A1-A13): the daemon keeps the whole prompt, reserves output room, fits history by whole turns, refuses before inference when the system messages and newest turn do not fit, never cuts, refuses routes it cannot enforce, and reports what it did in the final record; clients read a capability first and never send the carrier outside the contract (A12, line 253). A2 (line 243) currently makes `/api/generate` refuse any body with an `admission` member. Capacity is declared at 20,000 conversation bindings, with no eviction (§3, line 119).

The operator has ruled (`docs/_design/ADR_2026-09-27_G1_OQ4_OPERATOR_CONTRACT.md`, D-11) that G1 also covers the classic `ollama run` paths: the classic interactive chat and one-shot or piped runs, with the admission contract extended to `/api/generate`. That ruling settles whether; this packet asks how. The same record's D-1 to D-5 state the agent chat's contract, which you may use as a reference but need not copy: the chat opens and refuses each turn with an attributed error entry carrying `context_unavailable` and the daemon's version (D-1, D-2); the refusal report is a mandatory verdict beside the failure number (D-4); `/instructions` output names values and failures name codes (D-5); and adapters read the capability before every carrier-bearing request (D-1(a)).

The paths, in the unchanged candidate:

- `ollama run <model>` is interactive only when standard input and output are both terminals and no prompt argument is given; otherwise it is a one-shot run, whose prompt comes from the arguments and from piped standard input (`cmd/cmd.go:792-816`). Both first preload the model with a prompt-less `/api/generate` request (`cmd/cmd.go:587-597`, called at `:900`).
- The classic interactive chat, `generateInteractive` (`cmd/interactive.go:34`), keeps its history in `opts.Messages` and sends every turn through `chat` (`cmd/cmd.go:1758`, request built at `:1840-1852`) to `/api/chat`. `/set system` stores the text in `opts.System` and replaces the last history message when it is a system message, otherwise appends one (`cmd/interactive.go:385-397`; multi-line form at `:182-185`). `/clear` empties the history and re-adds the `/set system` text (`:279-286`). `/load <model>` switches model and empties the history (`:209-253`). `/save <model>` creates a model from `opts.System` and the session's messages (`:254-277`; `NewCreateRequest` at `:548`, which copies them at `:568-580`). A failed turn returns its error, which ends the session and exits with the error, except for two thinking errors that print and continue (`:529-539`).
- The one-shot run, `generate` (`cmd/cmd.go:1884`), sends one `/api/generate` request with `System: opts.System`, which is empty for a one-shot run, so the daemon uses the model's own system text (`cmd/cmd.go:1972-1984`; `server/routes.go:520-525`). Its errors reach `main`, which prints `Error: <err>` and exits 1 (`main.go`, `cobra.CheckErr`).
- The daemon's `/api/generate` handler (`server/routes.go:250`) proxies explicit cloud models first (`:272`) and remote-host stubs (`:313`); refuses `raw` with a template, system or context (`:427`); and, when a request has no suffix and no template, builds `[system, the model's MESSAGE history, user]` and renders it through the same fitting functions `/api/chat` uses, native (`prepareNativeChatRequest`) or Go-rendered (`chatPrompt`) (`:519-615`); otherwise it executes the template directly (`:617-624`).

## Questions

**C-1. In the classic chat, what is the base the carrier is composed onto, and what happens to `/set system` text?** Today a leading system message from `/set system` suppresses the model's own system text at the daemon (`server/routes.go:2634-2636`), and later `/set system` messages sit inside the history.
1. The base is the model's system text; `/set system` messages stay in the history where they are, so a request can carry the composed message and later system messages.
2. The base is the most recent `/set system` text when there is one, otherwise the model's system text; that message is sent only as the base, not also in the history, preserving today's "replace the model's system text" meaning.
3. The base is the model's system text followed by the `/set system` text, joined as the desktop joins its blocks, and `/set system` messages are sent only in the base.
4. While saved instructions are enabled, `/set system` is refused with a message naming the conflict.

**C-2. What are the classic chat's conversation boundaries and instruction commands?**
1. The session start binds a new conversation at the current revision; `/clear` and `/load` each start a new conversation (bound before the old state is cleared); `/save` never includes the carrier; the chat gains `/instructions`, `/instructions reload` and `/instructions off` with the agent chat's outputs as plain lines.
2. As 1, but `/load` keeps the conversation and its selection, as a model switch does in the agent chat, although the history is emptied.
3. As 1, but without `/instructions` commands in the classic chat: reload by starting a new session, disable from the desktop or the agent chat.
4. The whole classic session is one conversation regardless of `/clear` and `/load`.

**C-3. How does the classic chat refuse a turn?** (A missing capability, or a daemon refusal under A6, A10 or A13.)
1. Print one error line carrying the code and the daemon's version (or the refusal's counts), drop the refused turn from the history because nothing was sent, and keep the session open for the next line.
2. Return the error as the classic chat does for other failures, ending the session with `Error:` and exit status 1.
3. Check at session start and refuse to open the session.

**C-4. How does a one-shot run carry the carrier and the contract?**
1. On `/api/generate`: the request's `system` field is `Compose(base, selection)` with the model's system text as the base; the request carries the admission member; the daemon applies the contract on the chat-like path it already shares with `/api/chat`, and refuses the member on the raw, template, suffix and context forms.
2. The one-shot run switches to `/api/chat` (one user message after the composed system message) whenever saved instructions are enabled, and `/api/generate` keeps refusing the member.
3. On `/api/generate`, with the carrier placed in the prompt text rather than the system field.

**C-5. Which capability does a one-shot run read?** `chat.admission.v1` is designated but not built (A12).
1. One capability for both routes, renamed (for example `admission.v1`), advertised only when both are implemented.
2. A second capability, `generate.admission.v1`, advertised independently; each adapter reads the one for its route.
3. `chat.admission.v1` is widened to mean both routes, without renaming.

**C-6. How does a one-shot run select its revision, given 20,000 bindings with no eviction?** A script can call `ollama run <model> <prompt>` many thousands of times.
1. Each run is a new conversation with its own binding and `opened` event, like any other, subject to the cap.
2. Each run reads the current profile without writing a binding (an unbound selection used for one request), with a stated header value for the selection generation.
3. One-shot runs share one binding in their own namespace, reloaded to the current revision at each run.
4. One-shot bindings are written and are exempt from the cap through their own retention rule.

## For every answer

Choose, give reasons with citations, name what would change your choice, and name the risk. If two questions constrain each other, say how. Say also how a one-shot run should report the admission record's notices (A9's `context_exhausted`, the 80% cloud notice) and a refusal without disturbing a script that reads its standard output.
