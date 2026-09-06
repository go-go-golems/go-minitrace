// Node 24's native TypeScript stripping runs the actual frontend decoder.
// Usage: go run scripts/06-proto-outcomes.go | node scripts/07-check-proto-outcomes.mjs <repo>
import assert from 'node:assert/strict';
import { readFileSync } from 'node:fs';
import { resolve } from 'node:path';
import { pathToFileURL } from 'node:url';

const repo = resolve(process.argv[2] || process.cwd());
const { decodeSessionDetail } = await import(pathToFileURL(resolve(repo, 'web/src/api/sessionProtoAdapters.ts')));
const wire = JSON.parse(readFileSync(0, 'utf8'));
const detail = decodeSessionDetail(wire);
assert.equal(detail.metrics.execution_record_count, 5);
assert.equal(detail.metrics.file_touch_count, 5);
assert.equal(detail.metrics.confirmed_file_target_count, 0);
const calls = detail.blocks[0].turns[0].tool_calls_in_turn;
assert.equal(calls.length, 5);
for (const call of calls) {
  const expected = call.id === 'succeeded' ? true : call.id === 'failed' ? false : null;
  assert.equal(call.record_kind, 'execution');
  assert.equal(call.input.file_targets.length, 1);
  assert.equal(call.input.file_targets[0].success, null);
  assert.equal(call.input.file_targets[0].status, 'attempted');
  assert.equal(call.input.file_targets[0].source_reference, 'native#L3');
  assert.equal(call.output.success, expected, `${call.id} success changed`);
  assert.equal(call.output.status, call.id);
  assert.equal(call.output.exit_code, call.id === 'failed' ? -1 : null);
}
console.log(JSON.stringify({ passed: true, outcomes: calls.map(c => ({ id: c.id, success: c.output.success, status: c.output.status, exit_code: c.output.exit_code })) }, null, 2));
