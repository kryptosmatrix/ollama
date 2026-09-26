#!/usr/bin/env python3
"""Bounded cloud check for G1 (Thole, continuation 13). Implements QUALIFICATION_PLAN.md exactly.

Usage: cloud_check.py <out_dir>
"""
import hashlib, json, os, re, secrets, sys, time, urllib.error, urllib.request

OUT = sys.argv[1]
BASE = 'http://127.0.0.1:11434'
MODELS = ['deepseek-v4.1-flash:cloud', 'glm-5.2:cloud', 'glm-5.3:cloud']
CAP = 500_000
Q2_LINES = 2000
Q3_TARGET_TOKENS = 100_000
NUM_PREDICT = int(os.environ.get('CLOUD_CHECK_NUM_PREDICT', '48'))
ONLY = os.environ.get('CLOUD_CHECK_ONLY')  # amendment 1: 'glm-5.3:cloud:Q1,Q2'
SPENT_BEFORE = int(os.environ.get('CLOUD_CHECK_SPENT_BEFORE', '0'))
spent = 0


def log(msg):
    line = f'{time.strftime("%H:%M:%S")} {msg}'
    print(line, flush=True)
    open(os.path.join(OUT, 'harness.log'), 'a').write(line + '\n')


def post(body, timeout=900):
    data = json.dumps(body).encode('utf-8')
    req = urllib.request.Request(BASE + '/api/chat', data=data, method='POST', headers={'Content-Type': 'application/json'})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, r.read().decode('utf-8', 'replace'), time.time() - t0, len(data)
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', 'replace'), time.time() - t0, len(data)


def carrier(code):
    return ('User-configured instructions; revision 1; selection 1:\n'
            f'Carrier code: {code}. When asked for the carrier code, give exactly this value.')


def filler(n_lines):
    first, last = secrets.randbelow(900000) + 100000, secrets.randbelow(900000) + 100000
    lines = []
    for i in range(1, n_lines + 1):
        v = first if i == 1 else last if i == n_lines else secrets.randbelow(900000) + 100000
        lines.append(f'Line {i:06d}: value {v}.')
    return '\n'.join(lines), first, last


