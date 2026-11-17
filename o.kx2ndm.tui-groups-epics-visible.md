---
id: kx2ndm
title: "TUI: groups and epics visible and collapsible"
status: "doing"
type: feature
priority: "high"
assignee: ["vole"]
labels: ["needs-discussion"]
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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

2026-11-16 Jørgen answered 'I like your lint plan. Yes' right after the overview proposal; taken as approval of the severity plan (pbh9xq). The overview LINT section question (list findings there, or drop the section) is still open with him.

2026-11-16 Jørgen: the IN PROGRESS row already carries everything (the ! lint mark and the blocked status), but the overview has no legend for '!'. Add one, compact, no extra line: the list view footer already says '! lint finding · status blocked = unmet dependency', the overview footer does not, and the header only says '! 1 findings'. Coordinator's reading of the overview layout, stated to him: one row per task, in IN PROGRESS when taken (status drawn blocked, with what it waits on); BLOCKED only for tasks nobody started; the LINT section shows the finding text and fix, not the task row again.

vole 2026-11-17 question for Jørgen (also on tmp/lanes/morning.md). Proposal, building the first part now: (A) every row gets one short dim cell after the title's start, no extra line: 'later · ↳gem59b' (group, then the epic id with ↳); shown only when set, cut first when the row is tight (102bc0 order). (B) NEXT UP leaves out tasks in group 'later' (like hug task next should), and a folded heading 'LATER [3]' under NEXT UP holds them; Space on it unfolds. Questions: 1) Should 'later' be special by name, or should any group be markable as deferred (e.g. hug.toml deferred_groups = ["later"])? 2) The grouping toggle (check 2): one key, g, cycling flat → by group → by epic, header says 'grouped by epic'. g is free outside gg; or use z (vim's fold prefix)? 3) Do group headings start folded for 'later' only, or for all groups except the one holding the selection?
