---
id: 16dxsr
title: "Depend on a group or epic, and on a stated condition"
state: "done"
type: feature
priority: "high"
assignee: ["kestrel"]
labels: []
group: ""
rank: "p"
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A task can wait on a whole group or epic (blocked until every task in it is done or dropped) and on a condition hug can check (load below N, a branch merged, a release published, a date reached). Blocked shows the open condition; next skips the task until it holds.

## Purpose
Jørgen 2026-11-16, seeing a coordinator's todo list full of 'Blocked: after phase 1 landings' entries: dependencies belong on the board, between tickets and groups, not in an agent's todo list; and some waits are conditions, not tickets ('blocked on load: load below something').

## Principle
Depends on accepts a task, an epic or group (all children done), or a condition line with a checkable predicate. hug evaluates conditions when it reads the board; an unknown or unevaluable condition counts as unmet and lint names it with a fix: line. No agent-side polling: next simply skips blocked tasks. Today Blocked() (huginn/internal/task/query.go:220) only resolves single tasks; a dep on an epic waits for the epic's own status, and groups have no dependency at all.

## Relevant details
Condition kinds to start: load below N (5-min average), branch merged into main, tag/release exists, date reached. Same vocabulary as ez50rb's memory conditions, so share one evaluator.

## Depends on

## Acceptance
- [x] a task depending on a group is blocked until every task in the group is done or dropped (test)
- [x] a task depending on an epic is blocked until all its children are closed (test)
- [x] a condition 'load below 120' blocks next while the 5-min load is above 120 and frees it when below (test with an injected load)
- [x] hug task show and the TUI print the unmet condition; lint flags an unknown condition with a fix: line (test)

## Notes

kestrel 12:19, 8fb5a6c on ci/16dxsr (CI 44072850549 pending). Model: internal/cond, one evaluator for task waits and memory holds (muninn/internal/holds now maps until/while to cond events). Depends on line '- when PREDICATE [wakes NAME]' or '- group:NAME'; predicates: date YYYY-MM-DD, load below N, branch NAME merged [at TIP] (dep --add records the tip, so merged+deleted still reads merged), release/tag NAME exists, task:ID closed, group NAME closed. Unmet: next skips, take refuses (exit 3, fix: hug task show ID); task/group waits also hold back done; date/load/branch/tag only gate starting. Unknown (unparseable, branch not in clone, no loadavg on Windows) never holds; lint [unknown-condition] with fix: hug task dep ID --remove 'when ...'. wakes is recorded and reported (default: assignee); acting on it is 5dxzxe's supervisor. Epic dependency: Closed() = done/dropped or all children closed (memoized child index, linear at 10k). Tests: TestDependOnGroup (3 open in group auth -> 1 open after subgroup -> every task in group auth is closed; next then offers it; empty group refused), TestDependOnEpicWaitsForChildren (blocked until kid1 done + kid2 dropped, epic itself todo; show [todo, epic: 2 open]), TestLoadConditionBlocksNext (injected 140 -> next skips, 80 -> next offers; done not held by load; unknown load -> unknown-condition finding), TestUnknownConditionLintAndRefusal, TestWaitsThroughCLI (show 'waits: when load below 120  [waiting: load is 140.0]  wakes kestrel', JSON waits, lint '[unknown-condition] ... (no branch feat/nowhere in this clone)'), TestBranchWaitRecordsTip, TestDetailShowsUnmetCondition (TUI '⧗ when load below 120 WAITING · load is 140.0 · wakes kestrel', '↳ ID Epic [todo] BLOCKED (epic: 1 open)'). Full niced suite: 0 FAIL (cli 127.2s, tui 77.6s, task 8.3s, muninn app 40.6s). Cache schema 11.

kestrel 12:40: rebased twice over main (last onto 6d88d74, tb0dra); now 0d795c4 on ci/16dxsr, fresh CI 44076137798 running (44073740045 was green on ubuntu/macos/windows/bench/secrets for the earlier base 0aa3abb). Full niced suite on 0d795c4: exit 0. Check 1: TestDependOnGroup (task/waits_test.go). Check 2: TestDependOnEpicWaitsForChildren. Check 3: TestLoadConditionBlocksNext (injected load 140 -> next = other task; 80 -> next = it). Check 4: TestWaitsThroughCLI (show 'waits: when load below 120  [waiting: load is 140.0]  wakes kestrel'; lint '[unknown-condition] unknown condition: "when branch feat/nowhere merged" cannot be checked here (no branch feat/nowhere in this clone), so ID waits' + 'fix: hug task dep ID --remove ...'), TestUnknownConditionLintAndRefusal, TestDetailShowsUnmetCondition (TUI '⧗ when load below 120 WAITING · load is 140.0 · wakes kestrel').

kestrel 13:10: landed on main 893cd50..c9ea6ed (fast-forward). CI 44078738328 on c9ea6ed: ubuntu, macos, windows, bench, secrets all success. Cache schema 12 (6ephqk took 11).
