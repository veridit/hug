---
id: mkzcqr
title: "TUI sort builder: order by one field, then another, then Enter"
status: "doing"
type: "feature"
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
In hug tui you build a nested sort step by step: pick a field, pick its direction, then either add another field (then by ...) or press Enter to apply; Esc at any point leaves the order as it was. The header shows the active sort, e.g. 'sort: priority ↓ · type · updated ↓'.

## Depends on

## Purpose
Jørgen 2026-11-16: 'select an additional ordering, additional ordering, or stop at some point and enter to apply or escape to stop it'. Today rows have one fixed order (rank).

## Principle
Built on the palette tree: ': sort' opens the builder; each step lists the fields not used yet with letters from assignShortcuts (status, priority, type, rank, updated, created, assignee, title); the same letter again flips its direction; the prompt shows the chain so far and 'then by: … · Enter apply · Esc cancel'; Backspace removes the last field. Rank is the default and the only order where J/K, grab and paste move tasks; with another sort they are not shown and the footer says why. The sort applies to every section and view in the session.

## Acceptance
- [x] ': sort', p (priority), then t (type), Enter orders rows by priority then type, and the header shows 'sort: priority ↓ · type' (test and golden)
- [x] Esc in the middle of building leaves the previous order unchanged (test)
- [x] the same field letter twice flips its direction; Backspace removes the last field (test)
- [ ] with a sort other than rank, J/K, m, y and P/p are not shown and say 'moving needs rank order: : sort rank' when pressed (test)

## Notes

CI run 43919869737 on ci/mkzcqr (32f4fa9, now origin/main, ff from 9283f8f): test (ubuntu-latest) success, test (macos-latest) success, test (windows-latest) success. TestSortBuilderPriorityThenType (sort_test.go:26): ': sort', p, t, Enter; asserts every adjacent pair ordered by priority then type and header contains 'sort: priority ↓ · type'; '--- PASS: TestSortBuilderPriorityThenType (0.04s)'. Golden testdata/tmux/*/sort.txt (4 profiles), line 1: 'HUG TASKS · list   60/60 tasks · ! 7 findings · sort: priority ↓ · type', rows start 000001 feature critical, 000021 feature critical, 000041 feature critical, 000002 bug critical.

TestSortBuilderEscKeepsOrder (sort_test.go:60): sort by type, then ': sort' p n Esc leaves sort 'type' and the same row order; from rank order ': sort' p Esc leaves no sort; '--- PASS: TestSortBuilderEscKeepsOrder (0.07s)'

TestSortBuilderFlipAndBackspace (sort_test.go:79): p t p -> 'priority ↑ · type', t -> 'priority ↑ · type ↓', Backspace -> 'priority ↑', Backspace on empty keeps the builder open; p p Enter puts low first; '--- PASS: TestSortBuilderFlipAndBackspace (0.03s)'
