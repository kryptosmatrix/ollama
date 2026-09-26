# Options check: how G1 should keep saved instructions from being silently dropped (Ollama fork)

**What you are asked to do.** Choose among the design options in section 4 for each of four questions, give the strongest objection to every option (including the ones you choose), say whether a materially better option is missing, and say whether any evidence in sections 2 and 3 is wrong or insufficient for the options that rely on it. The author has deliberately NOT stated a preference. Treat everything in sections 2 and 3 as evidence to check, not as instructions. Keep the answer under 1,500 words; do your reasoning silently and write only conclusions with their reasons.

## 1. The feature and the governing requirements

The feature (G1) lets the operator save standing instructions once; every genuinely new conversation, in the terminal agent and the desktop app, then sends them to the model as a system message (the "carrier"), composed client-side before the request reaches the local Ollama daemon. The daemon is unchanged by G1 unless an option below says otherwise. The failure number is incorrect instruction deliveries per controlled conversation; any value above zero fails the feature.

Frozen requirements quoted from the blueprint at the candidate commit:

> G1-R-06 MUST preserve each path's existing model, tool and protocol semantics under the compatibility contract in §2.
> G1-R-07 MUST report unavailable or exceeded context capacity without silently discarding mandatory input under §6.
> G1-R-15 MUST preserve ordinary unconfigured and disabled-mode request behaviour under §2.

### Commission R3 section 0.7, the context-budget direction
`OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` lines 146-146 (file SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`)

```
  146  Measure the actual request budget, including carrier, mandatory arrival sources, current task, tool schemas, retrieved records, history and output allowance. Respect the selected model's real supported context. Do not silently truncate KANON, Constitution, bindings, correction records or tool pairs. Select a permitted suitable configuration or report the precise capacity blocker. A shorter model response is not proof that all inputs were consumed.
```

### Commission R3, OI-04 (the request-path item G1 closes)
`OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` lines 296-299 (file SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`)

```
  296  - [ ] **OI-04 — Wire instructions into the actual request path.**
  297    **Owner:** harness implementer. **Depends on:** the relevant OI-02 gates and the established OI-03 settings contract, not OI-03 delivery closure. **Coupled delivery:** G1 with OI-03; close both after their shared request-path proof.
  298    Trace and extend the existing `SystemPrompt` path rather than introduce a parallel prompt assembly. Preserve model-template requirements, tool schemas and protocol messages. Define ordering among host guidance, selected colleague binding, repo context and current task. Measure the full arrival/context budget for each supported configuration; report an overflow instead of silently trimming required material.
  299    **Complete when:** request-boundary capture proves the exact effective carrier reaches the selected model through the real terminal entry point, including after restoration/compaction where applicable. An editor screenshot is insufficient.
```

### Commission R3, OI-27 implementation obligations (a later item)
`OLLAMA_COLLEAGUE_INTEGRATION_WORK_LIST.md` lines 410-410 (file SHA-256 `74c6bb39fbcd62ebe6aa1c9807349a0faad9472b4b485999244872ca182a71ec`)

```
  410    **Implementation obligations:** filter by permission/scope before ranking; retrieve, deduplicate and rank bounded candidates using the actual supported interfaces; preserve corrections and evidence provenance. Summary entries point to exact recoverable source messages. Do not replace raw authoritative facts with an unsupported synthesis or regenerate an ever-growing summary of summaries. Permit exact source expansion when needed and declare insufficient context instead of guessing. Cache/index incrementally with invalidation for edits, corrections and access changes; do not scan or resummarise the entire archive for every request. Preserve the complete archive and export independently. Bound every stage and reserve room for output/tools; when mandatory content cannot fit, expose the precise capacity issue. Integrate startup recovery and compaction coherently, keep a tested rollback/compatibility path, and make selective context the qualified normal path for the new colleague/agent mode—not an unused toggle. A failure must not silently send an unbounded or unauthorised archive as fallback.
```

## 2. Measurements (executed 2026-09-26 on the operator's Mac, Apple Silicon, 128 GB)

Setup: the candidate daemon built from commit `04009c432e313eef65172293e75feb038ebe476b` (unchanged), the pinned native llama-server b10091 (`b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45`, server files byte-identical to the pinned blobs), model qwen3:8b (Q4_K_M), `num_ctx` 512, `think` false, non-streaming `/api/chat`. Every probe used a fresh daemon and runner in an isolated sandbox. The "native" arm sends messages to llama-server's chat endpoint (`OLLAMA_GO_TEMPLATE=false`); the "rendered" arm renders the prompt with the Go template and calls llama-server's completion endpoint (the default for this model). The carrier is a system message; `carrier_span` gives its token positions in the prompt, counted by the runner's own `/tokenize` with the flags the native inference path uses. `n_keep`/`n_discard` lines are the native server's own log lines.

