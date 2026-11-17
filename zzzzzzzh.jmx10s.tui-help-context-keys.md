---
id: jmx10s
title: "TUI help: context keys first, easy to scan and search"
state: "doing"
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
created: 2026-11-16
updated: 2026-11-16
doing_since: "2026-11-17T21:37"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The ? screen shows the keys for where you are first (this task, this list, this prompt), then the global ones, in a layout that is quick to scan and can be searched

## Purpose
Jørgen 2026-11-16: with keys moving (n/p step tasks, N new task) the shortcut list must be easy to read and search, and must lead with the keys relevant to the current context instead of one global list

## Principle
Context first, global second, each under a short heading. One key per row, key column aligned, plain words. Show only keys that work in that context. Same grouping and wording as the context palette (05r3pg), so the two teach each other

## Relevant details
Code: huginn/internal/tui/keys.go (help lines, keyBar hints), help_search_test.go (existing help search), render.go (help view). Contexts: overview, list, board, lint, detail with page/acceptance/dependency focus (from mv4vye), palette, filter picker, confirm prompt. Do after mv4vye and 05r3pg, which settle the keys and the grouping

## Depends on

## Acceptance
- [ ] ? opened inside a task lists that task's keys first under a heading naming the context, then global keys
- [ ] ? opened in a list leads with list keys, and no row shows a key that does nothing there
- [ ] typing in help filters rows by key or word, and Esc clears the filter, then closes
- [ ] a test walks every context and presses each listed key to prove it is bound
- [ ] fits 80x24 without a line wider than the terminal; goldens reviewed

## Notes


## History
- 2026-11-17T21:37 todo -> doing/active (take)
