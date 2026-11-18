---
id: mbdfzs
title: "hug mcp serves with the installed hug, never an old binary"
state: "doing"
type: bug
priority: "urgent"
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T02:44"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A running hug mcp server serves every request with the hug that is installed now: when the binary on disk changes, it re-executes itself between requests, keeping its stdio connection, so no write comes from an older version.

## Purpose
kestrel 2026-11-18 (3nhp67): commit 5408358f on hug/tasks was written by an older hug's resume rule. ps at 02:32 local shows 8 hug mcp servers started 15-17 Nov (pids 3175, 9014, 13638, 15206, 56816, 65214, 77273), each still running the binary it started from while brew has installed v0.7.52. An old writer undoes new rules (stage fields, lint at write) and, after w40cmk, would write the old layout into a migrated board.

## Principle
One hug writes a board at a time: the installed one. The re-exec happens between requests, never mid-write, and the client sees no disconnect. No version negotiation or compatibility layer (hug upgrade migrates).

## Relevant details
Check os.Executable()'s resolved target (the Homebrew symlink moves to the new Cellar path) or its inode/mtime before each request; on change, finish the reply, then syscall.Exec the new binary with the same args and fds. Windows has no exec: there, spawn the new server and proxy, or exit and let the client restart; pick one and say which on the ticket. Must land before w40cmk.

## Depends on

## Acceptance
- [ ] A test starts hug mcp, replaces the binary under it, sends a request, and the reply comes from the new binary (version field) on the same connection (fails first on main)
- [ ] A request in flight when the binary changes completes with the old binary; the next one runs on the new (test)
- [ ] After the next brew upgrade, ps shows no hug mcp whose binary is older than the installed version (crab checks on Jørgen's machine)

## Notes

Jørgen 02:39 local: 'auto-reloading the MCP is immensely important, good catch, get it out.' Ruling (crab): mbdfzs goes first for kestrel, ahead of 3nhp67, and ships in the first hourly release after it is green.


Landed 8dee4a86. Windows choice: exit cleanly between requests. Each request already read gets JSON-RPC error -32000 'hug was upgraded while this server ran: it stops now so the app starts the installed hug; run the call again', then exit 0, and the client restarts the server. No proxy (crab scope 02:40). Fails first: with main's StdioTransport path, TestMCPReloadsChangedBinary fails with 'reload_test.go:141: after the upgrade: version "0.0.1-old", err <nil>'. Mutants, 12 run, 11 killed: never-reload, no-drain (check 2: 'running request: version "", err context deadline exceeded'), no-replay, replay-reply-leaks, held-request-lost, inflight-never-clears, windows-no-answer, main. 'reload always' was not reached before the 600 s cap. 1 did not compile and was redone as a compiling variant, which was killed. Race on cmd/hug: ok 28.2s. Servers started before this release need one app restart.
## History
- 2026-11-18T02:44 todo -> doing/active (take)
