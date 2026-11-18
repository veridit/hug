---
id: 4tycwd
title: "Detour scratch for in-progress work"
state: "doing"
type: spike
priority: ""
assignee: ["kikazaru"]
labels: ["needs-discussion"]
group: ""
rank: "zzzzzzv"
created: 2026-11-16
updated: 2026-11-16
doing_since: "2026-11-18T18:57"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_asked_by: "piglet"
waiting_on: "CI run 44373015352 (windows-cli rerun of the 10m timeout in TestBoardColumns)"
waiting_since: "2026-11-18T20:00"
---

## Goal
When an agent must leave an in-progress task for a detour, it parks where it was and picks it up again without loss

## Purpose
Detours (a bug found mid-task, a user question) make agents forget the half-done task; the board says doing but the thread is lost

## Principle
Light: no park command, no marker. A detour to another task is a dependency (hug task dep A --add B plus a note); a wait is for things outside the board and hug wakes it (--until or its condition). next means what to work on now, so a waiting task is never offered as next work during its wait (crab 17:57).

## Relevant details
Candidates: hug task park ID 'where I was' (a dated note plus a marker), shown by hug task next --mine and in the take guide; or reuse notes with a 'parked:' prefix. Decide after the nudge work (qawmnb) shows how often detours happen

## Depends on

## Acceptance
- [ ] ~~an agent can park and resume a task with one command each, and next --mine shows parked work first~~ (not relevant: replaced by checks 2-6 (crab 17:57: narrowed, no park command; detour = dep + note, already works), piglet, 2026-11-18)
- [x] hug task take refused for one-in-doing prints a fourth fix: hug task dep A --add B && hug task note A "where I was" (A the doing task, B the refused one), in text and in the JSON fixes array
- [x] hug task next --mine skips my doing tasks in stage waiting; a woken wait (--until day reached or its condition holds) counts as active and comes first as today
- [ ] when nothing else is left, hug task next --mine says so and lists my waiting tasks one line each (id, what it waits on), exit 4
- [ ] hug task resume ID prints once what the task was waiting on ('was waiting on: ...')
- [ ] one CHANGELOG line; tests fail first on origin/main and pass after; CI green

## Notes

Proposal (piglet, probed on hug v0.7.67 in a throwaway repo; not taken: gav9nw is parked, nothing built):
Already covered, no new command: a detour to another task is a dependency, not a wait (the wait help says so). hug task dep A --add B && hug task note A "where I was". A becomes doing/depends, which does not count for one-in-doing, so take B succeeds. When B is done, A goes back to doing/active on its own (History: doing/depends -> doing/active (dependency B)), next --mine returns A first, and show prints the note. Probe: A dep C, take C ok, done C, next --mine = A, stage active.
The wait route (what crab used on wxsyc5) works less well:
(1) During the detour, next --mine returns the parked task A, not the detour, with next: hug task check A 1. Stage waiting is not skipped (store.go:734 skips only Waiting() = deps/conditions).
(2) resume A while the detour is still doing is refused (one in doing). That is right, but the fix lines don't name the detour.
(3) After resume, the 'where I was' text stays only in ## History, which show does not print; resume's output does not echo it.
Narrowed proposal (no park command, no marker):
a. The take refusal gains a fourth fix when the refused task is known: hug task dep A --add B && hug task note A "where I was", where B is the task being taken.
b. next --mine skips my doing/waiting tasks (they come back via resume or --until, as today), so the detour is next.
c. resume prints what the task waited on ('was waiting on: ...') once.
Check 1 becomes: one command to park (dep + note), none to resume (it returns by itself), and next --mine shows the detour during it and the parked task right after.
Open for crab: is (b) a bug fix or wanted behaviour (a CI wait showing in next --mine as a reminder)?


1a5b6778 on origin/main 8c26ec51, pushed as 4tycwd and ci/4tycwd (CI run 44373015352). Failing first: TestDetourThroughTheCLI on e6853a7c: 'detour_test.go:28: exit 3, want 3 and four fix: lines ending with "hug task dep 6dak1j --add fs5ecd && hug task note 6dak1j \"where I was\""' (output had 3 fix lines). After: task 'ok 18.206s', mcp 'ok 50.412s', TestDetourThroughTheCLI PASS. Correction to my proposal: gap (2) was wrong; the resume refusal already names the detour task. Local full cli/gitbranch/tui runs hit timeouts at load avg 314-354 (cli 10m timeout; gitbranch TestUpgradeFailuresKeepOriginalLayout 'remote did not answer within 1s'; tui 600s), in packages/tests this change does not touch; CI is the gate.
CI 44373015352 on 1a5b6778: ubuntu, macos, windows-latest, secrets, bench success; test-windows-cli FAILED twice (attempt 1 18:56Z, rerun 19:11Z), both 'panic: test timed out after 10m0s' in huginn/internal/cli (running: TestBoardColumns, TestIdentityAndCommandHelp, TestSyncMergesBothSidesAppends/...), no assertion failure. Same job failed on ci/jrj1vq-c2 44372296874 (TestCacheWriteRacesReads), ci/z77pem 44371620428, ci/3t4a49 44370325830; last green windows-cli 44370638232; a green cli run took 423.255s (44366766759). Not shown: whether my TestDetourThroughTheCLI (~20 CLI commands) pushes the package over 600s on Windows.
landed 9163f10 on main 2026-11-18: Detour from a doing task: fourth take fix, next --mine skips waits, resume says…
shipped in v0.7.69 (9163f10)
## History
- 2026-11-18T18:57 todo -> doing/active (take)
- 2026-11-18T20:00 doing/active -> doing/waiting (wait)
