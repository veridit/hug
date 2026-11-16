---
id: 05r3pg
title: "TUI: context command palette, inline at the bottom"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Pressing : opens a palette in the bottom part of the screen, leaving the task or list visible above, and offers first the actions for what you are in (this task, this check, this list), then the global ones, each as a tree

## Purpose
Jørgen 2026-11-16: he opened the palette inside a task and got a full-screen global list that hid the task and offered nothing about it. He wants to say 'edit this ticket' and have the palette know which ticket, because that is where he is

## Principle
Context first, global second, each under a short heading. The palette never hides what it acts on. A tree, not a flat list: one entry per branch (filter, then field, then value). Every action reachable by a key is also in the palette, with its key shown, so the palette teaches the keys. Compact, vim keys, Esc backs out one level then closes

## Relevant details
Today (v0.7.15): huginn/internal/tui/palette.go holds a global tree (overview, list, board, lint, filter > field > value, goto, clear, help), completions() and executeCommand(); render.go paletteText() replaces the whole body. Wanted: (1) render the palette over the bottom rows only, about a third of the height, with the current view still drawn above; (2) a context group whose heading names the subject, e.g. 'TASK s6ff5k' with take, done, drop, edit, set status, set type, rank up/down, note, and when a check is selected 'CHECK 2' with tick/untick and promote; in a list the subject is the selected row; (3) a 'GLOBAL' group with the existing tree; (4) each row shows its direct key where one exists (t take, d done, x drop, e edit, Space tick); (5) typing filters across both groups; (6) the footer must describe the palette, not the view underneath (today it still says 'Detail · 100% · Esc back'). Actions run through the existing requestAction/executeAction path so confirmation and commits stay the same. Depends on mv4vye (focus states decide what 'the selected check' is); do it after mv4vye lands

## Depends on

## Acceptance
- [ ] with a task open, : shows the task above and the palette in the bottom part of the screen
- [ ] the first group is headed by the open task's id and lists its actions; choosing edit acts on that task without asking which
- [ ] with a check focused, tick/untick and promote for that check are offered
- [ ] global commands follow under their own heading as a tree; filter is one branch
- [ ] rows show the direct key for actions that have one, and the footer describes the palette
- [ ] tests cover context in detail, in a list, and at 80x24; goldens reviewed

## Notes
