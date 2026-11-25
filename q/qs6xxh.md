---
id: qs6xxh
title: "Task page: H and L jump between headings; a section folds by its size"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "000i"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
On a task's page, H and L jump to the next or previous heading, and only a section taller than the screen folds, opening as you jump to it.

## Purpose
Jørgen 2026-11-25 on 9a479v: folding is hard coded (his 8a0ybb Acceptance was large but should not fold, Notes did), and he has no key to reach the big sections.

## Principle
Fold by size, not by name; one key pair for headings, the lower-case pair stays for the next stop.

## Details
Today (v0.7.112): Details folds when over 6 wrapped lines (hya1st); answered Questions, Notes and History always fold (8a0ybb); Acceptance never folds. h/l and Tab step through every stop (ids, checks, links). Jørgen 2026-11-25: 'I would like for maybe uppercase H and L to jump to the next major headline heading and for each heading if it is bigger than the whole screen to start off as compacted when I scroll down but when I jump to it maybe open up as I jump to it.' Proposal: a section folds when its text is taller than the page area (not at a fixed count); H/L move the view to the next or previous heading and open it if folded; scrolling past (j/k) leaves it folded. H/L already mean 'mark column' in lists (hktdgs); on a task page they are free. Keys provisional until bfxq93. Check with frames before building, since this changes the page.

## Depends on

## Acceptance
- [ ] Jørgen picks from ANSI frames of 8a0ybb at 100x30: which sections fold, and where H and L land
- [ ] H and L on a task's page jump to the next and previous heading and open it if folded; j and k scroll past a folded section without opening it (tmux test)
- [ ] A section folds only when taller than the page area, whatever its name; Acceptance of 8a0ybb shows open at 100x30 (golden)

## Notes

Jørgen 2026-11-25 10:42 (request 1 on 9a479v): 'a bit like when I jump to the active section in the overview, it expands' (a section opens when you land on it). 10:43 (request 2): 'I do wonder if we should use dot like comma and period to jump ... to different sections instead of uppercase H and uppercase L ... maybe those uppercase things are free'. On v0.7.112: comma and period are unbound everywhere in hug tui; H and L are bound only in lists (mark a column, tui.go:1936), free on a task page. Frames show both key pairs for him to pick; keys stay provisional until bfxq93.
