# Agent Guidelines for go-minitrace

go-minitrace converts AI agent transcripts (Claude Code, Codex, Pi, Copilot, claude.ai, ChatGPT, turnsdb) into `.minitrace.json` archives and queries them with a normalized SQLite engine (`query run`, `query commands`, JS runtime, `serve` web UI).

## Build Commands

- Run the CLI: `go run ./cmd/go-minitrace <args>` (prefer this over build + ./binary)
- Build: `go build ./...`
- Test: `go test ./...`
- Run single test: `go test ./pkg/path/to/package -run TestName`
- Lint: `golangci-lint run -v` or `make lint`
- Format: `go fmt ./...`
- Frontend/embedded web UI: `go generate ./...` runs `cmd/build-web`, a **Dagger** pipeline (requires Docker) that builds `web/` with pnpm and embeds the dist into the serve command. For iterative frontend work use `make dev` (tmux: serve `--dev` backend + Vite dev server) or run `pnpm dev` in `web/` yourself.

IMPORTANT: To run a server and do some interaction with it, use tmux, this makes it very easy to kill a server.
Use capture-pane to read the output.

## Project Structure

- `cmd/go-minitrace/` — CLI entry point; subcommand groups live in `cmd/go-minitrace/cmds/` (`convert`, `discover`, `query`, `serve`, `annotate`, `validate`, `preview`)
- `cmd/build-web/` — Dagger-based frontend build invoked by `go generate`
- `pkg/minitrace/` — core schema types, archive/manifest writing, metrics
- `pkg/minitracedb/` — normalized SQLite engine: schema, conversion/materialization, sandboxed read-only query runner, embedded presets (`presets/`)
- `pkg/minitracecmd/` — sqleton-style structured query command loading/rendering (`core/` embedded commands)
- `pkg/minitracejs/` — `require("minitrace")` Goja module: `mt.db()` builder API over normalized SQLite
- `pkg/adapters/` — one package per source format (claudecode, codex, pi, copilot, claudeai, chatgpt, turnsdb)
- `pkg/annotate/`, `pkg/validate/` — annotation store and archive validation
- `pkg/doc/` — embedded glazed help pages (markdown with YAML frontmatter; malformed frontmatter breaks the help system — smoke with `go run ./cmd/go-minitrace help --all`)
- `web/` — React 19 + TypeScript + Vite + MUI (Material UI) SPA, **pnpm** workspace (no bun, no bootstrap, no templ), Storybook under `web/.storybook`
- `proto/` + `gen/` — protobuf definitions and generated code for the serve API
- `queries/` — saved SQL files (normalized schema); default `--query-dir` for `serve`
- `testdata/query-repositories/` — showcase structured-command repositories used by tests and docs
- `ttmp/` — docmgr ticket workspace (design docs, diaries); do not treat as source code

<runningProcessesGuidelines>
- When testing TUIs, use tmux and capture-pane to interact with the UI.
- When using tmux, try to batch as many commands as possible when using send-keys.
- When running long-running processes (servers, etc...), use tmux to more easily interact and kill them.
- Kill a process using port $PORT: `lsof-who -p $PORT -k`. When building a web server, ALWAYS use this command to kill the process.
</runningProcessesGuidelines>

<goGuidelines>
- When implementing go interfaces, use the var _ Interface = &Foo{} to make sure the interface is always implemented correctly.
- Always use a context argument when appropriate.
- CLI commands are glazed commands wired into cobra; follow the existing patterns in `cmd/go-minitrace/cmds/`.
- Use the "defaults" package name, instead of "default" package name, as it's reserved in go.
- Use github.com/pkg/errors for wrapping errors.
- When starting goroutines, use errgroup.
- Only use the toplevel go.mod, don't create new ones.
- When writing a new experiment / app, add zerolog logging to help debug and figure out how it works, add --log-level flag to set the log level.
- When using go:embed, import embed as `_ "embed"`
- When using build tagged features, make sure the software compiles without the tag as well
</goGuidelines>

<webGuidelines>
- The frontend is React 19 + TypeScript + Vite + MUI with Redux Toolkit / RTK Query; package manager is pnpm.
- Do NOT introduce bun, bootstrap, or templ — they are not used in this repo.
- API types come from protobuf (`proto/` → `gen/`, consumed in `web/src/gen`); prefer regenerating over hand-editing.
- The production SPA is embedded into the Go binary via `go generate` (Dagger build); `serve --dev` runs API-only mode for use with the Vite dev server.
</webGuidelines>

<debuggingGuidelines>
If me or you the LLM agent seem to go down too deep in a debugging/fixing rabbit hole in our conversations, remind me to take a breath and think about the bigger picture instead of hacking away. Say: "I think I'm stuck, let's TOUCH GRASS".  IMPORTANT: Don't try to fix errors by yourself more than twice in a row. Then STOP. Don't do anything else.

</debuggingGuidelines>

<generalGuidelines>
Don't add backwards compatibility layers or adapters unless explicitly asked. If you think there is a need for a backwards compatibility or adapting to an existing interface, STOP AND ASK ME IF THAT IS NECESSARY. Usually, I don't need backwards compatibility.

If it looks like your edits aren't applied, stop immediately and say "STOPPING BECAUSE EDITING ISN'T WORKING".

Run the format_file tool at the end of each response.
</generalGuidelines>
