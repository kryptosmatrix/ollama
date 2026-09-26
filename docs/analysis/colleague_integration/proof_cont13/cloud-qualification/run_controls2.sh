#!/bin/bash
# Prove cloud_check2.py against fake_cloud.py in every mode before any real request (amendment 2 controls).
# Each mode must produce exactly the expected verdicts, in order: T0a T0b T0c for each of the three models,
# then T1 for each model. Any mismatch exits non-zero. HARNESS and CONTROLS_OUT select a copy and an output
# directory, for the deliberately broken copies.
set -u
D="$(cd "$(dirname "$0")" && pwd)"
OUT="${CONTROLS_OUT:-$D/amendment2-controls}"
rm -rf "$OUT"; mkdir -p "$OUT"
PORT=18765
fail=0
rep() { local out="" i; for i in $(seq "$2"); do out="$out $1"; done; printf '%s' "${out# }"; }
expect() {
  local mode="$1" want="$2"
  python3 "$D/fake_cloud.py" "$PORT" "$mode" & local fpid=$!
  sleep 0.5
  CLOUD_CHECK_BASE="http://127.0.0.1:$PORT" CLOUD_CHECK_SPENT_BEFORE=0 \
    python3 "${HARNESS:-$D/cloud_check2.py}" "$OUT/$mode" > "$OUT/$mode.stdout" 2>&1
  local rc=$?
  kill "$fpid" 2>/dev/null; wait "$fpid" 2>/dev/null
  local got
  got="$(python3 -c "import json;print(' '.join(r['verdict'] for r in json.load(open('$OUT/$mode/results.json'))))" 2>/dev/null)"
  if [ "$rc" -eq 0 ] && [ "$got" = "$want" ]; then echo "PASS control $mode: $got"
  else echo "FAIL control $mode (rc=$rc): got [$got] want [$want]"; fail=1; fi
}
expect echo        "$(rep PASS 12)"
expect drop_start  "$(rep NO_EVIDENCE 9) $(rep START_LOST 3)"
expect drop_middle "$(rep PASS 9) $(rep MIDDLE_LOST 3)"
expect tool_call   "$(rep NO_EVIDENCE_TOOL_CALL 12)"
expect error       "$(rep NO_EVIDENCE_ERROR 9)"
expect garbage     "$(rep NO_EVIDENCE 12)"
# Request shape: T0a/T0b think false, T0c think omitted, T1 streamed with think true, five tools, one historical
# tool call carrying an id, and a tool result carrying tool_call_id; the allowance is derived and applied.
python3 - "$OUT/echo" <<'PY' || fail=1
import glob, json, os, sys
d = sys.argv[1]
rs = json.load(open(d + '/results.json'))
by = {(r['model'], r['probe']): r for r in rs}
models = ['deepseek-v4.1-flash:cloud', 'glm-5.2:cloud', 'glm-5.3:cloud']
assert len(rs) == 12 and len(by) == 12, 'probe count'
for m in models:
    tag = m.replace(':', '_')
    body = {p: json.load(open(f'{d}/{tag}.{p}.request.json')) for p in ('T0a', 'T0b', 'T0c', 'T1')}
    assert body['T0a'].get('think') is False and 'tools' not in body['T0a'], 'T0a shape'
    assert body['T0b'].get('think') is False and len(body['T0b']['tools']) == 1, 'T0b shape'
    assert 'think' not in body['T0c'] and len(body['T0c']['tools']) == 1, 'T0c shape'
    t1 = body['T1']
    assert t1['stream'] is True and t1['think'] is True and len(t1['tools']) == 5, 'T1 flags'
    calls = [c for msg in t1['messages'] for c in (msg.get('tool_calls') or [])]
    assert len(calls) == 1 and calls[0]['id'] == 'call_1', 'T1 historical call id'
    assert any(msg.get('role') == 'tool' and msg.get('tool_call_id') == 'call_1' for msg in t1['messages']), 'T1 tool_call_id'
    assert 'MID=<the value on line' in t1['messages'][-1]['content'], 'T1 middle question'
    assert by[(m, 'T1')]['thinking_chars'] > 0, 'streamed thinking not accumulated'
    assert by[(m, 'T1')]['tool_allowance'] >= 64 and by[(m, 'T0b')]['tool_allowance'] >= 64, 'allowance applied'
    assert by[(m, 'T0a')]['tool_allowance'] == 0, 'allowance on a tool-free probe'
    for p in ('T0a', 'T0b', 'T0c', 'T1'):
        assert json.load(open(f'{d}/{tag}.{p}.result.json')) == by[(m, p)], f'per-probe file differs from results.json: {p}'
al = json.load(open(d + '/tool_allowance.json'))
assert al['allowance'] >= 64 and al['allowance'] % 16 == 0 and len(al['excess_per_T0b_T0c']) == 6, 'allowance rule'
print('PASS control shape: T0 think false/false/omitted; T1 streamed, think true, 5 tools, call id and tool_call_id; allowance', al['allowance'])
PY
exit $fail
