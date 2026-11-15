---
id: 5g86n6
title: "hug task layout: .hug/tasks, epics, promote, task files"
status: todo
type: feature
priority: ""
assignee: []
labels: ["hug", "layout"]
created: 2026-11-15
updated: 2026-11-15
---

## Goal
Tasks live at .hug/tasks/ on branch hug/tasks with three fixed levels (epic, task, acceptance item), groups and epics as independent views, and per-task file folders with checked references

## Depends on

## Acceptance
- [ ] hug upgrade moves .huginn/ (branch huginn) to .hug/tasks/ (branch hug/tasks) in one commit, idempotent; hug setup detects the old layout and names the command; every other task command on an old repo says fix: hug upgrade
- [ ] parent field: an epic is a task with children and has no parent; a task has at most one parent, an epic; done EPIC refuses while a child is open (exit 3); lint flags missing parent, cycles, too-deep chains
- [ ] import backlog maps parent_task_id through aliases: seid-lang's 190 links come back; re-run unchanged
- [ ] hug task promote ID N: item N becomes a sibling task under the same epic (standalone if none); the item reads 'done when task:<id> is done' and ticks when that task is done
- [ ] ls/next/TUI: --group and --epic views; next skips epics
- [ ] task files in .hug/tasks/<group>/<id>/; attach/show --file/detach; text only, 256 KiB, secret check; references task:<id>/<path> checked by lint; backlog .backlog/evidence imported with citations rewritten

## Notes
