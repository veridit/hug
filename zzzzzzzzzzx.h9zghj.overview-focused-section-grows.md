---
id: h9zghj
title: "Overview: the focused section grows, others keep at least 3 rows"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
In the TUI overview the section you are in takes all spare rows; every other section keeps at least 3 rows (or fewer if it has fewer tasks); RECENTLY DONE lists all closed tasks newest first, shows 3 rows when not focused and scrolls through all of them when focused; each section remembers its scroll position

## Purpose
Jørgen 2026-11-16: RECENTLY DONE showed one row ('[3 of 1263 closed] · 1-1 shown') on seid; 'it should expand when you go into it ... expand anything you're focused on because then it can grow'. Minimum for unfocused sections: 'at least three'.

## Principle
Replace the share-by-need row split (overviewRows in huginn/internal/tui/render.go) with: each section gets min(its task count, 3) rows, plus a line for its header; the focused section gets everything left. RECENTLY DONE stops capping at 3 tasks: it covers every closed task. Sections never vanish. The layout stays calm because each unfocused section keeps its first rows in place.

## Relevant details
Seid board for checking (read only): ~/Veridit/src/seid-lang, 1263 done. Overview sections: IN PROGRESS, NEXT UP, BLOCKED, RECENTLY DONE, LINT.

## Depends on

## Acceptance
- [ ] on a 40-row terminal with 5 sections, the focused section shows all spare rows and each other non-empty section shows min(count, 3) rows (test and golden)
- [ ] RECENTLY DONE focused scrolls through every closed task newest first; unfocused it shows 3 (test)
- [ ] moving between sections with Tab keeps each section's scroll position (test)

## Notes
