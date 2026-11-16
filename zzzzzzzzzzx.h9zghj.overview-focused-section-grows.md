---
id: h9zghj
title: "Overview: the focused section grows, others keep at least 3 rows"
status: "doing"
type: feature
priority: "high"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
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
- [x] on a 40-row terminal with 5 sections, the focused section shows all spare rows and each other non-empty section shows min(count, 3) rows (test and golden)
- [x] RECENTLY DONE focused scrolls through every closed task newest first; unfocused it shows 3 (test)
- [x] moving between sections with Tab keeps each section's scroll position (test)
- [ ] an empty section is one line, None to the right of its title (e.g. "BLOCKED · unmet dependencies  [0] · None"), in every overview section and board column (test and golden)

## Notes

2026-11-16 Jørgen: an empty section takes one line: 'None' goes on the right of its title (e.g. 'BLOCKED · unmet dependencies  [0] · None'), not on a row of its own, since there is nothing to navigate to. Applies to every overview section and board column.

2026-11-16 Jørgen (screenshot on v0.7.27): show the hug version in the lower right corner of the TUI, on the key-hint line (e.g. right-aligned 'hug v0.7.27'). Small; can ride with this ticket or the next TUI one.

gh run view 43910574753 (ci/h9zghj @ 05ac572 = local main HEAD): secrets success / test (windows-latest) success / test (macos-latest) success / test (ubuntu-latest) success. Local: go test -count=1 -race ./huginn/internal/tui/ -> 'ok github.com/veridit/hug/huginn/internal/tui 57.105s'.

check 1: TestOverviewFocusedSectionGrows at 140x40 for focus IN PROGRESS, NEXT UP, RECENTLY DONE: others rows == drawn == min(count,3), focused == min(count, spare); golden 140x40 overview: 'IN PROGRESS [15] · 1-11 shown', NEXT UP/BLOCKED/LINT '1-3 shown', RECENTLY DONE '[30] · 1-3 shown'. check 2: TestRecentlyDoneScrollsThroughAll: 30 of 30 closed, newest first, unfocused 3 rows, j selects every one in order and each is on screen. Seid board read-only (2508 tasks): 'RECENTLY DONE · last closed  [1263] · 26-41 shown', G selects znpa2b (oldest). check 3: TestSectionsKeepScroll: IN PROGRESS scrolled to 000050 (first row 2), Tab/j/Tab/Shift-Tab/Shift-Tab back: same task, same first row; drawn while unfocused; board TODO column same. check 4: TestEmptySectionIsOneLine: 'IN PROGRESS · assigned to  [0] · None', BLOCKED, RECENTLY DONE likewise, no line that is only None; board 'DOING [0] · None', BLOCKED, DONE, DROPPED; 'TODO [1]' without None.
