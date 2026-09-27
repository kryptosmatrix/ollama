## 1. Real requirement vs author-added restriction

**Real (from G1-R-07 + commission):** measure the *complete actual request budget* for the carrier that will be dispatched; retain carrier/model protocol/tools/required context; reserve output explicitly; fail honestly on unknown/overflow; do not silently discard mandatory input.

**Author-added restrictions not implied by the requirement:**
- "Exact tokenizer for every provider" — not derivable; the source itself shows only local tokenization and `len/4` for Anthropic.
- "Cloud/local must both pass admission in G1" — restricting a backend is *explicitly permitted* by the commission ("restricting a backend must be explicit… cannot be sold as full multi-backend delivery").
- "Single strict ingress must be `/api/chat/strict`" — the excerpt flags this only as *one possible* mechanism; the requirement is that an old daemon cannot silently accept a new strict flag.
- "One-token measurement must be production default" — evidence shows `num_predict=0`→400 and one-token calls *infer*; that is a candidate, not a mandate.

## 2. Source-backed strongest objection to each viable option

**(A) one-token full-input inference as measurement.** The source shows real inference happens (one-token probes returned 25/502/502 input tokens with one output token) and that `num_predict=0` is rejected upstream (`max_tokens must be positive`). Cost, latency, and side effects are real. `promptEvalCount = CacheN + PromptN` (`llm/llama_server.go:1473`) is a *post-hoc* measure, not a preflight, and does not prove the *untruncated* requested input was counted — only what the runtime saw. It cannot be sold as zero-cost tokenization, and it does not exist for every cloud path.

**(B) opt-in strict admission at final prepared-input boundary.** Strongest objection: on the local path there are at least two final rendered forms — completion prompt (`completionPromptForRequest`, respects `tokenizerAddsBOS`) and native chat (`ApplyChatTemplate` + `Tokenize`, `server/routes.go:3029–3037`). Public `Tokenize` undercounts relative to what the chat template emits (`add_special`/BOS differ, `completionPrompt` trims BOS when the runner adds it). So "share actual rendering/tokenization" is only sound if the strict path consumes the *exact* function that will be dispatched (`ApplyChatTemplate` output fed to `Tokenize` for native; the completion prompt bytes fed to `tokenize(prompt, true, …)` for completion), which the source does contain — but only on the local runner. For MLX/native-with-remote/projector/media, no such function exists in the excerpt, so `context_unavailable` is the only honest outcome.

**(C) preserve whole outgoing messages + post-run counts without claiming preflight.** This *meets* G1-R-07 literally (report capacity rather than silently discard). Objection: post-run counts arrive *after* inference, so overflow is detected by the model, not by admission; if the local runner errors with the existing "prompt is longer than context length" branch (`llm/llama_server.go:293–296`), that is a legitimate `context_exceeded` signal — but the client cannot distinguish it from an unrelated 400 without a typed contract. It also cannot pre-empt cost/latency on cloud.

**(D) derived minimal design — strict admission only where an exact shared renderer+tokenizer exists, and honest `context_unavailable` elsewhere.** Objection: it declines to claim parity across backends and must accept that G1-R-07 is *satisfied per-backend*, with disabled/unqualified backends returning `context_unavailable` rather than passing. This is what the commission already licenses.

## 3. Concrete minimal mechanism + qualification obligations

**Contract (bounded, per backend):**

