---
id: fkpyg3
title: "Every check reports every problem at once, with one fix line"
status: todo
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
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

## Notes

Jørgen 2026-11-17 11:00, on zjedy0 in Seid: the detail shows ONE finding (no acceptance), but the task also lacks Purpose and Principle; its whole description sits in Goal, including backlog's <!-- SECTION:DESCRIPTION --> markers; the title carries tags (.Refactor .Compiler.Lang); and backlog's leftover fields remain (created_date, updated_date, ordinal, backlog_status, import_type_inference). Cause: query.go:237 PatternSince = 2026-11-16. Purpose and Principle are checked only on tasks created since then, which hides it on 1,215 of Seid's 1,236 open tasks. The leftover markers and fields have no check at all (1,208 open tasks have the markers, 1,214 the fields). Scope added: (a) drop PatternSince, so every open task is held to the pattern (done and dropped stay exempt); (b) new findings for import leftovers: backlog markers and fields that hug doesn't use; hug upgrade removes them; (c) tags in the title go to txjkrm's cleanup; (d) the fix line must be runnable, but 'hug task edit ID' opens an editor, and set has no --accept, so add 'hug task set ID --accept "..."' (and --purpose/--principle) so agents can fix everything with one command.
