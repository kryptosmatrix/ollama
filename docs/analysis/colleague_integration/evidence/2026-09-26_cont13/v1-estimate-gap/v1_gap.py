#!/usr/bin/env python3
"""G1 §6.5 V1: gap between the daemon's token estimate and the runner's actual prompt count (Thole, cont 13).

For each model and route, and each prompt case, records three counts of the same request:
  go_estimate      - runner /tokenize of the rendered prompt with the flags Go's Tokenize sends today
                     (both omitted: runner defaults add_special=false, parse_special=true)
  inference_flags  - runner /tokenize with add_special=true, parse_special=true
  actual           - prompt_eval_count (cache_n + prompt_n) of a real /api/chat request, num_predict 1
The rendered prompt comes from the daemon's own `_debug_render_only` for that request, so it is the text
the route builds (Go template, renderer, or native apply-template on the native route).

Usage: v1_gap.py <sandbox_rt_dir> <out_dir>
"""
import json, os, re, signal, subprocess, sys, time, urllib.error, urllib.request

RT, OUT = sys.argv[1], sys.argv[2]
PORT = 11500
BASE = f'http://127.0.0.1:{PORT}'
CARRIER = ('User-configured instructions; revision 1; selection 1:\n'
           'Use Australian English. Keep answers short. Finish with ZEBRA-7Q.')
TOOLS = [
    {'type': 'function', 'function': {'name': 'get_weather', 'description': 'Current weather for a city',
     'parameters': {'type': 'object', 'properties': {'city': {'type': 'string', 'description': 'City name'}},
                    'required': ['city']}}},
    {'type': 'function', 'function': {'name': 'add', 'description': 'Add two integers',
     'parameters': {'type': 'object', 'properties': {'a': {'type': 'integer'}, 'b': {'type': 'integer'}},
                    'required': ['a', 'b']}}},
]
SYS = {'role': 'system', 'content': CARRIER}
CASES = {
    'plain': {'messages': [SYS, {'role': 'user', 'content': 'What is the capital of Australia?'}]},
    'multiturn': {'messages': [SYS] + sum(([{'role': 'user', 'content': f'Question {i}: name a Brisbane suburb.'},
                                          {'role': 'assistant', 'content': f'Answer {i}: New Farm.'}] for i in range(6)), [])
                  + [{'role': 'user', 'content': 'And one more?'}]},
    'tools': {'messages': [SYS, {'role': 'user', 'content': 'What is the weather in Hobart?'}], 'tools': TOOLS},
    'toolcalls': {'messages': [SYS, {'role': 'user', 'content': 'Add 17 and 25, then tell me.'},
                               {'role': 'assistant', 'content': '', 'tool_calls': [{'function': {'name': 'add', 'arguments': {'a': 17, 'b': 25}}}]},
                               {'role': 'tool', 'content': '42', 'tool_name': 'add'},
                               {'role': 'user', 'content': 'Thanks. Now double it.'}], 'tools': TOOLS},
    'unicode': {'messages': [SYS, {'role': 'user', 'content': 'Café naïve résumé — 東京 🦘🌏 ñ Ω ∑ ✓ «quote» é z‍z'}]},
    'thinking': {'messages': [SYS, {'role': 'user', 'content': 'Is 91 prime?'},
                              {'role': 'assistant', 'content': 'No.', 'thinking': '91 = 7 x 13, so not prime.'},
                              {'role': 'user', 'content': 'Is 97 prime?'}], 'think': True},
}
RUNS = [('qwen3:8b', 'rendered'), ('qwen3:8b', 'native'), ('gpt-oss:20b', 'rendered'), ('gemma4:26b', 'rendered')]


def log(msg):
    line = f'{time.strftime("%H:%M:%S")} {msg}'
    print(line, flush=True)
    open(os.path.join(OUT, 'harness.log'), 'a').write(line + '\n')


def http(method, url, body=None, timeout=600):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method, headers={'Content-Type': 'application/json'})
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, r.read().decode('utf-8', 'replace')
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', 'replace')


