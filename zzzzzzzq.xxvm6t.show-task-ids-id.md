---
id: xxvm6t
title: "Bare task ids without brackets; task:ID accepted and followable"
status: "doing"
type: feature
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
A task id shows bare in its colour wherever a person reads it (TUI rows, detail, prompts), with no brackets and no sigil. In running text the reference form is task:ID: every command that takes a ref accepts it, and in the TUI a task:ID in notes or description can be followed.

## Purpose
Jørgen 2026-11-16: the TUI wraps ids in brackets, [9taa9d], which mean nothing and cost two columns. He wants a hash sign, and the hash to do linking everywhere

## Principle
One way to write an id for people: #id. Brackets go. What you see is what you can type: the # key in the TUI already jumps to an id. Commands keep taking the bare id, because an unquoted #id at the start of a shell word is a comment and the argument would vanish. The stored reference form task:ID keeps working; #id is accepted alongside it, not instead

## Relevant details
TUI: taskRowMark and the overview rows in huginn/internal/tui/render.go render "["+id+"]" in a width of 8; detail.go badges render [status] [type] [priority] [id]. Row becomes #9taa9d in 7 columns; decide whether the other badges keep brackets. CLI: show/ls/next output and next:/now:/fix: lines print commands, so ids inside commands stay bare. References: lint and promote use task:ID today ('done when task:X is done'); lint ignores placeholders like task:REF/NAME. Add #id as a recognised reference in the same resolver, with the same placeholder rule, and make sure '#' followed by a non-id word (markdown heading, '#1', colour codes like #ff7373) is not taken as a reference: require the id alphabet and length, and an existing task for a link. TUI detail: Enter on a #id in notes or text follows it (today only dependencies can be followed). Memory: hug mem text may mention #id; linking there is display only. Goldens change in every view

## Depends on

## Acceptance
- [x] no TUI view shows an id in brackets or with a sigil; rows, detail and prompts show the bare id in its colour (goldens)
- [x] every command that takes a ref accepts task:ID as well as the bare id (today 'hug task show task:9taa9d' prints not-found) (test)
- [x] every printed command still uses the bare id and runs when pasted into a shell (test)
- [x] in the TUI a task:ID in notes or description can be followed to that task, and Esc comes back (test)
- [ ] full tests pass; goldens reviewed

## Notes

2026-11-16 ON HOLD, design under discussion with Jørgen: '#id' cannot be pasted into a shell command (bash treats an unquoted word starting with # as a comment, so the id vanishes). Do not start until the written form of an id is decided. Options discussed: bare coloured id, task:ID, another sigil.

2026-11-16 DECIDED with Jørgen, hold lifted. This note overrides the goal, details and checks above wherever they say '#id'. No sigil: not '#', not '@', and no '.task:' form for now. (1) TUI shows the bare id in its colour, no brackets, in rows, detail and prompts. (2) In running text (notes, goals, checks) the reference form stays task:ID, and in the TUI a task:ID in notes or description can be followed. (3) Commands accept task:ID as well as the bare id; today 'hug task show task:9taa9d' prints 'not-found: task:9taa9d'. (4) Printed commands keep the bare id. Checks to read as: 1 no brackets and no sigil around ids in any TUI view; 2 task:ID accepted by every command that takes a ref; 3 dropped (no '#' parsing); 4 unchanged; 5 a task:ID in notes or description can be followed; 6 unchanged. Evidence for dropping '#': in bash, zsh and sh an unquoted '#9taa9d' argument is a comment and vanishes.

2026-11-16 15:39 crab: worker poodle's session ended at the 15:21 server reload. Its uncommitted work is in the tree: huginn/internal/cli/cli.go (+23), huginn/internal/task/query.go (+15), new huginn/internal/cli/taskref_test.go and huginn/internal/mcp/taskref_test.go (task:ID accepted). taskref_test.go uses a #!/bin/sh editor script that fails on Windows: switch to useEditor(t, "", "") from editor_test.go.

check 2: gh run view 43899146680 (ci/xxvm6t @ ea0bb25, same files as main 32fa085): secrets success / test (ubuntu-latest) success / test (windows-latest) success / test (macos-latest) success; TestEveryRefCommandAcceptsTaskColon (21 commands with task:ID) and TestToolsAcceptTaskColon (MCP) pass

checks 1,4,5: gh run view 43902441497 (ci/xxvm6t @ 43299f3 = main HEAD): secrets success / test (macos-latest) success / test (ubuntu-latest) success / test (windows-latest) success. Local: go test -race ./huginn/internal/tui/ -> 'ok github.com/veridit/hug/huginn/internal/tui 57.591s'. TestIDsBare (overview, list, board, detail: no [ID], no #ID), TestFollowTaskRefAndBack (Tab to LINKS, j, Enter opens Beta, trail 'ID › ', Esc back to Main, missing task:zzzzzz says so). Goldens: 28 files, only [ID] -> ID and 'list › ▸ ID' in the trail, reviewed.

check 3: TestEveryRefCommandAcceptsTaskColon fails if any next:/fix: line printed after 21 task:ID commands contains task: (run 43899146680 success on all OSes)
