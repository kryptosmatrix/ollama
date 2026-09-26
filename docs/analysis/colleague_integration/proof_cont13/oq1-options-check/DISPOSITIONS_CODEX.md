# OQ-1 options check — dispositions of the Codex answer

Reviewer: Codex CLI 0.153.4 (OpenAI), requested model `gpt-6-astra` (the served identity is not exposed in
the event stream), `--sandbox read-only --ephemeral`, hooks, memories, plugins and multi-agent disabled,
web search disabled. Attempt 1 failed before review (configured default `gpt-6-sol` unsupported on the
account; see `reviews/codex/SUBSTITUTION.md`). Attempt 2: identical prompt bytes (`reviews/codex/prompt.md`,
SHA-256 in `prompt.sha256`: the frozen packet `8631e331…` plus a preface granting read access to source under
agent/, api/, app/, cmd/, llm/, server/, x/ and the pinned native `tools/server/`, and withholding docs/).
Audit of its 36 commands: none touched docs/, none used another absolute path. Usage: 552,580 input tokens
(462,080 cached), 11,612 output (6,626 reasoning). Answer: `reviews/codex/attempt2/last_message.md`.

The reviewer chose P1 + R2 + V2 + K2 and proposed a missing option, an opt-in lossless-admission contract.
Its central claims were tested by execution before disposition (round 3 and round 3b under
`oq1-measurements/`, run after this answer arrived; results files cited below).

| # | Claim (abridged) | Disposition | Evidence |
|---|---|---|---|
| X1 | `num_keep: -1` does not unconditionally preserve the prompt: the native shift clamps the keep to `n_ctx - 4`, so a near-capacity prompt loses input tokens. | **Accepted, confirmed by execution.** | server-context.cpp:2874. run3b `clampedge_keep_all` (both arms): prompt 510 of 512, 118 shifts each `n_keep = 508, n_left = 3, n_discard = 1`; the tail instruction was lost and output degenerated (`PAD PAD…`; `</think> ZEBRA-7Q` loops). |
| X2 | Without a reservation, output starves or input is lost once history fills the context; "R1" is unsound. | **Accepted, confirmed by execution.** | run3b after a history packed to 494 / 498 of 512: shift off stopped mid-sentence after 18 / 14 tokens (`done_reason: "length"`); default shift degenerated with the carrier gone; keep-whole-prompt completed and ended with the carrier's sentinel. run3 shows the short-reply case survives on slack alone (9 tokens), which is why the defect needs a long reply to appear. |
| X3 | `done_reason: "length"` alone cannot distinguish a full context from an exhausted output budget. | **Accepted.** | server-context.cpp:1858-1863 and the budget check below it both set `STOP_TYPE_LIMIT`; llama_server.go:1672-1675 and 1973-1976 map both to `length`. The client needs the counts, or an explicit flag, to tell them apart. |
| X4 | History truncation on both routes selects message suffixes without protecting tool-call/result groups. | **Accepted.** | prompt.go:37-72 and routes.go:3019-3053 pick the first suffix that fits; nothing prevents the suffix starting at a tool result whose call was dropped. G1's carrier moves that boundary later, so it makes the pre-existing split more frequent. |
| X5 | P1's reload cost and output shortening. | **Accepted, measured.** | run2 `reload_sequence`; run3b `longreply_shift_false`. |
| X6 | Modelfile `MESSAGE` entries are prepended before client messages, so a client carrier need not remain leading. | **Accepted as a fact; bears on P3 only.** | routes.go:2633 (`msgs := append(m.Messages, req.Messages...)`). Keep-whole-prompt protects the carrier wherever it sits. |
| X7 | P4 needs a public full-request token count that does not exist. | **Accepted.** | Inference routes only (routes.go:1888-1908); tokenisation is the runner's private endpoint (llama_server.go:2431). |
| X8 | Choose P1 as the generation policy. | **Refuted by execution for G1's use.** In a long conversation P1 cuts the reply at the few tokens of slack the packed history leaves, and forces reloads; keep-whole-prompt with a reserve preserves the carrier without either. | run3b `longreply_shift_false` vs `longreply_keep_all`; run2 `reload_sequence`. |
| X9 | R2's mandatory set must include bindings, corrections, mandatory arrival material. | **Narrowed.** Those records do not exist in G1: non-null colleague bindings are refused (blueprint §3) and correction/arrival material belongs to G2 and OI-27. G1's mandatory set is the carrier, the model's baseline system text, existing MCP instructions, and the newest turn with its whole tool group. The contract is versioned so a later item can widen it. | Blueprint §3 binding admission; commission OI-07/OI-08/OI-27. |
| X10 | Choose K2: refuse carrier-bearing cloud requests until the provider path is qualified; a label cannot prevent silent provider truncation, and later probes cannot qualify earlier deliveries. | **Accepted.** The local daemon cannot enforce the contract on a proxied request. Qualification needs cloud requests, which spend the operator's quota and send content to the provider, so it is Ash's to authorise. | routes.go:2440-2448 (proxied untouched); commission OI-21 ("Cloud runs require current disclosure/spending authority"). |
| X11 | Choose V2; define capabilities by enforcement behaviour and route coverage; a missing capability must be an explicit refusal, not a silent degrade. | **Accepted** (agrees with DeepSeek D8). | — |
| X12 | Missing option: an opt-in, atomic lossless-admission contract. | **Adopted in reduced form** — see the decision record. The daemon applies it only when the request asks for it; unclassified input is not in play because G1's mandatory set is defined by position (system messages, newest turn and its tool group). | — |
| X13 | The native "at or above" rejection holds for ordinary generation; the non-splittable branch tests strictly greater. | **Accepted** as a precision correction; it does not change the chat result. | server-context.cpp:3121-3131. |
| X14 | The §2 measurements are one model, one slot, `think:false`, non-streaming; they do not establish tools, media, MLX, concurrency or provider behaviour. | **Accepted.** These become named verification obligations, not claims. | — |
