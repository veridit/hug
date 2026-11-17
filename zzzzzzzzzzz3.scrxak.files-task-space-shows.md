---
id: scrxak
title: "Files in a task: Space shows a file inline, n/p step, Esc or u back"
status: "doing"
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T15:02"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In a task's FILES section, Space shows the selected file in the detail pane, read-only, with a header 'path · size · 3 of 26'. j/k and PgUp/PgDn scroll, n/p show the next and previous file of the task, and Esc or u go back to the FILES list with the file you last saw selected. Enter still edits and E still opens the folder. A binary file, or a file over 1 MiB, shows one line instead of content ('binary, 32.6 KiB: Enter opens it'). Long lines wrap, tabs expand, control characters are escaped.

## Purpose
Jørgen 2026-11-17 11:05, looking at 26 evidence files under backlog-evidence/TASK-2034.02/ in a Seid task: to see what an evidence file says, he now leaves hug for the editor. That costs focus. Glancing through a few files is the common case, and editing them is rare.

## Principle
A glance, not a viewer: plain text only, no syntax highlighting, no Markdown rendering, no search, no editing. Same keys as elsewhere (j/k scroll, n/p step, Esc/u up one level, as 9xfnwr). Read the file only when it is shown, never on list or detail load. Never let a file's bytes reach the terminal raw.

## Relevant details
FILES section: huginn/internal/tui/detail.go:421 (hint 'j/k or [ ] step · Enter edits · E folder'). Done task zgr01t added the files section, the icons and Enter to the editor. Detail footer today: 'j/k scroll · n/p task · Esc back'. Inside the file view, n/p step files; outside it they step tasks. File reads go through the existing file-content checks (binary, too large).

## Depends on

## Acceptance
- [ ] Space on a file shows its text in the detail pane with the header 'path · size · i of N'; Esc and u return to FILES with that file selected (golden at 80 and 140 columns)
- [ ] n/p step to the next and previous file, stopping at the ends with a footer note; j/k and PgUp/PgDn scroll (test)
- [ ] a binary file and a file over 1 MiB show one line with the size and 'Enter opens it', not their bytes; ESC sequences and control characters in a text file are shown escaped (test with a file holding \x1b[2J)
- [ ] opening the detail and the list reads no file contents; showing a file reads only that file (test counts reads)
- [ ] Jørgen looks through the TASK-2034.02 evidence on Seid with Space and n/p, and says it works

## Notes


## History
- 2026-11-17T15:02 todo -> doing/active (take)
