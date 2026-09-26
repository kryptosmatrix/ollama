# Dispositions of the Codex options check on cloud admission (Thole, continuation 13)

Reviewer: Codex `gpt-6-astra` (OpenAI), read-only sandbox over the worktree at `4f27c8e3`, docs withheld, packet SHA-256 `2ca31ab42bbbf42c09d0f0b3df6eb176bce120fb8b3d1a2143c79340e419c640`, exit 0 at 2026-09-26T21:06:47+1000 (`reviews/codex/`). Every source claim below was re-read at the cited file before it was dispositioned; line numbers are at `4f27c8e3`.

## Source claims

| Claim | Check | Disposition |
|---|---|---|
| Tool calls carry variable-length IDs the bound omits | `api/types.go:221-224` (`ID string json:"id,omitempty"`); the message also carries `ToolCallID` at `:206` | Confirmed (Codex cited `:197`, the start of `Message`; the fields are at `:206` and `:222`). Adopted: the bound counts every rendered field of `Message` and `ToolCall` |
| `format` may be rendered by the provider | `api/types.go:143-144`; terminal sets it from `opts.Format` (`cmd/tui/chat/chat.go:1163`), desktop never | Confirmed as unmeasured. Adopted: refused on cloud (narrow; no default client path sends it) |
| Both clients default `think` to true for capable models | `cmd/cmd.go:2592-2593`; `app/ui/ui.go:879-887`; desktop omits false (`:1978-1986`) | Confirmed. Adopted: amendment 2 probes think false, omitted and true |
| The desktop skips records without content, thinking or tool calls | `app/ui/ui.go:992-994` | Confirmed. Adopted as a B2b obligation: read the final record's `admission` before that skip, on every route |
| `/api/show` cloud metadata can be stale | `server/model_show_cache.go:177-186` (`GetCloudSWR`, stale-while-revalidate) | Confirmed. Q5(b) rejected: `modified_at` and parameter size are not an identity |
| The Go stream reader keeps only error text and sign-in URL | `api/client.go:221-237` | Confirmed. Adopted: refusals carry `code` and details, and the Go client's error type keeps them (B2a) |
| Desktop classifies errors by substring; a count containing "402" becomes a billing error | `app/ui/ui.go:598-606` | Confirmed. Adopted: adapters classify admission refusals by code before any substring rule (B2b) |
| Compatibility endpoints can bypass the chat handler through the cloud passthrough | `server/routes.go:1898-1903` (`cloudPassthroughMiddleware` on `/v1/chat/completions`, `/v1/completions`, `/v1/responses`); `server/cloud_proxy.go:73-136` (`cloudPassthroughMiddleware`) forwards the raw body at `:133` | Confirmed. Adopted: those routes and `/api/generate` refuse a body with a top-level `admission` member (`invalid_admission`), before passthrough |
| Terminal tools can be switched off | `cmd/tui/chat/input.go:402` (`/tools`) | Confirmed; no design change (fewer features is inside the profile) |
| Native cloud routing, re-marshal and byte copying | `server/routes.go:2440-2448`; `server/cloud_proxy.go:165-173`, `:242` | Confirmed, as the packet stated |
| Remote-host system insertion is conditional | `server/routes.go:2520` (only when the first message is not a system message) | Confirmed; the packet's wording was loose. That route stays refused |
| The eight passes total 273,596 tokens; "11 requests" omits the HTTP 410 attempt | Arithmetic: 51+20,080+99,680+53+25,004+103,671+58+24,999 = 273,596; the 410 was a twelfth request | Confirmed. Records now say 11 requests to the live models plus one refused request |
| First, last and carrier values cannot exclude middle truncation | Probe design, `cloud_check.py:37-63` | Confirmed, and not previously recorded. Runs 1-2 give no middle evidence; amendment 2 adds a middle value |

## Choices

**Q1 — the measure.** Codex chose the byte bound conditionally and objected that calling it guaranteed overstates it: the provider's rendered prompt adds template text, and normalisation can change bytes. Adopted with changes. The measure is named a *prompt bound*, not a guarantee. It counts, per field, the larger of the UTF-8 length and the NFKC-normalised UTF-8 length. For tool definitions, tool calls and tool results, which templates may render as JSON, it counts the ASCII-escaped JSON length. It adds fixed allowances that every qualification probe checked against the provider's own count. Its premise and falsifier are stated in the blueprint. Codex's missing option — a provider-side count bound to the inference with refusal instead of truncation — does not exist on this route; it is recorded under reconsider-when. Options (b)-(d) rejected for the reasons Codex gave, each consistent with the source.

**Q2 — over the limit.** Codex chose refusal. Adopted. Two reasons settle it. The operator's words are "larger is refused clearly". And the cloud route drops no history today, so fitting would introduce message loss there, driven by a bound that is two to four times the real count, discarding history the model could have used (G1-R-06). The local routes keep A5's fitting, because truncating history is what they do today. Codex's improvement is adopted: every admitted cloud response reports the limit and the request's bound, so the clients can warn before the limit is reached.

**Q3 — unexercised features.** Codex chose to qualify tools, tool history, streaming and thinking before admitting them, and to refuse images, `num_ctx`, `num_keep`, `truncate` and `shift` when supplied. Adopted, plus `format`. Amendment 2 of the qualification plan does this within the plan's existing cap, and it is described as compatibility evidence at the probes' sizes, not as boundary qualification. Codex notes the remaining 97,507 tokens cannot repeat a probe of about 100K. The combined profile above the probed size is therefore an inference, and the blueprint says so.

**Q4 — reporting.** Codex chose to annotate the final record, for non-streaming responses too, keeping unknown fields. Adopted in the least invasive form: the daemon inserts one `admission` member into the successful final JSON object without re-encoding the rest of it. Every other record and every error passes through byte for byte.

**Q5 — identity.** Codex chose a dated table keyed by provider origin, exact forwarded model name and request profile, recording V and the counting method. Adopted. Q5(b) is rejected on the stale-cache finding above. The table's evidence is described as dated, never as continuous certification.

**Q6 — remaining requirements.** All seven adopted, with the source work named above. The admission field is consumed and removed before forwarding, after version validation. The carrier and the admission field travel together on every controlled request, including tool continuations. No reserve, `context_length` or `context_exhausted: false` is reported on cloud. Retention during long generation is stated as unobserved, with the arithmetic that bounds its reach, and no output limit is imposed on clients. Codex also asked that the byte gate's cost be described honestly: it admits roughly a quarter to a half of V in real tokens. The report to the operator says so.

**Process caveat reported by Codex.** It read "one unrelated external tooling-instruction file" and git attempted sandbox-blocked cache writes; nothing changed. No held content applies to this repository (checked: the programme's HANDOFF, STATE and the successor document hold no hold), so the read breaches no hold.