#### run1 / native / shift_default_long (result file SHA-256 `72d79167c084e1310623ae1cff2bf83b702f708553c20d48c4321db6b09fab64`)
```json
{
 "arm": "native",
 "probe": "shift_default_long",
 "http_status": 200,
 "done_reason": "stop",
 "prompt_eval_count": 106,
 "eval_count": 1102,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 106
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400
 },
 "error": null,
 "native_or_go_log_lines": [
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253"
 ],
 "native_or_go_log_line_count": 3,
 "llama_server_launch_tail": [
  "0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512 --context-shift --keep 4\""
 ],
 "reply_tail_last_160_chars": "s reflect the ongoing progression of bridge engineering, driven by technological innovation and the need for resilient infrastructure in an ever-changing world."
}
```

#### run1 / native / shift_false_long (result file SHA-256 `dceef636d6fd594430bb4dc3711d6f68b63d4ca476508e42e12004a9e9b225d7`)
```json
{
 "arm": "native",
 "probe": "shift_false_long",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 106,
 "eval_count": 406,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 106
 },
 "request_extra": {
  "shift": false
 },
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400
 },
 "error": null,
 "native_or_go_log_lines": [],
 "native_or_go_log_line_count": 0,
 "llama_server_launch_tail": [
  "-port 54452 --host 127.0.0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512\""
 ],
 "reply_tail_last_160_chars": "The use of mortar and precise stone-cutting techniques enabled the creation of solid, long-lasting structures that could withstand the test of time. This period"
}
```

#### run1 / native / oversize_default (result file SHA-256 `2dd9e89476ddda05a321274c388cabae5b6491b5839f27c30984d262982a4ab6`)
```json
{
 "arm": "native",
 "probe": "oversize_default",
 "http_status": 400,
 "done_reason": null,
 "prompt_eval_count": null,
 "eval_count": null,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4285
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64
 },
 "error": "{\"error\":{\"code\":400,\"message\":\"request (4285 tokens) exceeds the available context size (512 tokens), try increasing it\",\"type\":\"exceed_context_size_error\",\"n_prompt_tokens\":4285,\"n_ctx\":512}}",
 "native_or_go_log_lines": [
  "srv    send_error: task id = 0, error: request (4285 tokens) exceeds the available context size (512 tokens), try increasing it"
 ],
 "native_or_go_log_line_count": 1,
 "llama_server_launch_tail": [
  "0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512 --context-shift --keep 4\""
 ],
 "reply_tail_last_160_chars": ""
}
```

#### run1 / native / oversize_shift_false (result file SHA-256 `5d0e42b2d7c13e1beb8a9b2146b80eb3656d0104ae7ce4c0e18e056c2cf37d90`)
```json
{
 "arm": "native",
 "probe": "oversize_shift_false",
 "http_status": 400,
 "done_reason": null,
 "prompt_eval_count": null,
 "eval_count": null,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4285
 },
 "request_extra": {
  "shift": false
 },
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64
 },
 "error": "{\"error\":{\"code\":400,\"message\":\"request (4285 tokens) exceeds the available context size (512 tokens), try increasing it\",\"type\":\"exceed_context_size_error\",\"n_prompt_tokens\":4285,\"n_ctx\":512}}",
 "native_or_go_log_lines": [
  "srv    send_error: task id = 0, error: request (4285 tokens) exceeds the available context size (512 tokens), try increasing it"
 ],
 "native_or_go_log_line_count": 1,
 "llama_server_launch_tail": [
  "-port 54527 --host 127.0.0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512\""
 ],
 "reply_tail_last_160_chars": ""
}
```

#### run2 / native / keep_all_long (result file SHA-256 `a5860137cae8a48f22bf85aaf835e7bca8aeff8a0288d8dc775ef3bd67ae4a0c`)
```json
{
 "arm": "native",
 "probe": "keep_all_long",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 106,
 "eval_count": 1400,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 106
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400,
  "num_keep": -1
 },
 "error": null,
 "native_or_go_log_lines": [
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 106, n_left = 405, n_discard = 202",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 106, n_left = 405, n_discard = 202",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 106, n_left = 405, n_discard = 202",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 106, n_left = 405, n_discard = 202",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 106, n_left = 405, n_discard = 202"
 ],
 "native_or_go_log_line_count": 5,
 "llama_server_launch_tail": [
  "st 127.0.0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512 --context-shift\""
 ],
 "reply_tail_last_160_chars": " 19th century, allowing for the construction of longer spans than previously possible. The use of strong cables and anchorages made it possible to build bridges"
}
```

