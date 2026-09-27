# Codex attempt 1 failed before any review; attempt 2 names the model explicitly

Attempt 1 (files in this directory: events.jsonl, stderr.txt, exit.txt) used the account's configured
default model `gpt-6-sol` from ~/.codex/config.toml. The service returned HTTP 400
"The 'gpt-6-sol' model is not supported when using Codex with a ChatGPT account." No review ran.

Attempt 2 (attempt2/) sends the identical prompt bytes (prompt.md, SHA-256 in prompt.sha256) with
`-m gpt-6-astra -c service_tier="default"`, the model a one-word probe accepted at 2026-09-26T20:04:05+1000.
Same provider family (OpenAI), same packet, same overrides (hooks, memories, plugins, multi_agent off;
web search disabled; read-only sandbox; ephemeral). Written before attempt 2 was launched.
