// Loaded explicitly only in the analyst's Pi session.
import { randomUUID } from "node:crypto";
import { execFile } from "node:child_process";
import { mkdir, readFile, writeFile } from "node:fs/promises";
import { join } from "node:path";

export default function (pi: any) {
  const root = process.env.GMT_EVAL_ROOT!;
  const work = process.cwd();
  const binary = join(root, "input", "go-minitrace");
  const run = async (args: string[], kind: string) => {
    const captured = await new Promise<{ error: any; stdout: string; stderr: string }>((resolve) => {
      execFile("/usr/bin/sandbox-exec", ["-f", join(work, "query.sb"), binary, ...args], {
        cwd: work, timeout: 25000, maxBuffer: 2 * 1024 * 1024, env: {
          PATH: "/usr/bin:/bin", HOME: work, TMPDIR: work,
        },
      }, (error, stdout, stderr) => resolve({ error, stdout, stderr }));
    });
    const { error, stdout, stderr } = captured;
    const exit = error ? (typeof error.code === "number" ? error.code : null) : 0;
    const processError = error?.code === "ERR_CHILD_PROCESS_STDIO_MAXBUFFER" ? "capture_limit" :
      error?.killed ? "timeout_or_signal" : error && exit === null ? "execution_failed" :
      exit ? "nonzero_exit" : null;
    const captureComplete = !processError || processError === "nonzero_exit";
    const receipt = join("tool-results", randomUUID());
    const folder = join(work, receipt);
    const previewTruncated = stdout.length > 22000 || stderr.length > 3000;
    let queryComplete: boolean | null = null;
    if (exit === 0 && captureComplete && kind !== "help") {
      try {
        const output = JSON.parse(stdout);
        const envelope = Array.isArray(output) && output.length === 1 ? output[0] : output;
        if (typeof envelope?.complete === "boolean" && Array.isArray(envelope.rows)) {
          queryComplete = envelope.complete;
        }
      } catch { /* Plain output carries no query envelope. */ }
    }
    const status = { exit, processError, captureComplete, queryComplete, previewTruncated, receipt };
    await mkdir(folder, { recursive: true, mode: 0o700 });
    await Promise.all([
      writeFile(join(folder, "stdout"), stdout),
      writeFile(join(folder, "stderr"), stderr),
      writeFile(join(folder, "receipt.json"), JSON.stringify(status) + "\n"),
    ]);
    return { ...status, stdout: stdout.slice(0, 22000), stderr: stderr.slice(0, 3000) };
  };
  const string = { type: "string" };
  const schema = (properties: Record<string, unknown>, required = Object.keys(properties)) =>
    ({ type: "object", properties, required, additionalProperties: false });
  const result = (value: unknown) => ({ content: [{ type: "text", text: JSON.stringify(value) }] });
  const artifact = (name: string) => {
    if (!/^[a-z][a-z0-9_-]{0,45}\.(js|sql|md)$/.test(name)) throw Error("Use a flat .js, .sql or .md filename");
    return join(work, "artifacts", name);
  };

  const catalogArgs = ["--query-repository", join(work, "artifacts")];
  type Catalog = { commands: { path: string[]; sourceRoot: string }[]; diagnostics: unknown[] };
  const registrations = async (name?: string) => {
    const output = await run(["query", "catalog", ...catalogArgs, ...(name ? [name] : [])], "help");
    if (output.exit !== 0) return { paths: [], diagnostics: [output.stderr || output.processError], receipt: output.receipt };
    const catalog: Catalog = JSON.parse(output.stdout);
    return { paths: catalog.commands.filter(command => !name || command.sourceRoot === join(work, "artifacts"))
      .map(command => command.path), diagnostics: catalog.diagnostics };
  };

  pi.registerTool({
    name: "save_artifact", label: "Save artifact",
    description: "Write or replace a flat .js, .sql or .md artifact. For a JS command, returns its registered paths and scanner diagnostics.",
    parameters: schema({ name: string, content: string }),
    async execute(_id: string, { name, content }: { name: string; content: string }) {
      if (Buffer.byteLength(content) > 100_000) throw Error("Artifact exceeds 100 kB");
      await writeFile(artifact(name), content);
      return result({ saved: name, ...(name.endsWith(".js") ? await registrations(name) : {}) });
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
    description: "Run pinned go-minitrace. Use help with text for API help or with path to list JS command groups and verbs. Use schema with optional text for normalized tables or one table’s columns. Use sql with text, or js with a path array such as [\"ledger\"] or [\"report\", \"summary\"]. Scope defaults to all. Returns stdout/stderr previews, queryComplete when an envelope declares it, previewTruncated, captureComplete, processError, and a local receipt.",
    parameters: schema({ kind: { enum: ["help", "schema", "sql", "js"] }, text: string,
      path: { type: "array", maxItems: 8, items: { type: "string", pattern: "^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$" } },
      scope: { enum: ["all", "codex", "claude-code", "pi"] } }, ["kind"]),
    async execute(_id: string, { kind, text, path, scope = "all" }: {
      kind: string; text?: string; path?: string[]; scope?: string;
    }) {
      if (!["all", "codex", "claude-code", "pi"].includes(scope)) throw Error("Invalid scope");
      const archives = join(root, "input", "archives", (scope === "all" ? "*" : scope + "-*") + ".minitrace.json");
      const catalog = ["query", "commands", ...catalogArgs];
      if (path !== undefined && (!Array.isArray(path) || path.length > 8 ||
          !path.every(word => typeof word === "string" && /^[a-zA-Z0-9][a-zA-Z0-9_-]{0,63}$/.test(word)))) {
        throw Error("Pass path as an array of command words without flags or file extensions");
      }
      let args: string[];
      if (kind === "js") {
        if (!path?.length || text !== undefined) throw Error('Use path, for example ["ledger"] or ["report", "summary"]');
        args = [...catalog, ...path, "--archive-glob", archives, "--output", "json"];
      } else if (kind === "schema") {
        if (path !== undefined || (text !== undefined && !/^[a-z][a-z0-9_]*$/.test(text))) {
          throw Error("Omit text to list tables, or pass a normalized table name in text");
        }
        const output = await run(["query", "schema", ...(text ? [text] : [])], "help");
        return result(output.exit === 0 ? JSON.parse(output.stdout) : output);
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
      const output = await run(args, kind);
      if (kind === "js" && output.exit !== 0 && output.stderr.includes("available commands:")) {
        const listed = await registrations();
        const parent = path!.slice(0, -1);
        const matches = listed.paths.filter(words => parent.every((word, index) => words[index] === word));
        return result({ ...output, availablePaths: (matches.length ? matches : listed.paths).slice(0, 20) });
      }
      return result(output);
    },
  });
}
