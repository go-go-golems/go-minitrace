import assert from "node:assert/strict";
import childProcess from "node:child_process";
import { syncBuiltinESMExports } from "node:module";
import { join } from "node:path";

const calls = [];
const original = childProcess.execFile;
childProcess.execFile = (executable, argv, options, callback) => {
  calls.push({ executable, argv, options });
  callback(null, "[]", "");
};
syncBuiltinESMExports();

try {
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
  console.log("Pi command discovery, grouped invocation and argument validation passed");
} finally {
  childProcess.execFile = original;
  syncBuiltinESMExports();
}
