For Q1, I choose **(a) conditionally**: its claimed guarantee is currently unsupported. Request-field bytes do not necessarily bound the provider’s rendered prompt.

[Inference] The recommendations below support bounded cloud availability; they do not establish the stronger zero-error delivery claim.

For **Q1**, the strongest objections are:

- **(a):** The tokenizer premise must apply after normalisation and template expansion. Fixed allowances need justification for special tokens, repeated material and tool rendering. The enumeration also misses variable-length tool-call IDs ([api/types.go:197](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/types.go:197)) and potentially `format` ([api/types.go:143](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/types.go:143)). It is substantially conservative: it rejects the passing DeepSeek probe because 269,245 exceeds 99,680.
- **(b):** Equal byte lengths can produce different token counts. The measured bytes-per-token ratios cannot safely become conversion factors.
- **(c):** “Same family” does not establish identical vocabulary, normalisation, special tokens or rendering. Installation-dependent refusal also undermines cloud availability.
- **(d):** The previous count covers the previous rendered request. Changed tools, thinking modes, history or template treatment can invalidate additive accounting; an already-truncated previous prompt is especially dangerous.

A materially better missing option is provider-side counting bound to the actual inference request, with enforced refusal instead of truncation. No such facility is established here. Until (a)’s assumptions are established, calling it “guaranteed” substitutes confidence for evidence.

For **Q2, choose (b)**. It most directly implements “larger is refused clearly” and preserves the submitted conversation.

Its strongest objection is unnecessary refusal when disposable history could have been removed. **(a)** changes conversational meaning and stretches the operator’s wording by admitting a rewritten oversized request. **(c)** makes that change explicit, but introduces another policy switch and inconsistent client behaviour. No materially better independent policy is missing; early size notices would improve (b).

For **Q3**, my choices and objections to every alternative are:

- **Tool definitions — (b).** Against (a): provider tool rendering is unverified. Against (b): finite examples cannot cover arbitrary MCP schemas. Against (c): it blocks normal terminal-agent use.
- **Assistant tool calls and tool results — (b).** Against (a): history rendering and call/result associations were not exercised. Against (b): one successful round trip does not cover parallel calls or varied histories. Against (c): conversations stop when tools become useful.
- **Streaming — (b).** Against (a): transport-only equivalence is plausible but unverified upstream. Against (b): successful probes do not establish correct handling of fragmented records, errors or premature termination. Against (c): both clients normally stream.
- **Thinking true or a level — (b).** Against (a): false does not establish other modes’ rendering or output behaviour. Against (b): each supported mode and reasoning-budget interaction needs coverage. Against (c): thinking defaults to true for capable models in both clients ([cmd/cmd.go:2592](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/cmd.go:2592), [app/ui/ui.go:879](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:879)).
- **Images — (c).** Against (a): text bytes do not establish image-token cost. Against (b): media probes exceed the text-only authorisation. Against (c): this excludes otherwise supported vision conversations.
- **`num_ctx` and `num_keep` — (c), when explicitly supplied.** Against (a): they could invalidate capacity or retention assumptions. Against (b): selected values cannot establish arbitrary-value semantics. Against (c): useful overrides are excluded.
- **`truncate` and `shift` — (c), when explicitly supplied.** Against (a): forwarding flags does not establish that the provider honours them. Against (b): small probes cannot establish behaviour at context exhaustion. Against (c): protective false settings are rejected alongside dangerous settings.

Refusing streaming makes the ruling hollow for both clients; refusing tools or thinking makes it hollow for the terminal’s normal workflow and substantially restricts desktop use.

A better testing arrangement is combined, realistic client profiles with synthetic tool results, plus deterministic proxy-framing tests. Include omitted thinking, explicit false and supported enabled modes. The remaining 97,507 tokens cannot fund even one repeat of either largest approximately 100K probe. Short compatibility probes must not be described as new boundary qualification.

For **Q4, choose (a)**, covering non-streaming JSON too.

