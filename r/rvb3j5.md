---
id: rvb3j5
title: "An idle hug mcp reloads into a new hug without waiting for a request"
state: "doing"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "000001"
created: 2026-11-25
updated: 2026-11-25
doing_since: "2026-11-25T21:32"
stage: "active"
model: "claude-opus-5-5"
branch: "rvb3j5"
worktree: "/Users/jhf/.jcode/scratch/wt-rvb3j5"
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
- [x] A test starts hug mcp, replaces its binary, sends nothing, and within 60 s the process runs the new binary and still answers on the same pipes
- [x] hug lists the running hug processes with their versions (doctor or version), and flags one older than the installed hug
- [ ] After the next release, no hug mcp on Jørgen's Mac runs an older hug after 2 minutes (crab checks with that command; Jørgen is not asked)

## Notes


check 1: TestMCPReloadsWhileIdleAfterBrewUpgrade (cmd/hug/reload_idle_test.go): brew-shaped Cellar/hug/0.0.1 + bin/hug symlink; hug mcp started from bin/hug; symlink moved to Cellar/hug/0.0.2, old keg removed; no request sent; 'reloaded as hug 0.0.2-new (was 0.0.1-old), same connection' at 10.0 s; lsof shows same pid on the 0.0.2 keg; next task_list answers 0.0.2-new on the same pipes. Failing-first 65dbca1f: 'no reload within 60 s of the upgrade, with no request sent'. Branch rvb3j5 a7a1e4a6.
check 2: hug version --running (cmd/hug/running.go): TestVersionRunningFlagsOlderHug starts 0.0.1-old and 0.0.2-new hug mcp, runs 0.0.2-new version --running: rows with pid, mcp, version, start, file; old row says 'older than this hug', exit 5, fix: line. Read-only on this Mac: 12 processes listed, 9 mcp + 3 tui + 1 task sync flagged (6 on Cellar 0.7.108/0.7.113, the rest on removed files). Unsupported OS: exit 1 + fix: line (TestVersionRunningUnsupported). Branch rvb3j5 a7a1e4a6.
rvb3j5 branch a7a1e4a6 (CI 45709839858). Design: 5 s stat timer on the launch path (follows brew's moved symlink like the exec does), file must hold still 1 tick, no request running, no half-read message; idle wait is poll(2) on fd 0 so no client byte is held in the old process; SIGHUP via self-pipe reloads at once, logs 'is unchanged' otherwise. hug upgrade does NOT yet send SIGHUP: hugs before this release exit on SIGHUP (Go default), and version --running cannot tell a reload-capable pid from an old one when macOS names its removed file '/hug'. Mutants 12/12 killed. Check 3 command: hug version --running (exit 0 = none stale; after brew upgrade wait 2 min).
Final SHA dc365801 on rvb3j5 and ci/rvb3j5. CI 45711314196 attempt 2: 8/8 success. Attempt 1 failed only TestBigTaskKeys (huginn/internal/tui, untouched by this branch): 'tab, Notes open on a 700 KB task: p50 54.815666ms, want under 50ms' at load 18.06. The earlier run 45709839858 on a7a1e4a6 failed TestVersionRunningFlagsOlderHug: the test matched 'older' inside /var/folders; dc365801 now matches the flag phrases.
crab review: landed dc365801 (CI 45711314196, 8/8 jobs success) as main e51f93d9, v0.7.115; release.yml 45714231008 success; brew ../Cellar/hug/0.7.115/bin/hug, 'hug v0.7.115'. Mutants 12/12 killed (unicorn). Check 3, first look 22:43Z (2 min after install): 'hug version --running' exit=5, 12 processes, all started before v0.7.115: mcp 1402 and 91085 '?' (/hug removed), 8265 0.7.113, 95583 0.7.114, 40384/49138/58480/62272/96140 0.7.108 (Claude desktop's, since 11-22); tui 32504/19901/90079 '?' (since 11-23, tracked by mmfw71). Pre-0.7.115 hugs cannot reload themselves; each needs one restart by its app (Claude relaunches itself on update). crab rechecks with the same command; Jørgen is not asked.
landed 65dbca1 on main 2026-11-25: hug mcp: failing tests, idle reload after brew upgrade, SIGHUP, hug version…
landed a3e8a3a on main 2026-11-25: hug mcp reloads while idle; SIGHUP reloads now; hug version --running (rvb3j5)
landed d6b6b5d on main 2026-11-25: hug mcp: test that a cancelled request lets the idle reload go ahead (rvb3j5)
landed a7a1e4a on main 2026-11-25: hug version --running: flag a process whose hug file was removed or replaced…
landed dc36580 on main 2026-11-25: hug version --running test: match the flag phrases, not 'older' (/var/folders)…
shipped in v0.7.115 (65dbca1)
handed on after v0.7.115: check 3 open, jcode-rvb3j5's active work
## History
- 2026-11-25T21:32 todo -> doing/active (take)
