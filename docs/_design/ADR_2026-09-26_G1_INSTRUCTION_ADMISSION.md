# ADR 2026-09-26 — G1 instruction admission: keep the whole prompt, reserve room, refuse what cannot be verified

| Field | Value |
|---|---|
| Status | Decided by Thole under KANON 25 (tier 2, checked before deciding); governs G1 blueprint §6 |
| Decider | Thole (Claude Opus 5.5), G1 delivery owner from 26 September 2026 |
| Feature | OLLAMA-G1-INSTRUCTIONS (OI-03 + OI-04), open obligation OQ-1 |
| Candidate | Ollama fork `04009c432e313eef65172293e75feb038ebe476b`; native llama-server b10091 `b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45` |
| Evidence | `docs/analysis/colleague_integration/proof_cont13/oq1-measurements/` (runs 1, 2, 3, 3b) and `…/oq1-options-check/` |

## Context

G1 sends the operator's saved instructions (the "carrier") as a system message in every new conversation. G1-R-07 forbids silently discarding mandatory input and requires overflow to be reported; OI-04 asks for the budget to be measured for each supported configuration. Measured on this Mac with the unchanged candidate and qwen3:8b, on both local chat routes:

1. Under default settings the carrier is silently lost in any reply that outgrows the context. Each native context shift keeps `num_keep` tokens — Ollama sends 4 — and discards half of the rest; the carrier sat at tokens 16-66 (run1 `shift_default_long`, run3b `longreply_default`, where output then degenerated).
2. On the rendered route an oversized prompt returns HTTP 200: the Go runner keeps 4 tokens plus the tail and answers a mangled prompt (run1 rendered `oversize_default`; llama_server.go:276-316).
3. Shift off preserves input but, once history fills the context, cuts the reply after the few tokens of slack left (14-18 tokens, mid-sentence), and every change of the setting reloads the runner, about one second here (run3b `longreply_shift_false`, run2 `reload_sequence`; sched.go:1413-1418).
4. Keeping the whole prompt (`num_keep` -1) preserves the carrier through shifts with no reload and lets long replies finish (run2 `keep_all_long`, run3b `longreply_keep_all`, which ended with the carrier's sentinel). But the native shift clamps the keep to `n_ctx - 4` (server-context.cpp:2874), so a prompt within four tokens of capacity loses its own tail (run3b `clampedge_keep_all`: 118 shifts at `n_keep = 508`, tail instruction lost, output degenerate), and the rendered route then drops the prompt's tail on oversized input and returns 200 (run2 `keep_all_oversize`).
5. MLX refuses oversized input with HTTP 400 and never shifts (x/mlxrunner/pipeline.go:47-57). Explicit cloud models and remote-host stubs are forwarded without any local check (routes.go:2440-2448, 2495-2585).
6. Neither client shows `done_reason` today (no reference in app/ui, its frontend, cmd/tui or agent), and `length` covers both a full context and an exhausted output budget (server-context.cpp:1858-1863).

## Decision

G1 adds one opt-in, versioned **admission contract** to the daemon's `/api/chat`, used by both G1 adapters for every request that carries an enabled carrier, and advertised as a daemon capability. When a request carries the contract the daemon:

- keeps the whole prompt through context shifts (effective `num_keep` -1);
- fits history into the runner's context minus a reserve R, dropping whole turns oldest-first, never splitting an assistant tool call from its results, and always keeping every system message and the newest turn;
- refuses before inference with an explicit context error, carrying the counts, when the system messages and newest turn do not fit within the context minus R — the rendered route never cuts inside such a prompt;
- reports, in the final response, the context length, the reserve, the messages dropped and whether generation stopped because the context filled;
- refuses the contract, explicitly, on routes it cannot enforce: MLX (no MLX model is available to qualify it here), explicit cloud models and remote-host stubs.

Without the contract the daemon behaves exactly as today. Both clients check the capability before sending a carrier; if it is absent they refuse with the context-unavailable error rather than send the carrier unprotected or send without it. Both clients show, in the conversation, when a carrier-bearing reply stopped because the context filled. R defaults to one eighth of the context, clamped to at least 16 and at most 4,096 tokens, and a client may request a larger R up to half the context. Those numbers are **chosen, not measured**: 16 covers the native clamp of 4 plus the gap between the Go-side token estimate and the native count, which was then measured at zero in 24 cases across three models and both routes (blueprint §6.5 V1), leaving 11 tokens of margin for templates and models the measurement did not cover. One eighth keeps shifts infrequent (run3b `clampedge_keep_all` shows the thrash a tiny room causes).

## Alternatives considered

| Option | Why not chosen |
|---|---|
| Shift off for carrier-bearing requests (P1) | Preserves input, but starves replies once history fills the context and reloads the runner whenever carrier-bearing and ordinary requests alternate (run3b, run2). With a reserve it becomes sound, but the reserve needs the same daemon change as the chosen design, and the reloads and capped replies remain. |
| Keep the whole prompt with no reserve (P2 alone) | Unsound within four tokens of capacity (clamp) and on the rendered oversize path (run3b, run2). |
| The daemon protects leading system messages for every request (P3) | Changes ordinary and disabled-mode behaviour (G1-R-15); Modelfile messages can precede the carrier (routes.go:2633). |
| The client assembles the exact prompt (P4) | A second prompt assembler, which OI-04 forbids ("rather than introduce a parallel prompt assembly", commission:298); no public full-request count exists. |
| No reservation in G1, leave it to OI-27 (R1) | Refuted by run3b: without room, one lever loses input and the other starves output. |
| A version-number gate instead of a capability (V1) | Fork version strings need not map to behaviour; absence of a capability must fail closed. |
| Allow cloud models with a "provider-managed" label (K1) | A label cannot prevent silent provider truncation, and a later probe cannot qualify earlier deliveries. |
| Patch the native slot guard to enforce the reserve exactly | More exact, but a llama.cpp patch the fork must carry through every native bump. Not needed while the Go-side estimate's gap to the native count is bounded well below R; revisit if that bound fails. |

## Checks made before deciding (KANON 25.3)

The options, the measurements and the source excerpts — not a preferred answer — went blind to two non-Claude substrates: DeepSeek `deepseek-v4.1-flash` (text only) and Codex `gpt-6-astra` (reading the repository). They disagreed. Every claim was dispositioned against evidence (`DISPOSITIONS_DEEPSEEK.md`, `DISPOSITIONS_CODEX.md`); the deciding claims — the clamp and output starvation — were then tested by execution (run3, run3b) rather than argued. DeepSeek's choices of no reservation and allowing cloud were refuted by those runs and by the proxied path's source; Codex's choice of shift off was refuted by the long-reply run; Codex's reserve, capability and cloud-refusal arguments, and its lossless-admission proposal in reduced form, changed the decision.

## Consequences

- G1 now changes the daemon (api/types.go, server/routes.go, server/prompt.go, llm/llama_server.go, llm/server.go, the version handler) as well as the two adapters. The blueprint's estimate and reach closure grow accordingly.
- Carrier-bearing conversations on cloud and MLX models are refused until qualified. That is an explicit, visible limitation, not silent coverage.
- Replies in carrier-bearing conversations can still run long where the runner shifts: old generated text is shifted out, never the prompt. Where the runner cannot shift — multimodal runners (server-context.cpp:1215-1217) and model families launched without shift (sched.go:137-147) — the reply stops when the context fills, and the response says so.
- Existing truncation and shift behaviour for every other request is unchanged.

## Strongest case against

It adds a new wire contract to a fork that tracks upstream Ollama, which enlarges the rebase surface for as long as the fork lives, and it partly builds OI-27's admission layer ahead of that item's design, which may later want client-declared mandatory groups instead of positional ones. Mitigations: the contract is minimal, versioned and opt-in, and v1's positional mandatory set is designed to be widened by a later version, not replaced.

## Falsifiers — any one reopens this decision

- A carrier-bearing request on a supported route loses any carrier token, or any token of its newest turn, during generation or truncation.
- A shift occurs in a carrier-bearing request whose native keep is below its prompt length.
- A request whose Go-side estimate fits within the context minus R is found, by the runner's own count, to reach `n_ctx - 4` (the reserve's margin is wrong).
- A daemon without the capability receives a carrier-bearing request from either adapter.
- Disabled-mode requests differ from their pre-feature goldens.

## Reconsider when

OI-27 designs selective context; a native bump changes the shift or admission code cited above; the measured estimate gap approaches the reserve floor; or a cloud provider path is qualified (the refusal then narrows for that path only).
