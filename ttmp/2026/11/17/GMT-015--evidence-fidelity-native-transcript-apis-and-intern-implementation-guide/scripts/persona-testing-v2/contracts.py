"""Explicit V2 answer examples and mandatory/optional task boundaries."""
EXAMPLES={
'A':{'schema_version':2,'task':'A','sessions':[{'session_id':'id','turn_count':0,'tool_count':0}],'sql_js_equal':True,'evidence':[]},
'B':{'schema_version':2,'task':'B','synthetic':{'unicode_bytes':4,'clipped_text':'ab','clipped_complete':False,'strict_rows':0,'strict_error':True,'archive_source_ordinals':[0,1,2],'chronological_source_ordinals':[1,2,0],'manifest_duplicate_count':1,'manifest_conflict_count':1},'real_sessions':[],'evidence':[]},
'C':{'schema_version':2,'task':'C','path_base':'ABSOLUTE_CORPUS_RAW_DIRECTORY','synthetic_request_counts':{'complete':2,'incomplete':1},'real_request_counts':[{'path':'claude-code/file.jsonl','api_request_records':0}],'real_scans':[{'path':'claude-code/file.jsonl','sha256':'digest','bytes':0,'lines':0,'hashScope':'whole_source'}],'exact_replay_possible':False,'evidence':[]},
'D':{'schema_version':2,'task':'D','good_command_succeeded':True,'bad_sibling_diagnosed':True,'required_source_visible':True,'runtime_flags_visible':True,'negative_default_supported':False,'selected_session_ids':[],'evidence':[]}}
# Values above illustrate types/order semantics; prompts remove expected numerical answers.
SCHEMAS={'A': {'sessions': [{'session_id': 'string: exact real normalized session ID',
                     'turn_count': 'integer: actual turns rows',
                     'tool_count': 'integer: actual tool_calls rows'}],
       'sessions_scope': 'exactly nine rows, one per real normalized archive; preserve zero-tool '
                         'sessions',
       'sql_js_equal': 'boolean from actual two-path comparison'},
 'B': {'synthetic': {'unicode_bytes': 'integer',
                     'clipped_text': 'string',
                     'clipped_complete': 'boolean',
                     'strict_rows': 'integer',
                     'strict_error': 'boolean',
                     'archive_source_ordinals': 'array of integer source_ordinal values, sorted by '
                                                'source_ordinal; NOT row objects',
                     'chronological_source_ordinals': 'array of integer source_ordinal values, in '
                                                      'DEFAULT ToolRows order; NOT row objects',
                     'manifest_duplicate_count': 'integer',
                     'manifest_conflict_count': 'integer'},
       'real_sessions': [{'session_id': 'string: at least one real archive per framework',
                          'tool_count': 'integer: actual tools',
                          'token_evidence': 'observed token evidence object'}],
       'evidence': 'array of saved evidence paths'},
 'C': {'path_base': 'exact absolute path of corpus/raw',
       'synthetic_request_counts': 'complete/incomplete integers for fixture native.jsonl ONLY',
       'real_request_counts': 'one row per real native source, path relative to path_base, '
                              'api_request_records integer',
       'real_scans': 'same nine relative paths, sha256, bytes, lines, hashScope',
       'exact_replay_possible': 'boolean'},
 'D': {'good_command_succeeded': 'boolean',
       'bad_sibling_diagnosed': 'boolean',
       'required_source_visible': 'boolean',
       'runtime_flags_visible': 'boolean',
       'negative_default_supported': 'boolean: ONLY a negative numeric default in JS __verb__ metadata, not CLI values or YAML defaults',
       'selected_session_ids': 'array of exactly three session IDs, one real archive per framework',
       'evidence': 'array of saved evidence paths'}}
TASKS={
'A':('new intern','MANDATORY: Query all nine real normalized archives using SQL and runnable minitrace JS. Count actual turns and tools, include zero-tool sessions, compare results. Save sessions and sql_js_equal. OPTIONAL after saving mandatory answer: discover/convert one native source per framework, validate, inspect a failed-command candidate and try a history command.'),
'B':('Python analyst learning JavaScript','MANDATORY: With synthetic archive.json use JS/SQL to measure UTF-8 bytes of abé, permissive clipping at 3 bytes, and strict rejection. Separately report archive_source_ordinals ordered by source_ordinal and chronological_source_ordinals in DEFAULT ToolRows order. Run corpusManifest on exactly archive.json/archive-copy.json/archive-conflict.json. Put all fixture answers under synthetic. Then summarize one real archive per framework under real_sessions. OPTIONAL: row overflow, invalid query exceptions/envelopes and detailed token provenance. Never confuse archive order with default chronological order.'),
'C':('forensic reviewer','MANDATORY: Use mt.native in runnable JS to scan all nine real raw files and synthetic native.jsonl. synthetic_request_counts applies ONLY to native.jsonl fixture and counts complete/incomplete request graphs. real_request_counts counts api-request records for EACH real source; zero in Codex/Pi is a capture fact, not absence of instructions. real_scans reports whole-source sha256/bytes/lines/hashScope. Set path_base to the exact corpus/raw absolute directory and use framework/basename relative paths everywhere. Explain exact replay limits. OPTIONAL: early stop, malformed/oversized record details, compaction graph and instruction-read candidate.'),
'D':('SQL command author','MANDATORY: Create commands/good.js with root verb good and REQUIRED string source; source is a normalized archive file path. Open that source and return exactly one JSON row with top-level session_id and tool_count using a standard ToolRows recipe and bound SQL. Add commands/bad.js with one metadata numeric default -1 and keep it present. Test valid sibling execution, bad-sibling diagnostic, required flag visibility, runtime flags, and signed-default support. Pick one real archive per framework. Host evaluator will replay good three times with --source PATH, and normal --help with bad.js still present. OPTIONAL: db.schema/tables, forbidden SQL, memory cache and strict evidence. Do not replace nonzero statuses with zero or vice versa.')}
