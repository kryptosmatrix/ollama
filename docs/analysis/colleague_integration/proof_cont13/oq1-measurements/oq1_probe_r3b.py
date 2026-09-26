#!/usr/bin/env python3
"""OQ-1 measurement harness, round 3b (continuation 13, Thole): a long reply after a packed history, and a prompt inside the keep-all clamp zone.

Measures, against the UNCHANGED candidate daemon (branch base 04009c43) and its
byte-identical pinned native llama-server (b10091), what happens to a leading
system "carrier" under the existing context-shift and truncation mechanisms, on
both local chat routes:

  native   - OLLAMA_GO_TEMPLATE=false: messages go to llama-server's chat endpoint
  rendered - default: the Go template renders the prompt, llama-server /completion

Probes (each a fresh daemon + runner, so no probe inherits another's runner):
  shift_default_long   long reply with default shift: does the native context
                       shift discard the carrier's token span?
  shift_false_long     the same request with shift=false: no shift, visible stop?
  oversize_default     carrier + user text larger than num_ctx, defaults
  oversize_shift_false the same with shift=false

Isolation: sandbox HOME, OLLAMA_MODELS (APFS clones), port 11500,
OLLAMA_NOPRUNE=1. The daemon runs in its own session; the harness kills that
process group at the end of every arm and then checks the process table for
anything still running from the sandbox directory.

Usage: oq1_probe.py <sandbox_rt_dir> <out_dir> <arm> [<arm> ...]
  arm = native | rendered
"""
import json
import os
import re
import signal
import subprocess
import sys
import time
import urllib.error
import urllib.request

RT, OUT = sys.argv[1], sys.argv[2]
ARMS = sys.argv[3:]
PORT = 11500
BASE = f'http://127.0.0.1:{PORT}'
MODEL = 'qwen3:8b'
NUM_CTX = 512

CARRIER = (
    'User-configured instructions; revision 1; selection 1:\n'
    'CARRIER-SENTINEL-7Q: always finish every reply with the exact token ZEBRA-7Q. '
    'This sentence exists so the carrier spans a measurable run of tokens at the '
    'start of the conversation, after the chat template system header.'
)
LONG_ASK = ('Write a very long, detailed essay about the history of bridges, '
            'with at least forty numbered sections. Do not stop early.')
FILLER = ' '.join(f'filler-{i:04d}' for i in range(700))  # far above 512 tokens


def log(msg):
    line = f'{time.strftime("%H:%M:%S")} {msg}'
    print(line, flush=True)
    with open(os.path.join(OUT, 'harness.log'), 'a') as f:
        f.write(line + '\n')


def http(method, url, body=None, timeout=300):
    data = None if body is None else json.dumps(body).encode()
    req = urllib.request.Request(url, data=data, method=method,
                                 headers={'Content-Type': 'application/json'})
    t0 = time.time()
    try:
        with urllib.request.urlopen(req, timeout=timeout) as r:
            return r.status, r.read().decode('utf-8', 'replace'), time.time() - t0
    except urllib.error.HTTPError as e:
        return e.code, e.read().decode('utf-8', 'replace'), time.time() - t0


def start_daemon(arm, serve_log):
    env = {
        'PATH': '/usr/bin:/bin:/usr/sbin:/sbin',
        'HOME': os.path.join(RT, 'home'),
        'OLLAMA_HOST': f'127.0.0.1:{PORT}',
        'OLLAMA_MODELS': os.path.join(RT, 'models'),
        'OLLAMA_NOPRUNE': '1',
        'OLLAMA_DEBUG': '1',
        'OLLAMA_KEEP_ALIVE': '10m',
    }
    if arm == 'native':
        env['OLLAMA_GO_TEMPLATE'] = 'false'
    f = open(serve_log, 'ab')
    p = subprocess.Popen([os.path.join(RT, 'ollama'), 'serve'], cwd=RT, env=env,
                         stdout=f, stderr=subprocess.STDOUT, stdin=subprocess.DEVNULL,
                         start_new_session=True)
    for _ in range(120):
        try:
            st, _, _ = http('GET', BASE + '/api/version', timeout=2)
            if st == 200:
                return p
        except Exception:
            pass
        if p.poll() is not None:
            raise SystemExit(f'daemon exited early with {p.returncode}')
        time.sleep(0.5)
    raise SystemExit('daemon did not become ready')


def stop_daemon(p):
    pgid = os.getpgid(p.pid)
    if pgid == os.getpgid(0):
        raise SystemExit('refusing to signal my own process group')
    os.killpg(pgid, signal.SIGTERM)
    try:
        p.wait(timeout=30)
    except subprocess.TimeoutExpired:
        os.killpg(pgid, signal.SIGKILL)
        p.wait(timeout=10)
    time.sleep(2)
    left = subprocess.run(['pgrep', '-f', RT + '/'], capture_output=True, text=True).stdout.split()
    if left:
        for pid in left:
            try:
                os.kill(int(pid), signal.SIGKILL)
            except ProcessLookupError:
                pass
        log(f'LEFTOVER sandbox processes killed: {left}')
    else:
        log('teardown: no sandbox process left')


