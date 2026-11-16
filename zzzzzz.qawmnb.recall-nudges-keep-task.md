---
id: qawmnb
title: "Recall nudges: keep the task board current"
status: "todo"
type: feature
priority: ""
assignee: ["guppy"]
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

User direction 2026-11-16 09:39-09:41: the nudge replaces doctrine. Instead of instructions ('do an honest check', 'keep the board current'), the system says the right thing at the right moment, chosen by memory and semantics (the message and what the agent is about to do), not by a timer. Cases named: starting work -> take the task and put its acceptance items in your todos; checks met -> tick them and close the ticket; new ticket -> todos to follow it up and verify. Harnesses with built-in todos (jcode, Claude Code, Codex) get 'use your todos'; pi may later need a hug-provided session todo. Treat tool deficiencies, irrelevant memories and missed reminders as bugs to fix continuously.

part 1 committed a23d4e6: task.Guide lines printed as 'now:' (CLI) and 'guide' (MCP) on take/check/done/drop/promote. Per Jørgen 2026-11-16: one todo per task, not per check; a detoured task stays doing. go test ./huginn/... all ok.

Part 1 released in v0.7.15: now: lines on take/check/done/drop/promote, MCP guide field; take names the todo by task id.

2026-11-16 Jørgen: also nudge about lint findings, the way dream items are handed out, so a board with problems gets cleaned up over time (seid has thousands). Today nothing does: hug task next, take and the recall hook never mention findings; only 'hug task import' ends with 'next: hug task lint'. Counts on the seid copy: 1158 no-acceptance, 827 long-title, 432 done-unchecked, 24 missing-goal, 20 parent-too-deep, 10 duplicate-section, 4 missing-section, 2 dependency-cycle. Proposal for part 2: reuse the dream cue (muninn/internal/recall/dream.go: only at a break, at most once per 6 h per session), one line, one finding, with its fix command; prefer findings on open tasks and on tasks the session touched; never a count dump. Depends on rx74bp so each finding has a usable fix line.

2026-11-16 crab: back to todo. Part 1 shipped (v0.7.15); nobody is building part 2 yet, so doing was false. Staff it next.
