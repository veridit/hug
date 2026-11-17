---
id: bejd89
title: "Column cursor: Tab picks a column, o orders by it"
state: "doing"
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-17T19:38"
waiting_on: "tje4sx first (Jørgen's ruling); resumes after it lands"
waiting_since: "2026-11-17T20:03"
waiting_asked_by: "nautilus"
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

## Acceptance
- [ ] Tab and Shift-Tab move the column cursor and the header marks the active column; h/l still change section (golden at 80 and 140 columns)
- [ ] o cycles ▲, ▼ and off on the marked column; in a section the rows reorder within that section; off restores rank order (test)
- [ ] the active order is shown in the column header and the footer, and survives R reload and a filter change (test)
- [ ] o no longer switches to the overview; the overview is reached by the header tree key, and help lists both (test)
- [ ] Jørgen orders the Statbus overview by priority and then by owner with Tab and o, and says it works

## Notes

nautilus 14:09 (crab 14:08): dependency on 6ephqk removed. bejd89 needs only 6ephqk's columns (rows show stage, no status column: 25f5f5f, check 5), on main since d198e01. Tags (txjkrm) are not a column yet; the cursor covers the columns on screen, and tags join when txjkrm adds them.


file:column-sort.md (1143 bytes, sha256:504c83be) design, moved here from tmp/lanes (git-ignored, so other clones never saw it)
nautilus 15:31: given back, not started in code. Surveyed: tui/sort.go (sortKey, sortLabel, sortFields), render.go columns() and listText/overviewText/boardText, the list's Tab (keys.prev/next -> changePane), o = overview at tui.go and in the detail switch, shortcuts.go assignShortcuts gives views o l b !. Plan: new columns.go with the header cells and a column cursor (Tab/Shift-Tab, h/l stay sections), o cycles ▲ ▼ off on the marked column via m.sort, header marks it; views move to the v prefix (v o, v l, v b, v !). Mock: tmp/lanes/column-sort.md. Order after crab 15:00: x1ra0t, 89y0b7, 3jtfw9 landing, scrxak, 5z4zt4, then this.
Parked for tje4sx (Jørgen's ruling goes first). WIP at 63beb14 on origin/nautilus/bejd89-cols: Tab/Shift-Tab column cursor, o cycles ▲/▼/off, v view prefix, columns_test.go + testdata/columns/cursor.txt. Remaining: full tui suite + tmux goldens, board narrow T/P arrow (fixed after 63beb14, uncommitted in /Users/jhf/.jcode/scratch/6ephqk), CHANGELOG, -race, land.
## History
- 2026-11-17T15:09 todo -> doing/active (take)
- 2026-11-17T16:30 doing/active -> todo/ready (give back; was assignee: nautilus); was doing_since: 2026-11-17T15:09
- 2026-11-17T19:38 todo/ready -> doing/active (take)
- 2026-11-17T20:03 doing/active -> doing/waiting (wait)
