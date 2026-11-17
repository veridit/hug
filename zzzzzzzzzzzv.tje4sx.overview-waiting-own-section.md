---
id: tje4sx
title: "Overview: WAITING is its own section; IN PROGRESS holds only active work"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T20:03"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "check 3: open hug tui overview and say whether IN PROGRESS and WAITING read clearly (fb66000; brew upgrade after the next release, or go run ./cmd/hug tui from main)"
waiting_since: "2026-11-17T20:28"
waiting_asked_by: "nautilus"
waiting_for: "jhf"
---

## Goal
The overview gets a WAITING section between IN PROGRESS and NEXT UP. IN PROGRESS shows only doing tasks someone is actively on (stage active); every doing task that waits (waiting, design, review) goes to WAITING. The header reads 'WAITING · N · K on you'; rows waiting on the TUI user sort first. The ? mark stays only on rows that wait on you, so it is no longer on almost every row.

## Purpose
Jørgen 2026-11-17 19:54 local, looking at the hug overview: 12 of 12 shown IN PROGRESS rows were waiting, 11 on him: 'It is clear to me that waiting has to be its own section.' This reverses the earlier ruling 'work waiting on a person is a stage, not a section' (that ruling removed BLOCKED; a task with an open dependency still stays where it is). Seen at the same time: IN PROGRESS showed no active work at all, so what agents are doing now was hidden.

## Principle
A section answers one question: IN PROGRESS = what is being worked on now; WAITING = what is stuck on someone or something, you first. Same two-line rows. No new key needed: the section cursor (h/l) reaches it like the others.

## Relevant details
Code: overview sections in huginn/internal/tui (render.go, stage.go forYou()). The header's '? N for you' count moves to the WAITING header. The 1-4 filter keys and the doing count in the summary line stay as they are (doing still counts both). Golden at 80 and 140.

## Depends on

## Acceptance
- [x] a doing/active task is in IN PROGRESS; doing/waiting, design and review are in WAITING; a todo task with an open dependency stays in NEXT UP (test + goldens at 80 and 140)
- [x] WAITING's header counts its rows and those on you; rows on you sort first and only they carry ? (test)
- [ ] Jørgen opens the overview and says IN PROGRESS and WAITING read clearly
- [x] only the row under the cursor shows its second line (└─ waiting on ... since ...); every other row is one line, with ? on rows waiting on you; j/k and scrolling count the extra line (test + goldens at 80 and 140)

## Notes

Jørgen 20:00 local: only the focused row shows two lines; every other row is one line. Reverses the 89y0b7 rule that every waiting row takes two lines. So: unfocused waiting row = one line with the title (as before 89y0b7, plus the ? on rows on you); the row under the cursor opens its second line '└─ waiting on you: ... since ...'. Section heights and scrolling count the one extra line of the focused row.


crab 20:10 rulings (asked by nautilus): (1) doing/depends goes to WAITING (stuck on a task); Bravo's test changes. (2) WAITING = every open task whose stage waits on a person or thing: waiting, design, review, todo or doing alike (a question for Jørgen sits where he looks, not in NEXT UP); todo/depends stays in NEXT UP (not ready yet, nobody asked). (3) '? N for you' stays in the top header (the one count seen in every view); the WAITING header is 'WAITING · N', no second count: on-you rows come first and carry the ?. WAITING rows keep the owner column. One focused row with two lines applies in list and board too.
Landed fb66000 (0338d9c..fb66000 on main). Checks 1/2/4: TestRowsShowStage (IN PROGRESS=aaaaaa only; WAITING=bbbbbb review, cccccc waiting, dddddd doing/depends, eeeeee todo/design; NEXT UP=ffffff ready, gggggg deferred; header 'WAITING · on you first  [4]'; unfocused overview has no └─; each focused row opens exactly one └─ line), golden testdata/rows/stages.txt (overview, WAITING focused, list at 80 and 140). TestOverviewOneSectionPerTask: doing+open dep in WAITING, todo+open dep in NEXT UP. TestWaitingOnYou: on-you row first in WAITING and the only ? row, for design_for jhf, review_by Jørgen, waiting_for carol; '? 1 for you' in top header (crab 20:10). TestTwoLineRowsScrollByTask: list/overview/board, 40 j steps, at most one └─ on screen and it is under the cursor row, screen keeps its height. All 3 failed on origin/main 0338d9c first. tmux goldens 80x24/140x40 colour+no-colour regenerated and reviewed. Gates: go test -race ./huginn/internal/tui/ ok 173.4s; go test ./... 32 ok, exit 0.
landed fb66000 on main 2026-11-17: hug tui: the overview has a WAITING section; only the focused row opens its…
## History
- 2026-11-17T20:03 todo -> doing/active (take)
- 2026-11-17T20:28 doing/active -> doing/waiting (wait)
