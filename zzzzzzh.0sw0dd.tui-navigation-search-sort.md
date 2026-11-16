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

v0.7.15 released and installed 2026-11-16 (commit 6d36b9e, release run 'completed success', 'hug version' prints 'hug v0.7.15'): the review batch from the earlier note is shipped. Follow-ups are their own tasks: mv4vye (detail focus, prompts, hints), 05r3pg (context palette), kx2ndm (groups and epics).

2026-11-16 Jørgen answers the key-scheme question (item 1): the header is a tree, the same tree as the palette. Each header line is a branch with its letter marked in place, each value has its letter marked: 'view: overview list board lint', 'status: todo doing blocked done dropped', 'filter: ...'. Typing the branch letter then the value letter does it (v o = view overview, s d = status doing), and the palette spells the same path ('view overview', 'status doing'). Digits 1-4 are freed. He likes the f filter picker as it is. Today: s opens the palette prefilled 'status ', v and i both open list, o/b/! switch view, 1-4 filter status (tui.go). Open points: doing, done and dropped all start with d, so the status letters need choosing; the word is 'status' in the CLI while he said 'state'; whether one-key o/i/b/! stay beside v+letter. Pressing a branch letter should show its choices in place (no timeout), like the pending g. Build together with 05r3pg so header, palette and help (jmx10s) share one tree definition.

2026-11-16 Jørgen decides item 2 (header): 'HUG · <project>' where project defaults to the last piece of the checkout's path (the folder name, so two checkouts of one repo read differently, e.g. hug and hug-testing), with an override setting included from the start, not later. A key shows the full path of the checkout. Not built yet: the worker on mv4vye is editing the same files, so this waits for it to land.

2026-11-16 bug from Jørgen in v0.7.15: in the palette 'f g ' (short filter, short field, space) said 'No completions'. Fixed in a local commit (palette.go expandField, test TestPaletteShortFieldAfterFilter: 'ok github.com/veridit/hug/huginn/internal/tui'). Not pushed or released; ships with mv4vye.

2026-11-16 Jørgen approves the status letters t todo, d doing, b blocked, o done, r dropped, shown marked in place, and asks that letters are picked by a function, not by asking him. Built and committed locally (not wired in, not released): huginn/internal/tui/shortcuts.go assignShortcuts(names, reserved) gives each name its first letter no earlier name took. Test TestAssignShortcuts: 'ok github.com/veridit/hug/huginn/internal/tui'. It yields statuses tdbor, types fbcsd, priorities chml, filter fields stpglea (same as the f picker today), views o l b i (overview, list, board, lint), which differs from today's keys where list is i and lint is !. The header tree, the palette (05r3pg) and help (jmx10s) are to take their keys from it.
