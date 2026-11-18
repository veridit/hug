---
id: z77pem
title: "hug names running MCP servers too old to reload themselves"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Any hug run that talks to an agent (hook, mcp, mem/task write) notices a hug mcp process on this machine running a binary older than v0.7.54 (no reload) or a removed Cellar path, and says once per session which pids and how to restart them, with a fix: line

## Purpose
Statbus 2026-11-18: memory_remember failed twice with 'cannot modify graph_state because it is a view' from pid 3175, hug 0.7.2 started 15 Nov; the agent could not tell why. ps shows 6 such servers (0.7.2, 0.7.14, 0.7.43, 0.7.52); mbdfzs reload only helps servers started on v0.7.54+

## Principle
Say it, never kill another app's process; one line, once per session; the fix names the client's own restart (jcode mcp reload hug / restart the app)

## Relevant details
Detect via ps + the executable path (Cellar version or deleted file). macOS/Linux first; Windows prints nothing. Also: an old writer hitting the graph_state view gets SQLite's bare error; that cannot change in old binaries, so the message must come from the new hug side.

## Depends on

## Acceptance
- [ ] A test starts a fake old hug mcp (binary path under a Cellar version < 0.7.54) and the hook prints one line naming its pid and a fix: line
- [ ] On Jørgen's machine the line names the stale servers, and after they restart it is gone

## Notes

crab 18:20 local: mcp reload server=hug in crab's session replaced pid 15206 (0.7.14) with 18560 (Cellar 0.7.66): the restart fix works. Still stale: 3175 (Statbus, 0.7.2), 56816 truth_tribe (0.7.2), 65214 GenderedGames (0.7.2), 77273 Claude.app (0.7.43), 90712 (0.7.52).
