---
id: z77pem
title: "hug names running MCP servers too old to reload themselves"
state: "doing"
type: feature
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzxr"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T18:23"
stage: "waiting"
model: "claude-opus-5-5"
branch: "ec6z6j"
worktree: "/Users/jhf/.jcode/scratch/ec6z6j"
waiting_on: "check 2: release with 67d46348 and Jørgen's machine shows the stale-server line (crab stacks z77pem)"
waiting_since: "2026-11-18T20:44"
waiting_asked_by: "mizaru"
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
- [x] A test starts a fake old hug mcp (binary path under a Cellar version < 0.7.54) and the hook prints one line naming its pid and a fix: line
- [ ] On Jørgen's machine the line names the stale servers, and after they restart it is gone

## Notes

crab 18:20 local: mcp reload server=hug in crab's session replaced pid 15206 (0.7.14) with 18560 (Cellar 0.7.66): the restart fix works. Still stale: 3175 (Statbus, 0.7.2), 56816 truth_tribe (0.7.2), 65214 GenderedGames (0.7.2), 77273 Claude.app (0.7.43), 90712 (0.7.52).


mizaru: built on 4fb78339, commit baa6d5ea (branch z77pem, ci/z77pem). Check 1: TestHookNamesStaleMCPServers starts fake hug mcp servers at Cellar/hug/0.7.2 and Cellar/hug/0.7.54; the session's first jcode turn shows 'pid N hug 0.7.2 in jcode /srv/statbus/.jcode' plus fix: 'mcp action=reload server=hug' with its recall; 0.7.54 is not named; the second turn says nothing; the old pid still runs (signal 0). The Claude/Codex prompt hook says it once per session too. Fails first on 4fb78339's main.go (stalemcp_test.go:101, :144). Mutants 4/4 killed: z77pem-mut/mutants.txt. Smoke on this machine 17:41 named exactly 3175, 56816, 65214 (0.7.2), 77273 (0.7.43 Claude), 90712 (0.7.52) in 116 ms; 56357 (Brovs, started 13:08 after v0.7.54, lsof txt '/hug') is not named. Check 2 waits on the release + Jørgen's machine.
crab 18:48 local, review of baa6d5ea: detection, once-per-session, never-kill and tests accepted; asked for one change: tell the agent only about the server serving its own session (with its own fix line), log the others (hook.log + stats) instead of listing every app's servers to every session. CI 44362346273 cancelled for the respin.
mizaru: reworked per crab's review, 3197d0b8 (replaces baa6d5ea) on 4fb78339, ci/z77pem run 44363998313. The agent hears only of the hug mcp serving its own session (jcode: its jcode home; Claude/Codex: the app the hook runs under; pid 1 never counts), one line with fix: mcp action=reload server=hug. Other apps' stale servers, or a session whose server cannot be identified: no line, only hook.log; hug mem stats lists all (stale_mcp). Smoke 17:53: ~/.jcode session hears nothing (18560 at 0.7.66), an ssb session would hear pid 3175 (0.7.2), seid's would hear 90712 (0.7.52). Tests fail first with 4fb78339's main.go; 8/8 mutants killed (z77pem-mut/mutants.txt).
crab 18:56 local: rework 3197d0b8 accepted (own stale server -> one line + its own fix; others -> hook.log + hug mem stats stale_mcp; unidentified -> silent; TestHookSilentWhenOnlyOthersAreStale; 8/8 mutants). CI 44363998313; stacks after 8c26ec51.
CI fixes after 3197d0b8 went red: HUG_TEST_FAKE_MCP added to env.Known (Test kind); the prompt-hook test marshals its JSON (a Windows TempDir broke the hand-built string); on systems without ps/lsof/proc (Windows) the first turn does nothing at all, with no scan and no marker (staleMCPListed; TestStaleMCPNothingWhereProcessesAreNotListed, mutant m9 killed). The injected-scan tests now run on Windows too. CHANGELOG moved back to Unreleased after the rebase. Head 67d46348 on origin/main 477696be, branches z77pem + ci/z77pem. CI 44377886910 completed success 6/6 (macOS sat queued, was cancelled, rerun 19:38:58-19:43:49Z success).
crab 20:44 local: 67d46348 accepted on review (CI 44377886910 6/6 incl. macOS rerun; Windows failure was the test's string-built JSON, now json.Marshal; scan gated to darwin/linux, mutant m9 killed). Stacks after 9163f102.
landed 24e329e on main 2026-11-18: hug names an MCP server too old to reload itself, to its own session (z77pem)
shipped in v0.7.71 (24e329e)
## History
- 2026-11-18T18:23 todo -> doing/active (take)
- 2026-11-18T20:44 doing/active -> doing/waiting (wait)
