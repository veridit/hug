---
id: e3sevr
title: "TUI keys: H/L move the column marker; J/K page; marker sits on the cursor row"
state: todo
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
Keys in list, overview and board: j/k up/down, J/K page down/up, h/l left/right (in the overview: previous/next section, the same as Shift-Tab/Tab), H/L move the column marker left/right, o orders by the marked column. Moving a task in rank is m (grab) only; J/K no longer move tasks. The column marker is no longer drawn in the header row: on the cursor row a > stands between the columns, just before the marked column's cell, and the marked column is highlighted. So the cursor shows by the row's > sigil, the in-row marker and the column highlight.

## Purpose
Jørgen 2026-11-17 23:20 local, on the v0.7.48 overview: the header marker '▸✓' over the pie column should move into the cursor row between the columns; 'j/k up down, J/K page up/down, h/l left right, overlapping Tab/Shift-Tab for the overview navigation, H/L for column left right moving the marker. The marker plus the column highlight is the visual clue in addition to the > sigil.' Reverses bejd89's Tab for the column cursor and x1ra0t-era J/K as row moves outside the detail.

## Principle
One key family per axis: lower case moves the cursor, upper case moves it further (page, column). Nothing writes on a cursor key; moving a task in rank stays an explicit grab (m). Help (jmx10s) lists exactly the bound keys; its key test must keep passing.

## Relevant details
Mock, cursor on 3mavpt with the TYPE column marked:
    ●  3    3mavpt >bug      high    done   hug setup takes 34 s on Seid ...
 (the marked column's cells are highlighted down the section; the header row shows plain column names). Code: huginn/internal/tui columns (bejd89), actions.go J/K move (actionNames), tui.go key switch, help sections. Detail keys are unchanged: Tab steps ids, h/l sections (4bwcgp).

## Depends on

## Acceptance
- [ ] H/L move the column marker in list, overview and board; Tab/Shift-Tab in the overview change section like l/h; J/K page; m grabs to move (tests)
- [ ] the header row has no marker; the cursor row shows > before the marked column's cell and the marked column is highlighted (goldens at 80 and 140)
- [ ] help lists the new keys and TestHelpKeysAreBound passes
- [ ] Jørgen moves the marker with H/L, orders with o, and pages with J/K, and says it reads clearly

## Notes
