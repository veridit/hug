---
id: t01kqm
title: "Lint judges open tasks only; closed tasks are history, no hidden count"
state: "done"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzkx"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T19:48"
---

## Goal
hug task lint, the TUI LINT section and the header counts consider only open tasks (todo, doing). Done and dropped tasks get no findings and no 'N on closed tasks hidden' line. A closed task is checked once, when it closes (done already refuses open checks). Exception: a file hug cannot read as a task at all still shows, since it is not known to be closed.

## Purpose
Jørgen 00:25 local, on Statbus's LINT section (179 tasks, 298 broken, rows almost all 'done'): 'I don't think linting should show the done tasks'; 'a lint telling me we're ignoring 500 done tasks is not truly sensible.' Measured on Statbus by crab with hug v0.7.50: 308 broken findings, 290 on closed tasks (236 done-unchecked, 51 missing-goal, 3 duplicate-section); 18 on open tasks (10 branch-gone, 8 missing-goal); plus '170 advice findings on closed tasks hidden'.

## Principle
Lint is for work that can still change. History is neither nagged nor counted. Check at the moment of the write (wdkqzx), not after. No flag keeps the old behaviour (no backward compatibility). hug task lint ID on a closed task says it is closed and has nothing to judge.

## Relevant details
Code: huginn/internal/task lint severity rules; huginn/internal/cli/lint.go:134-147 (the hidden line, hiddenCommand, summary.Hidden and hidden_codes in JSON); huginn/internal/tui/render.go:326 (' (N on closed tasks hidden)'); header '! N broken · N advice'. The done-unchecked code goes: a done task with an open check only comes from import or a hand edit, and import (ggydj2) should say so at import time. The lint --all flag and the JSON hidden fields go. Owner: hatchling, after wdkqzx A+C.

## Depends on

## Acceptance
- [x] On Statbus's board hug task lint --summary counts only open tasks (about 18 broken), has no done-unchecked and no 'closed tasks hidden' line (numbers in a note)
- [x] A done or dropped task gives no finding; hug task lint ID on one prints 'ID is done: lint judges open tasks only' and exits 0 (test, fails on main)
- [x] The TUI LINT section and header counts match the CLI and show no hidden-on-closed text (golden)
- [x] A file hug cannot parse as a task still shows as broken (test)
- [x] The lint all-tasks flag and the JSON hidden fields are gone, and help says lint judges open tasks (help test)
- [x] Jørgen confirms the Statbus LINT section after release

## Notes


