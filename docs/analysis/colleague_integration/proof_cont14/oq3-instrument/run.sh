#!/bin/bash
# OQ-3 failure-number instrument: canonical run (continuation 14, Letterlock).
#
#   run.sh baseline <evidence-dir>   pre-G1 baseline: controls, run 1 freezes the goldens, run 2 must reproduce them
#   run.sh accept   <evidence-dir>   post-G1 acceptance: controls, one run in accept mode against the frozen goldens
#
# The gate reads child exit statuses directly and reconciles Go's -json test events; a printed
# status is never the gate. Nothing tracked may change. No network beyond loopback (the children
# carry a closed-port proxy for every other host).
set -u
MODE="${1:?mode: baseline|accept}"
EV="${2:?evidence directory}"
HERE="$(cd "$(dirname "$0")" && pwd)"
REPO="$(git -C "$HERE" rev-parse --show-toplevel)"
GOLDENS="$HERE/goldens"
SCRATCH="${OQ3_SCRATCH:?OQ3_SCRATCH (a disposable directory for the overlay and the built binary)}"
export GOPROXY=off GOTOOLCHAIN=local GOFLAGS=
mkdir -p "$EV" "$SCRATCH"
LOG="$EV/run.log"
: > "$LOG"
say() { echo "$*" | tee -a "$LOG"; }
fail() { say "GATE=FAIL: $*"; exit 1; }

cd "$REPO" || fail "cd $REPO"
[ "$MODE" = baseline ] || [ "$MODE" = accept ] || fail "unknown mode $MODE"
if [ "$MODE" = baseline ] && [ -e "$GOLDENS" ]; then fail "goldens already exist at $GOLDENS; a baseline never overwrites them"; fi
if [ "$MODE" = accept ] && [ ! -d "$GOLDENS" ]; then fail "no frozen goldens at $GOLDENS"; fi

# Preconditions: the tested commit and a clean tracked tree.
HEAD_SHA="$(git rev-parse HEAD)"
TRACKED_DIRTY="$(git status --porcelain --untracked-files=no)"
[ -z "$TRACKED_DIRTY" ] || fail "tracked changes present before the run: $TRACKED_DIRTY"
GOMOD_SHA="$(shasum -a 256 go.mod | cut -d' ' -f1)"
{
  echo "{"
  echo "  \"mode\": \"$MODE\","
  echo "  \"head\": \"$HEAD_SHA\","
  echo "  \"branch\": \"$(git rev-parse --abbrev-ref HEAD)\","
  echo "  \"go\": \"$(go version)\","
  echo "  \"uname\": \"$(uname -a)\","
  echo "  \"started\": \"$(date '+%Y-%m-%dT%H:%M:%S%z')\","
  echo "  \"go_mod_sha256\": \"$GOMOD_SHA\""
  echo "}"
} > "$EV/environment.json"
say "OQ3 $MODE run at $HEAD_SHA"

# The built binary (entry arm) and the overlay.
go build -o "$SCRATCH/ollama" . > "$EV/build.log" 2>&1
BUILD_EXIT=$?
[ $BUILD_EXIT -eq 0 ] || fail "go build exited $BUILD_EXIT"
BIN_SHA="$(shasum -a 256 "$SCRATCH/ollama" | cut -d' ' -f1)"
say "binary sha256 $BIN_SHA"
python3 - "$HERE" "$REPO" "$SCRATCH/overlay.json" <<'PY'
import json, sys
here, repo, out = sys.argv[1:4]
files = ["oq3_harness_test.go", "oq3_smoke_test.go", "oq3_oracle_test.go", "oq3_scenarios_test.go",
         "oq3_runner_test.go", "oq3_main_test.go", "oq3_controls_test.go"]
json.dump({"Replace": {f"{repo}/cmd/zz_{f}": f"{here}/{f}" for f in files}}, open(out, "w"), indent=1)
PY
cp "$SCRATCH/overlay.json" "$EV/overlay.json"
shasum -a 256 "$HERE"/oq3_*_test.go > "$EV/instrument.sha256"

go vet -overlay "$SCRATCH/overlay.json" ./cmd > "$EV/vet.log" 2>&1
VET_EXIT=$?
[ $VET_EXIT -eq 0 ] || fail "go vet exited $VET_EXIT"

