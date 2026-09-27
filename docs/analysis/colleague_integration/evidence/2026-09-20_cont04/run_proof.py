#!/usr/bin/env python3
"""Retain exact Go-test results for the isolated chat-read repair.

No production database or installed application is opened. Each invocation
uses a new home; inference is simulated by the tests themselves.
"""
from __future__ import annotations

import argparse
import hashlib
import json
import os
from pathlib import Path
import signal
import subprocess
import sys
import tempfile
import time

ROOT = Path('/Users/krypto/GitHub/ollama-eko-chat-read-integrity')
EVIDENCE = Path(__file__).resolve().parent
GO = '/opt/homebrew/bin/go'
REQUIRED = {
    'TestChatReadMissingHeaderIsNotFound',
    'TestChatReadValidEmptyAndPopulated',
    'TestChatReadRejectsMalformedMessage',
    'TestChatReadInitialisationFailureIsNotAbsence',
    'TestChatReadPreservesDatabaseErrorCause',
    'TestChatReadIntegrityAcrossProcesses',
    'TestChatReadIntegrityAcrossProcesses/seed',
    'TestChatReadIntegrityAcrossProcesses/fault',
    'TestChatReadIntegrityAcrossProcesses/recover',
    'TestChatReadIntegrityAcrossProcesses/positive',
}
SOURCES = ['app/store/database.go', 'app/store/store.go', 'app/ui/ui.go',
           'app/store/chat_read_errors_test.go', 'app/ui/chat_read_errors_test.go',
           'app/ui/tts_test.go', 'go.mod', 'go.sum']


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def write_json(path: Path, value: object) -> None:
    with path.open('x', encoding='utf-8') as output:
        json.dump(value, output, indent=2, sort_keys=True)
        output.write('\n')


def execute(args: list[str], cwd: Path, env: dict[str, str], target: Path,
            timeout: int) -> dict:
    started = time.monotonic()
    timed_out = False
    with (target / 'stdout.jsonl').open('xb') as out, (target / 'stderr.txt').open('xb') as err:
        process = subprocess.Popen(args, cwd=cwd, env=env, stdout=out, stderr=err,
                                   start_new_session=True)
        try:
            code = process.wait(timeout=timeout)
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(process.pid, signal.SIGTERM)
            try:
                code = process.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(process.pid, signal.SIGKILL)
                code = process.wait()
    return {'argv': args, 'cwd': str(cwd), 'exit_code': code,
            'timed_out': timed_out, 'elapsed_seconds': time.monotonic() - started,
            'stdout_sha256': digest(target / 'stdout.jsonl'),
            'stderr_sha256': digest(target / 'stderr.txt')}


def analyse(path: Path) -> dict:
    events = [json.loads(line) for line in path.read_text().splitlines() if line.strip()]
    started = {event['Test'] for event in events
               if event.get('Action') == 'run' and event.get('Test')}
    final = {event['Test']: event['Action'] for event in events
             if event.get('Test') and event.get('Action') in {'pass', 'fail', 'skip'}}
    builds_failed = [event for event in events
                     if event.get('Action') == 'build-fail' or event.get('FailedBuild')]
    outputs = ''.join(event.get('Output', '') for event in events)
    return {'started': sorted(started), 'finished': final,
            'missing': sorted(REQUIRED - final.keys()),
            'unfinished': sorted(started - final.keys()),
            'counts': {status: list(final.values()).count(status)
                       for status in ('pass', 'fail', 'skip')},
            'build_failures': builds_failed, 'output': outputs}


def main() -> int:
    parser = argparse.ArgumentParser()
    parser.add_argument('--label', required=True)
    parser.add_argument('--expect', choices=['pass', 'baseline-failure'], required=True)
    parser.add_argument('--overlay', type=Path)
    parser.add_argument('--failure-test')
    parser.add_argument('--failure-text')
    options = parser.parse_args()
    if not options.label.replace('-', '').replace('_', '').isalnum():
        parser.error('label must be alphanumeric with optional hyphens/underscores')
    target = EVIDENCE / options.label
    target.mkdir(mode=0o700, exist_ok=False)
    cache = json.loads(subprocess.check_output(
        [GO, 'env', '-json', 'GOCACHE', 'GOMODCACHE', 'GOPATH'], cwd=ROOT))
    before = {name: digest(ROOT / name) for name in SOURCES}
    write_json(target / 'source-before.json', before)
    with tempfile.TemporaryDirectory(prefix='eko-chat-read-proof-') as home:
        env = dict(os.environ)
        env.update(cache)
        env.update({'HOME': home, 'USERPROFILE': home, 'LOCALAPPDATA': home,
                    'OLLAMA_HOST': 'http://127.0.0.1:1', 'GOTOOLCHAIN': 'local',
                    'GOPROXY': 'off', 'GOSUMDB': 'off', 'CI': 'true',
                    'OLLAMA_MCP_CONFIG': str(Path(home) / 'mcp.json'),
                    'OLLAMA_MCP_APPROVALS': str(Path(home) / 'approvals.json'),
                    'OLLAMA_MCP_TOKENS': str(Path(home) / 'mcp-tokens.json')})
        args = [GO, 'test', '-count=1', '-json', '-timeout=150s', '-run=^TestChatRead']
        if options.overlay:
            args += ['-overlay', str(options.overlay.resolve())]
        args += ['./app/store', './app/ui']
        result = execute(args, ROOT, env, target, 240)
        result['environment_overrides'] = {key: env[key] for key in
            ('HOME', 'USERPROFILE', 'LOCALAPPDATA', 'OLLAMA_HOST', 'GOTOOLCHAIN',
             'GOPROXY', 'GOSUMDB', 'CI', 'OLLAMA_MCP_CONFIG',
             'OLLAMA_MCP_APPROVALS', 'OLLAMA_MCP_TOKENS')}
        result['cache_locations'] = cache
    result['source_head'] = subprocess.check_output(
        ['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    result['runner_sha256'] = digest(Path(__file__).resolve())
    result['overlay'] = None
    if options.overlay:
        replacements = json.loads(options.overlay.read_text())['Replace']
        result['overlay'] = {'file': str(options.overlay.resolve()),
            'sha256': digest(options.overlay),
            'replacements': {key: {'path': value, 'sha256': digest(Path(value))}
                             for key, value in replacements.items()}}
    result['source_unchanged_during_run'] = before == {name: digest(ROOT / name) for name in SOURCES}
    analysis = analyse(target / 'stdout.jsonl')
    result['test_results'] = {key: value for key, value in analysis.items() if key != 'output'}
    healthy_instrument = (not result['timed_out'] and not analysis['missing']
        and not analysis['unfinished'] and not analysis['build_failures']
        and analysis['counts']['skip'] == 0 and result['source_unchanged_during_run'])
    if options.expect == 'pass':
        accepted = healthy_instrument and result['exit_code'] == 0 and analysis['counts']['fail'] == 0
    else:
        test = options.failure_test or 'TestChatReadIntegrityAcrossProcesses/fault'
        text = options.failure_text or 'persisted records changed after failed read'
        accepted = (healthy_instrument and result['exit_code'] != 0
                    and analysis['finished'].get(test) == 'fail' and text in analysis['output'])
    result['expectation'] = options.expect
    result['expectation_met'] = accepted
    write_json(target / 'result.json', result)
    print(json.dumps(result, indent=2), flush=True)
    print(analysis['output'], flush=True)
    print('EKO_PROOF_EXPECTATION=' + ('PASS' if accepted else 'FAIL'), flush=True)
    return 0 if accepted else 1


if __name__ == '__main__':
    sys.exit(main())
