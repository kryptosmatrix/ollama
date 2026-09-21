#!/usr/bin/env python3
"""Bounded finalisation of the same saved review; no test or source evidence is reset."""
from pathlib import Path
import json
import re
import runpy

BASE=Path(__file__).resolve().parent
controller=runpy.run_path(str(BASE/'review_codegen.py'))
runner=runpy.run_path(str(BASE/'validate_codegen.py'))
sha,save,http=controller['sha'],controller['save'],controller['http']
MODEL,MANIFEST=controller['MODEL'],controller['MANIFEST']
ALLOWED=controller['ALLOWED']
REQUIRED={'contract','annotator','emission','generator','generator_tests','consumer_tests','go_responses','go_store','entry','api','renderer','runner','executor','package','tsconfig','vitest_config','self_audit','selector_correction'}

def main():
 target=BASE/'independent-review-final';target.mkdir(exist_ok=False)
 recovered=json.loads((BASE/'review-finalisation-context.json').read_text())
 messages=recovered['messages'];runs=recovered['runs'];reads=set(recovered['reads']);served=recovered['served']
 original=recovered['source_identity'];runner['assert_frozen']()
 if original['source']!=runner['fingerprints']() or original['instruments']!={k:sha(p) for k,p in ALLOWED.items()}:
  raise RuntimeError('Source/instrument changed since the executed review; do not reuse it')
 finaliser_sha=sha(Path(__file__))
 tags=[m for m in http('/api/tags').get('models',[]) if m.get('name')==MODEL]
 if len(tags)!=1 or tags[0].get('digest')!=MANIFEST:raise RuntimeError('Selected reviewer manifest changed')
 save(target/'model-observation.json',tags[0])
 tools=json.loads((BASE/'independent-review-r2/initial-request.json').read_text())['tools']
 messages.append({'role':'user','content':'''The transport stopped at its 26-round budget after your last source reads, without capturing a verdict. This is a bounded finalisation of that same review, not a new reviewer or a reset of evidence. The complete prior source/tool conversation has been restored. All eleven reviewer-requested executions completed, and the frozen source, instruments and installed dependency identities have been rechecked unchanged.

Before a passing verdict, six required source inspections remain missing in the tool record: api (the real getChat/ChatResponse path), renderer (the real StreamingMarkdownContent implementation), runner (validate_codegen.py), executor (the reused execution transport), tsconfig, and vitest_config. Also your go_store read covered only lines 1–60; the changed ToolFunction.Result annotation is at line 97, so read that actual region rather than claim to have checked it from the header. Inspect these with the same bounded tools. Use sufficiently sized reads rather than repeat small prefixes of already-returned JSON logs. Source reads are one-based, up to 450 lines; output reads are byte-based, up to 32000 bytes.

You have at most six further model rounds to inspect those concrete surfaces and give your evidence-led final report. The three unused execution slots remain available only for a substantiated need; existing fresh runs are not stale merely because this reporting turn was resumed. Do not claim completeness if evidence remains missing. Return JUDGE=PASS or JUDGE=FAIL only for the existing bounded generator contract; failing or incomplete review is acceptable. Preserve the known lint, warning, root/Keychain/package/main/installation and browser boundaries.'''} )
 save(target/'initial-request.json',{'messages':messages,'tools':tools,'previous_context_sha256':sha(BASE/'review-finalisation-context.json'),'finaliser_sha256':finaliser_sha})
 actual_reads=[];go_metadata_read=False
 for iteration in range(1,7):
  if original['source']!=runner['fingerprints']() or original['instruments']!={k:sha(p) for k,p in ALLOWED.items()} or sha(Path(__file__))!=finaliser_sha:
   raise RuntimeError('Review inputs changed')
  payload={'model':MODEL,'messages':messages,'tools':tools,'stream':False,'think':False,'options':{'temperature':0,'num_ctx':131072,'num_predict':8192}}
  if len(json.dumps(payload).encode())>600000:raise RuntimeError('Existing review context bound exceeded')
  response=http('/api/chat',payload);save(target/f'response-{iteration:02}.json',response)
  if not response.get('done') or response.get('done_reason')=='length' or response.get('model') not in {MODEL,'deepseek-v4.1-flash'}:raise RuntimeError('Incomplete or substituted reviewer response')
  served.append(response['model']);message=response['message'];messages.append(message);calls=message.get('tool_calls') or []
  if not calls:
   report=message.get('content','');(target/'report.md').write_text(report)
   cases=[x['case'] for x in runs];coverage=runner['CASES']<=set(cases) and cases[0]=='candidate' and cases[-1]=='candidate' and cases.count('candidate')>=2
   passed=coverage and REQUIRED<=reads and go_metadata_read and all(x['expectation_met'] for x in runs) and bool(re.search(r'^JUDGE=PASS\s*$',report,re.M)) and 'JUDGE=FAIL' not in report
   runner['assert_frozen']()
   result={'passed':passed,'source_commit':runner['HEAD'],'requested_model':MODEL,'served_models':sorted(set(served)),
     'tag_manifest':MANIFEST,'weight_immutability_claimed':False,'fresh_requested_cases':cases,'required_execution_coverage':coverage,
     'all_expectations_met':all(x['expectation_met'] for x in runs),'read_sources':sorted(reads),'missing_required_reads':sorted(REQUIRED-reads),
     'go_metadata_region_read':go_metadata_read,'finalisation_reads':actual_reads,'source_and_dependencies_unchanged':True,
     'preceding_attempts':['independent-review','independent-review-r2'],'resumed_same_full_conversation':True,
     'finaliser_sha256':finaliser_sha,'report_sha256':sha(target/'report.md'),'scope':'Generated-model maintenance only; not whole-programme/main/installation acceptance'}
   save(target/'result.json',result);save(target/'conversation.json',messages);print(report,flush=True);print('FINAL_INDEPENDENT_RESULT',json.dumps(result),flush=True);return 0 if passed else 1
  if len(calls)>12:raise RuntimeError('Existing tool fanout bound exceeded')
  for call in calls:
   name=call['function']['name'];args=call['function']['arguments'];args=json.loads(args) if isinstance(args,str) else args
   problem=controller['argument_problem'](name,args,runner['CASES'],len(runs))
   if problem is not None:result={'error':problem,'executed':False}
   elif name=='read_source':
    path=ALLOWED[args['name']];lines=path.read_text().splitlines();start,count=args['start'],args['count'];end=min(start-1+count,len(lines))
    text='\n'.join(f'{i+1}: {lines[i]}' for i in range(start-1,end))
    if len(text.encode())>75000:result={'error':'Source read too large; request fewer lines','executed':False}
    else:
     reads.add(args['name']);actual_reads.append({'name':args['name'],'start':start,'end':end,'file_sha256':sha(path)})
     if args['name']=='go_store' and start<=97<=end:go_metadata_read=True
     result={'path':str(path),'file_sha256':sha(path),'total_lines':len(lines),'text':text}
   elif name=='read_run_output':
    path=Path(runs[args['ordinal']-1]['tests'][args['stream']]);raw=path.read_bytes();offset=args['offset'];chunk=raw[offset:offset+args['count']]
    result={'path':str(path),'file_sha256':sha(path),'total_bytes':len(raw),'offset':offset,'next_offset':offset+len(chunk),'content':chunk.decode('utf8',errors='replace')}
   elif name=='run_validation':
    if len(runs)>=14:result={'error':'Original run budget exhausted; return an honest incomplete or failing verdict.','executed':False}
    else:
     result=runner['run'](args['case'],f'judge-final-{len(runs)+1}-{args["case"]}');runs.append(result);print('JUDGE_ADDITIONAL_RUN',args['case'],result['expectation_met'],flush=True)
   save(target/f'tool-{iteration:02}-{len(messages):03}.json',{'name':name,'arguments':args,'result':result})
   messages.append({'role':'tool','tool_name':name,'content':json.dumps(result)})
 save(target/'incomplete.json',{'passed':False,'reason':'Finite finalisation budget exhausted without a verdict','read_sources':sorted(reads),'missing_required_reads':sorted(REQUIRED-reads)})
 save(target/'conversation.json',messages)
 return 1

if __name__=='__main__':raise SystemExit(main())
