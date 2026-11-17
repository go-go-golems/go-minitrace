#!/usr/bin/env python3
"""Summarize emitted tool calls and API-attempt coverage, not inferred success."""
import argparse,json,pathlib,re
PATTERNS={
'discovery':r'\bdiscover\b','conversion':r'\bconvert\b','archive_validation':r'\bvalidate\b',
'cli_sql':r'query run','js_commands':r'query commands','schema':r'\.schema\(|minitrace-schema',
'recipe_execute':r'\.execute\(','query_envelope':r'\.queryResult\(',
'strict_evidence':r'StrictEvidence|strict-evidence','byte_limit':r'MaxCellBytes|max-cell-bytes',
'byte_length':r'utf8ByteLength','source_order':r'source_ordinal','raw_arguments':r'arguments_json',
'output_status':r'output_status','token_provenance':r'token_origin|TokenUsageRows',
'corpus_manifest':r'corpusManifest','native_stream':r'\.ForEach\(','native_inspect':r'\.Inspect\(',
'cache_control':r'cache-mode|SQLiteMemory','history':r'context-window|file-history|session-activity|ticket-timeline',
'query_receipt':r'run-record','normal_help':r'--help','catalog_signed_default':r'default\s*:\s*-1'}

def main():
 p=argparse.ArgumentParser(description=__doc__);p.add_argument('--matrix',type=pathlib.Path,required=True);p.add_argument('--output',type=pathlib.Path,required=True);a=p.parse_args()
 out=[]
 for f in sorted(a.matrix.glob('*/events.jsonl')):
  calls=[];errors=[];suspect=[]
  for line in f.read_text(errors='replace').splitlines():
   try:e=json.loads(line)
   except ValueError:continue
   if e.get('type')=='tool_execution_start':
    args=e.get('args',{});s=json.dumps(args,ensure_ascii=False)
    calls.append({'id':e.get('toolCallId'),'tool':e.get('toolName'),'args':args})
    # Candidates only: a model may mention a forbidden name while explicitly excluding it.
    if any(x in s for x in ['oracle.json','corpus/manifest.json','models-store.json','auth.json','.env']):suspect.append(calls[-1])
   if e.get('type')=='tool_execution_end' and e.get('isError'):
    errors.append({'id':e.get('toolCallId'),'tool':e.get('toolName'),'diagnostic':json.dumps(e.get('result',{}),ensure_ascii=False)[:1600]})
  text='\n'.join(json.dumps(c['args'],ensure_ascii=False) for c in calls)
  result={'run':f.parent.name,'tool_calls':len(calls),'error_events':errors,'attempted_features':{k:bool(re.search(v,text)) for k,v in PATTERNS.items()},'potential_scope_violations':suspect}
  out.append(result)
 a.output.write_text(json.dumps({'scope':'emitted call and code-text attempts; feature matches do not prove execution success; potential violations require manual review','runs':out},indent=2)+'\n')
if __name__=='__main__':main()
