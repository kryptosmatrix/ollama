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
