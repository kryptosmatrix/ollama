#!/bin/bash
# Dispatch the frozen OQ-4 classic-paths options packet (continuation 16, Abuttal) to Codex. KANON 25.3: the options,
# not the author's pick, go to a different substrate before the decisions are taken.
# usage: dispatch.sh C <PACKET_C.md sha256>
set -u
D="$(cd "$(dirname "$0")" && pwd)"
WHICH="${1:?usage: dispatch.sh C <packet sha256>}"
SHA="${2:?usage: dispatch.sh C <packet sha256>}"
case "$WHICH" in C) ;; *) echo "STOP: packet must be C"; exit 1 ;; esac
PACKET="$D/PACKET_$WHICH.md"
REPO=/Users/krypto/GitHub/ollama-eko-chat-read-integrity
got="$(shasum -a 256 "$PACKET" | cut -d' ' -f1)"
[ "$got" = "$SHA" ] || { echo "STOP: packet hash $got != frozen $SHA"; exit 1; }
out="$D/reviews/codex-$WHICH"
[ -e "$out" ] && { echo "STOP: $out exists"; exit 1; }
[ -z "$(git -C "$REPO" status --porcelain --untracked-files=no)" ] || { echo "STOP: tracked changes"; exit 1; }
committed="$(git -C "$REPO" show "HEAD:docs/analysis/colleague_integration/proof_cont16/oq4-options-check-classic/PACKET_$WHICH.md" | shasum -a 256 | cut -d' ' -f1)"
[ "$committed" = "$SHA" ] || { echo "STOP: committed packet hash $committed != frozen $SHA"; exit 1; }
mkdir -p "$out"
{ printf '%s\n' "You are an independent adviser working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/, and run read-only commands. Change nothing and run nothing that writes. Cite file:line for every claim you rest on."; printf '\n---\n\n'; cat "$PACKET"; } > "$out/prompt.md"
shasum -a 256 "$out/prompt.md" > "$out/prompt.sha256"
git -C "$REPO" rev-parse HEAD > "$out/head.txt"
date '+%Y-%m-%dT%H:%M:%S%z' > "$out/started.txt"
cd "$REPO" && /Users/krypto/.local/bin/codex exec -m gpt-6-astra -c 'service_tier="default"' \
  --disable hooks --disable memories --disable plugins --disable multi_agent -c 'web_search="disabled"' \
  --sandbox read-only --ephemeral --json -C "$REPO" -o "$out/last_message.md" \
  "$(cat "$out/prompt.md")" < /dev/null > "$out/events.jsonl" 2> "$out/stderr.txt"
echo "codex-exit=$? $(date '+%Y-%m-%dT%H:%M:%S%z')" > "$out/exit.txt"
