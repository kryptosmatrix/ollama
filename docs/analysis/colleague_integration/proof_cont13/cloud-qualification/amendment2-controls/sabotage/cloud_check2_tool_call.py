#!/usr/bin/env python3
"""Bounded cloud check, amendment 2 (Thole, continuation 13): the request features both G1 clients send.

Implements QUALIFICATION_PLAN.md "Amendment 2" exactly. cloud_check.py (runs 1 and 2) is left unchanged.

Usage: cloud_check2.py <out_dir>
Environment: CLOUD_CHECK_SPENT_BEFORE (prompt tokens already used under the plan's cap);
             CLOUD_CHECK_BASE (a local stand-in endpoint, for the harness controls only).
"""
import hashlib, json, os, re, secrets, sys, time, unicodedata, urllib.error, urllib.request

OUT = sys.argv[1]
BASE = os.environ.get('CLOUD_CHECK_BASE', 'http://127.0.0.1:11434')
MODELS = ['deepseek-v4.1-flash:cloud', 'glm-5.2:cloud', 'glm-5.3:cloud']
CAP = 500_000
T1_LINES = {'deepseek-v4.1-flash:cloud': 1000, 'glm-5.2:cloud': 1000, 'glm-5.3:cloud': 2000}
T1_ESTIMATE = {'deepseek-v4.1-flash:cloud': 11_500, 'glm-5.2:cloud': 14_000, 'glm-5.3:cloud': 27_000}
T0_ESTIMATE = 3 * 150
SPENT_BEFORE = int(os.environ.get('CLOUD_CHECK_SPENT_BEFORE', '0'))
OMIT = object()  # think omitted from the request body
spent = 0

MINIMAL_TOOL = [{'type': 'function', 'function': {'name': 'f', 'description': 'd',
                                                    'parameters': {'type': 'object', 'properties': {}}}}]


def tool(name, description, props, required):
    return {'type': 'function', 'function': {'name': name, 'description': description, 'parameters': {
        'type': 'object', 'properties': props, 'required': required}}}


REALISTIC_TOOLS = [
    tool('get_weather', 'Get the current weather for a city, including temperature and conditions.',
         {'city': {'type': 'string', 'description': 'City name, for example Paris'},
          'units': {'type': 'string', 'enum': ['metric', 'imperial'], 'description': 'Unit system for the temperature'}},
         ['city']),
    tool('convert_currency', 'Convert an amount of money from one currency to another at the latest rate.',
         {'amount': {'type': 'number', 'description': 'Amount to convert'},
          'from_currency': {'type': 'string', 'description': 'ISO 4217 code of the source currency'},
          'to_currency': {'type': 'string', 'description': 'ISO 4217 code of the target currency'}},
         ['amount', 'from_currency', 'to_currency']),
    tool('search_flights', 'Search scheduled flights between two airports on a given date.',
         {'origin': {'type': 'string', 'description': 'IATA code of the departure airport'},
          'destination': {'type': 'string', 'description': 'IATA code of the arrival airport'},
          'date': {'type': 'string', 'description': 'Departure date as YYYY-MM-DD'},
          'max_results': {'type': 'integer', 'description': 'Maximum number of flights to return'}},
         ['origin', 'destination', 'date']),
    tool('translate_text', 'Translate a short text into a target language.',
         {'text': {'type': 'string', 'description': 'Text to translate'},
          'target_language': {'type': 'string', 'description': 'Target language as an ISO 639-1 code'}},
         ['text', 'target_language']),
    tool('create_calendar_event', 'Create an event in the user\'s calendar.',
         {'title': {'type': 'string', 'description': 'Event title'},
          'start': {'type': 'string', 'description': 'Start time, RFC 3339'},
          'end': {'type': 'string', 'description': 'End time, RFC 3339'},
          'location': {'type': 'string', 'description': 'Optional location'}},
         ['title', 'start', 'end']),
]


