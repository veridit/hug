---
id: fkpyg3
title: "Every check reports every problem at once, with one fix line"
status: "doing"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] hug task new "x" names every missing field (--goal, --accept, --purpose, --principle) in one reply with one fix: line containing all of them (test)
- [ ] a task with missing Purpose, missing Principle and an 87-character title gets one lint entry listing all three and one fix: hug task set ID --purpose ... --principle ... --title ... (test)
- [ ] write-time checks and lint run the same check list: a task new/set accepts is lint-clean except for findings that need time (e.g. done-unchecked) (test enumerates the codes)
- [ ] MCP task_new/task_update return findings as a list plus the combined fix (test)
- [ ] on seid-lang, hug task lint output has one entry per task, and the count line still shows totals per code (shown before and after)
- [ ] every open task is held to the description pattern: PatternSince no longer exempts open tasks created before 2026-11-16 (done and dropped stay exempt); on seid-lang zjedy0's detail lists no acceptance, missing Purpose and missing Principle together (shown)
- [ ] new findings for backlog import leftovers: <!-- SECTION:... --> markers, and the unused fields created_date, ordinal, backlog_status, import_type_inference; their fix is hug upgrade (test; seid-lang counts before and after)
- [ ] hug task set ID --accept "..." --purpose "..." --principle "..." fixes those findings without an editor, so every combined fix: line is runnable (test)
- [ ] hug task lint is the board overview: totals per code and one compact line per task; hug task lint ID gives every finding of that task in detail with the one combined fix: line; hug task show (detail view) and hug task take show the same block as lint ID (test)
- [ ] rows in ls, board and the TUI list stay as they are: at most the existing ! marker, no finding text or counts (test)
- [ ] hug upgrade does only the mechanical part, shown as a diff: strips backlog SECTION markers, drops unused backlog fields, puts sections in order; it never splits Purpose or Principle out of Goal (test)

## Notes

Jørgen 2026-11-17 11:00, on zjedy0 in Seid: the detail shows ONE finding (no acceptance), but the task also lacks Purpose and Principle; its whole description sits in Goal, including backlog's <!-- SECTION:DESCRIPTION --> markers; the title carries tags (.Refactor .Compiler.Lang); and backlog's leftover fields remain (created_date, updated_date, ordinal, backlog_status, import_type_inference). Cause: query.go:237 PatternSince = 2026-11-16. Purpose and Principle are checked only on tasks created since then, which hides it on 1,215 of Seid's 1,236 open tasks. The leftover markers and fields have no check at all (1,208 open tasks have the markers, 1,214 the fields). Scope added: (a) drop PatternSince, so every open task is held to the pattern (done and dropped stay exempt); (b) new findings for import leftovers: backlog markers and fields that hug doesn't use; hug upgrade removes them; (c) tags in the title go to txjkrm's cleanup; (d) the fix line must be runnable, but 'hug task edit ID' opens an editor, and set has no --accept, so add 'hug task set ID --accept "..."' (and --purpose/--principle) so agents can fix everything with one command.

Jørgen 2026-11-17 11:01: the full findings matter in the detail view and in what the agent learns (CLI/MCP replies on show, take, new, set, lint). They must NOT pollute the listing: rows stay as they are (at most the existing ! marker), with no finding text or count on rows. Detail view: one block listing every finding of the task, then the one combined fix: line.

Jørgen 2026-11-17 11:08 (lint has two levels, so nobody is overwhelmed and we don't overwork): (1) board lint: 'hug task lint' gives the overview: totals per code, one compact line per task, no full fix text for every task. (2) task lint: 'hug task lint ID' (today it fails: 'expected 0 positional arguments, got 1') gives every finding for that one task, in detail, with the one combined fix: line. The detail view and 'hug task take' show the same as task lint, so taking a ticket means cleaning it up. hug upgrade does only the mechanical part (strip backlog SECTION markers, drop unused backlog fields, put sections in order, with a diff); splitting Purpose and Principle out of Goal is judgment, done per task when it is linted or taken, never in bulk.

guppy (hatchling) 12:40, plan, each step committed on ci/fkpyg3: (1) task.ByTask groups findings per task and task.CombinedFix merges every 'hug task set ID --x' fix into one command (others listed once, hug upgrade once); hug task lint = totals per code + one compact line per task ('ID  code, code  -> hug task lint ID'); hug task lint ID = every finding in detail + one fix: line; JSON keeps findings[] and adds tasks[] {id, path, codes, problems, fix}. (2) set gains --accept (repeatable, appends), --purpose, --principle, --details; missing-section and no-acceptance fixes become set commands. (3) new collects every missing field (title, --goal, --accept, --purpose, --principle, title length) in one error with one fix line; same check list as lint. (4) PatternSince dropped for open tasks; backlog leftovers findings (SECTION markers, created_date/updated_date/ordinal/backlog_status/import_type_inference/milestone/dependencies/origin? only the unused ones), fix hug upgrade; upgrade strips them with a diff. (5) show and take print the lint ID block; rows unchanged. (6) MCP returns findings[] + fix.
