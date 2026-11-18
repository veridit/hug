# huginn

<img src="docs/logo.svg" alt="Huginn, raven of thought" width="160">

A git-branch task tracker for people and agents: the tasks part of
[hug](../README.md), used as `hug task`. Markdown is the source of truth, Git is
the sync protocol, and the CLI, terminal UI and MCP server share the same
operations. Huginn, Odin's raven of thought, gives this part its folder name
and its mascot: what to do next, why, and how to know it is done. Its sister is
[muninn](../muninn/README.md), memory.
Licensed under MPL-2.0. Contributions need the contributor agreement in
[CONTRIBUTING.md](CONTRIBUTING.md).

## Install

Requires Git 2.42 or newer on PATH. No database service or account needed.
Install [hug](../README.md#install):

```sh
brew install veridit/tap/hug
hug version
```

## 60-second quickstart

Run inside an existing Git checkout. `take` needs the model doing the work
(`--model claude-opus-5-5`, or `HUG_MODEL`; a person gives `--model human`); a
name (`--as poodle`, `HUG_USER` or the agent session) is optional, and without
one the assignee is the model. `git config user.name` is never used. With an
origin, `take` claims through a remote push lock.
Without an origin, everything works locally.

```sh
hug task init
hug task new "Ship greeting" --goal "Visitors see a greeting" --purpose "First visit feels welcome" --principle "Plain text only" --accept "Greeting test passes"
hug task next
hug task take ship-greeting --model human
# Implement and test the change, then record the observed check:
hug task check ship-greeting 1
hug task done ship-greeting
```

Use `hug task board` to print todo, doing and done side by side (`--all` for every
done task), `hug tui` for an interactive view, and `hug task help COMMAND` for flags.
Dependencies and unchecked acceptance items block completion. `drop` preserves
history instead of deleting a task. Ranking and priority are separate concepts.

## Files, branches and groups

Tasks are `[rank.]id.slug.md`, with a stable six-character ID and readable slug.
They live on the dedicated `hug/tasks` branch, mounted as the `.hug/tasks/` Git
worktree inside your checkout. Folders are groups, not statuses. `hug task mv ID auth`
moves a task to the auth group. `hug task move ID --before OTHER` changes its rank.
Edit Markdown directly or use the CLI. Mutations commit the task branch, not
your application branch, and reach the other clones by themselves (`hug task sync`
does it at once). The SQLite cache under the Git common directory is disposable.

## Agents

Use `hug task COMMAND --json` for structured replies. Every reply includes a
`next` hint. Errors include an actionable `fix` and `exit_code`.

| Exit | Meaning |
|---|---|
| 0 | Success |
| 1 | Operational error |
| 2 | Usage error |
| 3 | Blocked |
| 4 | No next task |
| 5 | An upgrade is pending (`hug upgrade` previews it) |

`hug mcp` speaks MCP over stdio with the `task_*` and `memory_*` tools; start it
in the project checkout. `hug apps install` adds it to Claude Desktop, Claude
Code and ChatGPT/Codex. For Jcode, installation keeps backups and preserves other
MCP servers:

```sh
hug task install jcode --dry-run
hug task install jcode
hug task install jcode --all-instances
hug task install jcode --jcode-home /path/to/.jcode
hug task uninstall jcode
```

Restart the agent after changing configuration. The installer records the
absolute path of the executable with `args: ["mcp"]` and `env: {}`.
It preserves an existing `servers` or `mcpServers` schema, defaults to `servers`,
and repairs v0.1.0 mixed-schema files without dropping other servers. Re-run the
installer after upgrading if the executable path changes. MCP tools use
the same validation, completion checks and next-step hints as the CLI.

## Import backlog-md

```sh
hug task import backlog ./backlog
hug task lint
hug task board
```

The backlog folder or its `tasks/` folder both work. Archived tasks stay out
unless you name their folder.
The source is read-only. Source IDs remain aliases, dependencies are resolved,
and repeat imports conservatively preserve local edits. See
[importer details](internal/importer/README.md) for mapping and merge rules.

## Agent evaluation

D13 cheap-model gate at `26e82b5`: **238/240**, every cell at least **2/3**.
Four models, ten scenarios, three trials, CLI and MCP. State-only grading,
concurrency 4. These are the measured results, not a guarantee for other models.

# Huginn agent eval

Bar: every cell >= 2/3. State only. Concurrency 4.

## cli

| Model | create-bug | dependency | next-take | finish-by-slug | blocked-finish | reorder | move-group | drop-duplicate | priority-bug | sync |
|---|---|---|---|---|---|---|---|---|---|---|
| gpt-6-luna | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| claude-haiku-5-5 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| claude-sonnet-5-5 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| deepseek-flash | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |

## mcp

| Model | create-bug | dependency | next-take | finish-by-slug | blocked-finish | reorder | move-group | drop-duplicate | priority-bug | sync |
|---|---|---|---|---|---|---|---|---|---|---|
| gpt-6-luna | 3/3 | 3/3 | 3/3 | 3/3 | 2/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| claude-haiku-5-5 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| claude-sonnet-5-5 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 |
| deepseek-flash | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 3/3 | 2/3 | 3/3 | 3/3 | 3/3 |

PASS

## Design

See [DESIGN.md](DESIGN.md) for architecture and design decisions.