#### run2 / native / keep_all_oversize (result file SHA-256 `613a782fc3e3038bd35f48d92731c8747fc6cb79fda16b0d0b324896b4a96227`)
```json
{
 "arm": "native",
 "probe": "keep_all_oversize",
 "http_status": 400,
 "done_reason": null,
 "prompt_eval_count": null,
 "eval_count": null,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4285
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64,
  "num_keep": -1
 },
 "error": "{\"error\":{\"code\":400,\"message\":\"request (4285 tokens) exceeds the available context size (512 tokens), try increasing it\",\"type\":\"exceed_context_size_error\",\"n_prompt_tokens\":4285,\"n_ctx\":512}}",
 "native_or_go_log_lines": [
  "srv    send_error: task id = 0, error: request (4285 tokens) exceeds the available context size (512 tokens), try increasing it"
 ],
 "native_or_go_log_line_count": 1,
 "llama_server_launch_tail": [
  "st 127.0.0.1 --no-webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --flash-attn auto -b 512 -ub 512 --context-shift\""
 ],
 "reply_tail_last_160_chars": ""
}
```

#### run2 / native / reload_sequence (result file SHA-256 `32fdc146081e0c616ac42e8e50516fa7ac7360de263ab466fa8dc2316f73b089`)
```json
[
  {
    "step": 0,
    "shift": "default",
    "http_status": 200,
    "seconds": 1.29,
    "load_duration_ns": 906223958,
    "launches_so_far": 1
  },
  {
    "step": 1,
    "shift": false,
    "http_status": 200,
    "seconds": 1.38,
    "load_duration_ns": 990042791,
    "launches_so_far": 2
  },
  {
    "step": 2,
    "shift": "default",
    "http_status": 200,
    "seconds": 1.36,
    "load_duration_ns": 975777125,
    "launches_so_far": 3
  },
  {
    "step": 3,
    "shift": false,
    "http_status": 200,
    "seconds": 1.34,
    "load_duration_ns": 963317417,
    "launches_so_far": 4
  }
]

```

#### run1 / rendered / shift_default_long (result file SHA-256 `a09494316ce1fe70edaf122d562f17a79b4cc0ddf7edb2bab584ab727e8d89f9`)
```json
{
 "arm": "rendered",
 "probe": "shift_default_long",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 110,
 "eval_count": 1400,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 110
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400
 },
 "error": null,
 "native_or_go_log_lines": [
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 4, n_left = 507, n_discard = 253"
 ],
 "native_or_go_log_line_count": 4,
 "llama_server_launch_tail": [
  "p 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512 --context-shift --keep 4\""
 ],
 "reply_tail_last_160_chars": "ge engineering is comprehensive and well-structured. Here's a refined version with improved flow, clarity, and a more engaging tone, while retaining all the key"
}
```

#### run1 / rendered / shift_false_long (result file SHA-256 `03acc2f3280cc6f748beebb60e30893140c3f61a65be66807be685178abe5910`)
```json
{
 "arm": "rendered",
 "probe": "shift_false_long",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 110,
 "eval_count": 402,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 110
 },
 "request_extra": {
  "shift": false
 },
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400
 },
 "error": null,
 "native_or_go_log_lines": [],
 "native_or_go_log_line_count": 0,
 "llama_server_launch_tail": [
  "webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512\""
 ],
 "reply_tail_last_160_chars": "ods in the United States, is believed to have been formed by the collapse of a lava tube, creating a natural arch. Early humans likely recognized the utility of"
}
```

#### run1 / rendered / oversize_default (result file SHA-256 `ea3db0616b556d7def3c564938bb666c113fcd0ebc6faa94ce0384608b157d76`)
```json
{
 "arm": "rendered",
 "probe": "oversize_default",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 258,
 "eval_count": 64,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4289
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64
 },
 "error": null,
 "native_or_go_log_lines": [
  "time=2026-09-26T19:55:29.355+10:00 level=DEBUG source=prompt.go:77 msg=\"truncating input messages which exceed context length\" truncated=1",
  "time=2026-09-26T19:55:29.358+10:00 level=WARN source=llama_server.go:314 msg=\"truncating input prompt\" limit=258 prompt=4289 keep=4 new=258"
 ],
 "native_or_go_log_line_count": 2,
 "llama_server_launch_tail": [
  "p 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512 --context-shift --keep 4\""
 ],
 "reply_tail_last_160_chars": "er-0660\", \"filler-0661\", etc.) and then ended with `/no_think`. If you're looking for help with something specific, such as generating text, editing content, or"
}
```