def probe(model, name, n_lines):
    global spent
    code = secrets.token_hex(4).upper()
    if n_lines == 0:
        msgs = [{'role': 'system', 'content': carrier(code)},
                {'role': 'user', 'content': 'Reply exactly: CARRIER=<the carrier code>'}]
        first = last = None
    else:
        text, first, last = filler(n_lines)
        msgs = [{'role': 'system', 'content': carrier(code)},
                {'role': 'user', 'content': text + '\n\nReply in exactly this format and nothing else: '
                 'CARRIER=<the carrier code>; FIRST=<the value on line 000001>; '
                 f'LAST=<the value on line {n_lines:06d}>'}]
    body = {'model': model, 'messages': msgs, 'stream': False, 'think': False, 'options': {'num_predict': NUM_PREDICT}}
    raw = json.dumps(body, ensure_ascii=False).encode('utf-8')
    utf8_bytes = sum(len(m['content'].encode('utf-8')) for m in msgs)
    bound = utf8_bytes + 16 * (len(msgs) + 1)
    tries = 0
    while True:
        tries += 1
        try:
            st, text_resp, dt, sent_bytes = post(body)
            break
        except Exception as e:
            if tries >= 2:
                st, text_resp, dt, sent_bytes = None, f'transport error: {e!r}', 0, len(raw)
                break
            log(f'{model} {name}: transport error {e!r}; one retry')
            time.sleep(5)
    j = json.loads(text_resp) if text_resp.startswith('{') else {}
    reply = ((j.get('message') or {}).get('content') or '')
    pec = j.get('prompt_eval_count')
    if isinstance(pec, int):
        spent += pec
    got_c = re.search(r'CARRIER\s*=\s*([0-9A-Fa-f]{8})', reply)
    got_f = re.search(r'FIRST\s*=\s*(\d{6})', reply)
    got_l = re.search(r'LAST\s*=\s*(\d{6})', reply)
    ok_c = bool(got_c and got_c.group(1).upper() == code)
    ok_f = first is None or bool(got_f and int(got_f.group(1)) == first)
    ok_l = last is None or bool(got_l and int(got_l.group(1)) == last)
    if st != 200:
        verdict = 'NO_EVIDENCE_ERROR'
    elif ok_c and ok_f and ok_l:
        verdict = 'PASS'
    elif n_lines > 0 and ok_l and not (ok_c and ok_f):
        verdict = 'START_LOST'
    else:
        verdict = 'NO_EVIDENCE'
    rec = {'model': model, 'probe': name, 'lines': n_lines, 'http_status': st, 'seconds': round(dt, 1),
           'request_bytes': sent_bytes, 'content_utf8_bytes': utf8_bytes, 'conservative_bound': bound,
           'prompt_eval_count': pec, 'eval_count': j.get('eval_count'), 'done_reason': j.get('done_reason'),
           'bound_holds': (pec <= bound) if isinstance(pec, int) else None,
           'expected': {'carrier': code, 'first': first, 'last': last}, 'reply': reply[:400],
           'ok': {'carrier': ok_c, 'first': ok_f, 'last': ok_l}, 'verdict': verdict,
           'error': j.get('error') if st != 200 else None, 'tries': tries,
           'request_sha256': hashlib.sha256(raw).hexdigest(), 'spent_after': spent}
    tag = f'{model.replace(":", "_")}.{name}'
    open(os.path.join(OUT, tag + '.request.json'), 'wb').write(raw)
    open(os.path.join(OUT, tag + '.response.txt'), 'w').write(text_resp)
    open(os.path.join(OUT, tag + '.result.json'), 'w').write(json.dumps(rec, indent=1) + '\n')
    log(f'{model} {name}: http={st} {verdict} prompt={pec} bound={bound} bound_holds={rec["bound_holds"]} '
        f'{dt:.1f}s spent={spent} reply={reply[:90]!r}')
    if st in (402, 429) or 'quota' in text_resp.lower() or 'rate limit' in text_resp.lower():
        raise SystemExit(f'STOP: provider limit signalled on {model} {name}: {text_resp[:200]}')
    return rec


def main():
    global spent
    os.makedirs(OUT, exist_ok=True)
    results = []
    spent = SPENT_BEFORE
    if ONLY:
        model, probes = ONLY.rsplit(':', 1)
        for name in probes.split(','):
            if name == 'Q1':
                results.append(probe(model, 'Q1', 0))
            elif name == 'Q2':
                if spent + 26_000 > CAP:
                    log(f'{model} Q2: skipped, would pass the cap ({spent} spent)'); continue
                results.append(probe(model, 'Q2', Q2_LINES))
        open(os.path.join(OUT, 'results.json'), 'w').write(json.dumps(results, indent=1) + '\n')
        log(f'done: {len(results)} probes, prompt tokens spent {spent} of cap {CAP} (including {SPENT_BEFORE} before this run)')
        return
    for model in MODELS:
        results.append(probe(model, 'Q1', 0))
        q2 = probe(model, 'Q2', Q2_LINES)
        results.append(q2)
        per_line = (q2['prompt_eval_count'] / Q2_LINES) if isinstance(q2['prompt_eval_count'], int) else None
        if per_line is None:
            log(f'{model} Q3: skipped, Q2 gave no token count')
            continue
        q3_lines = int(Q3_TARGET_TOKENS / per_line)
        estimate = int(q3_lines * per_line)
        if spent + estimate > CAP:
            log(f'{model} Q3: skipped, estimate {estimate} would pass the cap ({spent} spent of {CAP})')
            continue
        results.append(probe(model, 'Q3', q3_lines))
    open(os.path.join(OUT, 'results.json'), 'w').write(json.dumps(results, indent=1) + '\n')
    log(f'done: {len(results)} probes, prompt tokens spent {spent} of cap {CAP}')


main()
