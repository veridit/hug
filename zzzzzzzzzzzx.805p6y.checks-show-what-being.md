---
id: 805p6y
title: "Checks show what is being worked on now, with a subtle line in the TUI"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
A ticket shows the work in progress, not only what is ticked: one acceptance check can be marked as current with a short now-text (for example 'waiting for CI run 44049348743'), and the TUI shows that line, dimmed, under the IN PROGRESS row and on the check in the detail view.

## Purpose
Jørgen 2026-11-17: tickets read like 'I did, I did, I did'; he wants to see what is happening now, such as waiting for CI as part of an acceptance check. The coordinator's own work this morning was invisible on the board: the landings waited on CI and on a worker fixing SQLITE_BUSY, and only the harness todo list showed it. This makes the board, not the harness, the place to see current work (see 5dxzxe, tzc9hb).

## Principle
Subtle: one dim line, only on tasks in progress, never more than one line per task. The task file stays the truth and plain markdown: the now-text is an indented line under the check, so other renderers show it as text. One current check per task. Ticking the check or writing a new now-text replaces the old one. No history kept beyond the ticket's Notes.

## Relevant details
Proposed command: hug task now ID N "text" (MCP task_update action now); hug task check ID N clears it. File form under the check: '  now: text (agent, time)'. TUI: IN PROGRESS row gets a dim second line '↳ 3. CI green on all three systems: waiting for run 44049348743 · 4 min'; the detail view marks the check ◐. Later, with 16dxsr, a now-text naming a condition (CI run, branch merged) can be checked by hug and nudge when it ends. Coordinate with 102bc0 (rows fit one line) and zhvkmj (status colors).

## Depends on

## Acceptance
- [ ] hug task now ID N "text" writes the now-line under check N; hug task check ID N removes it (test)
- [ ] the IN PROGRESS row shows one dim line with the check number, its text and the now-text, cut to the terminal width (golden)
- [ ] the detail view marks the current check and shows its now-text (golden)
- [ ] MCP task_update supports action now (test)
- [ ] Jørgen confirms the look on his board after release

## Notes