#### run1 / rendered / oversize_shift_false (result file SHA-256 `90b5b517e4061ef5cef32dcd2732de82bcd8103a3d27ef301c1fbd8e05324276`)
```json
{
 "arm": "rendered",
 "probe": "oversize_shift_false",
 "http_status": 400,
 "done_reason": null,
 "prompt_eval_count": null,
 "eval_count": null,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4289
 },
 "request_extra": {
  "shift": false
 },
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64
 },
 "error": "the prompt is longer than the context length currently available to the model; shorten the prompt, adjust the context length in settings, or use a model with a longer context length",
 "native_or_go_log_lines": [
  "time=2026-09-26T19:55:35.181+10:00 level=DEBUG source=prompt.go:77 msg=\"truncating input messages which exceed context length\" truncated=1"
 ],
 "native_or_go_log_line_count": 1,
 "llama_server_launch_tail": [
  "webui --offline -c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512\""
 ],
 "reply_tail_last_160_chars": ""
}
```

#### run2 / rendered / keep_all_long (result file SHA-256 `4693c3e00315a206acf87747ae94183ab88905bc5aa67f6da21dac19978f1374`)
```json
{
 "arm": "rendered",
 "probe": "keep_all_long",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 110,
 "eval_count": 1400,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 110
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 1400,
  "num_keep": -1
 },
 "error": null,
 "native_or_go_log_lines": [
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 110, n_left = 401, n_discard = 200",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 110, n_left = 401, n_discard = 200",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 110, n_left = 401, n_discard = 200",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 110, n_left = 401, n_discard = 200",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 110, n_left = 401, n_discard = 200"
 ],
 "native_or_go_log_line_count": 5,
 "llama_server_launch_tail": [
  "-c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512 --context-shift\""
 ],
 "reply_tail_last_160_chars": "oped the arch as a fundamental structural element, which allowed for the creation of durable and long-lasting bridges that could span wide rivers and valleys.\n\n"
}
```

#### run2 / rendered / keep_all_oversize (result file SHA-256 `99e930aa2e8d12a0870894eef029cdb11798bd8ee051c53e50e9836df75cb464`)
```json
{
 "arm": "rendered",
 "probe": "keep_all_oversize",
 "http_status": 200,
 "done_reason": "length",
 "prompt_eval_count": 511,
 "eval_count": 64,
 "runner_props": {
  "n_ctx": 512,
  "total_slots": 1
 },
 "carrier_span": {
  "carrier_first_token_index": 16,
  "carrier_end_token_index": 67,
  "prompt_tokens_runner_count": 4289
 },
 "request_extra": {},
 "request_options": {
  "num_ctx": 512,
  "num_predict": 64,
  "num_keep": -1
 },
 "error": null,
 "native_or_go_log_lines": [
  "time=2026-09-26T19:57:33.944+10:00 level=DEBUG source=prompt.go:77 msg=\"truncating input messages which exceed context length\" truncated=1",
  "time=2026-09-26T19:57:33.947+10:00 level=WARN source=llama_server.go:314 msg=\"truncating input prompt\" limit=511 prompt=4289 keep=511 new=511",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 508, n_left = 3, n_discard = 1",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 508, n_left = 3, n_discard = 1",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 508, n_left = 3, n_discard = 1",
  "slot   operator(): id  0 | task 0 | slot context shift, n_keep = 508, n_left = 3, n_discard = 1"
 ],
 "native_or_go_log_line_count": 65,
 "llama_server_launch_tail": [
  "-c 512 -np 1 --log-verbosity 4 --no-log-prefix --no-log-timestamps --no-jinja --chat-template chatml --flash-attn auto -b 512 -ub 512 --context-shift\""
 ],
 "reply_tail_last_160_chars": "072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler-0072 filler"
}
```

#### run2 / rendered / reload_sequence (result file SHA-256 `c0e276936a2dbe793cfbe9941ed76bb553e908d526bac9c602e2f524881cb3f5`)
```json
[
  {
    "step": 0,
    "shift": "default",
    "http_status": 200,
    "seconds": 1.34,
    "load_duration_ns": 964406000,
    "launches_so_far": 1
  },
  {
    "step": 1,
    "shift": false,
    "http_status": 200,
    "seconds": 1.37,
    "load_duration_ns": 996547542,
    "launches_so_far": 2
  },
  {
    "step": 2,
    "shift": "default",
    "http_status": 200,
    "seconds": 1.35,
    "load_duration_ns": 971823542,
    "launches_so_far": 3
  },
  {
    "step": 3,
    "shift": false,
    "http_status": 200,
    "seconds": 1.36,
    "load_duration_ns": 986069166,
    "launches_so_far": 4
  }
]

```

## 3. Source excerpts

### Rendered-route prompt truncation in the Go runner
`llm/llama_server.go` lines 276-328 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `eb6b90eab5122a509a841b54c57fb08ce91a000591d6abd18bd51cda625db46e`)

