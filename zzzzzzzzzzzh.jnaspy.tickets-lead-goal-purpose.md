---
id: jnaspy
title: "Tickets lead with Goal, Purpose, Principle; notes come last"
status: "doing"
type: bug
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
Opening a task anywhere (file, hug task show, TUI detail, MCP task_show) shows in this order: Goal, Purpose, Principle, Relevant details, Depends on, Acceptance, any other sections, then Notes (appended, oldest first), then History. hug upgrade puts every existing task file in that order.

## Purpose
Jørgen 2026-11-16, after reading several tickets: when you open a ticket you must see up front what it is about. Today the TUI detail shows Purpose, Principle and Relevant details after Notes and History (huginn/internal/tui/detail.go: they fall into the 'other sections' loop at the end), hug task show prints no Purpose or Principle at all (huginn/internal/cli/cli.go ~1088), and mkzcqr's file has Depends on before Purpose.

## Principle
One canonical section order, defined once in huginn/internal/task and used by the file writer, show, TUI and MCP. Notes only ever append at the end. hug upgrade reorders files without changing any section's text (no backwards compatibility kept; lint flags a file out of order with fix: hug upgrade). Unknown sections are kept, placed before Notes.

## Relevant details
Board today: 35 files Goal|Purpose|Principle|Relevant details|Depends on|Acceptance|Notes; 33 files Goal|Depends on|Acceptance|Notes; 1 Goal|Depends on|Purpose|Principle|Acceptance|Notes (zzzzzzo.mkzcqr). Writer: huginn/internal/task/document.go.

## Depends on

## Acceptance
- [x] TUI detail of a task with notes shows Goal, Purpose, Principle and Relevant details above Acceptance, and Notes then History last (golden)
- [x] hug task show prints Purpose, Principle and Relevant details after goal and before the checks (test)
- [ ] hug upgrade on a board with mkzcqr's order rewrites it to canonical order with identical section text, and a second run changes nothing (test)
- [ ] lint reports an out-of-order file with fix: hug upgrade (test)
- [ ] after release, hug upgrade on this repo's board leaves every file in canonical order (shown with a grep count)

## Notes

check 1: df60883 on origin/main (CI run 43936493365: ubuntu, macos, windows success). TestDetailSectionOrderGolden (huginn/internal/tui/sections_order_test.go) with testdata/detail/sections-000001-100.txt: Description, Purpose, Principle, Relevant details, then ACCEPTANCE, then Notes and History; tmux detail goldens regenerated in 4 profiles.

check 2: df60883 on origin/main (CI run 43936493365: ubuntu, macos, windows success). TestShowPrintsDescriptionBeforeChecks (huginn/internal/cli/sections_test.go): goal, purpose, principle, relevant details, then '1. [ ] Observable check', then notes; --json has purpose/principle/details.
