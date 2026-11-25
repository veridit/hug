---
id: vrx1ja
title: "Claude desktop: hug mcp can take over 120 s to answer initialize"
state: "doing"
type: bug
priority: "high"
assignee: ["vrx1ja-engineer"]
labels: []
group: ""
rank: "000000i"
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
- [x] The cause of the 46 s and >120 s starts is named with evidence (which step waited, how long)
- [ ] Jørgen sees no 'kunne ikke startes' popup for a week of releases

## Notes


Check 2 evidence (hippo, 2026-11-25). Setup: a copy of the live store made with sqlite3 .backup into the scratch folder vrx1ja-store (HUG_HOME pointed there, live store never written), hug v0.7.112 (c1056b46) with throwaway per-step timers, load 8-13.
Baseline: initialize 0.15-0.26 s. Memory open takes 0.12-0.23 s: Bound 0.05-0.18 s, store.Open 0.002-0.007 s, pickEmbedder 0.03-0.05 s, task RegisterMCP 0.001 s.
Write lock held by another process (BEGIN IMMEDIATE), nothing to migrate: initialize 0.185 s. Not the cause on its own.
Ollama down: pickEmbedder 0.001 s. Ollama answering only after 10 s: pickEmbedder 1.502 s (capped by Available's 1.5 s timeout), initialize 1.588 s.
Init lock muninn.db.init.lock held 20 s: 'VRX store: init lock 18.897s', initialize 19.268 s. flock has no timeout.
THE SLOW STEP: store.Open's schema exec runs while another process holds the write lock, on a store that needs schema added (a release that adds a table; on the copy holds_ended was dropped). Result: 'VRX store: schema exec 30.318s', then 'hug mcp: memory unavailable: schema: database is locked (5) (SQLITE_BUSY)', initialize 30.398 s. It waits busy_timeout (30 s) WHILE HOLDING the init lock, so openers queue. 4 hug mcp started together answered initialize at 30.437 s, 60.538 s, 90.627 s and 120.730 s ('init lock 30.209s / 60.303s / 90.392s' plus 'schema exec ~30.3s' each). The 4th is past Claude's 120 s.
Who holds the write lock that long: every hug mcp start runs a background Reindex inside one BEGIN IMMEDIATE. A full pass on the copy held it 409.86 s ('6m48.103s ... neighbors=6m14.24s').
Timing of the failures: a release execs the new hug in every jcode hug mcp (jcode logs show the reloads; 9 hug mcp were running on 11-24). Claude restarts its hug only when it relaunches itself for a stealth update. main1.log 11-21: 16:27:22 'stealth-update', 16:27:30 'Launching MCP Server: hug', 16:30:31 'Failed to connect to hug: Request timed out ... timeout: 60000', then retries at 120000. All 4 failing starts follow such a relaunch.
Gap: Claude does not log server stderr, so nothing records which process held the lock at each past failure. What is reproduced is the mechanism, up to 120.7 s: initialize waits behind memory open (the init lock, then 30 s per locked statement, one opener after another).
The task cache SchemaVersion is not on the start path: RegisterMCP takes 0.001 s and the cache builds on the first task call.
Failing-first test pushed: 841b2721 on vrx1ja, cmd/hug/mcp_open_test.go TestMCPAnswersBeforeMemoryOpens (memory db write-locked by the test, Ollama answering after 3 s). On main code: '--- FAIL: TestMCPAnswersBeforeMemoryOpens (3.03s)' 'mcp_open_test.go:142: hug mcp was not ready to answer initialize after 1 s: it waited 2.659s for memory'.
check 2: Slow step: store.Open's schema exec waits busy_timeout (30 s) on a write-locked muninn.db WHILE holding muninn.db.init.lock, when a release adds schema. Store copy: 'VRX store: schema exec 30.318s' then 'memory unavailable: schema: database is locked (5) (SQLITE_BUSY)'; 4 hug mcp started together answered initialize at 30.437/60.538/90.627/120.730 s (init lock 30.2/60.3/90.4 s each). Holder: hug mcp's start-up Reindex runs in one BEGIN IMMEDIATE, a full pass held it 409.86 s on the copy. Held init lock alone: 19.268 s for a 20 s hold. Write lock with nothing to migrate: 0.185 s. Slow Ollama: +1.5 s (probe cap). Full numbers in the 2026-11-25 hippo note.
## History
- 2026-11-25T12:54 todo -> doing/active (take)
