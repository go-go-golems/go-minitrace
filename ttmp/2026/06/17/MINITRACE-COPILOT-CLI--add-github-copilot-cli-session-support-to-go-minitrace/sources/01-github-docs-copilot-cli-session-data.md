---
Title: Github Docs Copilot Cli Session Data
Ticket: MINITRACE-COPILOT-CLI
DocType: reference
Topics: [go-minitrace, copilot, conversion]
Status: active
Intent: source-evidence
---

## Introduction

Every time you use Copilot CLI, your session data is recorded on your local machine. This includes 
your prompts, Copilot's responses, the tools that were used, and details of files that were 
modified. By default, session data is also synced to your GitHub account. This lets you query all 
of your past sessions, including sessions from Copilot CLI, Copilot cloud agent, Copilot code 
review, VS Code, JetBrains, and the GitHub Copilot app.

This session data powers several features, available from Copilot CLI, VS Code, JetBrains, the 
GitHub Copilot app, and GitHub.com:

- **Querying your session history**: Ask natural language questions about your past work, and 
Copilot will search your session data for answers.
- **Resuming sessions**: Pick up where you left off in any previous session.
- **The `/chronicle` slash command**: A set of purpose-built subcommands that generate standup 
reports, personalized tips, cost analysis, and suggestions for improving your custom instructions, 
all derived from your session history.

With user-scoped queries, you can only query your own sessions—no one else can access your 
session data.

This conceptual article explains how session data is stored and synced, and how you can use it to 
enhance your workflow. For a practical guide to resuming a session, asking Copilot about your CLI 
sessions, and using the `/chronicle` slash command, see [Using GitHub Copilot CLI session 
data](https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/chronicle). For 
querying sessions from VS Code, see [Managing agent 
sessions](https://docs.github.com/en/copilot/how-tos/copilot-on-github/use-copilot-agents/manage-and
-track-agents).

## How session data is stored

Every Copilot CLI session is persisted as a set of files in the `~/.copilot/session-state/` 
directory on your machine. The data for each session contains a complete record of the session. 
These files allow you to resume an interactive CLI session.

In addition to the session files, Copilot CLI stores structured session data in a local SQLite 
database, referred to as the session store. This data is a subset of the full data stored in the 
session files. The session store is what powers the `/chronicle` slash command and it also allows 
Copilot to answer questions you ask about your past work.

## Session syncing

By default, Copilot CLI syncs your session data to your GitHub account. This lets you query your 
past sessions from Copilot CLI, VS Code, JetBrains, the GitHub Copilot app, or GitHub.com, and 
includes sessions from Copilot cloud agent, Copilot code review, VS Code, and the GitHub Copilot 
app.

You can opt out of session syncing by setting `"remoteExport": false` in your CLI settings JSON. If 
you opt out, session data stays on your machine and you can only query it from Copilot CLI.

For Copilot Enterprise and Copilot Business users, an organization administrator must set the 
"Store local sessions in the Cloud" policy to at least "View from cloud." If the policy is disabled 
or unconfigured, sessions are stored locally only.

### Privacy and data locality

Local session data is stored in `~/.copilot/session-state/` and is only accessible to your user 
account on that machine.

Synced session data is stored on GitHub and is tied to your personal account. It is accessible only 
to you by default. Organization and enterprise administrators control whether syncing is available 
through the "Store local sessions in the Cloud" policy, but enabling the policy does not give 
administrators access to your session data.

