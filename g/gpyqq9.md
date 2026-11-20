---
id: gpyqq9
title: "ask is refused by the shipped-not-done finding that asking would settle"
state: "done"
type: bug
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzznl"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T15:29"
done_release: "v0.7.64"
---

## Goal
hug task ask on a shipped task whose remaining decision belongs to a person succeeds and settles the shipped-not-done finding, instead of being refused because of it.

## Purpose
Crab tried to ask Jørgen whether 4vd1mz (shipped, all checks ticked, owner decision pending) is done; ask refused with 'not ready to hand to jhf: shipped-not-done ... fix: hug task done 4vd1mz', and wait refused too, pointing back to ask. No command records 'the owner decides'.

## Principle
A finding whose fix is a person's decision must be settleable by asking that person. The readiness gate for ask excludes questions ask itself answers (stageSaysWhy already silences shipped-not-done once asked).

## Relevant details
questions.go noteQuestion: case shipped != nil && stageSaysWhy(t) returns nil, so asked would clear it; the ask command's readiness check runs lint before the stage changes. Repro: 4vd1mz at 2026-11-18 11:56Z.

## Depends on

## Acceptance
- [x] hug task ask on a shipped, all-ticked task succeeds, sets stage asked, and lint no longer reports shipped-not-done for it; a test fails first on main.
- [x] Other broken findings (e.g. missing-file) still block ask (test).

## Notes


commit ebf1c40 on branch gpyqq9 (ci/gpyqq9, run 44303286385). check 1: TestAskSettlesShippedNotDone (askgate_test.go) fails on main: 'ask on a shipped, all-ticked task: exit 2, want 0 / hug task ask: fanrse is not ready to hand to jhf: 1 broken finding: [shipped-not-done] ...'; passes with the fix (stage asked, lint has no shipped-not-done). check 2: TestAskOnShippedTaskStillRefusesOtherBrokenFindings: unknown-type still refuses, names only [unknown-type], fix 'hug task set ID --type feature', writes nothing. Local niced suite: ok huginn/internal/cli 246.793s, ok huginn/internal/task 13.743s, all huginn + cmd/hug ok. Fix: readyToHand excuses task.StageSettles codes for ask (replaces the one-open-check special case).
check 2 note: the test uses unknown-type, not missing-file. missing-file comes from LintFiles (board lint only), not from the per-task lint block readyToHand reads, so on main too the ask gate never sees a missing-file finding (pre-existing, not changed here). Tried adding a per-task LintFiles to the gate: TestEveryWriteEndsWithTheTasksLint then refused the ask because 'hug task detach' leaves a file: citation in the notes that lint reports as missing-file twice (also on main: 'hug task lint ID' after attach+detach shows 2 identical missing-file lines). Left both for crab to decide.
Second case 2026-11-18 12:33Z: 5z4zt4 (shipped v0.7.44, checks 4 and 5 open, both Jørgen's to confirm): ask refused with shipped-not-done. So it also hits tasks with open checks whose holder is the person, not only all-ticked tasks.
crab review 12:55Z: diff read (readyToHand: ask skips findings task.StageSettles names, onlyOpen removed; questions.go stageSettled = {shipped-not-done}). Tests cover all-ticked (4vd1mz shape), open checks asked of a person (5z4zt4 shape), and other broken findings still refuse. Exact SHA d177829e CI 44305350513 all 6 jobs success. Lands after xfv9pw and z82qsy (rebase plus exact-SHA CI).
CI green: run 44303286385 (ebf1c40) and run 44305350513 (d177829, branch head): secrets, bench, test ubuntu/macos/windows, test-windows-cli all success.
Landed on main 674336d1 (CI 44319483386, exact SHA, 6/6). Ships in the next release.
landed fe4dbc0 on main 2026-11-18: ask settles the shipped-not-done finding it asks about (gpyqq9)
landed 674336d on main 2026-11-18: test: ask about open checks of a shipped task settles shipped-not-done (gpyqq9…
shipped in v0.7.64 (fe4dbc0)
## History
- 2026-11-18T13:17 todo -> doing/active (take)
- 2026-11-18T13:22 doing/active -> doing/waiting (wait)
- 2026-11-18T15:29 doing/waiting -> done (done); was doing_since: 2026-11-18T13:17; waiting_asked_by: piglet; waiting_on: crab review; waiting_since: 2026-11-18T13:22
