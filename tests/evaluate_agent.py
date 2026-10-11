"""Explicit provider evaluation using the application's shared builder and agent.
Default: inspect fixtures without invoking inference.
Run: ../.venv/bin/python tests/evaluate_agent.py --run --output /tmp/agent-results.json
Outputs require manual quality review; group checks do not certify model safety.
"""
import argparse
import asyncio
import copy
import json
import logging
import sys
import time
from pathlib import Path
ROOT=Path(__file__).resolve().parents[1]
sys.path.insert(0,str(ROOT))
from briefing import generate_briefing_async,build_input
import briefing
from unittest.mock import patch

async def evaluate(cases,output_path,stop_file=None,diagnostics=False):
    results=[]
    for case in cases:
        if stop_file and stop_file.exists():
            print('Stopped between samples; no further model calls.',flush=True)
            break
        started=time.monotonic()
        reasons=[]
        original=briefing.validate_output
        def checked(output,evidence):
            try:return original(output,evidence)
            except ValueError as error:
                reasons.append(str(error))  # Application validation reasons, never provider request details.
                raise
        try:
            with patch('briefing.validate_output',side_effect=checked) if diagnostics else patch('briefing.validate_output',wraps=original):
                output=await asyncio.wait_for(generate_briefing_async(copy.deepcopy(case['input'])),timeout=90)
            groups=sorted(d['group'] for d in output.get('draft_operational_messages',[]))
            result={'case':case['case'],'output':output,'group_coverage_pass':groups==case['expected']['groups'],'expected':case['expected']}
            print(json.dumps({'case':case['case'],'result':'returned','group_coverage_pass':result['group_coverage_pass']}),flush=True)
        except Exception as error:
            # Never print provider exception text or request configuration.
            result={'case':case['case'],'error_type':type(error).__name__,'expected':case['expected']}
            if diagnostics:
                causes=[];current=error
                while current is not None and len(causes)<5:
                    causes.append(type(current).__name__);current=current.__cause__
                result['exception_types']=causes
            print(json.dumps({'case':case['case'],'result':'error','error_type':type(error).__name__}),flush=True)
        if diagnostics:result['validation_reasons']=reasons
        result['elapsed_seconds']=round(time.monotonic()-started,2)
        results.append(result)
        output_path.write_text(json.dumps(results,indent=2))
    print(json.dumps({'samples_attempted':len(results),'structured_outputs':sum('output'in r for r in results),'group_coverage_passes':sum(r.get('group_coverage_pass',False) for r in results),'manual_quality_review_required':True}),flush=True)

def main():
    parser=argparse.ArgumentParser(description=__doc__)
    parser.add_argument('--run',action='store_true',help='Explicitly invoke the configured provider once per fixture, no retries')
    parser.add_argument('--diagnostics',action='store_true',help='Record safe application validation reasons and exception class chains')
    parser.add_argument('--case',action='append',dest='case_names',help='Evaluate only these case names; may be repeated')
    parser.add_argument('--stop-file',type=Path,help='Stop before the next case when this file exists')
    parser.add_argument('--output',type=Path,default=Path('/tmp/weather-desk-agent-results.json'))
    args=parser.parse_args()
    cases=json.loads((ROOT/'tests/fixtures/agent-review-cases.json').read_text())
    if args.case_names:
        unknown=set(args.case_names)-{case['case'] for case in cases}
        if unknown:parser.error('Unknown case names: '+', '.join(sorted(unknown)))
        cases=[case for case in cases if case['case'] in args.case_names]
    for case in cases:
        built=build_input(case['input'])
        assert built['normalized_evidence']['mode']=='REPLAY'
        assert built['operational_review_context']['eligible']==bool(case['expected']['groups'])
    if not args.run:
        print(f'{len(cases)} fixtures validated; 0 model calls. Add --run to evaluate the configured model.')
        return
    logging.getLogger('strands').setLevel(logging.CRITICAL)
    asyncio.run(evaluate(cases,args.output,args.stop_file,args.diagnostics))

if __name__=='__main__':main()
