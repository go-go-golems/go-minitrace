---
Title: Jonmagic Copilot Session Search Resume
Ticket: MINITRACE-COPILOT-CLI
DocType: reference
Topics: [go-minitrace, copilot, conversion]
Status: active
Intent: source-evidence
---

Last week I switched to [copilot-cli](https://docs.github.com/en/copilot/how-tos/copilot-cli) as my 
full-time coding agent. I’d been splitting my time between it and 
[opencode](https://opencode.ai/) for months, using Copilot a few times a week, gravitating back to 
opencode for daily work. But after returning from two weeks of vacation to a fresh laptop, I 
committed to using only the latest copilot-cli for a full week. By the end of it I sent this to the 
team internally:

> I gotta say, things are finally at the point where I’m not tempted to switch back to opencode. 
Amazing work everyone. I’m going to start annoying my friends that still use claude code and get 
them to try switching to copilot again.

One friction point lingered though. Finding my way back to previous sessions.

## e-too-many-sessions

I run multiple Copilot sessions a day across different folders and repos. My 
[Brain](https://jonmagic.com/posts/context-rules-everything-around-me/) (personal knowledge base), 
work projects, side projects, home automation. Open a tab, start a session, do some work, close it, 
move on. An hour later I’d think “I need to pick that back up” and have no idea which session 
it was. My workaround was manually copy-pasting session IDs into my [weekly 
note](https://jonmagic.com/posts/context-rules-everything-around-me/) so I could `copilot 
--resume=<uuid>` later. It worked, but it felt wrong.

## ~/.copilot/ 👀

The copilot-cli stores all its state on disk at `~/.copilot/`. Each session gets its own directory 
under `session-state/` with an `events.jsonl` event log and a `workspace.yaml` metadata file. But 
the interesting discovery was `session-store.db`, a SQLite database copilot-cli maintains 
automatically.

```
~/.copilot/
├── config.json
├── session-state/           # One directory per session
│   └── <uuid>/
│       ├── events.jsonl     # Every event that happened
│       ├── workspace.yaml   # Session metadata
│       └── checkpoints/     # Snapshot history
├── session-store.db         # ← The good stuff
└── mcp-config.json
```

I opened `session-store.db` with `sqlite3` and found six tables.

| Table | What’s in it |
| --- | --- |
| `sessions` | ID, auto-generated summary, repository, branch, timestamps |
| `turns` | Every user message and assistant response |
| `checkpoints` | Titled snapshots with overviews and next steps |
| `session_files` | Every file touched during the session |
| `session_refs` | Commits, PRs, and issues linked to the session |
| `search_index` | FTS5 full-text search across all content |

That last one caught my eye. It looks like copilot-cli maintains an [FTS5 full-text search 
index](https://www.sqlite.org/fts5.html) across all session content. And the summaries in the 
`sessions` table? Already generated automatically, no LLM call needed to get a human-readable title 
for every session.

## copilot-sessions

Once I understood the data, the tool was straightforward. Query the database, pipe through 
[fzf](https://github.com/junegunn/fzf) for interactive fuzzy search, resume the chosen session:

```bash
# Browse recent sessions
cs

# Search for a session
cs home assistant

# List without fzf
cs -l
```

The output looks like this:

```
TIME     REPO             SESSION
11m ago  brain            Home Assistant Automation Ideas
1h ago   brain            Research Copilot CLI Theme
4h ago   brain            Review And Seed Spotify Playlist
13h ago  brain            Enable Ghostty SSH Terminal
```

Pick one, hit Enter, and Copilot resumes right in your terminal. `ctrl-y` copies the session ID to 
clipboard if you need it elsewhere.

Search uses fzf’s fuzzy matching against session summaries, all user messages, and repository 
names. That means it’s typo-tolerant. `mincraft` finds your Minecraft session, `sptoify` finds 
Spotify. It supports substring matching and multi-word queries too. The user messages are appended 
off-screen so fzf can match against them without cluttering the display.

All the formatting (relative timestamps, repo name truncation, column layout) happens inside a 
single SQLite query. No bash loops or per-row shell-outs. Startup is about 20ms for hundreds of 
sessions.

I published it as [copilot-sessions](https://github.com/jonmagic/copilot-sessions).

> [!tip] Tip
> If you use `copilot -p <prompt>` for automations (e.g., a cronjob running every few minutes), 
those sessions will show up in your search results too. I found it helpful to have the 
automation’s prompt include a step that deletes its own session from `session-store.db` at the 
end so it doesn’t clutter things up or become resumable.

## …one more thing

The `events.jsonl` file in each session directory is a streaming log of everything that happened. 
Every tool call, every file edit, every shell command, all with timestamps:

```
{"type":"session.start","timestamp":"...","data":{"context":{"repository":"jonmagic/copilot-sessions
","branch":"main"}}}
{"type":"tool.execution_start","timestamp":"...","data":{"toolName":"grep","arguments":{...}}}
{"type":"tool.execution_complete","timestamp":"...","data":{"success":true}}
{"type":"session.shutdown","timestamp":"...","data":{"modelMetrics":{"gpt-5.4":{"usage":{"inputToken
s":192048}}}}}
```

There’s a lot you could build on this. Session analytics, cross-session file search, timeline 
visualizations. For now I’m happy just being able to find my sessions without keeping a manual 
list.

> [!note] Note
> I work at GitHub, which helped me confirm some of these findings by reviewing internal code. But 
everything described here is observable by anyone with copilot-cli installed. Just open 
`~/.copilot/` and start exploring.
