# Bridge: CLI agent <-> Claude app agent

Two agents work on muninn + huginn together:

- **cli**: Claude Code CLI (or jcode) on the Mac. Full machine: builds, tests, git, releases.
- **app**: Claude in the Claude desktop app (cloud session linked to the Mac). Drives the
  user-facing test of the tools from inside a restricted harness.

The goal: one coherent muninn/huginn that is useful from the CLI, from other harnesses, and
from the app with its limits. The cli agent drives; the app agent prototypes and reports what
works from its side.

## How to talk

One file per message in `msgs/`, never edit someone else's file (the app cannot delete):

    msgs/<UTC yyyymmddThhmmssZ>-<from>.md      e.g. msgs/20261110T223000Z-cli.md

First line: `to: app` or `to: cli`, then `re: <file>` if it answers one. Keep it short:
what you did, what you need, one question at most. Long material goes in a separate file
next to it and gets linked.

Read new messages: `ls -1 msgs/ | sort | tail`.

## Waking the other side

- cli -> app: the app agent only runs when the user sends it a message. For now the user
  says "check the bridge". (Planned: an app-side mod that polls `msgs/` on a timer and wakes
  the app agent itself, see app constraints below.)
- app -> cli: the cli agent polls `msgs/` itself, or the user tells it.

## App agent constraints (verified 2026-11-10)

- Shell: Linux VM on the Mac that sees ONLY `~/Projects/muninn` and `~/Projects/huginn`
  (same files, live). No `~/.local/bin`, no `~/Library`, no worktrees outside these two.
  muninn/huginn binaries are not runnable there.
- Cannot delete files, so cannot `git commit` either (git needs to delete lock files).
  The cli agent commits.
- MCP works and runs on the Mac, started by the Claude app from
  `claude_desktop_config.json`: huginn (`list`, `next`, `take`, `update`, `done`, ...) and
  muninn (`memory_search`, `memory_recall`, `memory_remember`, `memory_forget`,
  `memory_related`, `memory_stats`). muninn appeared some time after huginn.
- `memory_stats` reports `binding: jcode@global` and no muninn project scope, so the app's
  muninn server is probably not bound to `~/Projects/muninn` (cwd or env not reaching it).
  Worth checking.
- No automatic recall here: the app has no UserPromptSubmit hook, so nothing is inserted
  unless the agent calls `memory_recall`. Verified with "banana mode": the hook block did not
  appear; `memory_recall` on the same text returned k0ya4p, vbqmx6, z1q348.
- Possible fix: an app "mod" (hooks module in the session's cloud container) has
  `prompt.submit` (can attach `context` before the model sees the message), `turn.complete`
  (sees the answer: extraction / save cue), `$.clock` timers, and `$.mcp.call(server, tool,
  args)` which can reach the Mac-side muninn through the device relay. Not built yet. It
  lives in the session's container unless packaged as a plugin.
