---
id: vrx1ja
title: "Claude desktop: hug mcp can take over 120 s to answer initialize"
state: "doing"
type: bug
priority: "high"
assignee: ["vrx1ja-engineer"]
labels: []
group: ""
rank: "zzzzzzzzzzzj"
created: 2026-11-24
updated: 2026-11-24
doing_since: "2026-11-25T12:54"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug mcp answers initialize within a second, whatever memory, the database or the machine load is doing.

## Purpose
Jørgen 2026-11-24 14:56 local: Claude desktop popup 'hug kunne ikke startes for Cowork- og Code-økter'. Claude gives up after 120 s and its retries fail too.

## Principle
The handshake never waits on memory: open the store, embedder and binding after answering, and report a slow or failed memory in the tools, not by hanging.

## Details
Claude's mcp-server-hug.log: 7 starts from 11-14 to 11-22; 4 never got an initialize result within 120 s (11-14 00:03Z, 11-17 17:48Z, 11-19 00:04Z, 11-21 15:27Z), each followed by 'Couldn't start for Cowork and Code sessions. Error: Request timed out' (shared-pool) retries. Successful starts answered after 0.6 s, 3.5 s, 7 s and 46 s (11-20 19:16:43 to 19:17:29). crab's probe at load 143 (11-24 13:57Z): 0.52 s. Before s.Run, cmd/hug/mcp.go:55 newMCPServer runs muninn.OpenMCP: config, store.Open (busy_timeout 30 s per statement, schema exec), pickEmbedder, Bound, SpawnSelf; huginn RegisterMCP runs cli.SpawnSelf. 9 hug mcp processes were running at 13:58Z. Each failure came soon after a hug release (Claude restarts the server). No log line for today's popup was found (mcp-server-hug.log ends 11-22 21:50Z).

## Depends on

## Acceptance
- [ ] A test holds the memory database write-locked (and makes the embedder slow) and hug mcp still answers initialize within 1 s; tools/list lists the task tools; a memory tool says memory is still opening, with a fix: line
- [ ] The cause of the 46 s and >120 s starts is named with evidence (which step waited, how long)
- [ ] Jørgen sees no 'kunne ikke startes' popup for a week of releases

## Notes


## History
- 2026-11-25T12:54 todo -> doing/active (take)
