---
id: 0sw0dd
title: "TUI navigation, search and sort"
status: "doing"
type: feature
priority: ""
assignee: ["Jørgen H. Fjeld"]
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

2026-11-16 review, built (uncommitted until goldens pass): (a) board rows compact: lint mark in the gutter left of the id, type and priority as one coloured letter, legend line under the board; (b) detail: { and } (also Ctrl-N/Ctrl-P) open the next/previous task of the list it was opened from, footer 'Task 3 of 22'; (c) palette uses the full height and is a tree: filter > field > value with task counts; ': f done' and ': filter done' find the value without the field; short forms (status todo, done) still run; (d) Space in a list no longer asks to tick a check you cannot see: it says 'Enter opens ID'; the prompt in detail reads 'Tick acceptance check N of ID? "text"'. Still open from the review: choosing board columns (wide vs letters), items 1-5.
