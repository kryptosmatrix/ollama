#!/bin/bash
# Dispatch the frozen round-4 review packets of OQ-3 instrument revision 4 to Codex (continuation 15, Treadle).
# KANON 19 and Method 20: a fresh reviewer each round on a different substrate, plus a reader whose only
# job is to attack the repair. Both run at once, read-only, on the frozen commit.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
JUDGE_SHA="${1:?usage: dispatch.sh <PACKET.md sha256> <PACKET_REPAIR.md sha256>}"
REPAIR_SHA="${2:?usage: dispatch.sh <PACKET.md sha256> <PACKET_REPAIR.md sha256>}"
REPO=/Users/krypto/GitHub/ollama-eko-chat-read-integrity
check() {
  local got; got="$(shasum -a 256 "$1" | cut -d' ' -f1)"
  if [ "$got" != "$2" ]; then echo "STOP: $1 hash $got != frozen $2"; exit 1; fi
}
check "$D/PACKET.md" "$JUDGE_SHA"
check "$D/PACKET_REPAIR.md" "$REPAIR_SHA"
for name in codex-judge codex-repair; do
  if [ -e "$D/reviews/$name" ]; then echo "STOP: $D/reviews/$name exists; a review is never overwritten"; exit 1; fi
done
if [ -n "$(git -C "$REPO" status --porcelain --untracked-files=no)" ]; then echo "STOP: tracked changes in the worktree"; exit 1; fi
run() {
  local name="$1" packet="$2" out="$D/reviews/$1"
  mkdir -p "$out"
  {
    printf '%s\n' "You are an independent reviewer working in a read-only sandbox whose working directory is an Ollama fork at the commit named by git HEAD. You may open any file in the repository, including everything under docs/, and run read-only commands such as git diff, git show and git log. Change nothing and run nothing that writes. Cite file:line for every finding."
    printf '\n---\n\n'
    cat "$packet"
  } > "$out/prompt.md"
  shasum -a 256 "$out/prompt.md" > "$out/prompt.sha256"
  git -C "$REPO" rev-parse HEAD > "$out/head.txt"
  date '+%Y-%m-%dT%H:%M:%S%z' > "$out/started.txt"
  ( cd "$REPO" && /Users/krypto/.local/bin/codex exec -m gpt-6-astra -c 'service_tier="default"' \
      --disable hooks --disable memories --disable plugins --disable multi_agent \
      -c 'web_search="disabled"' \
      --sandbox read-only --ephemeral --json \
      -C "$REPO" \
      -o "$out/last_message.md" \
      "$(cat "$out/prompt.md")" < /dev/null > "$out/events.jsonl" 2> "$out/stderr.txt"
    echo "codex-exit=$? $(date '+%Y-%m-%dT%H:%M:%S%z')" > "$out/exit.txt" ) &
  echo "$name pid $!"
}
run codex-judge "$D/PACKET.md"
run codex-repair "$D/PACKET_REPAIR.md"
wait
echo "both reviews finished $(date '+%Y-%m-%dT%H:%M:%S%z')"