def start(route, serve_log):
    env = {'PATH': '/usr/bin:/bin:/usr/sbin:/sbin', 'HOME': os.path.join(RT, 'home'),
           'OLLAMA_HOST': f'127.0.0.1:{PORT}', 'OLLAMA_MODELS': os.path.join(RT, 'models'),
           'OLLAMA_NOPRUNE': '1', 'OLLAMA_DEBUG': '1', 'OLLAMA_KEEP_ALIVE': '10m'}
    if route == 'native':
        env['OLLAMA_GO_TEMPLATE'] = 'false'
    p = subprocess.Popen([os.path.join(RT, 'ollama'), 'serve'], cwd=RT, env=env, stdout=open(serve_log, 'ab'),
                         stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL, start_new_session=True)
    for _ in range(120):
        try:
            if http('GET', BASE + '/api/version', timeout=2)[0] == 200:
                return p
        except Exception:
            pass
        if p.poll() is not None:
            raise SystemExit('daemon exited early')
        time.sleep(0.5)
    raise SystemExit('daemon not ready')


def stop(p):
    pgid = os.getpgid(p.pid)
    if pgid == os.getpgid(0):
        raise SystemExit('refusing to signal my own process group')
    os.killpg(pgid, signal.SIGTERM)
    try:
        p.wait(timeout=30)
    except subprocess.TimeoutExpired:
        os.killpg(pgid, signal.SIGKILL); p.wait(timeout=10)
    time.sleep(2)
    left = subprocess.run(['pgrep', '-f', RT + '/'], capture_output=True, text=True).stdout.split()
    for pid in left:
        try: os.kill(int(pid), signal.SIGKILL)
        except ProcessLookupError: pass
    log('teardown: ' + (f'LEFTOVER killed {left}' if left else 'no sandbox process left'))


def runner_port(serve_log):
    ports = re.findall(r'--port[ ",]+(\d+)', open(serve_log, errors='replace').read())
    return int(ports[-1]) if ports else None


def main():
    os.makedirs(OUT, exist_ok=True)
    rows = []
    for model, route in RUNS:
        tag = f'{model.replace(":", "_")}_{route}'
        serve_log = os.path.join(OUT, tag + '.serve.log')
        log(f'{tag}: starting daemon')
        p = start(route, serve_log)
        try:
            for case, spec in CASES.items():
                body = {'model': model, 'messages': spec['messages'], 'stream': False,
                        'options': {'num_ctx': 4096, 'num_predict': 1}}
                if 'tools' in spec: body['tools'] = spec['tools']
                body['think'] = spec.get('think', False)
                st, text = http('POST', BASE + '/api/chat', body)          # real request first: loads the runner
                j = json.loads(text) if text.startswith('{') else {}
                actual = j.get('prompt_eval_count')
                rb = dict(body); rb['_debug_render_only'] = True
                rst, rtext = http('POST', BASE + '/api/chat', rb)
                rendered = (json.loads(rtext).get('_debug_info') or {}).get('rendered_template') if rst == 200 else None
                port = runner_port(serve_log)
                def tok(flags):
                    if not (rendered and port): return None
                    s, b = http('POST', f'http://127.0.0.1:{port}/tokenize', dict({'content': rendered}, **flags))
                    return len(json.loads(b)['tokens']) if s == 200 else None
                go_est, inf = tok({}), tok({'add_special': True, 'parse_special': True})
                row = {'model': model, 'route': route, 'case': case, 'chat_status': st, 'render_status': rst,
                       'actual_prompt_eval_count': actual, 'go_estimate': go_est, 'inference_flags': inf,
                       'gap_actual_minus_go': (actual - go_est) if (actual is not None and go_est is not None) else None,
                       'gap_actual_minus_inference_flags': (actual - inf) if (actual is not None and inf is not None) else None,
                       'error': j.get('error') if st != 200 else None,
                       'rendered_prompt_sha256': __import__('hashlib').sha256(rendered.encode()).hexdigest() if rendered else None}
                rows.append(row)
                open(os.path.join(OUT, f'{tag}.{case}.rendered.txt'), 'w').write(rendered or '')
                log(f'{tag} {case:9s} status={st}/{rst} actual={actual} go={go_est} infflags={inf} '
                    f'gap_go={row["gap_actual_minus_go"]} gap_inf={row["gap_actual_minus_inference_flags"]}'
                    + (f' error={str(row["error"])[:100]}' if row['error'] else ''))
        finally:
            stop(p)
    json.dump(rows, open(os.path.join(OUT, 'results.json'), 'w'), indent=1)
    gaps = [r['gap_actual_minus_go'] for r in rows if r['gap_actual_minus_go'] is not None]
    log(f'done: {len(rows)} rows, {len(gaps)} with gaps; max undercount by the current Go estimate = {max(gaps) if gaps else None}')


main()
