---
id: pbh9xq
title: "Lint: separate broken from not up to par"
status: "done"
type: feature
priority: "medium"
assignee: ["wyvern"]
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T19:15"
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
- [x] each finding carries severity broken or advice in text and JSON, and the split per code is documented in help
- [x] hug task lint by default shows broken findings and advice on open tasks only; --all shows every finding
- [x] the TUI header counts broken findings; advice does not raise the warning count
- [x] on the seid copy the default lint output drops from 2,477 lines to the measured broken-plus-open-advice number, stated in the task notes
- [x] tests cover one code of each severity on an open and on a closed task

## Notes

2026-11-16 Jørgen decides: a done task with an unticked check is BROKEN, not advice: 'otherwise how can you have accountability'. The way out for a check that stopped applying is to strike it out with a reason (new ticket, see 'Acceptance checks can be struck out').

2026-11-16 Jørgen approves the plan: lint every task, give each finding a severity. Broken is always shown, counted in the TUI header and may block. Advice is shown only for open tasks and only when it can be acted on (take, new, edit, or one nudge at a break); closed tasks get none by default; --all shows everything.

2026-11-16 Jørgen: hide findings on closed tasks by default, CLI and TUI. Taken into 6neax1 now. Open point for him: he called done-with-unchecked-checks broken (accountability); with this default those 432 seid findings are hidden but counted in the one-line note. Keep that, or always list done-unchecked even on closed tasks?

Built on ci/pbh9xq (82ddba1 + CHANGELOG), parked behind g9dg32 per crab. Split per code: advice = long-title, no-acceptance, missing-section, section-order; broken = every other code (done-unchecked per Jørgen 2026-11-16), and so is a code missing from the table. Seid copy (2,508 tasks), default lint: before 696 shown (open only) / 1,192 hidden; after 1,141 shown = 445 broken (422 done-unchecked, 17 missing-goal, 6 duplicate-section) + 696 advice on open tasks; 747 advice hidden on closed tasks (422 no-acceptance, 325 section-order). Plain-text output stays bounded: summary plus the first 10. Not changed: recall's board nudge (a recall change needs a measured gain).

Back to todo (crab 13:31; wyvern resumes it after g9dg32). Code: ci/pbh9xq is pushed (82ddba1 plus a CHANGELOG commit), CI 44079521924 green. When rebased onto main, regenerate the narrow-60 golden: HUG_UPDATE_GOLDEN=1 go test -run TestNarrowRowsGolden ./huginn/internal/tui/


Rebase onto 4ecfa25 is in progress in worktree hug-lint (paused for wyvern's landing slot). Resolved: codes.go (Severity on every row of main's table; backlog-leftover advice, stale-owner and the stage codes broken), lint.go (main's overview replaces the old summary block; the overview header now says '(N broken)'), help.go, render.go, cli_test.go, edit_notice_test.go. Goldens are taken from main and must be regenerated. Still failing: TestLintOverviewDetailAndSameBlock and TestLintHidesClosedTasks.
Rebased onto 4ecfa25: ae0ec48 + 0acb158 on ci/pbh9xq, CI 44109039664. Conflicts resolved:
- codes.go: main's table with a Severity on each code. backlog-leftover is advice; stale-owner and the 4 stage codes are broken.
- The lint overview header now reads 'N findings (M broken) on T tasks'.
- Tests follow main: the overview is one line per task, done needs take first, and new refuses a long title so the test uses set.
- Goldens regenerated and reviewed: advice-only rows lose '!', and the header reads '! N broken · M advice'.
Local go test ./... is all ok.
Rebased onto main 06297ea (v0.7.43 merge) and pushed: 963df49, e3c65e2, 372ff7a, ea2d369 (main 06297ea..ea2d369). Conflicts: codes.go (main added unknown-person: Broken), help.go (kept main's search and deduce help), render.go (header 'N broken · M advice', then vnyz8z's spinner), tui.go (broken map beside main's people). initialSnapshot now uses ShownFindings like every load, so the counts do not change when the refresh lands; TestTmuxGolden waits for '⎘2' (the refreshed board): it drifted in 2 of 3 runs before, passed 3 of 3 after. Goldens: tmux 51 files and 3 row/detail files, header and '!'/'·' row marks only. -race: task, tui, cli, board, mcp ok. Check 4, measured on a fresh copy of Seid's hug/tasks (2,520 tasks; the board changed since the 2,498-task measure): --all 5,746 findings (269 broken: all done-unchecked; 5,477 advice); default 5,299 shown (269 broken + 5,030 advice on open tasks), 447 advice on closed tasks hidden. The text overview is 19 lines; its header: '5299 findings (269 broken) on 1299 tasks, in 4 codes'. Ticks follow green main CI.
landed 963df49 on main 2026-11-17: hug task lint: every finding is broken or advice; broken always shows, advice…
landed 372ff7a on main 2026-11-17: hug tui goldens: header reads 'N broken · M advice' in the glance, alias search…
shipped in v0.7.44 (963df49)
Ticks on green main: dcb20c2 contains 963df49..ea2d369 (and ceb9d1a, f9e5664), CI 44143180871 success on ubuntu -race, macos, windows, bench, secrets. 1: Severity column per code in codes.go, shown in text and JSON and in help (help.go 'broken or advice'). 2: default lint 5,299 shown, --all 5,746 on the Seid copy (note above). 3: header 'N broken · M advice', goldens 372ff7a. 4: numbers in the note above (5,299 shown, 447 hidden, 19-line overview). 5: TestLintHidesClosedTasks and TestLintOverviewDetailAndSameBlock cover a broken and an advice code on open and closed tasks.
Reflection (open, 2026-11-17, wyvern): The rebase met four parallel changes (codes, help, render, tui); the gap was TestTmuxGolden racing the refresh, fixed by waiting for the refreshed board's mark. Nothing in the instructions misled.
## History
- 2026-11-17T14:32 doing -> todo/ready (give back; was assignee: wyvern)
- 2026-11-17T15:23 todo/ready -> doing/active (take)
- 2026-11-17T15:59 doing/active -> doing/waiting (wait)
- 2026-11-17T19:15 doing/waiting -> done (done); was doing_since: 2026-11-17T15:23; waiting_asked_by: wyvern; waiting_on: main CI green on all jobs for ceb9d1a (contains ea2d369), then tick 1-5; waiting_since: 2026-11-17T18:16
