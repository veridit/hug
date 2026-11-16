---
id: 5g86n6
title: "hug task layout: .hug/tasks, epics, promote, task files"
status: "doing"
type: feature
priority: ""
assignee: ["Jørgen H. Fjeld"]
labels: ["hug", "layout"]
created: 2026-11-15
updated: 2026-11-15
---

## Goal
Tasks live at .hug/tasks/ on branch hug/tasks with three fixed levels (epic, task, acceptance item), groups and epics as independent views, and per-task file folders with checked references

## Depends on

## Acceptance
- [x] hug upgrade moves .huginn/ (branch huginn) to .hug/tasks/ (branch hug/tasks) in one commit, idempotent; hug setup detects the old layout and names the command; every other task command on an old repo says fix: hug upgrade
- [x] parent field: an epic is a task with children and has no parent; a task has at most one parent, an epic; done EPIC refuses while a child is open (exit 3); lint flags missing parent, cycles, too-deep chains
- [ ] import backlog maps parent_task_id through aliases: seid-lang's 190 links come back; re-run unchanged
- [ ] hug task promote ID N: item N becomes a sibling task under the same epic (standalone if none); the item reads 'done when task:<id> is done' and ticks when that task is done
- [ ] ls/next/TUI: --group and --epic views; next skips epics
- [x] task files in .hug/tasks/<group>/<id>/; attach/show --file/detach; text only, 256 KiB, secret check; references task:<id>/<path> checked by lint; backlog .backlog/evidence imported with citations rewritten

## Notes

Item 1 done in v0.7.3 (d955c8f, released 2026-11-15): hug upgrade = git branch -m + worktree move + remote rename; old layout fails with fix: hug upgrade; setup lists it. Verified: copy of seid-lang (2,494 tasks); this repo (31 tasks, remote hug/tasks); the seid agent upgraded seid-lang itself from the CLI/MCP instruction.

Item 6 built 2026-11-15 (9225b0a): attach/show --file/detach (CLI + MCP inline content), task folder <group>/<id>/ moves with the task, file:NAME and task:REF/NAME references, lint (missing refs, binary/oversized, forbid_cite in hug.toml). Generic import: 'hug task import files DIR --cite PREFIX' (not tied to Backlog); hug setup detects it. seid-lang copy: 4,738/4,741 files (3 under TASK-1958, whose task file does not parse), 700 citations rewritten, rerun unchanged, 0 reference findings.

v0.7.4 (2026-11-16) shipped item 6 (task files, import files, hug.toml) plus TUI fixes from user review: acceptance navigation (j/k, hanging indent, highlight), Space confirm names the item, o/i/b/! in detail, overview sizing with ↑/↓ and 'a-b shown', owner -> assigned; ls id-first columns + --jsonl; shorter slugs (4 meaningful words, <=32 chars); rebalance one-line output.

Item 3: import maps parent_task_id through the alias table (TestImportParent: resolves, rerun unchanged). seid-lang copy: 294 source parents, 282 restored, 20 flagged by lint as too deep with an exact fix ('hug task set X --parent EPIC'); the 12 unresolved point at tasks absent from the source.
