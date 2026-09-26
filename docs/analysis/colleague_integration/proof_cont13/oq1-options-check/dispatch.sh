#!/bin/bash
# Dispatch the frozen OQ-1 options packet, blind, to two non-Claude substrates (continuation 13, Thole).
# Codex (OpenAI) may read the repository to verify excerpts; DeepSeek receives the packet text only.
# Both get the identical packet bytes; neither sees the other's answer or any author preference.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
PACKET="$D/PACKET.md"
EXPECT_SHA="8631e33176d8e78fdf209fcb3defcc298d92f7c5a93634492698b6c2ee028a51"
GOT_SHA="$(shasum -a 256 "$PACKET" | cut -d' ' -f1)"
if [ "$GOT_SHA" != "$EXPECT_SHA" ]; then echo "STOP: packet hash $GOT_SHA != frozen $EXPECT_SHA"; exit 1; fi
mkdir -p "$D/reviews/codex" "$D/reviews/deepseek"

# ---- Codex: repository-reading, read-only sandbox, hooks/memories/plugins/multi-agent off, web search off ----
CPROMPT="$D/reviews/codex/prompt.md"
{
  printf '%s\n' "You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at commit 04009c432e313eef65172293e75feb038ebe476b. You may open source files under agent/, api/, app/, cmd/, llm/, server/ and x/ to verify the excerpts below, and the pinned native llama.cpp source under /Users/krypto/GitHub/ollama/build/darwin-sources/_deps/llama_cpp-src/tools/server/. Do not open anything under docs/ or any file outside those locations: earlier design drafts are withheld on purpose so that your answer is independent. Change nothing and run nothing that writes. Where you verify or refute an excerpt at source, cite file:line."
  printf '\n---\n\n'
  cat "$PACKET"
} > "$CPROMPT"
shasum -a 256 "$CPROMPT" > "$D/reviews/codex/prompt.sha256"
(
  cd /Users/krypto/GitHub/ollama-eko-chat-read-integrity || exit 1
  /Users/krypto/.local/bin/codex exec \
    --disable hooks --disable memories --disable plugins --disable multi_agent \
    -c 'web_search="disabled"' \
    --sandbox read-only --ephemeral --json \
    -C /Users/krypto/GitHub/ollama-eko-chat-read-integrity \
    -o "$D/reviews/codex/last_message.md" \
    "$(cat "$CPROMPT")" < /dev/null > "$D/reviews/codex/events.jsonl" 2> "$D/reviews/codex/stderr.txt"
  echo "codex-exit=$?" > "$D/reviews/codex/exit.txt"
) &
CODEX_PID=$!

# ---- DeepSeek via the estate's Ollama wrapper (text only), retrying only on the estate-wide lock ----
cd /Users/krypto/GitHub/TECHNE/Tools/ollama || exit 1
tries=0
while :; do
  tries=$((tries+1))
  ./ask.sh --model deepseek-v4.1-flash:cloud --prompt-file "$PACKET" --approve-local-content \
    --expect-digest e04da138d31e0c9468e982e1ae9503d06cb7e170caa16a90c17d931c4aa140f8 \
    --expect-served-model deepseek-v4.1-flash --think off --num-ctx 262144 --num-predict 16384 \
    --temperature 0 --seed 0 --num-gpu 0 > "$D/reviews/deepseek/stdout.txt" 2> "$D/reviews/deepseek/stderr.txt"
  rc=$?
  echo "try=$tries rc=$rc $(date '+%H:%M:%S')" >> "$D/reviews/deepseek/attempts.txt"
  if [ "$rc" -eq 9 ] && [ "$tries" -lt 60 ]; then sleep 15; continue; fi
  break
done
echo "deepseek-exit=$rc" > "$D/reviews/deepseek/exit.txt"
RUNDIR="$(tail -1 "$D/reviews/deepseek/stdout.txt" 2>/dev/null)"
if [ "$rc" -eq 0 ] && [ -d "$RUNDIR" ]; then
  cp "$RUNDIR/response.txt" "$D/reviews/deepseek/response.txt" 2>/dev/null
  cp "$RUNDIR/meta.json" "$D/reviews/deepseek/meta.json" 2>/dev/null
  echo "$RUNDIR" > "$D/reviews/deepseek/run_dir.txt"
fi
wait "$CODEX_PID"
echo "dispatch done $(date '+%H:%M:%S')"