```
  276  func (s *llamaServerRunner) completionPromptForRequest(ctx context.Context, req CompletionRequest) (any, error) {
  277  	prompt := s.completionPrompt(req.Prompt, req.LeadingBOS)
  278  	if !req.Truncate || len(req.Media) > 0 || s.options.NumCtx <= 1 || len(prompt) < s.options.NumCtx {
  279  		return prompt, nil
  280  	}
  281  
  282  	tokens, err := s.tokenize(ctx, prompt, true, nil)
  283  	if err != nil {
  284  		return nil, err
  285  	}
  286  
  287  	fullPromptLimit := s.options.NumCtx - 1
  288  	if len(tokens) <= fullPromptLimit {
  289  		return prompt, nil
  290  	}
  291  
  292  	if !s.launch.config.ContextShift {
  293  		return nil, api.StatusError{
  294  			StatusCode:   http.StatusBadRequest,
  295  			ErrorMessage: "the prompt is longer than the context length currently available to the model; shorten the prompt, adjust the context length in settings, or use a model with a longer context length",
  296  		}
  297  	}
  298  
  299  	nKeep := req.Options.NumKeep
  300  	if nKeep < 0 {
  301  		nKeep = len(tokens)
  302  	}
  303  	if s.tokenizerAddsBOS() {
  304  		nKeep++
  305  	}
  306  	nKeep = min(nKeep, fullPromptLimit)
  307  
  308  	limit := contextShiftPromptLimit(s.options.NumCtx, nKeep)
  309  	discard := len(tokens) - limit
  310  	truncated := make([]int, 0, limit)
  311  	truncated = append(truncated, tokens[:nKeep]...)
  312  	truncated = append(truncated, tokens[nKeep+discard:]...)
  313  
  314  	slog.Warn("truncating input prompt", "limit", limit, "prompt", len(tokens), "keep", nKeep, "new", len(truncated))
  315  	return truncated, nil
  316  }
  317  
  318  func contextShiftPromptLimit(numCtx, numKeep int) int {
  319  	if numCtx <= 1 {
  320  		return 0
  321  	}
  322  
  323  	numKeep = max(0, min(numKeep, numCtx-1))
  324  
  325  	// Match the old runners' first context shift: preserve num_keep, then free
  326  	// roughly half of the remaining context before generation needs the slot.
  327  	return numCtx - max((numCtx-numKeep)/2, 1)
  328  }
```

### Context shift is a llama-server launch argument
`llm/llama_server.go` lines 781-792 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `eb6b90eab5122a509a841b54c57fb08ce91a000591d6abd18bd51cda625db46e`)

```
  781  func appendContextShiftArgs(params []string, opts api.Options, enabled bool) []string {
  782  	if !enabled {
  783  		return params
  784  	}
  785  
  786  	params = append(params, "--context-shift")
  787  	if opts.NumKeep > 0 {
  788  		params = append(params, "--keep", strconv.Itoa(opts.NumKeep))
  789  	}
  790  
  791  	return params
  792  }
```

### Per-request fields Ollama sends to llama-server chat
`llm/llama_server.go` lines 2106-2118 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `eb6b90eab5122a509a841b54c57fb08ce91a000591d6abd18bd51cda625db46e`)

```
 2106  		}
 2107  		messages = append(messages, converted)
 2108  	}
 2109  
 2110  	body := map[string]any{
 2111  		"messages":          messages,
 2112  		"stream":            stream,
 2113  		"cache_prompt":      true,
 2114  		"n_predict":         req.Options.NumPredict,
 2115  		"n_keep":            req.Options.NumKeep,
 2116  		"temperature":       req.Options.Temperature,
 2117  		"top_k":             req.Options.TopK,
 2118  		"top_p":             req.Options.TopP,
```

### Default request options
`api/types.go` lines 1096-1104 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `2155658af3cdb27fc0f44d772ce94a7b52f8da1b51d7b26aa0d158d08ee5791d`)

```
 1096  func DefaultOptions() Options {
 1097  	return Options{
 1098  		// options set on request to runner
 1099  		NumPredict: -1,
 1100  
 1101  		// set a minimal num_keep to avoid issues on context shifts
 1102  		NumKeep:          4,
 1103  		Temperature:      0.8,
 1104  		TopK:             40,
```

### A shift mismatch forces a runner reload
`server/sched.go` lines 1409-1419 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `e4058db54c61b1af1a0c1799836731adbbf855d59c78757cce81d9d7dca3a07b`)

```
 1409  		optsExisting.NumGPU = -1
 1410  		optsNew.NumGPU = -1
 1411  	}
 1412  
 1413  	contextShift := req.contextShift
 1414  	if req.model.ModelPath != "" {
 1415  		contextShift = resolveContextShift(req.shift, req.model)
 1416  	}
 1417  	if runner.contextShift != contextShift {
 1418  		return true
 1419  	}
```

### Go history truncation on the rendered route keeps system messages
`server/prompt.go` lines 20-74 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `79411c4e15ff27fb8bac4dcd96076407e035ec407c0ab1a1840035f9668b4654`)

