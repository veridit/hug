---
id: 1phmre
title: "Lint reads a path quoted in a note on another board as a task link"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzq"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
missing-task-ref fires only for a real task: link, not for a .hug/tasks/<id>/... path quoted inside note prose that names another board's task.

## Purpose
w40cmk's note quotes Statbus's '.hug/tasks/37bryg/README.md' from a 2026-11-18 01:24 design discussion; lint reports it as a broken link on hug's board, and the only fix it offers is hug task edit (interactive).

## Principle
A finding must be true and its fix runnable non-interactively. Quoted history is not a link.

## Relevant details
Finding: w/w40cmk.md missing-task-ref '.hug/tasks/37bryg/README.md: no task 37bryg', fix 'hug task edit w40cmk'. 37bryg is a Statbus task.

## Depends on

## Acceptance
- [ ] Lint on w40cmk reports no missing-task-ref for the quoted Statbus path; a test with a quoted path in a note fails first on main.
- [ ] A real task: link to a missing id is still reported, with a non-interactive fix (test).

## Notes
