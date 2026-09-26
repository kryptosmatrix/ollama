# OQ-1 options check — dispositions of the DeepSeek answer

Reviewer: `deepseek-v4.1-flash` (DeepSeek), served through the operator's installed Ollama daemon
`0.32.5-47-geb8c93d` at 127.0.0.1:11434, wrapper `TECHNE/Tools/ollama/ask.sh`, model identity `PINNED`
(digest `e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8`), `--think off`, temperature 0,
seed 0, `done_reason: stop`. Packet: `PACKET.md`, 42,010 bytes, SHA-256
`8631e33176d8e78fdf209fcb3defcc298d92f7c5a93634492698b6c2ee028a51` (the wrapper's `meta.json` records the
same `prompt_sha256`). Packet contents: source excerpts from the Ollama fork at `04009c43` and llama.cpp at
`b4d6c7d8`, the continuation-13 measurement records, and requirement texts; no credentials, no held content
(held-content check: 2,001 held lines indexed, 0 hits; positive control on a held file fired).
Answer: `reviews/deepseek/response.txt`, 10,407 bytes.

Each claim below is dispositioned against the evidence in the packet or at source. "Refuted" means the cited
evidence says otherwise; "accepted" means the claim changes the design.

| # | Claim (reviewer's words, abridged) | Disposition | Evidence |
|---|---|---|---|
| D1 | "In every long probe the carrier span (tokens 16–67) sits inside n_keep (4 or 106/110), so shift discards history, not the carrier." | **Refuted.** With the default keep of 4, positions 16–66 lie in the discarded range. | run1 `shift_default_long` (both arms): `slot context shift, n_keep = 4, n_left = 507, n_discard = 253`; server-context.cpp:2884 removes `[n_keep, n_keep + n_discard)` = `[4, 257)`. The reviewer applies the same reasoning correctly to truncation in its own "facts" bullet 3, so the answer is internally inconsistent here. |
| D2 | P1 "does not stop the rendered-route truncation… returns 400 only because the post-truncation prompt then exceeded the slot; a slightly smaller oversize would return 200 with the carrier cut." | **Refuted.** Under shift off the Go runner refuses before any cut. | run1 rendered `oversize_shift_false` error text is the no-shift refusal at llama_server.go:292-297, which precedes the cut at :299-315. The `prompt.go:77` log line is `chatPrompt`'s message count after re-adding system messages (prompt.go:38-44, 76-77); it dropped nothing here. No path under shift off reaches the cut. |
| D3 | `truncateNativeChatMessages` "could still drop a system message if the client's estimate is wrong." | **Refuted.** | routes.go:3020-3025 and 3060-3065 re-add every system message from the skipped range. |
| D4 | OI-04 line 298 supports P4 (client assembles the prompt). | **Refuted.** The quoted line says the opposite. | Commission:298: "Trace and extend the existing `SystemPrompt` path rather than introduce a parallel prompt assembly." P4's client-side history selection and token accounting is a parallel assembly. |
| D5 | P4 sets shift off "once per session" whereas P1 toggles per request, so the reload cost counts against P1 only. | **Refuted.** Both send shift off on every carrier-bearing request; the reload arises only when one model alternates between carrier-bearing and ordinary requests, identically under both. | sched.go:1413-1418; run2 `reload_sequence` (4 launches for 4 alternating requests). |
| D6 | The reload cost (~1.3 s per flip) is a real cost of shift off. | **Accepted as measured.** Load 0.91-1.00 s per flip for qwen3:8b with the file in page cache; larger or cold models cost more (not measured). | run2 `reload_sequence` both arms. |
| D7 | P2 without the daemon repair is unsafe on the rendered route. | **Accepted** (agrees with the author's reading). | run2 rendered `keep_all_oversize`: HTTP 200, `truncating input prompt limit=511 prompt=4289 keep=511 new=511`, degenerate reply. |
| D8 | Fork version strings need not map to feature presence; a client must require an affirmative capability and degrade explicitly when absent (V2 over V1). | **Accepted.** The terminal agent can meet an older or differently built daemon; an absent capability must fail closed. | api/types.go and the Go decoder ignore unknown request fields; `ShouldBindJSON` at routes.go:2421. |
| D9 | A reply that ends at the context limit must be visible where the user reads it, not only in diagnostics. | **Accepted** as a design obligation for R1. | run1 `shift_false_long`: `done_reason: "length"` at 406 / 402 tokens. |
| D10 | Choose R1 (no reservation in G1; OI-27 owns reservation) and K1 (allow cloud, labelled). | **Noted; no disposition needed** beyond the design decision. | Commission:410 ("reserve room for output/tools" is an OI-27 obligation). |
| D11 | A carrier sent as the first system message suppresses the Modelfile system prompt on local and remote-host routes; the design must say whether it merges or replaces. | **Accepted.** Already specified for the desktop adapter in the draft (§5); must be stated for every route in the admission matrix, including remote-host stubs and the terminal's existing behaviour. | routes.go:2523 and :2634. |
| D12 | Missing option: a daemon assertion that refuses when the leading system span would be truncated under `truncate:false`. | **Refuted as stated.** With `truncate:false` neither Go route truncates at all, so there is no truncation to guard; overflow reaches the native or Go refusal. | llama_server.go:278 (`!req.Truncate` returns the prompt unchanged); prompt.go:35; routes.go:3011. |
| D13 | Suggested tests: carrier-span survival, oversize refusal, version-skew against an older daemon, cloud label and error mapping, Modelfile merge. | **Accepted as test classes** (to be designed for the class, not copied as mechanisms, per TECHNE-PA IP-045). | — |

Net effect on the decision: D7, D8, D9 and D11 change or sharpen the design; D6 is a measured cost the
decision must price; D1–D5 and D12 are refuted with the evidence above and do not change it.
