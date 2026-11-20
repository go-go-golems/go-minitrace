import assert from "node:assert/strict";
import childProcess from "node:child_process";
import { mkdir, mkdtemp, readFile, rm } from "node:fs/promises";
import { tmpdir } from "node:os";
import { syncBuiltinESMExports } from "node:module";
import { join } from "node:path";

const calls = [];
const original = childProcess.execFile;
const originalCwd = process.cwd();
const work = await mkdtemp(join(tmpdir(), "persona-tool-test-"));
let next = {error: null, stdout: "[]", stderr: ""};
childProcess.execFile = (executable, argv, options, callback) => {
  calls.push({ executable, argv, options });
  callback(next.error, next.stdout, next.stderr);
};
syncBuiltinESMExports();

try {
  process.chdir(work);
  process.env.GMT_EVAL_ROOT = "/fixture";
  const { default: register } = await import("./analyst.ts");
  const tools = new Map();
  register({ registerTool: tool => tools.set(tool.name, tool) });
  const invoke = params => tools.get("minitrace").execute("test", params);
  const catalogArgs = ["query", "commands", "--query-repository", join(process.cwd(), "artifacts")];

  await invoke({ kind: "help", path: [] });
  assert.deepEqual(calls.at(-1).argv.slice(3), [...catalogArgs, "--help"]);
  await invoke({ kind: "help", path: ["probe_sandbox"] });
  assert.deepEqual(calls.at(-1).argv.slice(3), [...catalogArgs, "probe_sandbox", "--help"]);
  await invoke({ kind: "js", path: ["probe_sandbox", "probe-sandbox"], scope: "codex" });
  assert.deepEqual(calls.at(-1).argv.slice(3), [
    ...catalogArgs, "probe_sandbox", "probe-sandbox", "--archive-glob",
    "/fixture/input/archives/codex-*.minitrace.json", "--output", "json",
  ]);
  assert.equal(calls.at(-1).executable, "/usr/bin/sandbox-exec");
  await invoke({ kind: "js", path: ["ledger"] });
  assert.equal(calls.at(-1).argv.at(7), "ledger");
  const dispatched = calls.length;
  for (const params of [
    { kind: "js", path: [] }, { kind: "js", path: ["--output"] },
    { kind: "js", path: ["probe_sandbox probe-sandbox"] }, { kind: "js", path: ["ledger.js"] },
    { kind: "js", text: "ledger" }, { kind: "help", path: ["--help"] },
  ]) {
    await assert.rejects(invoke(params), /path/);
  }
  assert.equal(calls.length, dispatched);
  const decode = value => JSON.parse(value.content[0].text);
  const long = JSON.stringify([{complete: false, rows: [{value: "x".repeat(23000)}]}]);
  next = {error: null, stdout: long, stderr: ""};
  const incomplete = decode(await invoke({kind: "sql", text: "select 1"}));
  assert.equal(incomplete.queryComplete, false);
  assert.equal(incomplete.previewTruncated, true);
  assert.equal(incomplete.captureComplete, true);
  assert.equal(incomplete.processError, null);
  assert.equal(incomplete.stdout.length, 22000);
  assert.equal(await readFile(join(work, incomplete.receipt, "stdout"), "utf8"), long);

  next = {error: {code: "ERR_CHILD_PROCESS_STDIO_MAXBUFFER", killed: true}, stdout: "partial", stderr: ""};
  const capped = decode(await invoke({kind: "js", path: ["ledger"]}));
  assert.equal(capped.queryComplete, null);
  assert.equal(capped.previewTruncated, false);
  assert.equal(capped.captureComplete, false);
  assert.equal(capped.processError, "capture_limit");
  assert.equal(await readFile(join(work, capped.receipt, "stdout"), "utf8"), "partial");
  const artifacts = join(process.cwd(), "artifacts");
  await mkdir(artifacts);
  next = {error: null, stdout: JSON.stringify({
    commands: [{path: ["report", "summary"], sourceRoot: artifacts}], diagnostics: [],
  }), stderr: ""};
  const saved = decode(await tools.get("save_artifact").execute("test", {name: "report.js", content: "// checked by catalog"}));
  assert.deepEqual(saved.paths, [["report", "summary"]]);
  assert.deepEqual(calls.at(-1).argv.slice(3), ["query", "catalog", "--query-repository", artifacts, "report.js"]);
  assert.equal(await readFile(join(artifacts, "report.js"), "utf8"), "// checked by catalog");

  next = {error: null, stdout: JSON.stringify({table: "files", columns: [{name: "path"}]}), stderr: ""};
  const schema = decode(await invoke({kind: "schema", text: "files"}));
  assert.deepEqual(schema.columns, [{name: "path"}]);
  assert.deepEqual(calls.at(-1).argv.slice(3), ["query", "schema", "files"]);
  console.log("Pi discovery, dispatch, completeness, preview and capture receipts passed");
} finally {
  process.chdir(originalCwd);
  await rm(work, {recursive: true, force: true});
  childProcess.execFile = original;
  syncBuiltinESMExports();
}
