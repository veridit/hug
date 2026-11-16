---
id: mkzcqr
title: "TUI sort builder: order by one field, then another, then Enter"
status: todo
type: "feature"
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
In hug tui you build a nested sort step by step: pick a field, pick its direction, then either add another field (then by ...) or press Enter to apply; Esc at any point leaves the order as it was. The header shows the active sort, e.g. 'sort: priority ↓ · type · updated ↓'.

## Depends on

## Acceptance
- [ ] ': sort', p (priority), then t (type), Enter orders rows by priority then type, and the header shows 'sort: priority ↓ · type' (test and golden)
- [ ] Esc in the middle of building leaves the previous order unchanged (test)
- [ ] the same field letter twice flips its direction; Backspace removes the last field (test)
- [ ] with a sort other than rank, J/K, m, y and P/p are not shown and say 'moving needs rank order: : sort rank' when pressed (test)

## Notes
