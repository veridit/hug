---
id: mbdfzs
title: "hug mcp serves with the installed hug, never an old binary"
state: "doing"
type: bug
priority: "urgent"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzll"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T02:44"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "check 3: crab's ps check after the next brew upgrade (8dee4a86 released); CI run 44223950419"
waiting_since: "2026-11-18T03:38"
waiting_asked_by: "kestrel"
---

## Goal
A running hug mcp server serves every request with the hug installed now, re-executing itself when the binary changes.

## Purpose
kestrel 2026-11-18 (3nhp67): commit 5408358f on hug/tasks was written by an older hug's resume rule. ps at 02:32 local shows 8 hug mcp servers started 15-17 Nov (pids 3175, 9014, 13638, 15206, 56816, 65214, 77273), each still running the binary it started from while brew has installed v0.7.52. An old writer undoes new rules (stage fields, lint at write) and, after w40cmk, would write the old layout into a migrated board.

## Principle
One hug writes a board at a time: the installed one. The re-exec happens between requests, never mid-write, and the client sees no disconnect. No version negotiation or compatibility layer (hug upgrade migrates).

## Relevant details
A running hug mcp server serves every request with the hug that is installed now: when the binary on disk changes, it re-executes itself between requests, keeping its stdio connection, so no write comes from an older version.

Check os.Executable()'s resolved target (the Homebrew symlink moves to the new Cellar path) or its inode/mtime before each request; on change, finish the reply, then syscall.Exec the new binary with the same args and fds. Windows has no exec: there, spawn the new server and proxy, or exit and let the client restart; pick one and say which on the ticket. Must land before w40cmk.

## Depends on

## Acceptance
- [x] A test starts hug mcp, replaces the binary under it, sends a request, and the reply comes from the new binary (version field) on the same connection (fails first on main)
- [x] A request in flight when the binary changes completes with the old binary; the next one runs on the new (test)
- [ ] After the next brew upgrade, ps shows no hug mcp whose binary is older than the installed version (crab checks on Jørgen's machine)

## Requests
- 2026-11-20T13:05 request 1 by jhf:
  This one should probably have been checked at this point because it's still waiting, and I think we've made many releases since.

## Notes

Jørgen 02:39 local: 'auto-reloading the MCP is immensely important, good catch, get it out.' Ruling (crab): mbdfzs goes first for kestrel, ahead of 3nhp67, and ships in the first hourly release after it is green.


Landed 8dee4a86. Windows choice: exit cleanly between requests. Each request already read gets JSON-RPC error -32000 'hug was upgraded while this server ran: it stops now so the app starts the installed hug; run the call again', then exit 0, and the client restarts the server. No proxy (crab scope 02:40). Fails first: with main's StdioTransport path, TestMCPReloadsChangedBinary fails with 'reload_test.go:141: after the upgrade: version "0.0.1-old", err <nil>'. Mutants, 12 run, 11 killed: never-reload, no-drain (check 2: 'running request: version "", err context deadline exceeded'), no-replay, replay-reply-leaks, held-request-lost, inflight-never-clears, windows-no-answer, main. 'reload always' was not reached before the 600 s cap. 1 did not compile and was redone as a compiling variant, which was killed. Race on cmd/hug: ok 28.2s. Servers started before this release need one app restart.
landed 8dee4a8 on main 2026-11-18: hug mcp: reload the installed hug between requests on the same connection…
Mutant 'reload always' (ignore start identity) killed: 'reload_test.go:89: connect: context deadline exceeded' in TestMCPReloadsChangedBinary and TestMCPNoReloadWhenUnchanged. Now 12 of 12 killed.
shipped in v0.7.54 (8dee4a8)
crab 18:19 local, check 3 FAILS on Jørgen's machine with v0.7.66 installed: hug mcp pids 3175 (ssb/.jcode, Statbus), 56816, 65214 run Cellar 0.7.2; 15206 0.7.14; 77273 0.7.43 (Claude.app); 90712 0.7.52. All started before v0.7.54, so they predate the reload and can never reload. Statbus hit it: memory_remember -> 'cannot modify graph_state because it is a view' (the 4vd1mz guard refusing a pre-v0.7.59 writer, as designed). Follow-up filed for servers older than the reload.
check 3: commit 09462c2e (branch mbdfzs3, on origin/main 939636ff). CI 44388310353 attempt 2: all jobs success (ubuntu, windows, macos, windows-cli rest+sync+shards, bench, secrets). Attempt 1 macOS job was cancelled with 0 steps after sitting queued 22 minutes with no runner; the rerun passed 20:42-20:47Z. Tests failed first: TestOldHugIsToldToRestart. 5 of 5 mutants killed (mbdfzs3/mut/mutants.txt). Live v0.7.2 evidence: scratch mbdfzs3/live-*.txt.
landed 056033a on main 2026-11-18: An old hug mcp server's failed memory write names its fix: restart it (mbdfzs…
shipped in v0.7.71 (056033a)
crab 2026-11-20 08:21Z, ps + lsof on Jørgen's machine (installed hug v0.7.87): 6 hug mcp processes. 2 still run Cellar/hug/0.7.2 (pids 56816 and 65214, started 15 Nov, --jcode-home truth_tribe_and_treasure and GenderedGames); 1 runs 0.7.87 (96239); 3 show a deleted binary path '/hug' (1402, 8265, 71259: their Cellar folder was removed by brew cleanup, so they run an old version). The fix (8dee4a86) shipped in v0.7.54; servers started before v0.7.54 cannot re-exec themselves, which is z77pem's case. Check 3 does not hold yet: those jcode instances must restart their MCP servers.
crab 2026-11-20 12:04Z ps: 4 hug mcp still run binaries older than the installed 0.7.90: pids 56816 (truth_tribe_and_treasure, since 11-15), 65214 (GenderedGames, 11-15), 1402 (~/.jcode, 11-18), 8265 (Brovs, 11-19). Check 3 still fails; those sessions need a restart.
## History
- 2026-11-18T02:44 todo -> doing/active (take)
- 2026-11-18T03:38 doing/active -> doing/waiting (wait)
- 2026-11-19T04:22 handed to crab: mizaru's session ended (closed 2026-11-19 03:11)
- 2026-11-20T13:05 request 1 by jhf
- 2026-11-20T13:05 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
