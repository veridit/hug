---
id: ge73ea
title: "Filing a task shows open tasks that look like it, before it is written"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-28
updated: 2026-11-28
---

## Goal
hug task new (CLI, MCP task_new, the TUI) finds open tasks close to the new one and asks the filer to check them before writing a duplicate.

## Purpose
Jørgen 2026-11-28: crab checks for an existing task before filing; automate it, so every agent and person gets it without knowing to search.

## Principle
The same shape as memory_remember's duplicate check (dedupe.go:20-27 DuplicateCheckError: NOT STORED, this looks like X; reinforce or store with a reason): above a closeness bar nothing is written and the reply lists the close tasks with ids, then two ways on: work on one of them, or file anyway with a reason (--new-reason, kept on the task). Below the bar it files and says nothing. Local and fast, no model call. Open tasks only; a done or dropped one is shown only when very close, as a likely repeat.

## Details
Today: hug task new writes at once; no similarity check exists (grep of huginn/internal/task for similar/duplicate finds only lint codes duplicate-id, duplicate-field, duplicate-origin). hug task search matches one exact phrase: 'MCP memory scope server working directory caller' returned no task although 5kn3n7 is exactly that, so the check needs word-level ranking over title, goal and purpose (e.g. BM25 over the board, ~400 tasks), not that search. Board size: hug task ls lists all tasks in milliseconds (search ran in 0.06 s). Bar to set from real data: replay the last 200 filed tasks against the board as it was, count the true duplicates found and the false alarms.

## Depends on

## Acceptance
- [ ] Filing a task close to an open one writes nothing and lists the close tasks (id, title, state) with the two ways on: hug task show ID, or file again with --new-reason "why it is different" (test, CLI and MCP)
- [ ] A task with no close open task files at once, with no extra output (test)
- [ ] On a replay of the last 200 filed tasks against the board as it was then, the check finds the known duplicates (tasks dropped as 'duplicate of') and alarms on at most 1 in 10 of the others (numbers noted)
- [ ] The check adds under 100 ms to hug task new on this board (measured)

## Notes
