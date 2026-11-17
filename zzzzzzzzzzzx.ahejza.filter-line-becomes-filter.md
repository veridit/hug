---
id: ahejza
title: "Filter line becomes 'filter: … · sort: …'; f and s open matching pickers"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
The line under the view line reads 'filter: <chips or none> · sort: <order or rank>', with f and s marked. f opens today's filter picker. s opens a sort picker built the same way: field letters listed with hints, each pick adds 'then by', the same letter flips direction, Backspace drops the last, Enter applies, Esc keeps the old order. The digit quick filters (1 todo 2 doing 3 done 4 depends) are removed, keys and hint.

## Purpose
Jørgen 23:37 local, on the line 'No filters · f picker · 1 todo  2 doing  3 done  4 depends': rework it into filter: and sort:, using a picker like the current f, and remove the 1-4 shortcuts. Filter and sort are the two things that shape the rows, so they share one line and one kind of picker.

## Principle
One line says what shapes the rows, and its letters are the keys that change it. Both pickers look and behave alike. No old aliases kept: s stops being ':status ', the digits stop filtering. Subtle colours, no red.

## Relevant details
Today: filters.go filterBar() prints 'No filters · f picker · 1 todo  2 doing  3 done  4 blocked' (the code still says blocked); tui.go:859 handles digits 1-4; tui.go:933 and :1027 bind s to ':status '. The sort builder already exists behind ': sort' (sort.go, mkzcqr: p t s u c g a n fields) and o orders by the marked column (e3sevr). The s picker reuses sortFields and shows the same QUICK FILTER-style panel ('SORT · then by …'). After: 'filter: [state:doing ×] · sort: priority ↓, updated ↓' or 'filter: none · sort: rank'. Lands after qh9ep1 (same header lines). Owner: nautilus.

## Depends on

## Acceptance
- [ ] The line reads 'filter: none · sort: rank' with f and s marked when nothing is set, and shows chips and the order when set (golden at 80 and 140)
- [ ] s opens a sort picker panel listing the sort fields with hints; picks chain as 'then by', the same letter flips direction, Backspace drops the last, Enter applies, Esc leaves the order unchanged (key test)
- [ ] f still opens the filter picker, and both panels share one look (golden)
- [ ] Keys 1-4 do nothing and no hint names them; s no longer opens ':status ' (key test)
- [ ] Context help (?) lists f filter and s sort with their picker keys (help test)
- [ ] Jørgen confirms on his board after release
- [ ] sc sorts by the marked column and repeated sc flips its direction; o no longer sorts; created moves to d; the picker lists 'c current column (NAME)' (key test)

## Notes

Jørgen 23:39 local: sc sorts by the current column, the one the column marker is on (e3sevr's H/L). This replaces e3sevr's o: o is unbound after ahejza. sortFields' c (created) moves to another letter: crab picks d (date created), u stays updated. In the picker, c reads 'c current column (priority)', naming the marked column.