```
   20  // chatPrompt accepts a list of messages and returns the prompt and media that should be used for the next chat turn.
   21  // chatPrompt truncates any messages that exceed the context window of the model, making sure to always include 1) the
   22  // latest message and 2) system messages
   23  func chatPrompt(ctx context.Context, m *Model, tokenize tokenizeFunc, opts *api.Options, msgs []api.Message, tools []api.Tool, think *api.ThinkValue, truncate bool) (prompt string, media []llm.MediaData, _ error) {
   24  	var system []api.Message
   25  
   26  	// TODO: This is only a truncation heuristic; llama-server handles the
   27  	// actual image/media inputs. Replace this with projector/model-aware media
   28  	// token accounting so image history is neither over-packed nor over-trimmed.
   29  	// Clip images are represented as 768 tokens, each an embedding.
   30  	imageNumTokens := 768
   31  
   32  	lastMsgIdx := len(msgs) - 1
   33  	currMsgIdx := 0
   34  
   35  	if truncate {
   36  		// Start with all messages and remove from the front until it fits in context
   37  		for i := 0; i <= lastMsgIdx; i++ {
   38  			// Collect system messages from the portion we're about to skip
   39  			system = make([]api.Message, 0)
   40  			for j := range i {
   41  				if msgs[j].Role == "system" {
   42  					system = append(system, msgs[j])
   43  				}
   44  			}
   45  
   46  			p, err := renderPrompt(m, append(system, msgs[i:]...), tools, think)
   47  			if err != nil {
   48  				return "", nil, err
   49  			}
   50  
   51  			s, err := tokenize(ctx, p)
   52  			if err != nil {
   53  				return "", nil, err
   54  			}
   55  
   56  			ctxLen := len(s)
   57  			if m.ProjectorPaths != nil {
   58  				for _, msg := range msgs[i:] {
   59  					ctxLen += imageNumTokens * len(msg.Images)
   60  				}
   61  			}
   62  
   63  			if ctxLen <= opts.NumCtx {
   64  				currMsgIdx = i
   65  				break
   66  			}
   67  
   68  			// Must always include at least the last message
   69  			if i == lastMsgIdx {
   70  				currMsgIdx = lastMsgIdx
   71  				break
   72  			}
   73  		}
   74  	}
```

### Go history truncation on the native route keeps system messages
`server/routes.go` lines 3010-3067 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `47b8e89a73ef76d118c89c101dbd9f74b0a63777ba3cbac153b46002b5397ec2`)

```
 3010  func truncateNativeChatMessages(ctx context.Context, m *Model, r llm.LlamaServer, opts *api.Options, req llm.ChatRequest, truncate bool) ([]api.Message, error) {
 3011  	if !truncate || opts == nil || opts.NumCtx <= 0 || len(req.Messages) <= 1 {
 3012  		return req.Messages, nil
 3013  	}
 3014  
 3015  	lastMsgIdx := len(req.Messages) - 1
 3016  	currMsgIdx := 0
 3017  	var system []api.Message
 3018  
 3019  	for i := 0; i <= lastMsgIdx; i++ {
 3020  		system = system[:0]
 3021  		for j := range i {
 3022  			if req.Messages[j].Role == "system" {
 3023  				system = append(system, req.Messages[j])
 3024  			}
 3025  		}
 3026  
 3027  		renderReq := req
 3028  		renderReq.Messages = append(slices.Clone(system), req.Messages[i:]...)
 3029  		prompt, err := r.ApplyChatTemplate(ctx, renderReq)
 3030  		if err != nil {
 3031  			return nil, err
 3032  		}
 3033  
 3034  		tokens, err := r.Tokenize(ctx, prompt)
 3035  		if err != nil {
 3036  			return nil, err
 3037  		}
 3038  
 3039  		ctxLen := len(tokens)
 3040  		if m != nil && m.ProjectorPaths != nil {
 3041  			for _, msg := range renderReq.Messages {
 3042  				ctxLen += 768 * len(msg.Images)
 3043  			}
 3044  		}
 3045  
 3046  		if ctxLen <= opts.NumCtx {
 3047  			currMsgIdx = i
 3048  			break
 3049  		}
 3050  		if i == lastMsgIdx {
 3051  			currMsgIdx = lastMsgIdx
 3052  			break
 3053  		}
 3054  	}
 3055  
 3056  	if currMsgIdx > 0 {
 3057  		slog.Debug("truncating native chat messages which exceed context length", "truncated", currMsgIdx)
 3058  	}
 3059  
 3060  	system = system[:0]
 3061  	for j := range currMsgIdx {
 3062  		if req.Messages[j].Role == "system" {
 3063  			system = append(system, req.Messages[j])
 3064  		}
 3065  	}
 3066  	return append(slices.Clone(system), req.Messages[currMsgIdx:]...), nil
 3067  }
```

