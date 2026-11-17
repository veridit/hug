---
id: n2hq8p
title: "One ticket in doing per agent; a second take says why or refuses"
status: "done"
type: feature
priority: "medium"
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "47sxn2"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T16:40"
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
kestrel 15:38: CI 44112302694 on 300abb8 was red on Windows only: '--- FAIL: TestHandEditedNotesSeenByLsAndShow (34.17s)' / 'scale_test.go:62: ls --json notes: "hand edited note", want "HAND EDITED NOTE"'. Same failure on ci/pbh9xq 44109039664; never on main in the last 40 runs. Cause: a timing assumption in the test, not n2hq8p and not the cache. Edit 1's mtime fell more than 2 s (racyWindow) behind the last sync, and cache.go:479 trusts a matching marker outside the window. Fix 21ee039 (crab 15:33, test only): both edits get an mtime inside the window. A 2.5 s pause stays, which failed the old test locally (failing-first evidence). Passes -count=3. Fails when the racy check is removed (mutation probe, reverted).
kestrel 15:41: landed on main as b8f9000..21ee039 (300abb8 rule plus 21ee039 test fix), fast-forward. Touched packages passed first: 'ok huginn/internal/cli 146.945s', 'ok huginn/internal/mcp 12.140s', 'ok huginn/internal/task 10.492s', 'ok huginn/internal/tui 97.987s'. Main CI 44116988262 running; a red from these commits gets fixed forward (crab 15:37).
Reflection (open, 2026-11-17, calf): A gap: since fkpyg3, hug task new requires --purpose and --principle, but my tests written before it still passed locally on the old base, and only CI on main showed it. Nothing told me to recheck test fixtures for 'new' after a rule change landed. Also, TestHandEditedNotesSeenByLsAndShow's comment said 'right after a sync' while the test kept an old mtime, so it hid an elapsed-time assumption.
kestrel 15:57: main CI 44116988262 on 21ee039 is green on all 5 jobs (secrets, bench, ubuntu, macos, windows success); Windows ran the fixed TestHandEditedNotesSeenByLsAndShow.
## History
- 2026-11-17T14:34 todo -> doing/active (take)
- 2026-11-17T15:02 doing/active -> doing/waiting (wait)
- 2026-11-17T16:40 doing/waiting -> doing/active (resume); was waiting_asked_by: calf; waiting_on: landing turn: right after hatchling (crab 15:09); rebase onto hatchling's push, one fresh CI; waiting_since: 2026-11-17T16:10
- 2026-11-17T16:40 doing/active -> done (done); was doing_since: 2026-11-17T14:34
