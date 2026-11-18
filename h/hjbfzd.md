---
id: hjbfzd
title: "Screen capture: attach to a chosen task, pick one, or create a new task"
state: "doing"
type: feature
priority: "high"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:16"
stage: "active"
model: "claude-opus-5-5"
branch: "hjbfzd"
worktree: "/Users/jhf/.jcode/scratch/hjbfzd"
---

## Goal
After S captures the screen in hug tui, the prompt offers where it goes: type a task ID (completion), s to pick a task from the list, or c to create a new task with the capture attached; Enter attaches, Esc cancels. Today the capture only attaches to the focused task.

## Purpose
Jørgen 2026-11-18 (yf39ya requests 1-2): 'when I take a screenshot, it would be nice if I could also attach it to ... ticket so-and-so or S for select or C for create. And if I press select, I can go pick one. And if I put create, I can create a new one.'

## Principle
One keystroke per choice, shown in the prompt. Selecting a task reuses the list's own navigation; creating one asks only for a title, and lint's missing sections are filled later through the ticket. The capture is never lost: Esc says 'not attached' and keeps nothing on disk.

## Relevant details
Capture today: S takes the frame (tui.go captured, picture.go attachCapture, render.go:1029 'Attach this screen to ID as NAME'). Attach via store.Attach. Pairs with 5z4zt4 (attach screenshots) and q2adr8 (one question per ask). The prompt follows the calm-TUI rules: underline marks the key letter, no background fills.

## Depends on

## Acceptance
- [ ] After S, the prompt reads e.g. 'Attach this screen to: ID_ · s select · c create · Esc' and typing a task ID (prefix completes) then Enter attaches it there (key test)
- [ ] s opens the task list in pick mode; Enter on a row attaches the capture to that task and returns (key test)
- [ ] c asks for a title, creates the task with the capture attached, and opens it (key test; the new task has the file in Files)
- [ ] Golden of the prompt at 80 and 140 columns
- [ ] Jørgen confirms on his board after release

## Notes


## History
- 2026-11-18T21:16 todo -> doing/active (take)