def log(msg):
    line = f'{time.strftime("%H:%M:%S")} {msg}'
    print(line, flush=True)
    open(os.path.join(OUT, 'harness.log'), 'a').write(line + '\n')


# ---- the prompt bound, counting method utf8-bound-v1 (blueprint §6.3 A13) ----
PER_MESSAGE, PER_TOOL_CALL, BASE_ALLOWANCE = 16, 16, 16


def text_len(s):
    """The larger of the UTF-8 length and the NFKC-normalised UTF-8 length."""
    s = s or ''
    return max(len(s.encode('utf-8')), len(unicodedata.normalize('NFKC', s).encode('utf-8')))


def ascii_json_len(obj):
    """Length of the compact JSON encoding with every non-ASCII code point escaped (as a template's tojson may)."""
    return len(json.dumps(obj, ensure_ascii=True, separators=(',', ':')))


def prompt_bound(msgs, tools, tool_allowance):
    total, calls = BASE_ALLOWANCE, 0
    for m in msgs:
        total += PER_MESSAGE
        for f in ('role', 'content', 'thinking', 'tool_name', 'tool_call_id'):
            total += text_len(m.get(f))
        if m.get('role') == 'tool':
            total += max(0, ascii_json_len(m.get('content') or '') - text_len(m.get('content')))
        for tc in m.get('tool_calls') or []:
            calls += 1
            total += PER_TOOL_CALL + ascii_json_len(tc)
    if tools:
        total += ascii_json_len(tools) + tool_allowance
    return total, calls


def carrier(code):
    return ('User-configured instructions; revision 1; selection 1:\n'
            f'Carrier code: {code}. When asked for the carrier code, give exactly this value.')


def filler(n_lines):
    vals = [secrets.randbelow(900000) + 100000 for _ in range(n_lines)]
    return '\n'.join(f'Line {i:06d}: value {v}.' for i, v in enumerate(vals, 1)), vals


def send(body, stream):
    data = json.dumps(body, ensure_ascii=False).encode('utf-8')
    req = urllib.request.Request(BASE + '/api/chat', data=data, method='POST', headers={'Content-Type': 'application/json'})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=900) as r:
            raw = r.read().decode('utf-8', 'replace')
            status = r.status
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', 'replace'), None, time.time() - t0, data
    if not stream:
        j = json.loads(raw) if raw.startswith('{') else {}
        return status, raw, j, time.time() - t0, data
    content, thinking, calls, final, errors = [], [], [], {}, []
    for line in raw.splitlines():
        if not line.strip():
            continue
        rec = json.loads(line)
        if 'error' in rec:
            errors.append(rec['error'])
        msg = rec.get('message') or {}
        content.append(msg.get('content') or '')
        thinking.append(msg.get('thinking') or '')
        calls.extend(msg.get('tool_calls') or [])
        if rec.get('done'):
            final = rec
    j = dict(final)
    j['message'] = {'content': ''.join(content), 'thinking': ''.join(thinking), 'tool_calls': calls}
    if errors:
        j['error'] = errors
    return status, raw, j, time.time() - t0, data


