#!/usr/bin/env python3
"""Bounded executions of the repository's real frontend runner; retain every arm."""
from __future__ import annotations
import argparse
import hashlib
import json
import os
from pathlib import Path
import re
import signal
import subprocess
import time

ROOT = Path('/Users/krypto/GitHub/ollama-eko-chat-read-integrity')
UI = ROOT / 'app/ui/app'
EVIDENCE = Path(__file__).resolve().parent
BASE = 'd559da25a4db89fa9369d8be4106100803e82a58'
FOCUS = ['src/components/StreamingMarkdownContent.integration.test.tsx',
         'src/utils/remarkCitationParser.test.ts']
EXPECTED = {
    'renders distinct upstream Markdown rather than fixed or empty output',
    'renders the input code through the loaded syntax highlighter',
    'uses the citation cursor to select the correct upstream page',
    'does not invent a link for a missing citation page',
    'does not turn model-provided images or raw HTML into embedded resources',
    'carries the exact range and surrounding text into the citation node',
    'preserves the distinct generic citation payload',
    'coalesces adjacent equal cursors without dropping a different cursor',
    'does not coalesce citations separated by ordinary text',
    'leaves code and malformed delimiters alone',
    'preserves an empty document without manufacturing a citation',
}
CASES = {'candidate', 'full', 'no-output', 'no-citations', 'wrong-page'}


def digest(path: Path) -> str:
    return hashlib.sha256(path.read_bytes()).hexdigest()


def save(path: Path, value: object) -> None:
    with path.open('x', encoding='utf-8') as f:
        json.dump(value, f, indent=2, ensure_ascii=False)
        f.write('\n')


def source_files() -> list[str]:
    output = subprocess.check_output(['git', 'ls-files', '-z', 'app/ui/app'], cwd=ROOT)
    return [s for s in output.decode().split('\0') if s]


def fingerprints() -> dict[str, str]:
    return {s: digest(ROOT / s) for s in source_files()}


def execute(argv: list[str], cwd: Path, target: Path, label: str,
            env: dict[str, str], bound: int) -> dict:
    started = time.monotonic()
    timed_out = False
    stdout = target / (label + '.stdout')
    stderr = target / (label + '.stderr')
    with stdout.open('xb') as out, stderr.open('xb') as err:
        proc = subprocess.Popen(argv, cwd=cwd, env=env, stdout=out, stderr=err,
                                start_new_session=True)
        try:
            code = proc.wait(timeout=bound)
        except subprocess.TimeoutExpired:
            timed_out = True
            os.killpg(proc.pid, signal.SIGTERM)
            try:
                code = proc.wait(timeout=5)
            except subprocess.TimeoutExpired:
                os.killpg(proc.pid, signal.SIGKILL)
                code = proc.wait()
    receipt = {'argv': argv, 'cwd': str(cwd), 'exit_code': code,
               'timed_out': timed_out, 'seconds': time.monotonic() - started,
               'stdout': str(stdout), 'stderr': str(stderr),
               'stdout_sha256': digest(stdout), 'stderr_sha256': digest(stderr)}
    save(target / (label + '.json'), receipt)
    return receipt


def apply_fault(path: Path, case: str) -> dict:
    old = path.read_text()
    changed = old
    edits: list[tuple[str, str]]
    if case == 'no-output':
        edits = [('{content}\n          </Streamdown>', '{""}\n          </Streamdown>')]
        expected = 'renders distinct upstream Markdown rather than fixed or empty output'
    elif case == 'no-citations':
        edits = [('import remarkCitationParser from "@/utils/remarkCitationParser";\n', ''),
                 ('        remarkCitationParser,\n', '')]
        expected = 'uses the citation cursor to select the correct upstream page'
    elif case == 'wrong-page':
        edits = [('pageStack[cursor]', 'pageStack[0]')]
        expected = 'uses the citation cursor to select the correct upstream page'
    else:
        raise ValueError('Unknown fault')
    for a, b in edits:
        if changed.count(a) != 1:
            raise RuntimeError('Fault anchor does not match exactly once')
        changed = changed.replace(a, b, 1)
    path.write_text(changed)
    return {'case': case, 'file': str(path), 'edits': edits,
            'before_sha256': hashlib.sha256(old.encode()).hexdigest(),
            'after_sha256': digest(path), 'required_failure_title': expected}


