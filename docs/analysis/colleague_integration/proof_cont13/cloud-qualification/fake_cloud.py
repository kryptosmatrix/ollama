#!/usr/bin/env python3
"""A local stand-in for /api/chat, used only to prove cloud_check2.py's parsing and verdicts before any real request.

Usage: fake_cloud.py <port> <mode>
Modes: echo (correct reply), drop_start (carrier and first value wrong, last right), drop_middle (middle value wrong),
       tool_call (a tool call, no text),
       error (HTTP 500), garbage (every value wrong).
"""
import json, math, re, sys
from http.server import BaseHTTPRequestHandler, ThreadingHTTPServer

PORT, MODE = int(sys.argv[1]), sys.argv[2]


def answer(body):
    msgs = body.get('messages') or []
    system = next((m.get('content') or '' for m in msgs if m.get('role') == 'system'), '')
    code = (re.search(r'Carrier code: ([0-9A-F]{8})', system) or [None, '00000000'])[1]
    last_user = next((m.get('content') or '' for m in reversed(msgs) if m.get('role') == 'user'), '')
    lines = re.findall(r'Line (\d{6}): value (\d{6})\.', last_user)
    first = lines[0][1] if lines else None
    last = lines[-1][1] if lines else None
    want_mid = re.search(r'MID=<the value on line (\d{6})>', last_user)
    mid = dict(lines).get(want_mid.group(1)) if want_mid else None
    if MODE == 'drop_middle' and mid is not None:
        mid = '333333'
    if MODE == 'drop_start':
        code, first = 'DEADBEEF', ('111111' if first else None)
    if MODE == 'garbage':
        code, first, last = 'DEADBEEF', ('111111' if first else None), ('222222' if last else None)
        mid = '333333' if mid is not None else None
    parts = [f'CARRIER={code}']
    if first is not None:
        parts += [f'FIRST={first}'] + ([f'MID={mid}'] if mid is not None else []) + [f'LAST={last}']
    return '; '.join(parts)


class H(BaseHTTPRequestHandler):
    def log_message(self, *a):
        pass

    def do_POST(self):
        n = int(self.headers.get('Content-Length') or 0)
        raw = self.rfile.read(n)
        body = json.loads(raw)
        if MODE == 'error':
            out = json.dumps({'error': 'fake failure'}).encode()
            self.send_response(500); self.send_header('Content-Type', 'application/json'); self.end_headers()
            self.wfile.write(out); return
        pec = math.ceil(len(raw) / 3)  # a stand-in count below the byte bound
        text = '' if MODE == 'tool_call' else answer(body)
        calls = [{'function': {'name': 'get_weather', 'arguments': {'city': 'Paris'}}}] if MODE == 'tool_call' else []
        final = {'model': body.get('model'), 'done': True, 'done_reason': 'stop', 'prompt_eval_count': pec,
                 'eval_count': max(1, len(text) // 4)}
        if body.get('stream') is False:
            rec = dict(final); rec['message'] = {'role': 'assistant', 'content': text, 'tool_calls': calls}
            out = json.dumps(rec).encode()
            self.send_response(200); self.send_header('Content-Type', 'application/json'); self.end_headers()
            self.wfile.write(out); return
        self.send_response(200); self.send_header('Content-Type', 'application/x-ndjson'); self.end_headers()
        recs = []
        if body.get('think'):
            recs.append({'message': {'role': 'assistant', 'content': '', 'thinking': 'Looking at the lines.'}, 'done': False})
        for i in range(0, len(text), 7):
            recs.append({'message': {'role': 'assistant', 'content': text[i:i + 7]}, 'done': False})
        if calls:
            recs.append({'message': {'role': 'assistant', 'content': '', 'tool_calls': calls}, 'done': False})
        last = dict(final); last['message'] = {'role': 'assistant', 'content': ''}
        recs.append(last)
        for r in recs:
            self.wfile.write(json.dumps(r).encode() + b'\n')


ThreadingHTTPServer(('127.0.0.1', PORT), H).serve_forever()