Built on ci/t01kqm (from 48b1151f). task.Judged replaces ShownFindings: findings on open tasks only, broken first; a path no task owns (a file hug cannot read as one) stays. done-unchecked code and its parse finding removed. CLI lint: closed lint ID answers 'ID is done: lint judges open tasks only' (exit 0, next: hug task show ID); --all, setHidden/writeHidden/hiddenCommand, summary.hidden/hidden_codes and reply.hidden removed; bounded lists end 'every one: hug task lint --json'. TUI: closedFindings gone; LINT header has no hidden text; a closed task's detail shows no finding. Help and MCP lint description say lint judges open tasks. Statbus measured read-only from /Users/jhf/ssb/statbus: brew v0.7.52 'hug task lint --summary' = '446 findings (314 broken) in 8 codes' (236 done-unchecked, 59 missing-goal, 11 branch-gone, 5 old-layout, 3 duplicate-section; 98 missing-section, 32 long-title, 2 no-acceptance) plus '170 advice findings on closed tasks hidden (170 no-acceptance)'; this build = '156 findings (24 broken) in 6 codes' (11 branch-gone, 8 missing-goal, 5 old-layout; 98 missing-section, 32 long-title, 2 no-acceptance), no hidden line, no done-unchecked. The ticket's 'about 18 broken' was 18 at crab's v0.7.50 count; the 5 old-layout are hug upgrade's (41hhgn) and open.
file:t01kqm-fail-before.txt (1037 bytes, sha256:f50cdac1)
file:t01-statbus-before.txt (2002 bytes, sha256:1de485d3)
file:t01-statbus-after.txt (1431 bytes, sha256:0b5ebc78)
Landed f6b63256 on main (CI 44223058200). Tests: cli TestLintJudgesOpenTasksOnly (check 2: a done task with a broken not-a-dependency, a dropped one with section-order advice, and a hand-edited done one with an open check give 'no findings'; lint ID on each prints 'ID is done|dropped: lint judges open tasks only' with next: hug task show ID, exit 0; JSON has no 'hidden'; lint --all exits 2; check 4: a header-less file zz.qqqqqq.garbage.md still shows; open broken before open advice); strike TestStrikeThroughCLI; task TestJudgedOpenTasksOnly and TestDoneWithOpenCheckIsNotAFinding; tui TestClosedTaskIsNotJudged (check 3: header, LINT section, marks and detail drop the closed task; overview has no 'hidden'), goldens overview/list/board x4 ('! 4 broken' -> '! 2 broken'; 'LINT ... [9] · 4 broken · 5 advice (1 on closed tasks hidden)' -> '[7] · 2 broken · 5 advice'), markdown-warning -> markdown-done (done 000003 renders, no ⚠). Check 5: help usage 'lint [ID] [--summary] [--code [CODE]]' (TestLintNeverSilentSummaryCodesHelp asserts it) and help says 'Lint judges open tasks only'. On main 48b1151f TestLintJudgesOpenTasksOnly and TestStrikeThroughCLI fail: '2 findings (2 broken) ... done-unchecked ... not-a-dependency ... 1 advice finding on closed tasks hidden (1 section-order): hug task lint --all' (file:t01kqm-fail-before.txt). Statbus before/after: file:t01-statbus-before.txt, file:t01-statbus-after.txt. -race: task 25.9s, cli 116.4s, mcp 9.7s, board 1.5s, tui -short 109.7s ok; TestTmuxGolden 21.2s ok; after rebase onto 28791dea: tui (with tmux) 101.4s, cli 123.6s, task 10.6s ok.
landed f6b6325 on main 2026-11-18: hug task lint judges open tasks only; closed tasks are history, nothing hidden…
CI 44223058200 for f6b63256: success, all 7 jobs (secrets, bench, test ubuntu/windows/macos, test-windows-cli). Check 1: 24 broken, not the ticket's 'about 18': 11 branch-gone + 8 missing-goal match crab's 18 on open tasks (10 branch-gone, 8 missing-goal at v0.7.50, one more branch since); the other 5 are old-layout, a code added after v0.7.50 (41hhgn), all on open tasks, which hug upgrade fixes.
shipped in v0.7.54 (f6b6325)
retriever 2026-11-18 13:35 (board check for crab, hatchling's session is gone): the wait on the release is over. f6b63256 shipped in v0.7.54 and brew hug is v0.7.62. Statbus today (read-only, ~/ssb/statbus, v0.7.62): '131 findings (13 broken) in 6 codes', no 'closed tasks hidden' line, no done-unchecked. Work remains: check 6 needs Jørgen's look, and nobody has asked him. Next owner: take it and run hug task ask t01kqm --to jhf --check 6 with the Statbus numbers above. Given back to todo because the owner (hatchling) is no longer running.
piglet 14:37: asked jhf about check 6 (hug task ask --check 6; the ask went through, stage asked; gpyqq9 is not needed here because the ask gate already excused shipped-not-done for the one open check). Own read-only Statbus check, hug v0.7.62 in ~/ssb/statbus: 'hug task lint --summary' = '130 findings (13 broken) in 7 codes' (6 branch-gone, 5 missing-goal, 1 doing-idle, 1 unknown-person; 84 missing-section, 31 long-title, 2 no-acceptance), no 'closed tasks hidden' line, no done-unchecked. The question says 131/6 codes from retriever's 13:35 read; one finding changed since.
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T01:43 todo -> doing/active (take)
- 2026-11-18T01:43 doing/active -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-18T01:43
- 2026-11-18T02:56 todo/ready -> doing/active (take)
- 2026-11-18T03:30 doing/active -> doing/waiting (wait)
- 2026-11-18T13:33 doing/waiting -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-18T02:56; waiting_asked_by: hatchling; waiting_on: the release carrying f6b63256, then Jørgen's look at the Statbus LINT section (check 6); waiting_since: 2026-11-18T03:30
- 2026-11-18T14:36 todo/ready -> doing/active (take)
- 2026-11-18T14:36 doing/active -> doing/asked (ask)
- 2026-11-18T19:48 answered asked (check 6), by jhf, for piglet (doing/asked -> doing/active):
  > t01kqm (lint judges open tasks only) shipped in v0.7.54; brew hug is v0.7.62. On Statbus today (read-only, ~/ssb/statbus) hug task lint --summary says '131 findings (13 broken) in 6 codes', with no 'closed tasks hidden' line and no done-unchecked (before: 446 findings, 314 broken, plus 170 hidden). Please open the Statbus LINT section in hug tui: does it now show only open tasks the way you wanted? If yes: hug task check t01kqm 6
  answered by ticking check 6
- 2026-11-18T19:48 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T19:48 doing/active -> done (done); was doing_since: 2026-11-18T14:36
- 2026-11-18T19:49 answer 1 acknowledged by crab