# gotest <name> <regex> <expected tests...>: runs, keeps the -json stream, reconciles discovery.
gotest() {
  local name="$1" regex="$2"; shift 2
  go test -count=1 -json -overlay "$SCRATCH/overlay.json" -run "$regex" -timeout 1800s ./cmd > "$EV/$name.jsonl" 2> "$EV/$name.stderr"
  local code=$?
  python3 - "$EV/$name.jsonl" "$@" <<'PY'
import json, sys
path, expected = sys.argv[1], sys.argv[2:]
actions = {}
for line in open(path):
    try:
        e = json.loads(line)
    except ValueError:
        continue
    if e.get("Test") and e.get("Action") in ("run", "pass", "fail", "skip"):
        actions.setdefault(e["Test"], []).append(e["Action"])
bad = []
for t in expected:
    a = actions.get(t, [])
    if "run" not in a:
        bad.append(f"{t}: not discovered")
    elif "pass" not in a:
        bad.append(f"{t}: {a[-1] if a else 'no result'}")
extra = [t for t in actions if t.split("/")[0] not in expected]
if extra:
    bad.append("unexpected tests ran: " + ", ".join(sorted(extra)))
print("RECONCILE " + ("OK " if not bad else "FAIL ") + json.dumps({t: actions.get(t) for t in expected}))
for b in bad:
    print("  " + b)
sys.exit(1 if bad else 0)
PY
  local rc=$?
  say "$name: go test exit $code, reconcile exit $rc"
  [ $code -eq 0 ] && [ $rc -eq 0 ]
}

if [ "${OQ3_GATE_SELFTEST:-}" = 1 ]; then
  # The gate must block (Method 04 T-08): a failing test, a missing required test, an empty selection.
  # A failing test: TestOQ3FailureNumber without its required environment fails at once.
  if gotest selftest-failing '^TestOQ3FailureNumber$' TestOQ3FailureNumber; then fail "gate passed a failing test"; fi
  if gotest selftest-missing '^TestOQ3ScorerControls$' TestOQ3ScorerControls TestOQ3RunnerControls; then fail "gate passed with a required test missing"; fi
  if gotest selftest-empty '^TestNoSuchOQ3Test$' TestOQ3ScorerControls; then fail "gate passed an empty selection"; fi
  say "GATE SELFTEST: all three faults blocked"
fi

gotest controls '^TestOQ3(ScorerControls|RunnerControls|HarnessSmoke)$' TestOQ3ScorerControls TestOQ3RunnerControls TestOQ3HarnessSmoke || fail "instrument controls"

run_number() {
  local out="$1" freeze="$2" mode="$3"
  OQ3_BINARY="$SCRATCH/ollama" OQ3_EVIDENCE_DIR="$out" OQ3_GOLDENS="$GOLDENS" OQ3_FREEZE="$freeze" OQ3_MODE="$mode" \
    OQ3_SCENARIOS=all OQ3_SOURCE="$HEAD_SHA binary $BIN_SHA" \
    gotest "$(basename "$out")" '^TestOQ3FailureNumber$' TestOQ3FailureNumber
}

if [ "$MODE" = baseline ]; then
  run_number "$EV/run1" 1 baseline || fail "baseline run 1"
  run_number "$EV/run2" 0 baseline || fail "baseline run 2 (repeat against the frozen goldens)"
  python3 - "$EV/run1/summary.json" "$EV/run2/summary.json" > "$EV/repeat-comparison.txt" <<'PY'
import json, sys
a, b = (json.load(open(p)) for p in sys.argv[1:3])
keys = ["controlled_conversation_trials", "observed_deliveries", "bad_observed_deliveries", "missing_required_deliveries",
        "failure_number", "category_counts_overlapping", "per_trial", "operation_verdicts", "required_refusal_verdicts", "valid"]
diff = [k for k in keys if a.get(k) != b.get(k)]
print("REPEAT " + ("IDENTICAL" if not diff else "DIFFERS: " + ", ".join(diff)))
sys.exit(1 if diff else 0)
PY
  REP=$?
  cat "$EV/repeat-comparison.txt" | tee -a "$LOG"
  [ $REP -eq 0 ] || fail "run 1 and run 2 disagree"
else
  run_number "$EV/run1" 0 accept || fail "acceptance run"
fi

# Postconditions: nothing tracked changed.
TRACKED_AFTER="$(git status --porcelain --untracked-files=no)"
[ -z "$TRACKED_AFTER" ] || fail "tracked changes after the run: $TRACKED_AFTER"
[ "$(shasum -a 256 go.mod | cut -d' ' -f1)" = "$GOMOD_SHA" ] || fail "go.mod changed"
echo "$(date '+%Y-%m-%dT%H:%M:%S%z')" > "$EV/finished.txt"
say "GATE=PASS ($MODE): instrument valid; see run1/summary.json"
exit 0