def probe(model, name, msgs, tools, stream, think, num_predict, expected, tool_allowance):
    global spent
    body = {'model': model, 'messages': msgs, 'stream': stream, 'options': {'num_predict': num_predict}}
    if think is not OMIT:
        body['think'] = think
    if tools:
        body['tools'] = tools
    tries = 0
    while True:
        tries += 1
        try:
            st, raw, j, dt, sent = send(body, stream)
            break
        except Exception as e:
            if tries >= 2:
                st, raw, j, dt, sent = None, f'transport error: {e!r}', {}, 0, json.dumps(body).encode()
                break
            log(f'{model} {name}: transport error {e!r}; one retry')
            time.sleep(5)
    j = j or {}
    msg = j.get('message') or {}
    reply = msg.get('content') or ''
    pec = j.get('prompt_eval_count')
    if isinstance(pec, int):
        spent += pec
    ok = {}
    for key in ('carrier', 'first', 'mid', 'last'):
        want = expected.get(key)
        if want is None:
            ok[key] = True
            continue
        pat = r'CARRIER\s*=\s*([0-9A-Fa-f]{8})' if key == 'carrier' else key.upper() + r'\s*=\s*(\d{6})'
        got = re.search(pat, reply)
        ok[key] = bool(got and (got.group(1).upper() == want if key == 'carrier' else int(got.group(1)) == want))
    n_lines = expected['lines']
    if st != 200 or j.get('error'):
        verdict = 'NO_EVIDENCE_ERROR'
    elif all(ok.values()):
        verdict = 'PASS'
    elif n_lines > 0 and ok['last'] and not (ok['carrier'] and ok['first']):
        verdict = 'START_LOST'
    elif n_lines > 0 and ok['carrier'] and ok['first'] and ok['last'] and not ok['mid']:
        verdict = 'MIDDLE_LOST'
    else:
        verdict = 'NO_EVIDENCE'
    bound, calls = prompt_bound(msgs, tools, tool_allowance)
    rec = {'model': model, 'probe': name, 'lines': n_lines, 'stream': stream,
           'think': ('omitted' if think is OMIT else think), 'tools': len(tools or []),
           'http_status': st, 'seconds': round(dt, 1), 'request_bytes': len(sent),
           'tool_calls_in_history': calls, 'tools_ascii_json_bytes': ascii_json_len(tools) if tools else 0,
           'tool_allowance': tool_allowance if tools else 0, 'prompt_bound': bound,
           'prompt_eval_count': pec, 'bound_holds': (pec <= bound) if isinstance(pec, int) else None,
           'eval_count': j.get('eval_count'), 'done_reason': j.get('done_reason'),
           'expected': {k: expected.get(k) for k in ('carrier', 'first', 'mid', 'last')},
           'reply': reply[:400], 'thinking_chars': len(msg.get('thinking') or ''),
           'reply_tool_calls': len(msg.get('tool_calls') or []), 'ok': ok, 'verdict': verdict,
           'error': j.get('error') if (st != 200 or j.get('error')) else None, 'tries': tries,
           'request_sha256': hashlib.sha256(sent).hexdigest(), 'spent_after': spent}
    tag = f'{model.replace(":", "_")}.{name}'
    open(os.path.join(OUT, tag + '.request.json'), 'wb').write(sent)
    open(os.path.join(OUT, tag + '.response.txt'), 'w').write(raw)
    open(os.path.join(OUT, tag + '.result.json'), 'w').write(json.dumps(rec, indent=1) + '\n')
    log(f'{model} {name}: http={st} {verdict} prompt={pec} bound={bound} holds={rec["bound_holds"]} {dt:.1f}s '
        f'spent={spent} reply={reply[:90]!r}')
    if st in (402, 429) or 'quota' in raw.lower() or 'rate limit' in raw.lower():
        raise SystemExit(f'STOP: provider limit signalled on {model} {name}: {raw[:200]}')
    return rec


