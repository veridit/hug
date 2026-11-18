---
id: jmx10s
title: "TUI help: context keys first, easy to scan and search"
state: "done"
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzh"
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T21:59"
done_release: "v0.7.48"
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
- [x] ? opened inside a task lists that task's keys first under a heading naming the context, then global keys
- [x] ? opened in a list leads with list keys, and no row shows a key that does nothing there
- [x] typing in help filters rows by key or word, and Esc clears the filter, then closes
- [x] a test walks every context and presses each listed key to prove it is bound
- [x] fits 80x24 without a line wider than the terminal; goldens reviewed

## Notes


Landed ac4409f (e971d24..ac4409f on main). New huginn/internal/tui/helpkeys.go: one key table per place (list, overview, board, lint, task by focus, file, palette, filter, find, jump, prompt, confirm), then EVERYWHERE; side by side from 120 columns. Check 1: TestHelpKeysAreBound asserts each context's help starts with its heading (IN THIS TASK · 000001, IN THIS FILE, IN THE : PALETTE, ...) then EVERYWHERE. Check 2: TestHelpShowsOnlyKeysThatWorkHere (list: no task keys, no h/l with one pane; task: no o/grab/J-K moves, Tab only where ids show). Check 3: TestHelpFindsByKeyOrWord ('strike' -> 1 row, Esc clears, Esc closes) and TestHelpSearchFindsOneActionAndEscBacksOut. Check 4: TestHelpKeysAreBound walks 16 contexts (4 views, 6 task focuses incl. ids, file, palette, filter, find, jump, confirm) and presses every row's probe keys there; it caught 9 rows that did nothing (c and the global row in a task, ↑↓ in palette/jump/find without completions, j/k in a file, the global letters in a file), fixed by listing only what is bound. Check 5: TestHelpFitsAndGolden (80x24 and 140x40, screen keeps its height, no line wider), golden testdata/help/contexts.txt reviewed; tmux help goldens 80x24/140x40 colour+no-colour regenerated and reviewed. Tests fail on main (helpSections undefined). Gates: go test -race ./huginn/internal/tui/ ok 150.9s; go test ./... 32 ok, exit 0. Built before 05r3pg (still todo); crab told; 05r3pg can reuse helpSections.
Reflection (dismissed, 2026-11-17, nautilus): AGENTS.md 'Tests must fail first' plus jmx10s check 4 ('press each listed key to prove it is bound') made me write the walk test before trusting my own key table: it found 9 rows I had listed that did nothing in that context, so the help now lists only bound keys instead of my belief about them
landed ac4409f on main 2026-11-17: hug tui: ? leads with the keys for where you are, then EVERYWHERE (jmx10s)
landed 58fcd3c on main 2026-11-17: Merge v0.7.47 into main (jmx10s, 3mavpt stay under Unreleased)
shipped in v0.7.48 (ac4409f)
## History
- 2026-11-17T21:37 todo -> doing/active (take)
- 2026-11-17T21:59 doing/active -> done (done); was doing_since: 2026-11-17T21:37
