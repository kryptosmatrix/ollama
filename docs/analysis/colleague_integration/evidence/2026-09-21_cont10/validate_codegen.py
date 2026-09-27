#!/usr/bin/env python3
"""Candidate-bound extension of cont08's retained real-runner instrument."""
from __future__ import annotations
import copy
import hashlib
import json
import os
from pathlib import Path
import re
import runpy
import shutil
import subprocess
import sys

ROOT = Path('/Users/krypto/GitHub/ollama-eko-chat-read-integrity')
UI = ROOT / 'app/ui/app'
EVIDENCE = Path(__file__).resolve().parent
BASE = EVIDENCE.parent / '2026-09-21_cont08/validate_frontend.py'
if hashlib.sha256(BASE.read_bytes()).hexdigest() != '76f9e131b609191188acbd20af9397577cf24e61a9e55660323ea326975d0ae6':
    raise RuntimeError('Previously inspected execution transport changed')
prior = runpy.run_path(str(BASE))
execute, save, digest = prior['execute'], prior['save'], prior['digest']
HEAD = '6a3778184d43b23c5cba14e0379922377f615f52'
FOCUS = ['src/utils/generatedModels.test.tsx']
EXPECTED = {
'materialises nested chat models through the real API consumer',
'delivers distinct decoded content to the real Markdown renderer',
'keeps string and object constructor inputs equivalent without mutating their values',
'preserves optional nested values without inventing arrays or objects',
'performs declared Date transforms on real generated classes',
'preserves primitive and structured JSON payloads',
'retains nested Settings construction and false or zero values',
'retains map conversion and its existing reference behaviour',
'does not swallow malformed JSON constructor errors',
'ships generated declarations without explicit-any syntax',
}
CASES = {'candidate', 'full', 'raw-generator', 'no-messages', 'fixed-content', 'generation', 'go-store', 'gate-self-checks', 'build', 'normal-entry'}
ENV = dict(os.environ, PATH='/opt/homebrew/bin:/usr/bin:/bin:/usr/sbin:/sbin', CI='true')


def source_files():
    raw = subprocess.check_output(['git', 'ls-files', '-z', 'app/ui/app', 'app/store', 'app/ui/responses', 'app/ui/ui.go', 'go.mod', 'go.sum'], cwd=ROOT)
    return [s for s in raw.decode().split('\0') if s]


def fingerprints():
    result = {s: digest(ROOT / s) for s in source_files()}
    for p in [UI/'node_modules/.package-lock.json', UI/'node_modules/typescript/lib/typescript.js', Path('/opt/homebrew/bin/node'), Path('/opt/homebrew/bin/go')]:
        result[str(p)] = digest(p)
    return result


def dependency_manifest():
    from concurrent.futures import ThreadPoolExecutor
    paths=[]
    for directory, dirs, files in os.walk(UI/'node_modules', followlinks=False):
        dirs[:] = sorted(d for d in dirs if d not in {'.tmp', '.vite', '.vite-temp', '.cache'})
        paths.extend(Path(directory)/name for name in sorted(files))
    def entry(p):
        return (str(p.relative_to(UI)), ('symlink:' + os.readlink(p)) if p.is_symlink() else digest(p))
    with ThreadPoolExecutor(max_workers=8) as pool:
        return dict(pool.map(entry, paths))


def frozen():
    return json.loads((EVIDENCE/'frozen.json').read_text())


def assert_frozen():
    identity = frozen()
    if subprocess.check_output(['git','rev-parse','HEAD'],cwd=ROOT,text=True).strip()!=HEAD or fingerprints()!=identity['source']:
        raise RuntimeError('Candidate/source changed; previous evidence invalidated')
    if dependency_manifest()!=identity['dependencies']:
        raise RuntimeError('Installed dependency bytes changed')
    return identity


def evaluate(report, exit_code, expected_count, fault_title=None):
    files=report.get('testResults', [])
    assertions=[a for f in files for a in f.get('assertionResults', [])]
    titles={a.get('title') for a in assertions}
    counts={k:sum(a.get('status')==k for a in assertions) for k in ['passed','failed','pending','skipped','todo']}
    collected=(len(assertions)==expected_count and report.get('numTotalTests')==expected_count and EXPECTED<=titles
               and bool(files) and all(f.get('assertionResults') for f in files))
    completed=(len(assertions)>0 and counts['passed']+counts['failed']==len(assertions)
               and report.get('numPassedTests')==counts['passed'] and report.get('numFailedTests')==counts['failed']
               and report.get('numPendingTests')==0)
    failures=[{'title':a.get('title'),'messages':a.get('failureMessages',[])} for a in assertions if a.get('status')=='failed']
    intended=(exit_code==1 and any(a['title']==fault_title and any('AssertionError:' in m for m in a['messages']) for a in failures))
    clean=exit_code==0 and report.get('success') is True and counts['failed']==0 and all(f.get('status')=='passed' for f in files)
    return {'expectation_met':bool(collected and completed and (intended if fault_title else clean)),
            'collected':collected,'completed':completed,'counts':counts,'required_titles_discovered':EXPECTED<=titles,
            'intended_fault_detected':intended,'clean_tests_passed':clean,'failing_assertions':failures,
            'file_load_errors':[{'name':f.get('name'),'message':f.get('message')} for f in files if not f.get('assertionResults')]}


