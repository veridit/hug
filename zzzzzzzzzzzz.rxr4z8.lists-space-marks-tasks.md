---
id: rxr4z8
title: "In lists, Space marks tasks and the footer offers actions for the marked ones"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
In the overview, list and board, Space toggles a mark on the focused task and steps to the next row. While one or more tasks are marked, the footer becomes an action bar for them, e.g. '2 marked · m move · t take · d done · x drop · w wait · e edit · Esc unmark', and each action applies to every marked task, with one confirm naming the count. Esc clears the marks.

## Purpose
Jørgen 23:42 local, with a screenshot of the overview where Space on 5z4zt4 only said 'Space ticks a check inside the task: Enter opens 5z4zt4': Space should mark the task, then offer context actions like move. Today the actions are scattered single keys that need knowing; marking first shows what can be done to what is marked.

## Principle
Mark, then act, and the screen says what acting will do. Only actions valid for every marked task are offered, and an action hug would refuse shows why with a fix line. Nothing writes without the confirm. Concurrent writes wait, never fail. Subtle colours: a marked row has a quiet tint and a mark in the pie column, never red.

## Relevant details
Today: Space in a list prints 'Space ticks a check inside the task: Enter opens ID' and does nothing (actions.go actionNames maps ' ' to toggle acceptance, only in the detail). Single-key actions live in actions.go (t take, d done, x drop, J/K move, e edit), grab is m (3xcesq). With marks, m grabs the marked tasks as one block and J/K move the block. Inside a task, Space keeps ticking the focused check. Owner: nautilus, after the keys work (e3sevr, qh9ep1, ahejza). Mock the footer in the ticket before building, so Jørgen sees it.

## Depends on

## Acceptance
- [ ] Space in overview, list and board marks or unmarks the focused task and moves to the next row; marked rows show a mark and a quiet tint (golden at 80 and 140)
- [ ] With N marked, the footer reads 'N marked ·' followed by the actions valid for all of them and 'Esc unmark'; Esc clears every mark (key test)
- [ ] t, d, x and w act on every marked task after one confirm naming the count, and a refused task is named with its fix: line while the others go ahead (key test with 3 marked, one refused)
- [ ] m grabs the marked tasks as one block; J/K and the drop move them together and keep their order (key test)
- [ ] Inside a task Space still ticks the focused check (existing tests stay green)
- [ ] Jørgen confirms on his board after release

## Notes

nautilus 23:44, footer mock for Jørgen (not built yet; queued after e3sevr, 0gztsy, qh9ep1, ahejza). 80 columns, 3 marked in NEXT UP, one of them an epic (d not offered: an epic finishes through its children):

    NEXT UP · rank order  [12]
  ◆ ○ 2   000041   feature critical ready     runtime Feature task 41
  ◆ ○ 2   000017   bug     low      ready     docs Bug task 17
  > ○ 1   000009   spike   high     ready     docs Spike task 09
  ◆ ◑ 1/2 a1b2c3   feature high     ready     Launch (epic)
  ...
  3 marked · m move · t take · x drop · w wait · e edit · Esc unmark
  j/k move · Space mark · Enter open · ? help · q quit

 ◆ (dim accent, never red) in gutter column 1 marks a row, with a quiet tint; the cursor row keeps >. t asks once: 'Take 3 tasks? Enter confirm · Esc cancel'. A refused task (e.g. taken by someone else) is named with its fix: line; the others go ahead. Narrow screens drop the action words to keys: '3 marked · m t x w e · Esc'.
