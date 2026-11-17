---
id: n2hq8p
title: "One ticket in doing per agent; a second take says why or refuses"
status: "doing"
type: feature
priority: "medium"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "47sxn2"
doing_since: "2026-11-17T14:34"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "landing turn: second push of my turn, after kbc99j+4vcwxk; CI 44096303450 green"
waiting_since: "2026-11-17T15:02"
waiting_asked_by: "calf"
---

## Goal
hug task take refuses when the taker already has a ticket in doing whose stage is active, naming it and offering the fixes: finish it (hug task done), park it (hug task wait ID "on what"), or put it back (hug task set ID --status todo with a note). Tickets in waiting, review or depends do not count. The board's doing count then means work in hand.

## Purpose
Epic 47sxn2. hatchling held 5 tickets in doing while working on one; Jørgen read them as stuck.

## Principle
Doing means someone is on it now. The refusal is the fix, not a lint later. Humans (--model human) get the same rule.

## Relevant details
Needs 6ephqk's stages (active, waiting, review, depends) and the wait command. Owner identity: model + --as name from take (t709he, x1t1t1).

## Depends on

## Acceptance
- [x] a second take by the same agent with one active ticket is refused with the three fix lines (test)
- [x] a take is allowed when the earlier ticket is waiting, review or depends (test)
- [x] MCP task_take returns the same refusal (test)

## Notes

Repro case (kestrel, 2026-11-17 13:10): with 4vcwxk in doing (taken 12:51 as kestrel, claude-opus-5-5, active work in hand), 'hug task take kbc99j --model claude-opus-5-5 --as kestrel' succeeded and printed 'now: as each check holds: hug task check kbc99j N ...'. One agent then held two active doing tickets. Check 1's test can replay exactly this: take A, then take B with the same --as/--model, and expect the refusal with the three fix lines.

kestrel 13:35, dependency on 6ephqk removed with crab's go (13:34): n2hq8p needs only the doing stages (active, waiting, review, depends) and hug task wait, both on main d198e01; 6ephqk's open steps (upgrade, look) do not touch take.


kestrel 13:48, built on main d198e01 as 8dc2ea8 (branch kestrel/n2hq8p, CI 44096303450 running). Rule in the store (task/busy.go: Active, BusyFixes): take into doing is refused when the taker has another doing task whose Stage is active; resume too. Taking a todo/review (it becomes doing/review) is allowed. Refusal: exit 3, 'not taking B: kestrel is on A (title): one ticket in doing at a time' + 3 fix: lines; --json and MCP: fix (first) and fixes (3). Tests: task TestSecondTakeWhileActiveIsRefused (repro, person, review take) and TestTakeAllowedWhenEarlierIsNotActive (waiting, review, depends; after done, wait, back to todo; resume refused), both fail before the fix ('want busy refusal naming aaaaaa, got <nil>'); cli TestSecondTakeRefusedWithThreeFixLines (exact text); mcp TestSecondTakeOverMCPIsRefused; tui TestTakeSecondTaskRefusedInTUI. 11 existing fixtures took several tasks as one name; they now use one name per agent or wait first. Niced full suite exit 0.
kestrel 14:03: CI 44096303450 on 8dc2ea8 green on all 5 jobs (secrets, bench, ubuntu, macos, windows). Check 1: task TestSecondTakeWhileActiveIsRefused + cli TestSecondTakeRefusedWithThreeFixLines (exact text, exit 3, three fix: lines). Check 2: task TestTakeAllowedWhenEarlierIsNotActive (waiting, review, depends). Check 3: mcp TestSecondTakeOverMCPIsRefused (fix, fixes, same problem text as the CLI). Lands as the second push of my turn, after kbc99j+4vcwxk (crab 14:02).
kestrel 14:36: push-2 CI 44102357077 on 07ffc27 was red on ubuntu: 'busy_test.go:16: [new First --goal G --accept C]: incomplete-task: missing section Purpose (why); missing section Principle ...' (cli) and the same in mcp busy_test.go:60. fkpyg3 (landed meanwhile) requires purpose and principle on new, and I had not rerun the tests after the rebase. Test-only fix in 812d1f0; touched packages (cli, mcp, task, tui) pass locally. The turn went to wyvern (crab 14:33); push after wyvern with a fresh CI.
## History
- 2026-11-17T14:34 todo -> doing/active (take)
- 2026-11-17T15:02 doing/active -> doing/waiting (wait)
