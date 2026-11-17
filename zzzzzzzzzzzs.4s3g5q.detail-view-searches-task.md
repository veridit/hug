---
id: 4s3g5q
title: "Detail view: / searches the task's own text, with n/N"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
In a task's detail, / works like less or vim: it finds the typed text inside this task (title, checks, notes, files, deps), highlights every match, scrolls to the first one at or after the cursor, and n/N step through the matches.

## Purpose
Jørgen 23:29 local, with a screenshot: in 6ephqk's detail he typed '/ row lay' to find 'row layout' in the notes, and got 'found 6ephqk, row 1 of 23 in WAITING', a task search that put him back on the task he was already reading. Long details (6ephqk has pages of notes) need an in-text find.

## Principle
One search key, and its meaning follows the view: in a list it finds tasks, in a detail it finds text. Matches use a subtle highlight, never red. Nothing is written. Every wait over 200 ms shows a spinner.

## Relevant details
Today's footer while typing: '↑↓ pick · Enter go to it · Tab keep as filter · Esc clear'. In the detail it becomes: 'match 2 of 5 · n next · N previous · Esc clear'. With no match: '0 matches in 6ephqk · Enter searches all tasks', so the task search stays one key away. Smart case like vim: all lower case ignores case. Matches inside a wrapped line still highlight both parts. Owner: nautilus, after e3sevr and 0gztsy (same files: render.go, helpkeys.go).

## Depends on

## Acceptance
- [ ] In the detail of a task, typing '/ row lay' highlights each match in the text and scrolls the first one into view (golden on a fixture with 3 matches, one in a wrapped note line)
- [ ] n and N step forward and back through the matches and wrap around, and the footer shows 'match K of M' (key test)
- [ ] With no match the footer reads '0 matches in ID · Enter searches all tasks', and Enter runs today's task search (key test)
- [ ] Esc clears the highlights and leaves the cursor where it is. In list, board and overview, / still finds tasks as before (existing tests stay green)
- [ ] Context help (?) and the footer list the detail search keys (help test)
- [ ] Jørgen confirms on his board after release
- [ ] A search typed in a detail or the file viewer never becomes a list filter: after u the overview shows every task, with no '[/ …]' line (key test, from Jørgen's 23:30 screenshot where '/ row lay' in 6ephqk hid all but one task)

## Notes

Jørgen 23:29: covers the inline file viewer too (glance): / finds text in the shown file, n/N step. Add as part of check 1-2's tests.
