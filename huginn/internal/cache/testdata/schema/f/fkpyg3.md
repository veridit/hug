---
id: fkpyg3
title: "Every check reports every problem at once, with one fix line"
state: "done"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzz3q"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T15:03"
---

## Goal
Whenever hug checks a task (task new, set, edit, done, take, lint, and MCP task_new/task_update), it runs every check and reports every problem with that task in one reply, each naming what is wrong (missing: --purpose, --principle; title 87 characters, max 80; sections out of order), followed by ONE fix: line that corrects all of them together where they can be fixed in one command (hug task set ID --purpose "..." --principle "..." --title "..."; reorder is done by the write itself or named once). Lint groups findings per task, not one line per finding.

## Purpose
Jørgen 2026-11-17 10:59: agents fix one thing, run again, get the next thing, five rounds for one task (purpose, goal, order...). Repro on v0.7.40: hug task new "x" fails with only 'missing-required' and a generic fix, not naming --goal or --accept; after adding them the task is written and lint then reports missing Purpose, Principle, then long-title as separate findings with separate fix lines (hug task edit ID vs hug task set ID --title). Each round costs an agent a turn.

## Principle
Collect, never stop at the first: validation returns a list. Every finding names the exact field or fact (no bare kind like missing-required). One combined fix: line per task, runnable once the <...> slots are filled; separate lines only when no single command can fix them. A write that can fix a problem itself (section order) does so instead of reporting it. Same set of checks at write time and in lint: what new/set accept, lint never flags later.

## Relevant details
Write-time check: huginn/internal/task/store.go New() (line ~168, returns on first error), Set/Edit paths. Lint: huginn/internal/task/codes.go (Codes list incl. section-order, missing-section, long-title, no-acceptance), cli lint output. Current lint lists one line per finding, so a task with 3 problems shows 3 entries with 3 different fix commands. MCP must return the same list (structured: findings [] + fix).

## Depends on

## Acceptance
- [x] hug task new "x" names every missing field (--goal, --accept, --purpose, --principle) in one reply with one fix: line containing all of them (test)
- [x] a task with missing Purpose, missing Principle and an 87-character title gets one lint entry listing all three and one fix: hug task set ID --purpose ... --principle ... --title ... (test)
- [x] write-time checks and lint run the same check list: a task new/set accepts is lint-clean except for findings that need time (e.g. done-unchecked) (test enumerates the codes)
- [x] MCP task_new/task_update return findings as a list plus the combined fix (test)
- [x] on seid-lang, hug task lint output has one entry per task, and the count line still shows totals per code (shown before and after)
- [x] every open task is held to the description pattern: PatternSince no longer exempts open tasks created before 2026-11-16 (done and dropped stay exempt); on seid-lang zjedy0's detail lists no acceptance, missing Purpose and missing Principle together (shown)
- [x] new findings for backlog import leftovers: <!-- SECTION:... --> markers, and the unused fields created_date, ordinal, backlog_status, import_type_inference; their fix is hug upgrade (test; seid-lang counts before and after)
- [x] hug task set ID --accept "..." --purpose "..." --principle "..." fixes those findings without an editor, so every combined fix: line is runnable (test)
- [x] hug task lint is the board overview: totals per code and one compact line per task; hug task lint ID gives every finding of that task in detail with the one combined fix: line; hug task show (detail view) and hug task take show the same block as lint ID (test)
- [x] rows in ls, board and the TUI list stay as they are: at most the existing ! marker, no finding text or counts (test)
- [x] hug upgrade does only the mechanical part, shown as a diff: strips backlog SECTION markers, drops unused backlog fields, puts sections in order; it never splits Purpose or Principle out of Goal (test)

## Notes

Jørgen 2026-11-17 11:00, on zjedy0 in Seid: the detail shows ONE finding (no acceptance), but the task also lacks Purpose and Principle; its whole description sits in Goal, including backlog's <!-- SECTION:DESCRIPTION --> markers; the title carries tags (.Refactor .Compiler.Lang); and backlog's leftover fields remain (created_date, updated_date, ordinal, backlog_status, import_type_inference). Cause: query.go:237 PatternSince = 2026-11-16. Purpose and Principle are checked only on tasks created since then, which hides it on 1,215 of Seid's 1,236 open tasks. The leftover markers and fields have no check at all (1,208 open tasks have the markers, 1,214 the fields). Scope added: (a) drop PatternSince, so every open task is held to the pattern (done and dropped stay exempt); (b) new findings for import leftovers: backlog markers and fields that hug doesn't use; hug upgrade removes them; (c) tags in the title go to txjkrm's cleanup; (d) the fix line must be runnable, but 'hug task edit ID' opens an editor, and set has no --accept, so add 'hug task set ID --accept "..."' (and --purpose/--principle) so agents can fix everything with one command.

Jørgen 2026-11-17 11:01: the full findings matter in the detail view and in what the agent learns (CLI/MCP replies on show, take, new, set, lint). They must NOT pollute the listing: rows stay as they are (at most the existing ! marker), with no finding text or count on rows. Detail view: one block listing every finding of the task, then the one combined fix: line.

