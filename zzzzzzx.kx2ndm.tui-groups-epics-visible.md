---
id: kx2ndm
title: "TUI: groups and epics visible and collapsible"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
In overview, list and board a reader sees which group (e.g. later) and which epic each task belongs to, and can fold a group or epic away

## Purpose
Jørgen 2026-11-16: e4w9ya is in group later, but the overview shows it among NEXT UP with nothing saying so; he pressed Space on a row expecting a group to expand or collapse

## Principle
Compact: no extra line per task. Group and epic are shown as section headings or a short dim cell, not repeated prose. Space folds or unfolds a heading; it never ticks a check outside the detail view. Only show keys that work

## Relevant details
CLI already shows epic:/child: lines and has --epic. TUI has filter fields g (group) and e (epic) only. Code: huginn/internal/tui/render.go (overviewText, listText, taskRowMark), tui.go (paneIndices, move). task.ChildIndex resolves parents once; do not resolve per row (ls on 2,498 tasks went 19 s to 0.8 s that way). Decide with Jørgen via the coordinator: view by group vs by epic as a toggle, and whether later/ groups start folded. Goldens: HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./huginn/internal/tui/

## Depends on

## Acceptance
- [ ] list and overview show each task's group and epic without adding a line per task
- [ ] a key toggles grouping by group, by epic, or flat; the header says which
- [ ] Space on a group or epic heading folds and unfolds it, and the count stays visible when folded
- [ ] tasks in group later are visibly set apart from NEXT UP
- [ ] tests cover grouping, folding and the 80-column width; goldens reviewed

## Notes

2026-11-16 Jørgen on the overview: qawmnb (doing, with an unmet dependency) shows in IN PROGRESS, again in BLOCKED and again in LINT. A task must appear in one section only. Under discussion, coordinator's proposal: a task someone is working on stays in IN PROGRESS with its status drawn as blocked (red) and what it waits on; BLOCKED holds only tasks nobody has started that cannot start; LINT lists the findings (problem and fix), not the task rows again. Not decided yet.
