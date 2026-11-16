---
id: 0sw0dd
title: "TUI navigation, search and sort"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Moving between states, searching and ordering in hug tui is quick and readable without reading help

## Purpose
The user's 2026-11-16 review: status line keys are hidden, status cannot be searched or sorted, the header lacks the project name, and memory and tasks are separate TUIs

## Principle
One line where two say the same; every key shown is a key that works; search is one syntax for free text and field:value; reuse SQLite FTS5 BM25 rather than inventing ranking

## Relevant details
Requests: (1) status counts line shows its keys (letters or matching column initials) and merges with the tab line, using a segment marker; (2) header 'HUG' plus project name (folder or hug.toml name); (3) switch between task and memory mode from the header; (4) search by field, e.g. status:done type:bug, with BM25 over title/goal/notes; (5) a sort key (sigil to decide: / is search, f is filter); (6) epics and groups visible in rows and detail. Screens: user screenshots 2026-11-16 09:36

## Depends on

## Acceptance
- [ ] status line keys visible and working; tab and status share one line
- [ ] header shows HUG and the project name
- [ ] status:done in / search returns done tasks
- [ ] a sort key orders the list by status, priority, updated or rank
- [ ] epic and group visible in list rows and detail

## Notes