### Explicit cloud models are proxied without local checks
`server/routes.go` lines 2440-2448 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `47b8e89a73ef76d118c89c101dbd9f74b0a63777ba3cbac153b46002b5397ec2`)

```
 2440  	if modelRef.Source == modelSourceCloud {
 2441  		req.Model = modelRef.Base
 2442  		if c.GetBool(legacyCloudAnthropicKey) {
 2443  			proxyCloudJSONRequestWithPath(c, req, "/api/chat", cloudErrRemoteInferenceUnavailable)
 2444  			return
 2445  		}
 2446  		proxyCloudJSONRequest(c, req, cloudErrRemoteInferenceUnavailable)
 2447  		return
 2448  	}
```

### Remote-host stub models: Modelfile system prompt added only if the first message is not system
`server/routes.go` lines 2515-2535 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `47b8e89a73ef76d118c89c101dbd9f74b0a63777ba3cbac153b46002b5397ec2`)

```
 2515  		req.Model = m.Config.RemoteModel
 2516  		if req.Options == nil {
 2517  			req.Options = map[string]any{}
 2518  		}
 2519  
 2520  		var msgs []api.Message
 2521  		if len(req.Messages) > 0 {
 2522  			msgs = append(m.Messages, req.Messages...)
 2523  			if req.Messages[0].Role != "system" && m.System != "" {
 2524  				msgs = append([]api.Message{{Role: "system", Content: m.System}}, msgs...)
 2525  			}
 2526  		}
 2527  
 2528  		msgs = filterThinkTags(msgs, m)
 2529  		req.Messages = msgs
 2530  
 2531  		for k, v := range m.Options {
 2532  			if _, ok := req.Options[k]; !ok {
 2533  				req.Options[k] = v
 2534  			}
 2535  		}
```

### Local routes: Modelfile system prompt added only if the first message is not system
`server/routes.go` lines 2633-2637 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `47b8e89a73ef76d118c89c101dbd9f74b0a63777ba3cbac153b46002b5397ec2`)

```
 2633  	msgs := append(m.Messages, req.Messages...)
 2634  	if req.Messages[0].Role != "system" && m.System != "" {
 2635  		msgs = append([]api.Message{{Role: "system", Content: m.System}}, msgs...)
 2636  	}
 2637  	msgs = filterThinkTags(msgs, m)
```

### MLX runner admission and generation cap
`x/mlxrunner/pipeline.go` lines 44-58 at commit `04009c432e313eef65172293e75feb038ebe476b` (file SHA-256 `021d6ad6f945dfce17c8ad686fd78f6d1fd82d822d8e81f445dce086e507f7aa`)

```
   44  		return errors.New("empty prompt")
   45  	}
   46  
   47  	if len(tokens) >= r.contextLength {
   48  		return fmt.Errorf("input length (%d tokens) exceeds the model's maximum context length (%d tokens)", len(tokens), r.contextLength)
   49  	}
   50  
   51  	// Cap generation to stay within the model's context length
   52  	maxGenerate := r.contextLength - len(tokens)
   53  	if request.Options.NumPredict <= 0 {
   54  		request.Options.NumPredict = maxGenerate
   55  	} else {
   56  		request.Options.NumPredict = min(request.Options.NumPredict, maxGenerate)
   57  	}
   58  
```

### Native: with context shift disabled, generation stops at the context limit
`tools/server/server-context.cpp` lines 1856-1864 at commit `b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45` (file SHA-256 `69ad37830dfc6d8ae25dbba6f2c7128b4df209bafb6b47ace2fb6aa0c8fdbe48`)

```
 1856          }
 1857  
 1858          // if context shifting is disabled, make sure that we don't run out of context
 1859          if (!params_base.ctx_shift && slot.prompt.n_tokens() + 1 >= slot.n_ctx) {
 1860              slot.truncated      = true;
 1861              slot.stop           = STOP_TYPE_LIMIT;
 1862              slot.has_next_token = false;
 1863  
 1864              SLT_DBG(slot, "stopped due to running out of context capacity, prompt.n_tokens() = %d, task.n_tokens = %d, n_decoded = %d, n_ctx = %d\n",
```

### Native: context shift keeps n_keep tokens and discards half of the rest
`tools/server/server-context.cpp` lines 2866-2886 at commit `b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45` (file SHA-256 `69ad37830dfc6d8ae25dbba6f2c7128b4df209bafb6b47ace2fb6aa0c8fdbe48`)

