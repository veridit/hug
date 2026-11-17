---
id: bejd89
title: "Column cursor: Tab picks a column, o orders by it"
status: todo
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
In list, overview sections and board, a column cursor marks one column header (pie, id, type, priority, tags, stage, title, owner). Tab and Shift-Tab move it right and left. h/l and the arrows keep moving between sections and panes. o on the marked column cycles: ascending ▲, descending ▼, off (back to rank order). The header shows the order where it applies ('title ▲'), and the footer says 'o order by title'. Pressing o on another column replaces the order. The sort builder (mkzcqr) stays for multi-column orders.

## Purpose
Jørgen 2026-11-17 11:43, after trying sort on the hug board: sorting today is hard (palette or sort builder, then pick a field). He sees the board as a spreadsheet: move the cursor to a column and press one key to order by it, like any spreadsheet header.

## Principle
One key, three states, shown in place: the active column and its direction are always visible, so you never guess why rows are in that order. Columns are the ones on screen, no hidden fields. Moving between sections stays on h/l. A key never does two things: o stops switching to the overview (the view keys move to the header tree, v o, as in 0sw0dd).

## Relevant details
Keys today (huginn/internal/tui/keys.go:20-21): h/←/Shift-Tab = previous pane or column, l/→/Tab = next. o = overview view (tui.go:950, render.go:490), so it conflicts. 0sw0dd notes Jørgen's 2026-11-16 header-tree key scheme (v o = view overview). The columns change with 6ephqk (state column out, stage in) and txjkrm (tags), so build after them. Mock: tmp/lanes/column-sort.md.

## Depends on

- 6ephqk
## Acceptance
- [ ] Tab and Shift-Tab move the column cursor and the header marks the active column; h/l still change section (golden at 80 and 140 columns)
- [ ] o cycles ▲, ▼ and off on the marked column; in a section the rows reorder within that section; off restores rank order (test)
- [ ] the active order is shown in the column header and the footer, and survives R reload and a filter change (test)
- [ ] o no longer switches to the overview; the overview is reached by the header tree key, and help lists both (test)
- [ ] Jørgen orders the Statbus overview by priority and then by owner with Tab and o, and says it works

## Notes
