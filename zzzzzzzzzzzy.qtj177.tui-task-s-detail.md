---
id: qtj177
title: "TUI: a task's detail draws no list hints ('H/L column · order by owner')"
state: todo
type: bug
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Inside a task, every line above the footer belongs to that task: the list's column hint ('H/L column · order by owner (▲)') is not drawn there, and the footer lists only keys that act in the detail.

## Purpose
Jørgen 2026-11-18 02:31 local, screenshot of zhvkmj's detail on v0.7.52: the line 'H/L column · order by owner (▲)' sits just above the footer. H/L and o act on list columns, which a detail has none of, so the hint 'doesn't seem relevant'. header-spec.md: in a task the global band is not drawn; this line is a list-only hint that leaked in with e3sevr.

## Principle
A hint is drawn only where its keys act. One test enumerates every hint line per place, so no list hint can leak into a detail again.

## Relevant details
Seen at the bottom of the detail, between HISTORY and the footer ('j/k scroll · edit · done · wait · x drop · yank · h/l Tab next · answer · New · up · next-, previous task · J/K ½ page'). Code: huginn/internal/tui, the column/order hint added by e3sevr (fa074661 'column hint drops · h/l section'). Same screenshot also shows the doubled wait lines and the truncated question, which 0gztsy already covers.

## Depends on

## Acceptance
- [ ] No 'H/L column' or 'order by' text in a detail at 80 and 140, one task open and two stacked (golden; test fails first on main)
- [ ] A test lists every hint and footer key per place (list, overview, board, detail, file viewer) and fails if a key shown there does nothing there
- [ ] Jørgen sees no list hint in a task on his board after release

## Notes

landed e89ebc2 on main 2026-11-18: tui: the column hint is drawn on the rows only, never kept into a task; file…

landed 48b1151 on main 2026-11-18: tui: glance golden: j/k not named for a file that fits (qtj177)