You can choose to share individual sessions, giving view-only access to others who have access to 
the repository. Shared sessions are not indexed for other users' session queries. For more 
information, see [Managing agent 
sessions](https://docs.github.com/en/copilot/how-tos/copilot-on-github/use-copilot-agents/manage-and
-track-agents#share-a-session).

Copilot reads session data when you ask questions about past interactions or use the `/chronicle` 
slash command. Session data such as your previous prompts, context, and responses may be sent to 
the AI model, just as in any normal Copilot CLI interaction.

### Managing your session data

You can manage your session data both locally and on GitHub.com.

#### Deleting sessions with the /session slash command

The simplest way to delete sessions is to use the `/session` slash command in an interactive CLI 
session.

- `/session delete` deletes the current session and starts a new one in its place.
- `/session delete SESSION-ID` deletes a specific session. This shows a preview first; add `--yes` 
to confirm, for example `/session delete SESSION-ID --yes`.
- `/session delete-all` deletes all of your local sessions except the current one. Add `--yes` to 
confirm: `/session delete-all --yes`. Sessions that are in use by another process are skipped.
- `/session prune --older-than DAYS` deletes sessions older than the specified number of days. Add 
`--dry-run` to preview what would be deleted.

When you delete a session that has been synced to your account, `/session delete` asks whether you 
also want to delete the synced (remote) copy. Deleting the synced copy also removes the session 
from your `/chronicle` insights and query results. The `/session delete-all` and `/session prune` 
subcommands only affect local sessions and do not delete synced data. To remove synced data for 
those sessions, manually delete it from GitHub.com.

For the full list of `/session` subcommands, see [GitHub Copilot CLI command 
reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)
.

#### Deleting session data manually

- **Local data**: To remove data for a particular CLI session locally, delete the relevant session 
directory from `~/.copilot/session-state/`. To clear all local session data, delete everything 
under `~/.copilot/session-state/`. After doing this you must manually reindex the session store. 
See [Reindexing the session store](#reindexing-the-session-store) later in this article. Deleting 
local files does not affect session data that has been synced to your account. You cannot delete 
synced data locally.
- **Synced data**: You can delete or hide synced CLI sessions from GitHub.com. Hiding a session 
removes it from your session index so it no longer appears in query results. Deleting a session 
removes it from your session list on GitHub.com. Deletion applies to CLI, VS Code, and GitHub 
Copilot app sessions.

## The /chronicle slash command

The `/chronicle` slash command provides purpose-built subcommands for generating insights from your 
session history, including standup reports, personalized tips, cost analysis, and suggestions for 
improving your custom instructions. For the full list of subcommands and usage examples, see [Using 
GitHub Copilot CLI session 
data](https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/chronicle#using-the-chr
onicle-slash-command).

## When to use session data

- **At the start of your day**: Run `/chronicle standup last 3 days` to generate a summary of 
recent work instead of reconstructing it manually.
- **Periodically, to level up**: Run `/chronicle tips` every week or two to discover features and 
workflow improvements you might be missing.
- **When Copilot keeps making the same mistake**: Run `/chronicle improve` to identify the pattern 
and generate custom instructions that make the agent more effective for your project.
- **To understand your token usage**: Run `/chronicle cost tips` to see where tokens are going and 
get suggestions for reducing costs.
- **To search for a specific topic**: Run `/chronicle search KEYWORD` to find sessions containing a 
specific term or topic. Unlike free-form questions, this searches session content directly for 
keywords rather than interpreting your query semantically.
- **To recall past work**: Ask a free-form question like "Have I worked on anything related to the 
payments API?" and Copilot will search your history.
- **To continue previous work**: Use `copilot --continue` or `copilot --resume` to pick up where 
you left off.

## Reindexing the session store

The session store is populated incrementally during a CLI session. Data for a session is written to 
disk in a session-specific subdirectory of `~/.copilot/session-state/`. This also happens 
periodically during a session, and also when the session ends.

You can reindex the session store from the session files on disk. Reindexing also syncs your 
session data to your account.

Situations where you might need to reindex include:

- **Indexing old sessions**: If you have old session files on disk that were created before the 
session store existed, reindexing will populate the session store with data from those sessions.
- **Session deletion**: To delete a session from your history, use the `/session delete`, `/session 
delete-all`, or `/session prune` slash commands. For a synced session, `/session delete` can also 
remove the synced copy, which removes the session from your `/chronicle` insights.
- **Migrating/recovering sessions**: If you moved your session files to another machine, or 
restored them from a backup, without also moving/restoring the session store file 
(`~/.copilot/session-store.db`), you can use the reindex command to recreate the session store.
- **File corruption**: If the session store file (`~/.copilot/session-store.db`) becomes corrupted, 
or is accidentally deleted, you can recover the session store from the session files.
- **Unexpected termination**: If a session terminates unexpectedly (for example, due to a crash or 
power loss) before data held in memory has been flushed to the session store you may be able to 
populate the session store with the missing data if it was written to disk, in the session files, 
prior to the termination.

To reindex the session store, use the following slash command in an interactive CLI session:

```
/chronicle reindex
```

## Further reading

- [Using GitHub Copilot CLI session 
data](https://docs.github.com/en/copilot/how-tos/copilot-cli/use-copilot-cli/chronicle)
- [Managing agent 
sessions](https://docs.github.com/en/copilot/how-tos/copilot-on-github/use-copilot-agents/manage-and
-track-agents)
- [GitHub Copilot CLI command 
reference](https://docs.github.com/en/copilot/reference/copilot-cli-reference/cli-command-reference)
