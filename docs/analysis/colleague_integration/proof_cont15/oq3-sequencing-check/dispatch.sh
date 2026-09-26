#!/bin/bash
# Dispatch the frozen sequencing options check (continuation 15, Treadle) to Codex. KANON 25.3: the
# options, not the author's pick, go to a different substrate before the order is decided.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
SHA="${1:?usage: dispatch.sh <PACKET.md sha256>}"
REPO=/Users/krypto/GitHub/ollama-eko-chat-read-integrity
got="$(shasum -a 256 "$D/PACKET.md" | cut -d' ' -f1)"
[ "$got" = "$SHA" ] || { echo "STOP: packet hash $got != frozen $SHA"; exit 1; }
out="$D/reviews/codex"
[ -e "$out" ] && { echo "STOP: $out exists"; exit 1; }
[ -z "$(git -C "$REPO" status --porcelain --untracked-files=no)" ] || { echo "STOP: tracked changes"; exit 1; }
mkdir -p "$out"
{ printf '%s\n' "You are an independent adviser working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/, and run read-only commands. Change nothing and run nothing that writes. Cite file:line for every claim you rest on."; printf '\n---\n\n'; cat "$D/PACKET.md"; } > "$out/prompt.md"
shasum -a 256 "$out/prompt.md" > "$out/prompt.sha256"
git -C "$REPO" rev-parse HEAD > "$out/head.txt"
date '+%Y-%m-%dT%H:%M:%S%z' > "$out/started.txt"
cd "$REPO" && /Users/krypto/.local/bin/codex exec -m gpt-6-astra -c 'service_tier="default"' \
  --disable hooks --disable memories --disable plugins --disable multi_agent -c 'web_search="disabled"' \
  --sandbox read-only --ephemeral --json -C "$REPO" -o "$out/last_message.md" \
  "$(cat "$out/prompt.md")" < /dev/null > "$out/events.jsonl" 2> "$out/stderr.txt"
echo "codex-exit=$? $(date '+%Y-%m-%dT%H:%M:%S%z')" > "$out/exit.txt"
