---
id: ahejza
title: "Filter line becomes 'filter: … · sort: …'; f and s open matching pickers"
state: "done"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzzzzkm"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T19:58"
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
- [x] The line reads 'filter: none · sort: rank' with f and s marked when nothing is set, and shows chips and the order when set (golden at 80 and 140)
- [x] s opens a sort picker panel listing the sort fields with hints; picks chain as 'then by', the same letter flips direction, Backspace drops the last, Enter applies, Esc leaves the order unchanged (key test)
- [x] f still opens the filter picker, and both panels share one look (golden)
- [x] Keys 1-4 do nothing and no hint names them; s no longer opens ':status ' (key test)
- [x] Context help (?) lists f filter and s sort with their picker keys (help test)
- [x] Jørgen confirms on his board after release
- [ ] ~~sc sorts by the marked column and repeated sc flips its direction; o no longer sorts; created moves to d; the picker lists 'c current column (NAME)' (key test)~~ (not relevant: Jørgen 23:39:44: o stays; o is the same as sc, crab, 2026-11-17)
- [x] o and sc both sort by the marked column, and repeating either flips its direction; in the s picker created moves to d and c reads 'c current column (NAME)' (key test)

## Notes

Jørgen 23:39 local: sc sorts by the current column, the one the column marker is on (e3sevr's H/L). This replaces e3sevr's o: o is unbound after ahejza. sortFields' c (created) moves to another letter: crab picks d (date created), u stays updated. In the picker, c reads 'c current column (priority)', naming the marked column.

Jørgen 23:39:34 local reverses the 23:39 note: o was good. o keeps sorting by the marked column (e3sevr). No sc, created stays on c in the s picker. The sc check is void.

Jørgen 23:39:44 local, final: o is the same as sc. Both stay: o is the one-key shortcut, sc is the picker's way to the same thing.

Header picture for all of tonight's rulings is attached to e3sevr as header-spec.md (crab 00:00 local). Build to it. Where it and this ticket differ, tell crab.


landed e36ecca4..4247303e (nautilus). CI run 44238251338 green on all 6 jobs (ubuntu with tmux goldens, macos, windows, windows-cli, bench, secrets); -race on tui ok (121.8s); tui + repocheck green after the final rebase. Tests: huginn/internal/tui/ahejza_test.go, all six fail first on main 03e783d0 (shim currentColumn): TestFilterLineNamesFilterAndSort 'at 80 the filter line reads "No filters · f picker · 1 todo  2 doing  3 done  4 depends"', TestSortPickerOnS 's opened ":", want the sort picker', TestFilterAndSortPickersLookAlike 'headings: "QUICK FILTER · choose a field" / ...', TestDigitsNoLongerFilter '1 changed something: ... State:todo', TestHelpListsFilterAndSortPickers 'help lacks "f filter picker: s t p g l a e"', TestSortCIsTheMarkedColumn 'sc with no column: prompt ":"'. Check 1: line 3 'filter: none · sort: rank' at 80 and 140, f and s in the key style; with a chip and an order 'filter: [state:doing ×] ⌫ last · F clear · sort: priority ↓ · updated ↓' (golden testdata/header/filter-line.txt); the order left line 1. Check 2: s opens the picker (also in a task), p t = 'priority ↓ · type', t again flips, ⌫ drops, Enter applies, Esc keeps (the old sort_test mkzcqr tests still pass via ': sort'). Check 3: both panels = accent heading, bold 'filter: …' / 'sort: …', then 'letter name hint' rows (golden testdata/header/pickers.txt; tmux quick-filter/sort frames). Check 4: 1-4 leave state() unchanged; no hint or help row names them; s no longer opens ': state ' (state palette is ': state', type T). Check 5: help 'f filter picker: s t p g l a e (state type priority …)', 's sort picker: c column, p t s u d g a n; again flips'; in the picker '? ' shows IN THE SORT PICKER with c and d created. Check 8: o and sc give the same single-key order on the marked column, again flips (o then sc flips too); with no column sc says 'mark one with H/L first'; s d sorts by created; the picker reads 'c  current column (priority)'.
landed 4247303 on main 2026-11-18: tui: 'filter: … · sort: …' line with f and s marked; s opens the sort picker (c…
shipped in v0.7.56 (4247303)
check 6: Jørgen 19:58 local: 'I like these pickers. Keep it'
Reflection (dismissed, 2026-11-18, crab): nothing: Jørgen's answer came as a hug answer event naming the check, and tick + done closed it
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G11, nothing misled (any gap named was fixed in this ticket).
## History
- 2026-11-18T05:01 todo -> doing/active (take)
- 2026-11-18T05:47 doing/active -> doing/asked (ask)
- 2026-11-18T19:57 answered asked (check 6), by jhf, for nautilus (doing/asked -> doing/active):
  > after the release: line 3 reads 'filter: none · sort: rank'. Press s for the sort picker (c is the marked column, the same as o; created moved to d) and f for the filter picker; 1-4 no longer filter (use :todo etc.). Does it read clearly?
  I like these pickers. Keep it
- 2026-11-18T19:57 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T19:58 answer 1 acknowledged by crab
- 2026-11-18T19:58 doing/active -> done (done); was doing_since: 2026-11-18T05:01
- 2026-11-18T19:58 answer 1 in progress by crab: hug task done ahejza
