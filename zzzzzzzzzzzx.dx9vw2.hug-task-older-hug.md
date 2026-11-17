---
id: dx9vw2
title: "hug task: an older hug says the board is newer, with an upgrade fix"
state: "done"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T21:57"
---

## Goal
When the hug binary is older than the board it reads (a task file says state:, the binary knows only status:), every command says so once, with fix: brew upgrade veridit/tap/hug (or hug upgrade from a source build), instead of failing on an empty state.

## Purpose
2026-11-17 19:20: hatchling's hug (built 14:36) ran 'hug task wait s6ff5k' on the board after abcf36b (status: -> state: in 126 tasks) and got 'not-allowed s6ff5k:  -> doing/waiting is not a move a command makes' (empty state) with fix 'hug task take', which sent it the wrong way. Note and attach from the old binary landed, so the wrong state was silent elsewhere.

## Principle
One clear line naming both sides (board layout vs this hug's version), not a lint per task. A newer hug reading an older board keeps working (old-layout lint already covers that side).

## Relevant details
Reflection on ce3dmq (2026-11-17). Possible signal: a board-level layout version (hug.toml or a marker on hug/tasks) that hug compares to the one it knows; or a task file with an unknown front-matter key that is a known future key. Crab 19:41: small bug, after typq1e.

## Depends on

## Acceptance
- [x] test: an old-layout binary (simulated layout version) on a newer board: a write command fails with 'this board is newer than this hug' and fix: brew upgrade veridit/tap/hug, before any write (test fails before)
- [x] read commands warn once on stderr and still print

## Notes


file:fail-before.txt (700 bytes, sha256:87239cef) TestOlderHug* (checks 1, 2 only) on origin/main 4fb0c92: note exits 0 and writes, ls warns 0 times
hatchling 20:35. Landed 8856b2b (4fb0c92..8856b2b), main CI 44170743115 running. Design: hug.toml top-level layout = N on hug/tasks (no key = 2, the state: board); task.Layout = 2 is what this hug knows; App.Layout lets a test play an older/newer hug. Gate in Execute after openBoard, before the board turn: every non-readOnly command (incl. take, tui, edit, sync) returns 'this board is newer than this hug: the board has layout 3 (hug.toml on branch hug/tasks), this hug knows layout 2 ...; it writes nothing to it' + fix: brew upgrade veridit/tap/hug; deduce pass and hug upgrade also stop. Reads: one stderr warning + fix line, or board_newer in --json/MCP replies. hug upgrade writes layout when the hug's is newer (TestUpgradeRecordsBoardLayout). Check 1: TestOlderHugRefusesToWriteNewerBoard (note, take, wait, new, set, sync: exit 1, HEAD unchanged, file unchanged, worktree clean). Check 2: TestOlderHugWarnsOnceOnRead (ls, show, next: exit 0, task printed, warning exactly once on stderr, none on stdout). Fail before: attached fail-before.txt (origin/main 4fb0c92: note exits 0 and writes; ls 0 warnings). -race cli 147.3 s ok, task 60.3 s ok; mcp, cmd/hug ok. Limit: guards the next layout change only; a hug from before 8856b2b has no check (CHANGELOG says so).
landed 8856b2b on main 2026-11-17: hug task: a hug older than its board writes nothing to it and says so, with fix…
file:fail-before-2.txt (324 bytes, sha256:e982f19c) TestUpgradeMarksStateBoardLayoutTwo on 8856b2b: upgrade preview exit 0, tasks: nothing to do (no layout written on a state: board)
hatchling 20:43, crab 20:35 question: on 8856b2b, hug upgrade --apply on a state: board wrote nothing (no key = 2 = known); attached fail-before-2.txt. Fixed in 9c9d4c4 (8856b2b..9c9d4c4, main CI 44172097835): BaseLayout 1 (no key, from before it), Layout 2 = the state: rename. upgrade --apply moves tasks to state: then commits 'hug upgrade: board layout 1 -> 2'; init writes layout = 2 on an empty board. Test TestUpgradeMarksStateBoardLayoutTwo (preview exit 5 'tasks: would: board layout 1 -> 2 in hug.toml', apply writes 'layout = 2\n', a layout-1 hug's note exits 1, init board needs no upgrade). Three tests updated for init's hug.toml (people, cleanup, sections). -race cli 140.2 s, task 57.6 s, mcp, cmd/hug ok. To arm the check, hug upgrade --apply once on hug's and Seid's boards with a hug at 9c9d4c4 or later; I have not run it on either (shared board, crab's call).
landed 9c9d4c4 on main 2026-11-17: hug board layout: the state: rename is layout 2; a board without the key is 1…
hatchling 20:57: main CI 44170743115 (8856b2b) and 44172097835 (9c9d4c4) success on all 5 jobs (secrets, bench, ubuntu, macOS, windows).
Reflection (open, 2026-11-17, hatchling): My first cut set 'no key' = the current layout, which made the check inert on every existing board: the case that bit me today would not have been caught on its first recurrence. Crab caught it by asking what upgrade writes on today's boards. Lesson for version markers: absent must mean oldest, and the test must start from a real board as it is today, not from one the test wrote.
## History
- 2026-11-17T21:13 todo -> doing/active (take)
- 2026-11-17T21:57 doing/active -> done (done); was doing_since: 2026-11-17T21:13