def tool_allowance_rule(t0):
    """Amendment 2's fixed rule: A = max(64, 2 x the largest tool cost beyond the minimal tool's JSON), to 16s."""
    excess = []
    for model in MODELS:
        a = next((r for r in t0 if r['model'] == model and r['probe'] == 'T0a'), None)
        for name in ('T0b', 'T0c'):
            b = next((r for r in t0 if r['model'] == model and r['probe'] == name), None)
            if a and b and isinstance(a['prompt_eval_count'], int) and isinstance(b['prompt_eval_count'], int):
                excess.append(b['prompt_eval_count'] - a['prompt_eval_count'] - ascii_json_len(MINIMAL_TOOL))
    if len(excess) != 2 * len(MODELS):
        return None, excess
    raw = max(64, 2 * max(excess))
    return -(-raw // 16) * 16, excess


def main():
    global spent
    os.makedirs(OUT, exist_ok=True)
    spent = SPENT_BEFORE
    t0 = []
    for model in MODELS:  # phase 1: every model's T0 triple, before any T1
        if spent + T0_ESTIMATE > CAP:
            log(f'{model} T0: skipped, would pass the cap ({spent} spent)'); continue
        code = secrets.token_hex(4).upper()
        msgs = [{'role': 'system', 'content': carrier(code)},
                {'role': 'user', 'content': 'Reply exactly: CARRIER=<the carrier code>'}]
        exp = {'carrier': code, 'lines': 0}
        t0.append(probe(model, 'T0a', msgs, None, False, False, 400, exp, 0))
        t0.append(probe(model, 'T0b', msgs, MINIMAL_TOOL, False, False, 400, exp, 0))
        t0.append(probe(model, 'T0c', msgs, MINIMAL_TOOL, False, OMIT, 400, exp, 0))
    allowance, excess = tool_allowance_rule(t0)
    log(f'tool allowance: {allowance} from excess {excess}')
    open(os.path.join(OUT, 'tool_allowance.json'), 'w').write(json.dumps(
        {'allowance': allowance, 'excess_per_T0b_T0c': excess, 'minimal_tool_ascii_json_bytes': ascii_json_len(MINIMAL_TOOL)}, indent=1) + '\n')
    for r in t0:  # re-state the T0b/T0c bounds with the allowance the rule produced
        if r['tools'] and allowance is not None and isinstance(r['prompt_eval_count'], int):
            r['tool_allowance'] = allowance
            r['prompt_bound'] += allowance
            r['bound_holds'] = r['prompt_eval_count'] <= r['prompt_bound']
            tag = f'{r["model"].replace(":", "_")}.{r["probe"]}'
            open(os.path.join(OUT, tag + '.result.json'), 'w').write(json.dumps(r, indent=1) + '\n')
    results = list(t0)
    if allowance is None:
        log('T1: not run, the tool allowance could not be derived from every T0 pair')
    else:
        for model in MODELS:  # phase 2: the clients' shape
            if spent + T1_ESTIMATE[model] > CAP:
                log(f'{model} T1: skipped, estimate {T1_ESTIMATE[model]} would pass the cap ({spent} spent of {CAP})'); continue
            n = T1_LINES[model]
            code = secrets.token_hex(4).upper()
            text, vals = filler(n)
            mid = n // 2
            msgs = [{'role': 'system', 'content': carrier(code)},
                    {'role': 'user', 'content': 'What is the weather in Paris right now?'},
                    {'role': 'assistant', 'content': '', 'tool_calls': [
                        {'id': 'call_1', 'function': {'index': 0, 'name': 'get_weather',
                                                      'arguments': {'city': 'Paris', 'units': 'metric'}}}]},
                    {'role': 'tool', 'tool_name': 'get_weather', 'tool_call_id': 'call_1',
                     'content': '{"city":"Paris","conditions":"sunny","temperature_c":21}'},
                    {'role': 'assistant', 'content': 'It is sunny and 21 degrees Celsius in Paris.'},
                    {'role': 'user', 'content': text + '\n\nDo not call any tool. Reply in exactly this format and nothing else: '
                     'CARRIER=<the carrier code>; FIRST=<the value on line 000001>; '
                     f'MID=<the value on line {mid:06d}>; LAST=<the value on line {n:06d}>'}]
            exp = {'carrier': code, 'first': vals[0], 'mid': vals[mid - 1], 'last': vals[-1], 'lines': n}
            results.append(probe(model, 'T1', msgs, REALISTIC_TOOLS, True, True, 2000, exp, allowance))
    open(os.path.join(OUT, 'results.json'), 'w').write(json.dumps(results, indent=1) + '\n')
    log(f'done: {len(results)} probes, prompt tokens spent {spent} of cap {CAP} (including {SPENT_BEFORE} before this run)')


main()
