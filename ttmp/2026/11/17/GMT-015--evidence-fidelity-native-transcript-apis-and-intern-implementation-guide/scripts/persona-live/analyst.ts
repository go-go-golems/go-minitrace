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
    description: "Run pinned go-minitrace. Use help with text for API help or with path to list JS command groups and verbs. Use sql with text, or js with a path array such as [\"ledger\"] or [\"report\", \"summary\"]. Scope defaults to all. Returns exit, stdout, stderr and deliveryTruncated.",
    parameters: schema({ kind: { enum: ["help", "sql", "js"] }, text: string,
      path: { type: "array", maxItems: 8, items: { type: "string", pattern: "^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$" } },
      scope: { enum: ["all", "codex", "claude-code", "pi"] } }, ["kind"]),
    async execute(_id: string, { kind, text, path, scope = "all" }: {
      kind: string; text?: string; path?: string[]; scope?: string;
    }) {
      if (!["all", "codex", "claude-code", "pi"].includes(scope)) throw Error("Invalid scope");
      const archives = join(root, "input", "archives", (scope === "all" ? "*" : scope + "-*") + ".minitrace.json");
      const catalog = ["query", "commands", "--query-repository", join(work, "artifacts")];
      if (path !== undefined && (!Array.isArray(path) || path.length > 8 ||
          !path.every(word => typeof word === "string" && /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(word)))) {
        throw Error("Pass path as an array of command words without flags or file extensions");
      }
      let args: string[];
      if (kind === "js") {
        if (!path?.length || text !== undefined) throw Error('Use path, for example ["ledger"] or ["report", "summary"]');
        args = [...catalog, ...path, "--archive-glob", archives, "--output", "json"];
      } else if (kind === "sql") {
        if (typeof text !== "string" || !text.trim() || path !== undefined) throw Error("Pass SQL in text");
        args = ["query", "run", "--sql", text, "--archive-glob", archives, "--cache-mode", "memory",
          "--max-rows", "200", "--max-cell-bytes", "2048", "--result-envelope", "--output", "json"];
      } else if (kind === "help" && path !== undefined) {
        if (text !== undefined) throw Error("Pass path for command help or text for a help topic");
        args = [...catalog, ...path, "--help"];
      } else {
        if (kind !== "help" || typeof text !== "string" || !/^[a-z0-9 -]{0,100}$/.test(text)) {
          throw Error("Pass a help topic in text or command words in path");
        }
        args = text.startsWith("query ") ? [...text.split(" "), "--help"] : ["help", ...text.split(" ").filter(Boolean)];
        if (args.some(x => x.startsWith("-") && x !== "--help")) throw Error("Help flags are supplied by the runner");
      }
      return result(await run(args));
    },
  });
}