def runner_port(serve_log):
    text = open(serve_log, 'r', errors='replace').read()
    ports = re.findall(r'--port[ ",]+(\d+)', text)
    return int(ports[-1]) if ports else None


def launch_lines(serve_log):
    text = open(serve_log, 'r', errors='replace').read()
    return [l for l in text.splitlines() if 'starting llama-server' in l]


def shift_lines(serve_log, since):
    text = open(serve_log, 'r', errors='replace').read()[since:]
    return [l for l in text.splitlines() if 'context shift' in l or 'truncating input' in l
            or 'exceeds the available context' in l or 'larger than the max context' in l]


def carrier_span(rport, rendered_prompt):
    """Token index span of the carrier inside the rendered prompt, using the
    runner's own tokenizer with the flags the native inference path uses."""
    def tok(content):
        st, body, _ = http('POST', f'http://127.0.0.1:{rport}/tokenize',
                           {'content': content, 'add_special': True, 'parse_special': True}, timeout=60)
        return json.loads(body)['tokens'] if st == 200 else None
    i = rendered_prompt.find('CARRIER-SENTINEL-7Q')
    j = rendered_prompt.find('start of the conversation, after the chat template system header.')
    if i < 0 or j < 0:
        return None
    j += len('start of the conversation, after the chat template system header.')
    before, upto = tok(rendered_prompt[:i]), tok(rendered_prompt[:j])
    whole = tok(rendered_prompt)
    return {'carrier_first_token_index': len(before) if before is not None else None,
            'carrier_end_token_index': len(upto) if upto is not None else None,
            'prompt_tokens_runner_count': len(whole) if whole is not None else None}


def chat(extra, messages, num_predict, timeout=600):
    body = {'model': MODEL, 'messages': messages, 'stream': False, 'think': False,
            'options': {'num_ctx': NUM_CTX, 'num_predict': num_predict}}
    extra = dict(extra)
    body['options'].update(extra.pop('options', {}))
    body.update(extra)
    st, text, dt = http('POST', BASE + '/api/chat', body, timeout=timeout)
    try:
        j = json.loads(text)
    except Exception:
        j = None
    return body, st, text, j, dt


TAIL = ' Finally, end your reply with the exact word LAST-SENTINEL-Q.'


def calibrate_clamp(extra, target=510):
    """Find a filler length whose rendered prompt is exactly `target` runner tokens, so the native
    keep-all shift clamp (n_keep = n_ctx - 4 = 508) falls inside the prompt's own tail."""
    def count(n):
        msgs = [{'role': 'system', 'content': CARRIER},
                {'role': 'user', 'content': 'Repeat nothing. ' + ' '.join('pad' for _ in range(n)) + TAIL}]
        rbody = {'model': MODEL, 'messages': msgs, 'stream': False, 'think': False,
                 '_debug_render_only': True, 'options': {'num_ctx': NUM_CTX, 'num_predict': 1}}
        ex = dict(extra); rbody['options'].update(ex.pop('options', {})); rbody.update(ex)
        st, text, _ = http('POST', BASE + '/api/chat', rbody, timeout=300)
        rendered = json.loads(text).get('_debug_info', {}).get('rendered_template')
        # load the runner once so its tokenizer is reachable
        if not getattr(calibrate_clamp, 'port', None):
            chat(extra, [{'role': 'user', 'content': 'hi'}], 1)
            calibrate_clamp.port = runner_port(calibrate_clamp.log)
        st, body, _ = http('POST', f'http://127.0.0.1:{calibrate_clamp.port}/tokenize',
                           {'content': rendered, 'add_special': True, 'parse_special': True}, timeout=60)
        return len(json.loads(body)['tokens']), msgs
    lo, hi = 1, 600
    best = None
    while lo <= hi:
        mid = (lo + hi) // 2
        c, m = count(mid)
        if c == target:
            best = m
            break
        if c < target:
            lo = mid + 1
        else:
            hi = mid - 1
    if best is None:
        c, best = count(lo)
    log(f'clamp calibration: filler words={lo if best is None else "found"} tokens={count(len(best[1]["content"].split()) - 11)[0] if False else "see result"}')
    return best


