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
- [ ] ': sort', p (priority), then t (type), Enter orders rows by priority then type, and the header shows 'sort: priority ↓ · type' (test and golden)
- [ ] Esc in the middle of building leaves the previous order unchanged (test)
- [ ] the same field letter twice flips its direction; Backspace removes the last field (test)
- [ ] with a sort other than rank, J/K, m, y and P/p are not shown and say 'moving needs rank order: : sort rank' when pressed (test)

## Notes