```
 2866  
 2867                  // Shift context
 2868                  int n_keep = slot.task->params.n_keep < 0 ? slot.task->n_tokens() : slot.task->params.n_keep;
 2869  
 2870                  if (add_bos_token) {
 2871                      n_keep += 1;
 2872                  }
 2873  
 2874                  n_keep = std::min(slot.n_ctx - 4, n_keep);
 2875  
 2876                  const int n_left    = slot.prompt.n_tokens() - n_keep;
 2877                  int       n_discard = slot.task->params.n_discard ? slot.task->params.n_discard : (n_left / 2);
 2878  
 2879                  // ref: https://github.com/ggml-org/llama.cpp/pull/24786
 2880                  n_discard = std::clamp(n_discard, 0, std::max(0, n_left - 1));
 2881  
 2882                  SLT_WRN(slot, "slot context shift, n_keep = %d, n_left = %d, n_discard = %d\n", n_keep, n_left, n_discard);
 2883  
 2884                  common_context_seq_rm (ctx_tgt, slot.id, n_keep            , n_keep + n_discard);
 2885                  common_context_seq_add(ctx_tgt, slot.id, n_keep + n_discard, slot.prompt.n_tokens(), -n_discard);
 2886  
```

### Native: a prompt at or above the slot context is rejected
`tools/server/server-context.cpp` lines 3134-3143 at commit `b4d6c7d8ff69c2e05e4e8ee7e6e710a08abd7b45` (file SHA-256 `69ad37830dfc6d8ae25dbba6f2c7128b4df209bafb6b47ace2fb6aa0c8fdbe48`)

```
 3134                                      string_format(
 3135                                          "input (%d tokens) is larger than the max context size (%d tokens). skipping",
 3136                                          slot.task->n_tokens(), slot.n_ctx),
 3137                                      ERROR_TYPE_EXCEED_CONTEXT_SIZE);
 3138                                  slot.release();
 3139                                  return;
 3140                              }
 3141                          } else {
 3142                              if (slot.task->n_tokens() >= slot.n_ctx) {
 3143                                  send_error(slot,
```

## 4. The options (authored text; no option is preferred)

**Q1 — How should an instruction-bearing request stop the carrier being silently discarded?**

- **P1, shift off.** The client sends `shift: false` on every request that carries an enabled carrier. Nothing in the daemon changes.
- **P2, keep the whole prompt, plus a daemon repair.** The client sends `options.num_keep: -1` on every carrier-bearing request, and the daemon's rendered-route prompt truncation is repaired so that a keep-whole-prompt request that does not fit is refused with an explicit error instead of being cut.
- **P3, the daemon protects leading system messages itself.** The daemon computes `n_keep` as the token count through the end of the leading system messages and never truncates or shifts inside that span, for every request.
- **P4, the client assembles the exact prompt.** The client sends `truncate: false` and `shift: false`, selects history itself using daemon token counts, and fails explicitly when mandatory content does not fit.

**Q2 — Room for the model's reply (the "output allowance").**

- **R1, no reservation in G1.** Replies may end at the context limit with `done_reason: "length"`, which the editor preview and conversation diagnostics surface; G1 measures and reports prompt tokens, the runner's context size and the remaining room for each request; reservation is left, explicitly and visibly, to the later item OI-27 whose obligations include reserving room for output.
- **R2, a daemon-side reservation.** A new request field asks the daemon to fit history into the context minus a reserve R, and to refuse before inference when the carrier and the newest message do not fit within that bound. Because Go's JSON decoding ignores unknown fields, an older daemon would silently ignore the field; the client must therefore check a capability the daemon advertises before relying on it.
- **R3, a client preflight.** The client estimates the full request with daemon token counts before dispatch and refuses or warns when the remaining room is below a declared minimum.

**Q3 — Clients newer than the daemon they talk to.** The desktop app ships and starts its own daemon; the terminal agent talks to whatever daemon is running, which may be older.

- **V1, existing fields only, plus a version gate.** G1 uses only request fields the daemon already honours; the terminal checks the daemon's reported version and refuses to send carrier-bearing requests to one that predates them.
- **V2, a capability handshake.** The daemon advertises named capabilities and clients refuse or degrade explicitly when one they need is absent.

**Q4 — Explicit cloud models and remote-host stubs**, where the daemon forwards the request and the provider enforces its own limits.

- **K1, allow, labelled.** Carrier-bearing cloud requests are sent; the preview and diagnostics label capacity "provider-managed, not verified locally"; recognisable provider overflow errors map to the feature's context error; the provider's actual oversize behaviour is measured later by a bounded, operator-authorised probe.
- **K2, refuse until qualified.** Carrier-bearing cloud requests are refused with an explicit capacity-unavailable error until the provider path is measured.
- **K3, client estimate.** The client estimates against the provider's advertised context length and refuses when the estimate does not fit.

## 5. What to return

For each of Q1 to Q4: your choice and why; the strongest objection to each option. Then: any materially better option missing; any fact above that looks wrong or does not support what relies on it (cite the section and line); and the tests that would expose a facade of your chosen combination. Under 1,500 words.