- **Inputs to admission:** one opaque *PreparedRequest* record containing (a) model identity, (b) the exact `[]api.Message` after every transformation the dispatch path will apply, (c) `Tools`, (d) `Think`, (e) `Format`, (f) `NumCtx` actually in effect (must equal `optionsForPrompt` result, not the requested value), (g) `OutputReserve` (explicit integer, ≥1, from `Options.NumPredict` after `boundedNumPredict`), (h) media list.
- **Admission function:** `Admit(PreparedRequest) → (used int, ok bool, reason)` where `used` is exact prompt tokens **of exactly what will be dispatched**.
- **Local completion path:** render via the existing `completionPromptForRequest` prefix logic minus truncation; tokenize that *exact* string with the same `addSpecial` setting the runner will use (`s.tokenize(ctx, prompt, true, …)` semantics), and account `tokenizerAddsBOS()` consistently.
- **Local native chat path:** call `r.ApplyChatTemplate(ctx, nativeReq)` on the *final* message set, then `r.Tokenize` on the returned prompt. Do **not** call `Tokenize` on individual messages; the excerpt already shows the templates do their own BOS/special insertion.
- **Decision:** if `used + OutputReserve > NumCtx` → `context_exceeded` and refuse before dispatch; if `used + OutputReserve > ContextLength()` reported by runner → `context_exceeded`; if the branch cannot produce an exact `used` → `context_unavailable` and refuse *unless* the caller has explicitly opted into post-hoc-only behavior (see below).
- **Mandatory input preservation:** carrier + tool schemas + thinking/format overhead are part of `PreparedRequest`; if any cannot be represented in the prompt form (e.g. image heuristic only), the branch is `context_unavailable`, never "approx pass".
- **`Truncate=false`, `Shift=false`** set on the outgoing request are necessary but insufficient; they do not substitute for admission.
- **Opt-in only:** the strict-admission path is off by default; when off, existing behavior (including `truncate` default) is byte-preserved (G1-R-15). A daemon that does not understand strict mode must reject the request, not ignore the flag — a distinct ingress (e.g. a header or subroute) whose 404/400 is observable satisfies this.
- **Per-branch named outcomes (required by commission):**
  - **local completion:** supported exact (`used` via `tokenize`); media → `context_unavailable`.
  - **local native chat:** supported exact via `ApplyChatTemplate`+`Tokenize`; media → `context_unavailable`.
  - **MLX:** `truncate` is forced off (`routes.go:2673–2675`); admission must use the same path as native if `ApplyChatTemplate`/`Tokenize` are available, otherwise `context_unavailable`.
  - **remote/provider branch (`routes.go:2495–2585`):** no exact tokenizer in source; `context_unavailable` unless the caller accepts post-hoc mode, in which case `context_exceeded` is reported from the provider error only when a typed upstream status is translated; otherwise unknown → `context_unavailable`.
  - **media:** `context_unavailable` (heuristic `imageNumTokens=768` at `prompt.go:30` is explicitly a heuristic).
- **Output reserve:** an integer carried on the PreparedRequest, sourced from the actual `NumPredict` after `boundedNumPredict`, never invented after a count. If absent → `context_unavailable`.

**Qualification obligations for this contract:**
1. Prove `used` computed by admission equals `promptEvalCount` (`CacheN + PromptN`) on a warmed local run for the *first* turn (cache empty), for both completion and native paths, for at least one model where `tokenizerAddsBOS()` is true and one where it is false.
2. Prove overflow → refusal occurs before any token leaves the process (no upstream request emitted).
3. Prove `context_unavailable` is returned (not silently passed) on: MLX, remote branch, any media present, any unqualified renderer.
4. Prove disabled/unconfigured preserves existing request bytes and adds no inference call (G1-R-15; I-06).
5. Prove a daemon lacking the strict ingress rejects the strict request observably.

## 4. Can a local-only stage count as full G1?

**No.** G1-R-07 is one of eighteen frozen outcomes and the commission explicitly treats "local AND permitted cloud use" as the eventual target, with "restricting a backend must be explicit… cannot be sold as full multi-backend delivery." A local-only strict-admission stage is therefore an **explicitly incomplete delivery stage** for G1-R-07: it may satisfy the outcome *for the branches it covers*, and it must declare `context_unavailable` for the others, but it cannot close OI-03/OI-04 on G1-R-07 alone. It can be a legitimate *first* pack boundary.

## 5. Unresolved questions (≤5)

1. **Which exact function is "the last point before dispatch" on the local native path in the current source revision** — does `handleNativeChat` pass the `msgs` slice it was given straight to `llamaServerChatRequest`, or is there an intermediate transformation not shown in the excerpt? Admission must bind to whatever that function is.
2. **Does `ApplyChatTemplate` on the running local runner already apply `add_special`/BOS consistently with `Tokenize`'s default?** The excerpt shows `completionPrompt` explicitly trims BOS when `tokenizerAddsBOS()` is true, implying the two tokenization entry points differ; the exact reconciliation for native chat is not shown.
3. **For MLX, is `ApplyChatTemplate`/`Tokenize` available via the same `LlamaServer` interface** (`llm/server.go:66–68` suggests yes), and does it respect the model's actual chat template (i.e., is it sound to reuse for admission)?
4. **What is the typed translation of a provider 400 carrying "prompt is longer than context length"** at the remote branch, and can it be distinguished from unrelated 400s without a new provider-specific contract?
5. **Is `OutputReserve` required to be ≥ a stated minimum** (e.g. 1) or is `NumPredict` allowed to be the caller's full context, in which case admission must treat `used + reserve > NumCtx` as satisfied only when `used` is effectively 0?
