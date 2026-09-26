#!/bin/bash
# Dispatch the frozen cloud-admission options packet, blind, to Codex (continuation 13, Thole).
# KANON 25 check before deciding: one non-Claude substrate reading the repository; no author preference stated.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
PACKET="$D/PACKET.md"
EXPECT_SHA="2ca31ab42bbbf42c09d0f0b3df6eb176bce120fb8b3d1a2143c79340e419c640"
GOT_SHA="$(shasum -a 256 "$PACKET" | cut -d' ' -f1)"
if [ "$GOT_SHA" != "$EXPECT_SHA" ]; then echo "STOP: packet hash $GOT_SHA != frozen $EXPECT_SHA"; exit 1; fi
mkdir -p "$D/reviews/codex"
CPROMPT="$D/reviews/codex/prompt.md"
{
  printf '%s\n' "You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at commit 4f27c8e3a3d9929f03ff7906b67b6df068f279a7. You may open source files under agent/, api/, app/, cmd/, internal/, llm/, server/ and x/ to verify the facts below. Do not open anything under docs/ or any file outside the repository: earlier design drafts are withheld on purpose so that your answer is independent. Change nothing and run nothing that writes. Where you verify or refute a fact at source, cite file:line."
  printf '\n---\n\n'
  cat "$PACKET"
} > "$CPROMPT"
shasum -a 256 "$CPROMPT" > "$D/reviews/codex/prompt.sha256"
cd /Users/krypto/GitHub/ollama-eko-chat-read-integrity || exit 1
/Users/krypto/.local/bin/codex exec -m gpt-6-astra -c 'service_tier="default"' \
  --disable hooks --disable memories --disable plugins --disable multi_agent \
  -c 'web_search="disabled"' \
  --sandbox read-only --ephemeral --json \
  -C /Users/krypto/GitHub/ollama-eko-chat-read-integrity \
  -o "$D/reviews/codex/last_message.md" \
  "$(cat "$CPROMPT")" < /dev/null > "$D/reviews/codex/events.jsonl" 2> "$D/reviews/codex/stderr.txt"
echo "codex-exit=$? $(date '+%Y-%m-%dT%H:%M:%S%z')" > "$D/reviews/codex/exit.txt"
