#!/usr/bin/env python3
"""Inventory repository-visible JS files without rewriting historical evidence.

Run from any directory: python3 path/to/04-inventory-js-api-migration.py
Uses rg's normal ignore rules; excludes dependencies/build output. Method tokens
are lexical evidence (including comments), not an AST or a claim of execution.
"""
import hashlib
import json
from pathlib import Path
import re
import subprocess
from collections import Counter

ROOT = Path(subprocess.check_output(["git", "rev-parse", "--show-toplevel"], cwd=Path(__file__).resolve().parent, text=True).strip())
TICKET = Path(__file__).resolve().parents[1]
paths = sorted(subprocess.check_output([
    "rg", "--files", "-g", "*.js", "-g", "*.mjs", "-g", "*.cjs",
    "-g", "!node_modules", "-g", "!web/dist", "-g", "!vendor",
], cwd=ROOT, text=True).splitlines())
rows = []
for name in paths:
    raw = (ROOT / name).read_bytes()
    source = raw.decode("utf-8")
    methods = sorted(set(re.findall(r"\.([A-Za-z_$][\w$]*)\s*\(", source)))
    imports = sorted(set(re.findall(r"require\([\"']([^\"']+)[\"']\)", source)))
    rows.append(dict(path=name, bytes=len(raw), sha256=hashlib.sha256(raw).hexdigest(),
                     methods=methods, imports=imports,
                     disposition=("historical; preserve original" if name.startswith("ttmp/")
                                  else "web/generated; outside query API" if name.startswith("web/")
                                  else "maintained migration review")))
report = dict(scope="rg-visible JS/MJS/CJS in this worktree; dependencies and build output excluded; inline JS in Go/Markdown is not counted", total=len(rows),
              bytes=sum(x["bytes"] for x in rows),
              unique_content=len(set(x["sha256"] for x in rows)),
              groups=dict(Counter(x["path"].split("/")[0] for x in rows)), files=rows)
out = TICKET / "sources" / "js-api-migration-inventory.json"
out.write_text(json.dumps(report, indent=2) + "\n")
print(json.dumps({k:v for k,v in report.items() if k != "files"}, indent=2))
print(out)

# Every discovered path gets an explicit before/after-family assignment.
# These lexical tags index the manual review, not automated semantic migration.
def families(row):
    name = row['path']
    source = (ROOT / name).read_text()
    methods = set(row['methods'])
    tags = []
    if name.startswith('web/') or name.endswith('.mjs'):
        return ['M10 (unchanged external/web tooling)']
    if methods & {'db', 'sources', 'File', 'Content', 'RuntimeArchives', 'Build'}:
        tags.append('M1')
    if methods & {'query', 'queryOne', 'queryResult', 'execute'}:
        tags.append('M2')
    if methods & {'string', 'stringIn', 'like'} and 'sql' in source:
        tags.append('M3')
    if 'SessionToolCounts' in source:
        tags.append('M4')
    if methods & {'ToolRows', 'TokenUsageRows', 'TurnRows', 'SessionSummary', 'sql', 'args'}:
        tags.append('M5')
    if methods & {'reduce', 'filter', 'sort', 'map'}:
        tags.append('M6')
    if 'native' in source or 'inspection.compactions' in source:
        tags.append('M7')
    if methods & {'importer', 'CacheKey', 'cacheInfo', 'diagnostics'}:
        tags.append('M8')
    if '__verb__' in source or 'exports.' in source or 'mt.runtime' in source:
        tags.append('M9')
    if re.search(r'default\s*:\s*-\d|throw new Error|syntax', source) and '/generated/D-' in name:
        tags.append('M10 (review fixture intent)')
    return tags or ['M9/M10 (ordinary JS or discovery fixture)']

for row in rows:
    row['migration_families'] = families(row)
out.write_text(json.dumps(report, indent=2) + '\n')
doc = TICKET / 'design-doc/04-composable-javascript-query-grammar-architecture-api-and-intern-implementation-guide.md'
marker = '\n## 20. Complete discovered file inventory\n'
parts = [marker, '\nEach entry names its before/after families from section 19. Directory headings are repository-relative; filenames below them complete the path. Tags are lexical indexing aids; section 19 records semantic review. Historical files are preserved, and M10 external/generated files stay unchanged. Hashes and observed method names are in the JSON inventory.\n']
parent = None
for row in rows:
    p = Path(row['path'])
    if str(p.parent) != parent:
        parent = str(p.parent)
        parts.append('\n### ' + parent + '\n\n')
    parts.append('- `' + p.name + '` → ' + ', '.join(row['migration_families']) + '.\n')
doc.write_text(doc.read_text().split(marker)[0] + ''.join(parts))
