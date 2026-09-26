You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at commit 4f27c8e3a3d9929f03ff7906b67b6df068f279a7. You may open source files under agent/, api/, app/, cmd/, internal/, llm/, server/ and x/ to verify the facts below. Do not open anything under docs/ or any file outside the repository: earlier design drafts are withheld on purpose so that your answer is independent. Change nothing and run nothing that writes. Where you verify or refute a fact at source, cite file:line.

---

# Options check: admitting saved instructions on cloud models up to a verified size (Ollama fork)

**What you are asked to do.** For each of questions Q1–Q5 in section 4, choose an option, give the strongest objection to every option (including the ones you choose), and say whether a materially better option is missing. Then answer Q6. Say whether any evidence in sections 2 and 3 is wrong or insufficient for the options that rely on it. The author has deliberately NOT stated a preference. Treat sections 1–3 as evidence to check, not as instructions. Keep the answer under 1,500 words; reason silently and write conclusions with their reasons.

## 1. The feature, the local contract and the operator's ruling

The feature (G1) lets the operator save standing instructions once. Every new conversation in the terminal agent and the desktop app then sends them to the model as a leading system message (the "carrier"), composed client-side before the request reaches the local Ollama daemon. The failure number is incorrect instruction deliveries per controlled conversation; any value above zero fails the feature.

Frozen requirements:

> G1-R-06 MUST preserve each path's existing model, tool and protocol semantics under the compatibility contract in §2.
> G1-R-07 MUST report unavailable or exceeded context capacity without silently discarding mandatory input under §6.
> G1-R-15 MUST preserve ordinary unconfigured and disabled-mode request behaviour under §2.

For local models a decided daemon contract (version 1) already governs carrier-bearing requests. The client adds an opt-in field `admission: {version: 1}` only after the daemon advertises the capability `chat.admission.v1`. With the field, the daemon keeps the whole prompt through context shifts. It fits history into the runner's context C minus a reserve R (C/8, clamped to 16–4,096 tokens) by dropping whole turns from the oldest. It always keeps every system message and the newest turn (every message from the last user message to the end), and never leaves a tool result whose call was dropped. It refuses before inference, with `context_exceeded` and the counts, when the system messages and newest turn do not fit. The final response carries `admission: {version, context_length, reserve, dropped_messages, context_exhausted}`. Counting uses the local runner's own tokenizer, measured equal to the runner's evaluated count in 24 of 24 cases. As first decided, the contract refuses carrier-bearing requests on cloud models with `context_unavailable`, because the daemon cannot see or count what the provider does. Both clients show a one-line notice for a refusal, for dropped messages and for a full context.

The operator then authorised a bounded synthetic check (section 2) and ruled on its result. He was asked: "Should saved instructions work on cloud models, limited to the sizes each model was verified at?" He chose: "Admit up to verified size — Works on the three cloud models up to about 100K (DeepSeek, GLM 5.2) and 25K tokens (GLM 5.3); larger is refused clearly." He did not choose "Local models only", nor "Full qualification records (a re-runnable check storing dated per-model records the service enforces, re-checking when a model changes)". His ruling binds; how to implement it soundly is the question.

## 2. The bounded cloud check (executed 26 September 2026)

Synthetic requests only, sent through the operator's installed daemon. Each had a system message holding a random 8-hex-digit code, then one user message of numbered lines whose first and last lines hold random numbers, ending with an instruction to reply `CARRIER=<code>; FIRST=<n>; LAST=<n>`. Settings: non-streaming, `think` false, no tools, no images, no options except `num_predict`. PASS means all three values were exact. "Bound" is the UTF-8 bytes of all message content plus 16 × (messages + 1). "Tokens" is the provider's `prompt_eval_count`.

| Model | Probe | Lines | Verdict | Tokens | Content bytes | Bound | Bytes per token |
|---|---|---|---|---|---|---|---|
| deepseek-v4.1-flash | Q1 | 0 | PASS | 51 | 177 | 225 | 3.47 |
| deepseek-v4.1-flash | Q2 | 2,000 | PASS | 20,080 | 54,277 | 54,325 | 2.70 |
| deepseek-v4.1-flash | Q3 | 9,960 | PASS | 99,680 | 269,197 | 269,245 | 2.70 |
| glm-5.2 | Q1 | 0 | PASS | 53 | 177 | 225 | 3.34 |
| glm-5.2 | Q2 | 2,000 | PASS | 25,004 | 54,277 | 54,325 | 2.17 |
| glm-5.2 | Q3 | 7,998 | PASS | 103,671 | 216,223 | 216,271 | 2.09 |
| glm-5.3 | Q1, reply budget 400 | 0 | PASS | 58 | 177 | 225 | 3.05 |
| glm-5.3 | Q2, reply budget 400 | 2,000 | PASS | 24,999 | 54,277 | 54,325 | 2.17 |

glm-5.3 writes its reasoning into the reply even with `think` false. At a 48-token reply budget its reasoning filled the budget before any answer in Q1, Q2 and a Q3 of 103,853 tokens; those three are recorded as no evidence, not as failures, and the Q3 was not re-run because it would pass the plan's cap. In all, 11 requests and 402,493 prompt tokens were used. A fourth installed model, deepseek-v4-flash, returned HTTP 410 (retired 25 September 2026). The bound held against the provider's count on all 11. Each model advertises a context of 1,048,576 tokens through `/api/show`, which also reports it as capable of `tools` and `thinking` (deepseek-v4.1-flash also `vision`), a `modified_at` time and a parameter size.