def load_vitest(path):
    raw=Path(path).read_text()
    return json.loads(raw[raw.index('{"numTotalTestSuites"'):])


def lint_obligations(record):
    def rows(path):
        text=Path(path).read_text();return json.loads(text[text.index('[{'):])
    baseline=rows(EVIDENCE.parent/'2026-09-21_cont09/baseline/lint.stdout')
    current=rows(record['stdout'])
    def messages(data):
        return sorted((f['filePath'].split('/app/ui/app/',1)[-1],m['ruleId'],m['severity'],m['message'],m.get('line'),m.get('column'))
                      for f in data if '/codegen/' not in f['filePath'] for m in f['messages'])
    generated=[m for f in current if '/codegen/' in f['filePath'] for m in f['messages']]
    return {'bounded_delta_ok':messages(baseline)==messages(current) and not generated and record['exit_code']==1,
            'errors':sum(f['errorCount'] for f in current),'warnings':sum(f['warningCount'] for f in current),
            'generated_findings':generated,'whole_lint_pass':False}


def run(case, label):
    if case not in CASES or not re.fullmatch(r'[A-Za-z0-9_-]{1,70}', label): raise ValueError('Invalid case/label')
    identity=assert_frozen()
    target=EVIDENCE/'runs'/label;target.mkdir(parents=True,exist_ok=False)
    save(target/'source-before.json', {'head':HEAD,'frozen_sha256':digest(EVIDENCE/'frozen.json'),'runner_sha256':digest(Path(__file__))})
    def exe(args, cwd, name, bound=300): return execute(args,cwd,target,name,ENV,bound)
    result={'case':case,'head':HEAD,'expectation_met':False,'result_path':str(target/'result.json')}
    if case=='generation':
        check=exe(['/opt/homebrew/bin/npm','run','check:types'],UI,'check-types')
        tests=exe(['/opt/homebrew/bin/npm','run','test:codegen'],UI,'generator-tests')
        text=Path(tests['stdout']).read_text()
        counts={k:int(v) for k,v in re.findall(r'^# (tests|pass|fail|cancelled|skipped|todo) (\d+)$',text,re.M)}
        result.update(tests=tests,check_types=check,counts=counts,
            expectation_met=check['exit_code']==0 and tests['exit_code']==0 and counts=={'tests':22,'pass':22,'fail':0,'cancelled':0,'skipped':0,'todo':0})
    elif case=='normal-entry':
        tests=exe(['/opt/homebrew/bin/go','generate','-run','^//go:generate node ','./app/ui'],ROOT,'go-generate')
        text=Path(tests['stdout']).read_text()
        receipt=json.loads(text) if tests['exit_code']==0 and text.strip() else {}
        result.update(tests=tests,receipt=receipt,
          expectation_met=tests['exit_code']==0 and receipt.get('mode')=='write' and receipt.get('output')==str(UI/'codegen/gotypes.gen.ts')
            and receipt.get('output_sha256')==digest(UI/'codegen/gotypes.gen.ts') and receipt.get('classes')==35
            and receipt.get('helpers')==16 and receipt.get('emission',{}).get('identical') is True
            and len(receipt.get('upstream_runs',[]))==2 and all(x.get('exit_code')==0 for x in receipt.get('upstream_runs',[])))
    elif case=='go-store':
        tests=exe(['/opt/homebrew/bin/go','test','-count=1','-json','./app/store','./app/ui/responses'],ROOT,'go-tests')
        rows=[json.loads(l) for l in Path(tests['stdout']).read_text().splitlines() if l.startswith('{')]
        started={row['Test'] for row in rows if row.get('Test') and row['Action']=='run'}
        passed={row['Test'] for row in rows if row.get('Test') and row['Action']=='pass'}
        bad=[row for row in rows if row['Action'] in {'fail','skip'} and row.get('Test')]
        package_pass=any(row.get('Package')=='github.com/ollama/ollama/app/store' and not row.get('Test') and row['Action']=='pass' for row in rows)
        result.update(tests=tests,counts={'started':len(started),'passed':len(passed),'top_level':sum('/' not in s for s in started),'bad':len(bad)},
          expectation_met=tests['exit_code']==0 and package_pass and started==passed and not bad and sum('/' not in s for s in started)==17)
    elif case=='build':
        tests=exe(['/opt/homebrew/bin/npm','run','build'],UI,'build')
        result.update(tests=tests,expectation_met=tests['exit_code']==0,warning_limit='Existing dependency and build warnings retained; not a clean whole-programme gate')
    elif case=='gate-self-checks':
        original=load_vitest(EVIDENCE/'validation-first/frontend.stdout')
        positive=evaluate(original,0,164)['expectation_met']
        child=exe([sys.executable,'-c','raise SystemExit(7)'],UI,'failing-child')
        missing=copy.deepcopy(original);missing['testResults'][0]['assertionResults'].pop()
        empty={'numTotalTests':0,'numPassedTests':0,'numFailedTests':0,'numPendingTests':0,'success':True,'testResults':[]}
        controls={'clean_positive':positive,'failing_child_blocks':not evaluate(original,child['exit_code'],164)['expectation_met'],
                  'missing_case_blocks':not evaluate(missing,0,164)['expectation_met'],'empty_selection_blocks':not evaluate(empty,0,0)['expectation_met']}
        result.update(tests=child,controls=controls,expectation_met=all(controls.values()))
    else:
        cwd=UI;fault_title=None;fault=None
        if case not in {'candidate','full'}:
            cwd=target/'frontend';cwd.mkdir()
            for rel in source_files():
                p=ROOT/rel
                if not p.is_relative_to(UI):continue
                tail=p.relative_to(UI)
                if p.is_symlink() or '.env' in tail.name or p.suffix in {'.pem','.key','.crt'}:raise RuntimeError('Unsafe copy input')
                dest=cwd/tail;dest.parent.mkdir(parents=True,exist_ok=True);dest.write_bytes(p.read_bytes())
            (cwd/'node_modules').symlink_to(UI/'node_modules',target_is_directory=True)
            p=cwd/'codegen/gotypes.gen.ts';before=p.read_text();changed=before
            if case=='raw-generator':
                changed=(EVIDENCE/'raw-generator.ts').read_text()
                fault_title='ships generated declarations without explicit-any syntax';edits=[{'whole_file_fixture':'raw-generator.ts','sha256':digest(EVIDENCE/'raw-generator.ts')}]
            else:
                if case=='no-messages':
                    old='this.messages = this.convertValues((source as Record<string, unknown>)["messages"], Message) as Chat["messages"];'
                    new='this.messages = [] as Chat["messages"];'
                    fault_title='materialises nested chat models through the real API consumer'
                else:
                    old='this.content = (source as Record<string, unknown>)["content"] as Message["content"];'
                    new='this.content = "constant fault" as Message["content"];'
                    fault_title='delivers distinct decoded content to the real Markdown renderer'
                if changed.count(old)!=1:raise RuntimeError('Fault anchor count mismatch')
                changed=changed.replace(old,new);edits=[{'old':old,'new':new}]
            if changed==before:raise RuntimeError('Inactive fault')
            p.write_text(changed)
            fault={'case':case,'path':'codegen/gotypes.gen.ts','edits':edits,'before_sha256':hashlib.sha256(before.encode()).hexdigest(),
                   'after_sha256':digest(p),'required_failure_title':fault_title}
            save(target/'fault.json',fault)
        compile_result=exe(['/opt/homebrew/bin/npm','exec','--no','--','tsc','-b','--force','--pretty','false'],cwd,'typecheck')
        args=['/opt/homebrew/bin/npm','test','--','--run']+([] if case=='full' else FOCUS)+['--reporter=json']
        tests=exe(args,cwd,'tests')
        observed=evaluate(load_vitest(tests['stdout']),tests['exit_code'],164 if case=='full' else 10,fault_title)
        result.update(observed,compile=compile_result,tests=tests,fault=fault)
        result['expectation_met'] &= compile_result['exit_code']==0 and not compile_result['timed_out'] and not tests['timed_out']
        if case=='full':
            lint=exe(['/opt/homebrew/bin/npm','run','lint','--','--format=json'],UI,'lint')
            delta=lint_obligations(lint);result.update(lint=lint,lint_delta=delta);result['expectation_met'] &= delta['bounded_delta_ok']
        if case=='raw-generator':
            lint=exe(['/opt/homebrew/bin/npm','exec','--no','--','eslint','codegen/gotypes.gen.ts','--format=json'],cwd,'raw-lint')
            raw_lint=Path(lint['stdout']).read_text();lint_files=json.loads(raw_lint[raw_lint.index('[{'):])
            messages=[m for f in lint_files for m in f['messages']]
            rejected=lint['exit_code']==1 and bool(messages) and all(m['ruleId']=='@typescript-eslint/no-explicit-any' and m['severity']==2 for m in messages)
            result.update(raw_lint=lint,raw_lint_findings=len(messages),raw_lint_rejected=rejected)
            result['expectation_met'] &= rejected
        if fault:
            result['fault_bytes_unchanged_through_execution']=digest(cwd/'codegen/gotypes.gen.ts')==fault['after_sha256']
            result['expectation_met'] &= result['fault_bytes_unchanged_through_execution']
            # This directory is an exact owned copy of committed files plus the retained fault.
            # No source, private data or unique result lives here; logs and patch remain outside it.
            shutil.rmtree(cwd)
            result['isolated_copy_retired']=not cwd.exists()
    assert_frozen()
    result['source_and_dependencies_unchanged']=True
    save(target/'result.json',result)
    return result

if __name__=='__main__':
    import argparse
    p=argparse.ArgumentParser();p.add_argument('--case',choices=sorted(CASES),required=True);p.add_argument('--label',required=True)
    a=p.parse_args();r=run(a.case,a.label);print(json.dumps(r,indent=2));raise SystemExit(0 if r['expectation_met'] else 1)
