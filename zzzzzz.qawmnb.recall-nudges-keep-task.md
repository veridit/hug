---
id: qawmnb
title: "Recall nudges: keep the task board current"
status: "doing"
type: feature
priority: ""
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Agents keep the board true while they work: they take a task before starting, note evidence, and close it when every check is ticked

## Purpose
The board is how the user and other agents see progress; overnight 2026-11-16 eight tasks were closed and none taken, so nothing ever showed as in progress

## Principle
Use the recall hook we already run on every user message; nudge only when there is a concrete mismatch (work happening but nothing in doing, a doing task idle for hours, a task done in conversation but open on the board). One short line, never every turn. A setting turns it off

## Relevant details
Signals the hook already has: the session's last task commits on hug/tasks, its doing tasks, time since last update, the user message. Harnesses with a built-in todo list (jcode, Claude Code, Codex) get 'add the hug task to your todos'; harnesses without one (pi) could later get a minimal hug-provided session todo. Overnight evidence: hug/tasks log has 8 'hug task done' and 0 'hug task take' commits between 00:00 and 09:38

## Depends on

## Acceptance
- [ ] recall injects at most one board line per session hour, only on a detected mismatch, with the exact command (hug task take ID / done ID)
- [ ] setting nudge_board = false turns it off
- [ ] a replay on real sessions shows the nudge fires on the overnight session and not on sessions with a current board

## Notes