Its strongest objection is protocol damage: parsing and reserialising can lose unknown fields, alter errors or mishandle termination. Preserve other records and errors; augment only a successful final object, retaining unknown fields. **(b)** requires new client header plumbing and cannot carry outcomes discovered after headers are sent. **(c)** cannot distinguish protected admission from accidental ordinary success.

No materially better channel is missing. Client handling is essential: the desktop currently skips records without message content, thinking or tool calls, including ordinary empty final records ([app/ui/ui.go:992](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:992)).

For **Q5, choose (c)**: a minimal dated source table keyed by provider origin, exact forwarded model name and supported request profile, recording V and the counting method. Changes require a recorded manual check; retirement disables eligibility. This does not require automatic synthetic rechecking.

**(a)’s** strongest objection is undetected alias or provider changes. **(b)’s** is false assurance: modification time and parameter size do not identify the inference deployment or template, and `/api/show` can return stale cloud metadata ([server/model_show_cache.go:180](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/model_show_cache.go:180)). **My (c)** still cannot detect invisible provider changes; its dated evidence must not be presented as continuous certification.

A materially better identity would be an immutable provider revision covering model, tokenizer and rendering, enforced on inference. Its availability is unestablished.

For **Q6**, these requirements remain:

- Consume and remove `admission` before forwarding. Validate its version before any eligible routing; never retry without protection.
- Preserve structured refusal codes and diagnostic counts through both clients. The Go stream reader currently retains only error text and sign-in information ([api/client.go:222](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/client.go:222)). Avoid desktop substring classification: a count containing “402” could otherwise become a billing error ([app/ui/ui.go:600](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:600)).
- Distinguish local admission refusals from authentication, retirement, rate limits and upstream failures. Map provider context errors only where their meaning is established.
- Report V as a **verified prompt limit**, not actual context capacity. Report the bound as a bound, not an evaluated count. Do not invent a cloud reserve or `context_exhausted: false`; exhaustion is unknown without a trustworthy provider signal.
- Input admission does not establish retention during long generation. The tested reply budgets do not justify silently imposing equivalent limits on ordinary clients. A provider retention guarantee or separately justified output policy remains missing.
- Require carrier and admission together on every controlled request, including tool continuations. Unsupported daemons must not trigger unprotected fallback.
- Guard or explicitly reject admission on compatibility endpoints before their cloud passthrough: these can bypass `ChatHandler` ([server/routes.go:1898](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:1898), [server/cloud_proxy.go:133](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/cloud_proxy.go:133)). Keep unconfigured requests unchanged.

Blanket cloud refusal, silent feature removal and mandatory automatic qualification would contradict the ruling. A conservative byte gate must also be described honestly: it does not provide full practical utilisation of the advertised approximately 100K allowance.

On **sections 2 and 3**: native cloud routing, remarshal and response copying are verified ([server/routes.go:2440](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:2440), [server/cloud_proxy.go:165](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/cloud_proxy.go:165), [server/cloud_proxy.go:242](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/cloud_proxy.go:242)). Remote manifest system insertion is conditional ([server/routes.go:2520](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/server/routes.go:2520)); its admission refusal is a proposed policy, not implemented here. `ChatRequest` currently lacks admission ([api/types.go:133](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/api/types.go:133)). Terminal tools can be disabled ([cmd/tui/chat/input.go:402](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/cmd/tui/chat/input.go:402)); desktop false thinking is omitted ([app/ui/ui.go:1978](/Users/krypto/GitHub/ollama-eko-chat-read-integrity/app/ui/ui.go:1978)).

[Unverified] Section 2’s execution results remain supplied evidence. The remaining-budget arithmetic is correct; the eight displayed passes total 273,596 tokens. Three exact recalled values cannot exclude selective middle truncation or establish universal instruction compliance. “11 requests” also excludes the additional HTTP 410 attempt.

Process caveat: I read one unrelated external tooling-instruction file. Git attempted sandbox-blocked cache writes. No files were changed.