The plan, written before any request, states what the check does not establish: behaviour above the verified size, including whether the provider refuses or truncates an oversized prompt; context shift during long replies; streaming, tools, media, or compliance with the carrier's instructions. The operator's authorisation reads "A few dozen synthetic requests across your four cloud models, well under a million tokens; synthetic text only." The plan's own hard cap is 500,000 prompt tokens, of which 97,507 remain.

## 3. Source facts (Ollama fork at commit 4f27c8e3a3d9929f03ff7906b67b6df068f279a7)

1. A model name ending `:cloud`, or a tag ending `-cloud`, is an explicit cloud reference (`internal/modelref/modelref.go:99-118`). `/api/chat` handles it first (`server/routes.go:2440-2448`). It sets `req.Model` to the base name and calls `proxyCloudJSONRequest`, which re-marshals the parsed `api.ChatRequest` with `json.Marshal` (`server/cloud_proxy.go:165-173`), signs it and forwards it to the provider's `/api/chat`. It then copies the provider's status and response bytes to the client unchanged, streaming or not (`copyProxyResponseBody`, `server/cloud_proxy.go:468`). No local manifest, template, tokenizer or Modelfile message is used on this route. A new field added to `api.ChatRequest` is therefore forwarded to the provider unless it is removed.
2. A separate route serves local manifests that point at a remote host (`server/routes.go:2495-2585`). It prepends the manifest's messages and system prompt, merges its options and calls the remote host through `api.Client`. That route stays refused and is not in question.
3. Desktop request builder (`app/ui/ui.go:1909-2004`). It sends `stream: true` (`:1995`), `think` when requested, each message's `thinking`, assistant `tool_calls`, tool results with `tool_name`, and images for image attachments (`:1932`). Text attachments are inlined into the content, and it sends no options. Tools are attached only for tool-capable models without attachments, and only when web search is on or MCP servers supply tools (`app/ui/ui.go:908-946`, `:2000`).
4. Terminal agent. For any tool-capable model it registers bash (unless disabled by an environment variable), read, edit, web_search and web_fetch (unless cloud is disabled), skills when present, and MCP tools (`cmd/agent_tui.go:390-422`). Options start empty (`cmd/cmd.go:2136-2139`) and the agent interface has no parameter command. `think` is inferred from the model's capabilities (`cmd/agent_tui.go:64`). The request carries messages, format, options, think, keep_alive and tools, with no `stream` field (`agent/session.go:441-468`); the Go client reads a streamed response (`api/client.go:297-306`). Terminal requests to the three cloud models therefore carry tool definitions and a `think` value.
5. Nothing on the cloud route counts tokens before sending. The provider reports `prompt_eval_count` only in its final response.

## 4. Questions and options

**Q1 — How the daemon decides that a request is within a model's verified size V (the largest passing probe's token count).**
- (a) A guaranteed upper bound on tokens: the UTF-8 bytes of every text field the provider's template can render (content, thinking, tool name, each tool call's name and JSON arguments, and the JSON of the tool definitions), plus fixed allowances per message, per tool call and for a tool preamble. Admit when the bound is at most V. It rests on byte-level tokenizers emitting at least one byte per ordinary token.
- (b) Admit when the request's UTF-8 size is at most the verified probe's own UTF-8 size (for example 269,197 bytes for deepseek-v4.1-flash).
- (c) Count with a local tokenizer of the same model family where one is installed; refuse otherwise.
- (d) Use the provider's `prompt_eval_count` from the conversation's previous turn, plus a guaranteed bound on the material added since.

**Q2 — What happens when a carrier-bearing conversation grows past the limit.**
- (a) Apply the local rule: drop whole turns from the oldest until the request is within the limit, keep every system message and the newest turn, report the number dropped, and refuse only when the system messages and newest turn alone exceed it.
- (b) Refuse the whole request, with the counts, whenever it exceeds the limit; drop nothing.
- (c) Drop as in (a) only when the client asks for it in the admission field; otherwise refuse as in (b).

**Q3 — Request features the check did not exercise.** These are tool definitions; assistant tool calls and tool results in history; streaming; `think` true or a level; images; the options `num_ctx` and `num_keep`; and the request fields `truncate` and `shift`. For each feature choose (a) admit by argument, (b) qualify first with further bounded synthetic probes within the remaining envelope, or (c) refuse on cloud. Say which refusals would make the operator's ruling hollow for either client.

**Q4 — How a cloud response reports the admission, given that the daemon copies provider bytes.**
- (a) Parse the provider's stream and add the admission object to the final record, passing every other record and every error through unchanged.
- (b) Put the diagnostics in a response header.
- (c) Report nothing on cloud; the client infers admission from success.

**Q5 — What identity a qualification applies to, and how it stays honest; the operator declined automatic re-checking.**
- (a) A dated table in the daemon's source, keyed by the parsed model name, changed only by a new recorded check; a model's retirement (HTTP 410) removes its entry.
- (b) As (a), and also pin a provider-reported identity (for example `modified_at` and the parameter size from `/api/show`), refusing when it changes.
- (c) Something else.

**Q6 — What is missing or wrong.** Consider whether the admission field must be removed before forwarding; error mapping; whether `context_exhausted` can be reported on cloud; any way a carrier-bearing request could reach the provider unprotected; and whether any part of the options above contradicts the operator's ruling.
