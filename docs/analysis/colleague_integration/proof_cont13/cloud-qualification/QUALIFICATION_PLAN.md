# Bounded cloud check — plan, budget and pass criteria (written before any request)

**Authority.** Ash's chip answer on 26 September 2026: "Bounded check (Recommended) — A few dozen synthetic requests across your four cloud models, well under a million tokens; synthetic text only." Recorded in DECISIONS.md as `ash-cloud-bounded-check-2026-09-26`.

**Models.** The installed cloud stubs, through Ash's installed daemon at 127.0.0.1:11434 (the only local route holding his cloud sign-in). `deepseek-v4-flash:cloud` returns HTTP 410 ("retired at 2026-09-25"), so the check covers the three live ones: `deepseek-v4.1-flash:cloud`, `glm-5.2:cloud`, `glm-5.3:cloud`. Each advertises a context of 1,048,576 tokens through `/api/show`.

**Question.** Is a leading system carrier delivered intact, at realistic sizes, on these routes, with default request fields (the fields G1 would send)? A carrier-bearing request can only be admitted on a route where delivery is shown up to a stated size.

**Probes, per model, sequential, non-streaming, `think` false, `num_predict` 48, default `truncate`/`shift`:**

- **Q1** — about 150 tokens: the carrier plus a short user message.
- **Q2** — about 30,000 tokens: the carrier (a system message holding a random code), then one user message of numbered lines whose first and last lines hold random numbers, ending with the instruction to reply exactly `CARRIER=<code>; FIRST=<number>; LAST=<number>`.
- **Q3** — about 100,000 tokens: as Q2, sized from Q2's measured tokens per line for that model.

Codes and numbers are drawn fresh per probe from `secrets`, so a correct answer cannot be guessed or remembered.

**Budget.** Hard cap of 500,000 prompt tokens in total, summed from each response's `prompt_eval_count`; before each probe the harness estimates its size from the model's measured tokens per line and skips the probe if the running sum would pass the cap. Planned total about 390,000. At most one retry per probe, and only for a transport failure; any 402, 429 or quota message stops the run.

**Pass criteria, fixed now.** A probe passes when the reply contains all three exact values. `CARRIER` and `FIRST` both correct means the start of the prompt and the system message were delivered at that size. `LAST` correct with `CARRIER` or `FIRST` wrong is recorded as **start lost** — the silent-truncation signature. All wrong, or an error, is recorded as **no evidence** for that size.

**What a result establishes.** For each model, the verified size V is the largest passing probe's `prompt_eval_count`. It supports admitting carrier-bearing requests on that route whose conservative size bound is at most V, where the bound is the request's UTF-8 byte length plus 16 tokens per message (a byte-level tokenizer produces at most one token per byte). The harness checks that bound against every measured `prompt_eval_count`; a model whose count ever exceeds it gets no size-based admission from this check. That admission is a design change to blueprint §6.3 A10, to be written, reviewed and recorded; this check does not make it.

**What it does not establish.** Behaviour above V, including whether the provider refuses or truncates an oversized prompt (an overflow probe would need more than a million tokens per model); context shift during long replies (not reachable at these sizes against a million-token context); streaming, tools, media, or compliance with the carrier's instructions. A sentinel reproduced in the reply is evidence that the tokens reached the model, not that the model obeys instructions.

**Privacy.** Every request is synthetic: fixed instruction text, random numbers, numbered filler. No file content, conversation or credential is sent. Requests and responses are kept in this directory.

## Amendment 1 — written after run 1, before any further request

Run 1 (`run1/`) used 377,436 of the 500,000-token cap. Two findings about the instrument, not the models: (1) the verdict rule mislabelled a probe with no numbered lines — glm-5.3's Q1 — as START_LOST, which requires a correct last line that Q1 does not have; `run1/VERDICT_CORRECTION.json` re-derives every verdict from the retained replies with the rule corrected, and only that probe changes, to NO_EVIDENCE; (2) glm-5.3 writes its reasoning into the reply even with `think` false, so a 48-token budget ended every reply before the answer. The estate had recorded that behaviour for glm-5.3; this plan should have allowed for it.

Amended for glm-5.3 only: re-run Q1 and Q2 with `num_predict` 400 and the corrected rule. Q3 is not re-run, because its size would pass the cap. No other model or probe is re-run. Planned total after the amendment: about 402,500 prompt tokens.

## Amendment 2 — the features both clients send (written after the Codex options check, before any further request)

