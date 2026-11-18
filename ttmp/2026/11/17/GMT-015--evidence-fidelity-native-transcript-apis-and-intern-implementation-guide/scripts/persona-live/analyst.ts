// Loaded explicitly only in the analyst's Pi session.
import { execFile } from "node:child_process";
import { readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";

export default function (pi: any) {
  const root = process.env.GMT_EVAL_ROOT!;
  const work = process.cwd();
  const binary = join(root, "input", "go-minitrace");
  const run = (args: string[]) => new Promise((resolve) => {
    execFile("/usr/bin/sandbox-exec", ["-f", join(work, "query.sb"), binary, ...args], {
      cwd: work, timeout: 25000, maxBuffer: 512 * 1024, env: {
        PATH: "/usr/bin:/bin", HOME: work, TMPDIR: work,
      },
    }, (error, stdout, stderr) => {
      resolve({ exit: error ? (error.code ?? error.message) : 0,
        stdout: stdout.slice(0, 22000), stderr: stderr.slice(0, 3000),
        deliveryTruncated: stdout.length > 22000 || stderr.length > 3000 || error?.code === "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" });
    });
  });
  const string = { type: "string" };
  const schema = (properties: Record<string, unknown>, required = Object.keys(properties)) =>
    ({ type: "object", properties, required, additionalProperties: false });
  const result = (value: unknown) => ({ content: [{ type: "text", text: JSON.stringify(value) }] });
  const artifact = (name: string) => {
    if (!/^[a-z][a-z0-9_-]{0,45}\.(js|sql|md)$/.test(name)) throw Error("Use a flat .js, .sql or .md filename");
    return join(work, "artifacts", name);
  };

  pi.registerTool({
    name: "save_artifact", label: "Save artifact",
    description: "Write or replace a flat .js, .sql or .md artifact in the run workspace.",
    parameters: schema({ name: string, content: string }),
    async execute(_id: string, { name, content }: { name: string; content: string }) {
      if (Buffer.byteLength(content) > 100_000) throw Error("Artifact exceeds 100 kB");
      await writeFile(artifact(name), content);
      return result({ saved: name });
    },
  });
  pi.registerTool({
    name: "read_artifact", label: "Read artifact",
    description: "Read an artifact you previously saved.",
    parameters: schema({ name: string }),
    async execute(_id: string, { name }: { name: string }) {
      return result(await readFile(artifact(name), "utf8"));
    },
  });
  pi.registerTool({
    name: "minitrace", label: "Run go-minitrace",
    description: "Run pinned go-minitrace. Use help for CLI and API help, sql for a result-envelope query, or js for a verb from the saved JS artifact directory. Scope defaults to all. Returns exit, stdout, stderr and deliveryTruncated.",
    parameters: schema({ kind: { enum: ["help", "sql", "js"] }, text: string,
      scope: { enum: ["all", "codex", "claude-code", "pi"] } }, ["kind", "text"]),
    async execute(_id: string, { kind, text, scope = "all" }: { kind: string; text: string; scope?: string }) {
      if (!["all", "codex", "claude-code", "pi"].includes(scope)) throw Error("Invalid scope");
      const archives = join(root, "input", "archives", (scope === "all" ? "*" : scope + "-*") + ".minitrace.json");
      let args: string[];
      if (kind === "help") {
        if (!/^[a-z0-9 -]{0,100}$/.test(text)) throw Error("Pass a help topic or command path");
        args = text.startsWith("query ") ? [...text.split(" "), "--help"] : ["help", ...text.split(" ").filter(Boolean)];
        if (args.some(x => x.startsWith("-") && x !== "--help")) throw Error("Help flags are supplied by the runner");
      } else if (kind === "sql") {
        args = ["query", "run", "--sql", text, "--archive-glob", archives, "--cache-mode", "memory",
          "--max-rows", "200", "--max-cell-bytes", "2048", "--result-envelope", "--output", "json"];
      } else {
        if (!/^[a-z][a-z0-9_-]{0,45}$/.test(text)) throw Error("Pass one root verb name");
        args = ["query", "commands", "--query-repository", join(work, "artifacts"), text,
          "--archive-glob", archives, "--output", "json"];
      }
      return result(await run(args));
    },
  });
}
