---
id: pbh9xq
title: "Lint: separate broken from not up to par"
status: "todo"
type: feature
priority: "medium"
assignee: []
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "ready"
---

## Goal
Every lint finding has a severity: broken (the tool cannot read or trust the task) or advice (the task works but is below the house style), and each place that shows findings decides by severity what to show

## Purpose
Jørgen 2026-11-16: should we lint all tasks? Reported all the time, findings are noise forever; but being broken differs from being not up to par. Measured on the seid copy (2,498 tasks, 2,477 findings): 2,424 are style (no-acceptance 1,158, long-title 827, done-unchecked 432, and so on) and only about 50 are structural (missing-goal 24, parent-too-deep 20, duplicate-section 10, dependency-cycle 2); 986 findings sit on closed tasks

## Principle
Broken is always shown and can block; advice is offered at the moment it can be acted on and is otherwise quiet. Closed tasks get no advice: nobody will fix a done task's title. Nudge one finding at a time with its fix, never a count. Imported history is not held to today's style (as task.PatternSince already does for missing-section)

## Relevant details
Codes today (huginn/internal/task/document.go): candidates for broken: dangling-dependency, dependency-cycle, parent-missing, parent-too-deep, duplicate-section, missing-goal, unreadable file. Candidates for advice: long-title, no-acceptance, done-unchecked, missing-section. Surfaces: hug task lint (default shows broken plus advice on open tasks, --all shows everything, --code as now, JSON gets a severity field); TUI header count and the lint view (count broken; advice shown dim or behind a toggle); take/new/edit guide lines (advice for the task in hand only); the recall nudge (qawmnb part 2: one finding at a break). Decide with Jørgen: the exact split per code, and whether done-unchecked on a closed task is broken (the board says done but checks are open) or history

## Depends on

## Acceptance
- [ ] each finding carries severity broken or advice in text and JSON, and the split per code is documented in help
- [ ] hug task lint by default shows broken findings and advice on open tasks only; --all shows every finding
- [ ] the TUI header counts broken findings; advice does not raise the warning count
- [ ] on the seid copy the default lint output drops from 2,477 lines to the measured broken-plus-open-advice number, stated in the task notes
- [ ] tests cover one code of each severity on an open and on a closed task

## Notes

2026-11-16 Jørgen decides: a done task with an unticked check is BROKEN, not advice: 'otherwise how can you have accountability'. The way out for a check that stopped applying is to strike it out with a reason (new ticket, see 'Acceptance checks can be struck out').

2026-11-16 Jørgen approves the plan: lint every task, give each finding a severity. Broken is always shown, counted in the TUI header and may block. Advice is shown only for open tasks and only when it can be acted on (take, new, edit, or one nudge at a break); closed tasks get none by default; --all shows everything.

2026-11-16 Jørgen: hide findings on closed tasks by default, CLI and TUI. Taken into 6neax1 now. Open point for him: he called done-with-unchecked-checks broken (accountability); with this default those 432 seid findings are hidden but counted in the one-line note. Keep that, or always list done-unchecked even on closed tasks?

Built on ci/pbh9xq (82ddba1 + CHANGELOG), parked behind g9dg32 per crab. Split per code: advice = long-title, no-acceptance, missing-section, section-order; broken = every other code (done-unchecked per Jørgen 2026-11-16), and so is a code missing from the table. Seid copy (2,508 tasks), default lint: before 696 shown (open only) / 1,192 hidden; after 1,141 shown = 445 broken (422 done-unchecked, 17 missing-goal, 6 duplicate-section) + 696 advice on open tasks; 747 advice hidden on closed tasks (422 no-acceptance, 325 section-order). Plain-text output stays bounded: summary plus the first 10. Not changed: recall's board nudge (a recall change needs a measured gain).

Back to todo (crab 13:31; wyvern resumes it after g9dg32). Code: ci/pbh9xq is pushed (82ddba1 plus a CHANGELOG commit), CI 44079521924 green. When rebased onto main, regenerate the narrow-60 golden: HUG_UPDATE_GOLDEN=1 go test -run TestNarrowRowsGolden ./huginn/internal/tui/


## History
- 2026-11-17T14:32 doing -> todo/ready (give back; was assignee: wyvern)
