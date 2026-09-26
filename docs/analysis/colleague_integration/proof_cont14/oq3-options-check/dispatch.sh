#!/bin/bash
# Dispatch the frozen OQ-3 instrument options packet, blind, to Codex (continuation 14, Letterlock).
# KANON 25 check before deciding: one non-Claude substrate that can read the repository; options only, no author pick.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
PACKET="$D/PACKET.md"
EXPECT_SHA="${1:?usage: dispatch.sh <frozen packet sha256>}"
GOT_SHA="$(shasum -a 256 "$PACKET" | cut -d' ' -f1)"
if [ "$GOT_SHA" != "$EXPECT_SHA" ]; then echo "STOP: packet hash $GOT_SHA != frozen $EXPECT_SHA"; exit 1; fi
OUT="$D/reviews/codex"
if [ -e "$OUT" ]; then echo "STOP: $OUT exists; a review is never overwritten"; exit 1; fi
mkdir -p "$OUT"
CPROMPT="$OUT/prompt.md"
{
  printf '%s\n' "You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at commit a7c35fd65e732cd378dc6d2e93ccc43f152efcef. You may open source and test files anywhere under agent/, api/, app/, cmd/, envconfig/, internal/, llm/, server/ and x/, and the two design files docs/_design/G1_PERSISTED_INSTRUCTIONS.md and docs/_design/G1_PERSISTENT_INSTRUCTIONS_CONCEPT.md. Do not open anything else under docs/ (earlier evidence and reviews are withheld on purpose so that your answer is independent) or any file outside the repository. Change nothing and run nothing that writes. Where you verify or refute a fact at source, cite file:line."
  printf '\n---\n\n'
  cat "$PACKET"
} > "$CPROMPT"
shasum -a 256 "$CPROMPT" > "$OUT/prompt.sha256"
cd /Users/krypto/GitHub/ollama-eko-chat-read-integrity || exit 1
date '+%Y-%m-%dT%H:%M:%S%z' > "$OUT/started.txt"
/Users/krypto/.local/bin/codex exec -m gpt-6-astra -c 'service_tier="default"' \
  --disable hooks --disable memories --disable plugins --disable multi_agent \
  -c 'web_search="disabled"' \
  --sandbox read-only --ephemeral --json \
  -C /Users/krypto/GitHub/ollama-eko-chat-read-integrity \
  -o "$OUT/last_message.md" \
  "$(cat "$CPROMPT")" < /dev/null > "$OUT/events.jsonl" 2> "$OUT/stderr.txt"
echo "codex-exit=$? $(date '+%Y-%m-%dT%H:%M:%S%z')" > "$OUT/exit.txt"