Jørgen 2026-11-17 11:08 (lint has two levels, so nobody is overwhelmed and we don't overwork): (1) board lint: 'hug task lint' gives the overview: totals per code, one compact line per task, no full fix text for every task. (2) task lint: 'hug task lint ID' (today it fails: 'expected 0 positional arguments, got 1') gives every finding for that one task, in detail, with the one combined fix: line. The detail view and 'hug task take' show the same as task lint, so taking a ticket means cleaning it up. hug upgrade does only the mechanical part (strip backlog SECTION markers, drop unused backlog fields, put sections in order, with a diff); splitting Purpose and Principle out of Goal is judgment, done per task when it is linted or taken, never in bulk.

guppy (hatchling) 12:40, plan, each step committed on ci/fkpyg3: (1) task.ByTask groups findings per task and task.CombinedFix merges every 'hug task set ID --x' fix into one command (others listed once, hug upgrade once); hug task lint = totals per code + one compact line per task ('ID  code, code  -> hug task lint ID'); hug task lint ID = every finding in detail + one fix: line; JSON keeps findings[] and adds tasks[] {id, path, codes, problems, fix}. (2) set gains --accept (repeatable, appends), --purpose, --principle, --details; missing-section and no-acceptance fixes become set commands. (3) new collects every missing field (title, --goal, --accept, --purpose, --principle, title length) in one error with one fix line; same check list as lint. (4) PatternSince dropped for open tasks; backlog leftovers findings (SECTION markers, created_date/updated_date/ordinal/backlog_status/import_type_inference/milestone/dependencies/origin? only the unused ones), fix hug upgrade; upgrade strips them with a diff. (5) show and take print the lint ID block; rows unchanged. (6) MCP returns findings[] + fix.

hatchling 12:28, seid-lang evidence (copy of hug/tasks at ccbd527d5, real board untouched; files in ~/.jcode/scratch/fk_seid/). BEFORE (hug v0.7.40) lint --summary: '190 findings in 2 codes' (189 no-acceptance, 1 branch-gone); plain lint 27 lines. AFTER (fkpyg3 cb6daa9+tests): '4967 findings on 1196 tasks, in 4 codes': 2390 missing-section, 2387 backlog-leftover, 189 no-acceptance, 1 branch-gone; plain lint is 19 lines (totals, first 10 tasks one line each, 'next: hug task set q70jjy --purpose "..." --principle "..."; then hug upgrade'). zjedy0 detail: 5 findings, [no-acceptance], [missing-section] Principle, [missing-section] Purpose, [backlog-leftover] 4 unused fields, [backlog-leftover] 2 SECTION marker lines; next: hug task set zjedy0 --purpose "..." --principle "..." --accept "observable check"; then hug upgrade. STRIP (Store.Tidy, what hug upgrade runs, on the copy): 2494 files tidied, 12479 marker lines and 9652 field lines removed, 0 lines added, second run changes 0 files; after it backlog-leftover is 0 and 4 open tasks newly show missing-goal (6k4091, xnram6, hnw3aq, q1aw7w): their Goal held only the SECTION marker, the text sits under later headings. Pre-strip lint counted the marker as goal text; fine to leave, upgrade then names them.

hatchling 12:32, evidence per check (ci/fkpyg3, local suite green incl. cli 205s): 1 TestNewNamesEveryMissingPart + TestNewNamesEveryMissingFlag; 2 TestCombinedFixForOneTask; 3 TestNewAndLintShareTheCheckList; 4 TestNewAndUpdateReturnFindingsAndOneFix (mcp); 5 seid before/after, note 12:28; 6 TestPatternHoldsForOldOpenTasks + seid zjedy0 detail; 7 TestBacklogLeftovers + seid counts 0 -> 2387 -> 0 after strip; 8 TestSetTextFixesWithoutEditor; 9 TestLintOverviewDetailAndSameBlock; 10 same test, ls part (no finding text in rows), TUI row goldens unchanged; 11 TestBacklogLeftovers + upgrade tests, seid strip adds 0 lines and a second run changes 0 files. Fixtures from 6ephqk/16dxsr tests got Purpose/Principle (stageDoc, draft doc) or depLint; TestWaitsThroughCLI now reads lint ID (the overview has no full finding text by design).

hatchling 13:04: rebased on f583cac (sfa8w4 strike) as a0dfe06: kept main's strike in cli/mcp allowed maps, help usage and MCP update schema, re-added set's goal/purpose/principle/details/accept; strike tests got Purpose/Principle and read lint ID. CI 44088940602; x325pp stacked on it as 75dc78b, CI 44088956385.

hatchling 13:21: rebased on d74af5d (wyvern, 0sbvnq parallel tests, m1c28r env): boardFindings now takes the app's JCODE_HOME (a.getenv) for stale-owner, tests use useEditor(t, a, ...). ci/fkpyg3 3415936, CI 44091815701; x325pp stacked 18474c9, CI 44091816614. Local full suite green on 18474c9.


hatchling 14:04: on main b023624 (with x325pp 38ff464). Landing CI 44096348851 (see x325pp note); full matrix green on the same code in 44095242756.
Reflection (dismissed, 2026-11-17, hatchling): AGENTS.md's TUI golden line said HUGINN_UPDATE_GOLDEN while tests read HUG_UPDATE_GOLDEN (since fixed on main by m1c28r). The 'every error ends with a fix: line' rule shaped the one combined fix design well.
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G8, goldens env var name: fixed by m1c28r (2c27e422); AGENTS.md says HUG_UPDATE_GOLDEN.
## History
- 2026-11-17T14:36 doing -> doing/waiting (wait)
- 2026-11-17T15:03 doing/waiting -> done (done); was waiting_asked_by: hatchling; waiting_on: landing turn; waiting_since: 2026-11-17T14:36
