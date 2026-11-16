---
id: 3xcesq
title: "TUI: move a task far in one go (grab and drop, top and bottom)"
status: "doing"
type: feature
priority: "high"
assignee: ["poodle"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
Reordering in hug tui takes one action per decision, not one per row: grab a task, move it freely with j/k (or jump with g/G, a count or a search), and drop it in place with a single confirm and a single commit

## Purpose
Jørgen 2026-11-16: moving heype5 up NEXT UP took 'up, accept, up, accept, up, accept. I go crazy.' He asked for yank and paste, or a reorganize/prioritize mode to move quickly. Today each J/K is a confirm and a commit

## Principle
Vim-like, Esc always backs out. Two ways, same rank code (Store.Move/rankPlan): (1) grab mode: m grabs the selected task; j/k, g/G, counts and / move it on screen live (nothing written); Enter drops it with one commit and one result line ('moved heype5 above a1dx7s, row 3 of 31'); Esc puts it back. (2) yank and paste: y on a task, go anywhere in the same section, P puts it above the cursor row, p below; one commit. Single J/K stays for one step but needs no confirm (it is undoable by the opposite key). Keys shown only when they work; letters chosen by assignShortcuts; no { }.

## Relevant details
Code: huginn/internal/tui/move.go (moveTarget, from ksv323), actions.go. A move across sections (e.g. into BLOCKED) is refused with a message. Rank is sparse, so a drop is one write.

## Depends on

## Acceptance
- [x] grab mode: m, then 5 presses of k and Enter moves a task 5 rows up with one confirm-free drop and exactly one commit (test)
- [x] Esc in grab mode leaves the order and the board unchanged (test)
- [ ] y then P on another row puts the task above that row with one commit (test)
- [ ] the footer in grab mode shows only the keys that work there, e.g. 'j/k move · g/G top/bottom · Enter drop · Esc cancel' (golden)

## Notes

fc2c3cc + 88b07b9: grab (m, j/k with count, gg/G, Enter one commit, Esc nothing), yank y + P/p paste one commit, J/K no confirm. Tests: TestGrabFiveUpOneCommit, TestGrabEscCancels, TestYankAndPaste, TestGrabMovesFarWithOneCommit, TestGrabCountsAndEnds, TestMoveFarRefusals, TestGrabDefersReload all PASS. Golden testdata/tmux/*/grab.txt key bar: 'j/k move · g/G top/bottom · Enter drop · Esc cancel'; tmux: PASS (44 real-binary goldens). Choice noted: the ticket goal says 'a single confirm'; check 1 says confirm-free drop: I followed check 1 (Enter in grab is the confirm). List y now yanks; type palette is :type, f t, or y inside a task.