def run_arm(arm):
    adir = os.path.join(OUT, arm)
    os.makedirs(adir, exist_ok=True)
    msgs_long = [{'role': 'system', 'content': CARRIER}, {'role': 'user', 'content': LONG_ASK}]
    msgs_over = [{'role': 'system', 'content': CARRIER}, {'role': 'user', 'content': 'Summarise: ' + FILLER}]
    history = []
    for k in range(30):
        history.append({'role': 'user', 'content': f'Note {k:02d}: the harbour crane number {k:02d} was repainted blue and green in the spring maintenance cycle.'})
        history.append({'role': 'assistant', 'content': f'Understood, note {k:02d} is recorded: crane {k:02d} repainted blue and green in spring.'})
    ask = {'role': 'user', 'content': 'Final request: list every crane number mentioned above, one per line, each followed by its two colours.'}
    msgs_hist = [{'role': 'system', 'content': CARRIER}] + history + [ask]
    probes = [
        ('longreply_default', {}, msgs_hist, 300),
        ('longreply_shift_false', {'shift': False}, msgs_hist, 300),
        ('longreply_keep_all', {'options': {'num_keep': -1}}, msgs_hist, 300),
        ('clampedge_keep_all', {'options': {'num_keep': -1}}, 'CLAMP', 120),
    ]
    results = []
    for name, extra, msgs, npred in probes:
        serve_log = os.path.join(adir, f'{name}.serve.log')
        calibrate_clamp.log = serve_log
        calibrate_clamp.port = None
        log(f'[{arm}] {name}: starting fresh daemon')
        p = start_daemon(arm, serve_log)
        try:
            if msgs == 'CLAMP':
                msgs = calibrate_clamp(extra)
            # render-only first (no inference) to locate the carrier in the prompt the route builds
            rbody = {'model': MODEL, 'messages': msgs, 'stream': False, 'think': False,
                     '_debug_render_only': True, 'options': {'num_ctx': NUM_CTX}}
            ex = dict(extra); rbody['options'].update(ex.pop('options', {})); rbody.update(ex)
            rst, rtext, _ = http('POST', BASE + '/api/chat', rbody, timeout=300)
            rendered = None
            try:
                rendered = json.loads(rtext).get('_debug_info', {}).get('rendered_template')
            except Exception:
                pass
            mark = os.path.getsize(serve_log)
            body, st, text, j, dt = chat(extra, msgs, npred)
            rport = runner_port(serve_log)
            props = None
            if rport:
                pst, ptext, _ = http('GET', f'http://127.0.0.1:{rport}/props', timeout=30)
                try:
                    pj = json.loads(ptext)
                    props = {'n_ctx': pj.get('default_generation_settings', {}).get('n_ctx'),
                             'total_slots': pj.get('total_slots')}
                except Exception:
                    props = {'status': pst, 'raw': ptext[:300]}
            span = carrier_span(rport, rendered) if (rport and rendered) else None
            ps_st, ps_text, _ = http('GET', BASE + '/api/ps', timeout=30)
            rec = {
                'arm': arm, 'probe': name, 'request': body,
                'render_status': rst, 'rendered_prompt_chars': len(rendered) if rendered else None,
                'http_status': st, 'seconds': round(dt, 2),
                'done_reason': (j or {}).get('done_reason'),
                'prompt_eval_count': (j or {}).get('prompt_eval_count'),
                'eval_count': (j or {}).get('eval_count'),
                'error': (j or {}).get('error') if j else text[:2000],
                'reply_tail': ((j or {}).get('message') or {}).get('content', '')[-300:],
                'native_log_lines': shift_lines(serve_log, mark),
                'launch_lines': launch_lines(serve_log),
                'runner_props': props,
                'carrier_span': span,
                'api_ps': ps_text[:1500],
            }
            with open(os.path.join(adir, f'{name}.rendered_prompt.txt'), 'w') as f:
                f.write(rendered or '')
            with open(os.path.join(adir, f'{name}.response.txt'), 'w') as f:
                f.write(text)
            with open(os.path.join(adir, f'{name}.result.json'), 'w') as f:
                f.write(json.dumps(rec, indent=2) + '\n')
            results.append(rec)
            log(f'[{arm}] {name}: http={st} done={rec["done_reason"]} prompt={rec["prompt_eval_count"]} '
                f'eval={rec["eval_count"]} shift_lines={len(rec["native_log_lines"])} props={props} span={span}')
        finally:
            stop_daemon(p)
    return results


def reload_sequence(arm):
    adir = os.path.join(OUT, arm)
    os.makedirs(adir, exist_ok=True)
    serve_log = os.path.join(adir, 'reload_sequence.serve.log')
    msgs = [{'role': 'system', 'content': CARRIER}, {'role': 'user', 'content': 'Say hello in five words.'}]
    p = start_daemon(arm, serve_log)
    seq = []
    try:
        for i, extra in enumerate([{}, {'shift': False}, {}, {'shift': False}]):
            body, st, text, j, dt = chat(extra, msgs, 16)
            seq.append({'step': i, 'shift': extra.get('shift', 'default'), 'http_status': st,
                        'seconds': round(dt, 2), 'load_duration_ns': (j or {}).get('load_duration'),
                        'launches_so_far': len(launch_lines(serve_log))})
            log(f'[{arm}] reload step {i} shift={extra.get("shift", "default")} http={st} '
                f'{dt:.2f}s load_ns={(j or {}).get("load_duration")} launches={len(launch_lines(serve_log))}')
    finally:
        stop_daemon(p)
    with open(os.path.join(adir, 'reload_sequence.result.json'), 'w') as f:
        f.write(json.dumps(seq, indent=2) + '\n')
    return seq


if __name__ == '__main__':
    os.makedirs(OUT, exist_ok=True)
    allres = []
    for arm in ARMS:
        allres += run_arm(arm)
    with open(os.path.join(OUT, 'summary.json'), 'w') as f:
        f.write(json.dumps(allres, indent=2) + '\n')
    log(f'done: {len(allres)} probe records')
