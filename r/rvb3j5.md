---
id: rvb3j5
title: "An idle hug mcp reloads into a new hug without waiting for a request"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
Every running hug mcp runs the installed hug within a minute of an upgrade, whether or not a request arrives, proven by a test.

## Purpose
Jørgen 2026-11-25 on chxk50: 'MCP should always reload automatically ... Surely this is something you have automated tests for'. At 12:00Z five hug mcp processes from 11-22 still ran v0.7.108 after v0.7.112 was installed.

## Principle
The machine proves it, not Jørgen: an automated test, and a check hug runs itself; never ask him to restart processes.

## Details
Today (cmd/hug/reload.go, mbdfzs): the reloader checks the binary only before it passes on a NEW request, so a server with no traffic keeps the old hug forever. Seen 2026-11-25 12:00Z via lsof on the txt mapping: pids 40384, 49138, 58480, 62272, 96140 (Claude desktop's, started 11-22 22:50) on Cellar/hug/0.7.108; 95583 on 0.7.112; 1402, 8265, 91085 unreadable. Three hug tui (32504, 19901, 90079) since 11-23. Fix: also watch the binary on a timer (or fsnotify) and reload while idle, with the same drain and replayed initialize; old binaries' processes cannot be fixed from new code, so hug upgrade/brew post-install could signal them (SIGHUP to reload) if they understand it. Also: hug doctor (or hug version --running) lists running hug processes and their versions, so no one needs lsof.

## Depends on

## Acceptance
- [ ] A test starts hug mcp, replaces its binary, sends nothing, and within 60 s the process runs the new binary and still answers on the same pipes
- [ ] hug lists the running hug processes with their versions (doctor or version), and flags one older than the installed hug
- [ ] After the next release, no hug mcp on Jørgen's Mac runs an older hug after 2 minutes (crab checks with that command; Jørgen is not asked)

## Notes