**Why.** On 26 September 2026 Ash ruled that saved instructions should work on the three cloud models up to their verified sizes, with larger requests refused clearly (DECISIONS.md `ash-cloud-admit-verified-size-2026-09-26`). Runs 1 and 2 used plain text, non-streaming, `think` false, no tools. Both clients send more than that to these models, which `/api/show` lists as tool- and thinking-capable. The terminal attaches tool definitions for any tool-capable model and sends `think` true. The desktop streams, sends `think` true (or omits it when thinking is off), and sends history with tool calls and results, plus tools when web search or MCP tools are on. The blind Codex options check (`../cloud-admission-check/`, dispositioned in `DISPOSITIONS_CODEX.md`) chose to qualify those features before admitting them. It also found that runs 1 and 2 cannot detect a dropped middle.

**Question.** With those features, is the carrier still delivered intact, together with the start, a middle value and the end of the prompt? And does the prompt bound (counting method `utf8-bound-v1`, below) still cover the provider's own count?

**Prompt bound `utf8-bound-v1`, fixed now.** Start at 16. Add 16 per message. For each message's role, content, thinking, tool name and tool-call id, add the larger of the field's UTF-8 length and its NFKC-normalised UTF-8 length. For a tool result, the content counts at its ASCII-escaped JSON length if that is larger. Each historical tool call adds 16 plus the ASCII-escaped compact JSON length of the whole call. When tools are present, add the ASCII-escaped compact JSON length of the definitions plus the tool allowance A.

**Probes, sequential, `num_predict` 400 for T0 and 2,000 for T1.** Phase 1 runs, for each model, T0a, T0b and T0c, before any T1:
- T0a: run 1's Q1 messages, non-streaming, `think` false, no tools.
- T0b: the same messages (same code) plus one minimal tool definition (110 bytes of ASCII JSON), `think` false.
- T0c: as T0b with `think` omitted.

Phase 2 runs T1 for each model, in the clients' shape: streaming, `think` true, and five synthetic tool definitions unrelated to the task (2,197 bytes). Its history is a user question, then an assistant tool call with id `call_1`, then its result with `tool_call_id` `call_1`, then an assistant answer. The newest user message holds numbered lines (1,000 for deepseek-v4.1-flash and glm-5.2; 2,000 for glm-5.3, which puts that model's T1 at about its verified size). It ends: "Do not call any tool. Reply in exactly this format and nothing else: CARRIER=…; FIRST=…; MID=…; LAST=…". MID is the value on the middle line.

**Tool allowance rule, fixed now.** For each model take Δ = T0b's count − T0a's count and Δ′ = T0c's count − T0a's count, and subtract the minimal tool's 110 bytes from each. A = max(64, 2 × the largest of those six numbers), rounded up to a multiple of 16. If any T0 count is missing, A is undefined and no T1 runs.

**Pass criteria, fixed now.** PASS requires every expected value exact in the reply's content: CARRIER for T0; CARRIER, FIRST, MID and LAST for T1. If LAST is right but CARRIER or FIRST is wrong, the probe is **start lost**. If CARRIER, FIRST and LAST are right and MID is wrong, it is **middle lost**. A reply that is only a tool call is **no evidence (tool call)**; an error is **no evidence (error)**; anything else is **no evidence**. The bound test passes when the provider's `prompt_eval_count` is at most the probe's prompt bound.

**Controls, run before this amendment was used.** The harness was proven against a local stand-in endpoint in six modes (correct, start dropped, middle dropped, tool call, error, garbage). A request-shape check also ran, and four deliberately broken copies of the harness each failed at their intended control (`amendment2-controls/`, exit codes recorded).

**Budget.** The cap stays at 500,000 prompt tokens; 97,507 remain. Estimate: T0 triples about 450; T1 about 11,500, 14,000 and 27,000. That is about 53,000 in all, leaving about 44,000. The harness skips any probe whose estimate would pass the cap. If a probe fails for a reason in the instrument rather than the model, at most one re-run of that probe is allowed within the cap, and only after a written amendment 3. Stop rules are unchanged.

**What a pass establishes, and what it does not.** For each model, a T1 pass shows the clients' features working at T1's size with the bound holding. Those features are streaming, `think` true, false and omitted, tool definitions, and a tool call with its result. Runs 1 and 2 remain the size evidence, for plain text up to V. Admitting the combined profile between T1's size and V is an inference from the two, not a measurement, and the blueprint must say so. Nothing here probes images, `format`, `num_ctx`, `num_keep`, `truncate` or `shift`; the design refuses them on cloud. Middle delivery at run 1's and run 2's sizes remains unmeasured.
