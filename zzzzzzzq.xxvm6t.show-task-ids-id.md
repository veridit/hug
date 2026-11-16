---
id: xxvm6t
title: "Show task ids as #id, and link #id everywhere"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
A task id is written #9taa9d wherever a person reads it (TUI rows, detail, prompts, CLI text), without brackets, and #id in any text (notes, goals, checks, memories) is a link to that task

## Purpose
Jørgen 2026-11-16: the TUI wraps ids in brackets, [9taa9d], which mean nothing and cost two columns. He wants a hash sign, and the hash to do linking everywhere

## Principle
One way to write an id for people: #id. Brackets go. What you see is what you can type: the # key in the TUI already jumps to an id. Commands keep taking the bare id, because an unquoted #id at the start of a shell word is a comment and the argument would vanish. The stored reference form task:ID keeps working; #id is accepted alongside it, not instead

## Relevant details
TUI: taskRowMark and the overview rows in huginn/internal/tui/render.go render "["+id+"]" in a width of 8; detail.go badges render [status] [type] [priority] [id]. Row becomes #9taa9d in 7 columns; decide whether the other badges keep brackets. CLI: show/ls/next output and next:/now:/fix: lines print commands, so ids inside commands stay bare. References: lint and promote use task:ID today ('done when task:X is done'); lint ignores placeholders like task:REF/NAME. Add #id as a recognised reference in the same resolver, with the same placeholder rule, and make sure '#' followed by a non-id word (markdown heading, '#1', colour codes like #ff7373) is not taken as a reference: require the id alphabet and length, and an existing task for a link. TUI detail: Enter on a #id in notes or text follows it (today only dependencies can be followed). Memory: hug mem text may mention #id; linking there is display only. Goldens change in every view

## Depends on

## Acceptance
- [ ] no TUI view shows an id in brackets; rows and detail show #id
- [ ] #id in a note, goal or check resolves like task:ID in lint, show and promote; task:ID still works
- [ ] a markdown heading, a colour code and '#1' are not treated as task references (test)
- [ ] every printed command still uses the bare id and runs when pasted into a shell (test)
- [ ] in the TUI a #id in notes or description can be followed to that task
- [ ] full tests pass; goldens reviewed

## Notes