def run(case: str, label: str) -> dict:
    if case not in CASES or not re.fullmatch(r'[A-Za-z0-9_-]{1,70}', label):
        raise ValueError('Invalid case or label')
    target = EVIDENCE / 'runs' / label
    target.mkdir(parents=True, exist_ok=False)
    head = subprocess.check_output(['git', 'rev-parse', 'HEAD'], cwd=ROOT, text=True).strip()
    before = fingerprints()
    save(target / 'source-before.json', before)
    cwd = UI
    fault = None
    if case not in {'candidate', 'full'}:
        cwd = target / 'frontend'
        cwd.mkdir()
        for rel in source_files():
            p = ROOT / rel
            tail = p.relative_to(UI)
            if p.is_symlink() or '.env' in tail.name or p.suffix in {'.pem', '.key', '.crt'}:
                raise RuntimeError('Unexpected sensitive/symlink source in copy scope')
            dest = cwd / tail
            dest.parent.mkdir(parents=True, exist_ok=True)
            dest.write_bytes(p.read_bytes())
        (cwd / 'node_modules').symlink_to(UI / 'node_modules', target_is_directory=True)
        fault = apply_fault(cwd / 'src/components/StreamingMarkdownContent.tsx', case)
        save(target / 'fault.json', fault)
    env = dict(os.environ)
    env.update(PATH='/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin', CI='true')
    invocation = {'case': case, 'head': head, 'baseline': BASE,
                  'controller_sha256': digest(Path(__file__)),
                  'environment_overrides': {'PATH': env['PATH'], 'CI': env['CI']},
                  'scope': 'Real Node SSR renderer, Shiki, remark; not browser styling, app launch or live services',
                  'fault': fault}
    save(target / 'invocation.json', invocation)
    # All arms must compile before a failed assertion can count as fault detection.
    compile_result = execute(['/opt/homebrew/bin/npm', 'exec', '--no', '--',
                              'tsc', '-b', '--force', '--pretty', 'false'], cwd, target,
                             'typecheck', env, 150)
    args = ['/opt/homebrew/bin/npm', 'test', '--', '--run']
    if case != 'full':
        args.extend(FOCUS)
    args += ['--reporter=json']
    test_result = execute(args, cwd, target, 'tests', env, 240)
    raw = Path(test_result['stdout']).read_text()
    try:
        report = json.loads(raw[raw.index('{'):])
    except (ValueError, json.JSONDecodeError):
        report = {}
    files = report.get('testResults', [])
    assertions = [a for f in files for a in f.get('assertionResults', [])]
    titles = {a.get('title') for a in assertions}
    count = 154 if case == 'full' else 11
    failures = [a for a in assertions if a.get('status') == 'failed']
    statuses = {k: sum(a.get('status') == k for a in assertions)
                for k in ['passed', 'failed', 'pending', 'skipped', 'todo']}
    collected = (len(assertions) == count and report.get('numTotalTests') == count
                 and EXPECTED <= titles and all(f.get('assertionResults') for f in files))
    completed = all(a.get('status') in {'passed', 'failed'} for a in assertions)
    clean = (report.get('success') is True and not failures
             and all(f.get('status') == 'passed' for f in files)
             and test_result['exit_code'] == 0)
    detected = (fault is not None and test_result['exit_code'] == 1
                and any(a.get('title') == fault['required_failure_title']
                        and a.get('failureMessages') for a in failures))
    unchanged = before == fingerprints()
    met = (collected and completed and unchanged and compile_result['exit_code'] == 0
           and not compile_result['timed_out'] and not test_result['timed_out']
           and (clean if fault is None else detected))
    result = {'case': case, 'head': head, 'expectation_met': met,
              'compile': compile_result, 'tests': test_result,
              'expected_count': count, 'collected_assertions': len(assertions),
              'required_titles_discovered': EXPECTED <= titles,
              'counts': statuses, 'complete': completed,
              'original_source_unchanged': unchanged,
              'clean_tests_passed': clean, 'intended_fault_detected': detected,
              'failing_assertions': [{'title': a.get('title'), 'messages': a.get('failureMessages')}
                                     for a in failures],
              'file_load_errors': [{'name': f.get('name'), 'message': f.get('message')}
                                   for f in files if not f.get('assertionResults')],
              'result_path': str(target / 'result.json')}
    save(target / 'result.json', result)
    return result


if __name__ == '__main__':
    parser = argparse.ArgumentParser()
    parser.add_argument('--case', choices=sorted(CASES), required=True)
    parser.add_argument('--label', required=True)
    args = parser.parse_args()
    result = run(args.case, args.label)
    print(json.dumps(result, indent=2))
    raise SystemExit(0 if result['expectation_met'] else 1)
