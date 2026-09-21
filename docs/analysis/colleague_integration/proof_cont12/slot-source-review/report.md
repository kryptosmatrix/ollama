## Counterexample check on the claimed guard

The excerpts support the selection, with two precise limitations:

1. **`slot.task->n_tokens() > slot.n_ctx` at `:3131` and `>= slot.n_ctx` at `:3142` fire against `slot.n_ctx`, which is set from `llama_n_ctx_seq(ctx_tgt)` capped to `llama_model_n_ctx_train` (`:1250–1256`).** So the guard's denominator is the *actual allocated slot*, not requested `NumCtx`. Confirmed against `llm/llama_server.go:330–332`, where `ContextLength()` returns `s.options.NumCtx` — the Go wrapper cannot substitute for the capping.

2. **The guard fires only when the slot enters `SLOT_STATE_PROCESSING_PROMPT` at `:3069–3083`, i.e. after the slot was already assigned a task.** "Before prompt evaluation" is satisfied — `n_past` and cache reuse are computed *after* the checks (`:3099–3150`). "Before queueing" is not.

**Counterexample to guard-as-sole-admission:** if a request's tokens exceed capacity, the native server *accepts the task, allocates a slot, and then sends an error via `send_error(..., ERROR_TYPE_EXCEED_CONTEXT_SIZE)`* (`:3132–3137`, `:3143–3148`). That is a runtime rejection, not a pre-queue refusal. G1-R-07 ("report unavailable or exceeded without silently discarding") is satisfied — no silent discard occurs — but the rejection is post-allocation and post-slot-bind. So this guard is a **sufficient correctness site but not a pre-queue admission gate**; any design claiming it prevents *queuing* oversize input is wrong.

**Counterexample to output reserve:** the guard checks input only. Output is bounded separately at `:1858–1873` (`!params_base.ctx_shift` → stop when `prompt.n_tokens() + 1 >= n_ctx`; also `has_budget(params_base)` / `n_predict`). So "input + reserved output" is not enforced by `:3131/:3142`; it must be added as extra arithmetic against the same `slot.n_ctx`, and `Shift` handling must match `routes.go:2611` semantics. The draft already says this.

**Counterexample to `can_split()` branch:** for `!can_split()` the threshold is `> n_ubatch` first (`:3120`), then `> slot.n_ctx` (`:3131`). For `can_split()`, only `>= slot.n_ctx` (`:3142`). These are different failure modes at different thresholds; the extension must preserve both, not collapse them.

**No counterexample found** to the claim that this site is the correct place to see `slot.task->tokens` as the actual prepared input.

## Remaining integration decisions

1. **Pre-queue vs pre-eval contract.** The doc says "reject before prompt evaluation". That matches this guard's firing point. If the intent is "reject before the daemon allocates a slot for it", the guard does not do that and would need an earlier check (in `handle_completions_impl` or via a count-only admission service). Decide which; the draft's current wording permits the former.
2. **Output reserve representation.** Additive to `slot.n_ctx` in a new comparison alongside `:3131/:3142`, with `can_split`/`!can_split` parity and `params_base.ctx_shift` semantics matched to `:1858–1866`. Whether the reserve is `n_predict`, `n_predict + n_keep`, or a policy value is not yet fixed.
3. **Versioned ingress / capability so an older daemon cannot silently ignore the reserve.** Not addressed by this guard alone; unchanged from the previous consultation.
4. **Backend coverage for the guard-extension.** `slot.task->tokens` exists only on the local llama-server path; MLX, media (`has_mtmd` disables cache reuse at `:3164–3166`), and remote/provider have no equivalent slot. Their outcome must be named (`context_unavailable` or refusal) — the guard cannot cover them.

## Correction to my previous consultation

- I implied a **new pre-task hook between tokenization and task build** was the only sound site. That is wrong: the existing guard at `:3068–3150` already compares actual dispatched tokens to the actual capped `slot.n_ctx`, which is a stronger denominator than `llm/llama_server.go:330–332`'s `ContextLength()` (= requested `NumCtx`). My earlier "existing prepared-input boundary" was vague; this guard is a concrete one, and my "actual fully prepared input that will be queued" preference was satisfied here for *input size*, not for *queue timing* — which is the distinction I should have drawn earlier.
- **Version-only timeout vs executable behavior:** the timeout proves the command did not return in budget. It does **not** prove what the executable did or did not do internally before/after the timeout — no model-specific probe was launched, and internal binary activity was not observed. I make no claim about either.

No promotion, permission, or verdict.
