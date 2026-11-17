---
id: 4bwcgp
title: "Detail: Tab reaches a task's links (deps, waited-on ticket, parent); Enter opens"
state: "done"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:39"
---

## Goal
In a task's detail, every task it refers to (its dependencies, a ticket id in its waiting_on, its parent epic, its children, task:ID in the text) is reachable by Tab / Shift-Tab, marked, and Enter opens it; Esc or u returns. The 'waiting on after 46ek1f' line in jr55s8's detail is the case: 46ek1f must be one Tab away.

## Purpose
Jørgen 2026-11-17 19:43 local, on jr55s8's detail ('waiting on after 46ek1f (crab 17:52)'): 'there is no way to tab to the dependency, so I can't quickly navigate to it.' The Links section (task:ID in text) exists, but ids inside a wait reason, dependencies and the parent are not reachable from the top of the detail.

## Principle
Keys act on what the view shows: an id shown in the detail is a link. One cursor for links in the detail, in reading order. No new section that crowds the screen: the ids are marked where they already stand.

## Relevant details
Sources: dependencies, waiting_on text (any 6-char id or alias that resolves), parent, children, task:ID refs. Unknown ids are not links. Check what Tab does in the detail today (bejd89 leaves it as is there) and fold links into it.

## Depends on

## Acceptance
- [x] in a detail whose waiting_on names a task id, Tab moves to that id, Enter opens it, Esc returns to the first detail (test + golden)
- [x] dependencies, parent, children and task:ID refs are all on the same Tab cycle, in reading order; an id that resolves to no task is not a stop (test)
- [ ] ~~Jørgen tabs from a waiting row's detail to the ticket it waits on and back, and says it is quick~~ (not relevant: Tab stepping ids is replaced by Jørgen's 23:27 ruling (Tab = l, one focus ring, e3sevr); he confirms the new keys on e3sevr check 4, crab, 2026-11-18)

## Notes


Landed 4fb0c92 (3bc5d72..4fb0c92 on main). Design agreed with crab 19:50: in the detail Tab/Shift-Tab step task ids in reading order, h/l step sections (as in bejd89); Notes ids added per crab. Check 1: TestTabReachesWaitedOnID (Main waits 'on after <Blocker> (crab 17:52)'; 2nd Tab marks '└─ waiting on after ▸<Blocker>' (1st is the epic badge), Enter opens Blocker, Esc returns to Main; golden testdata/detail/tab-ids.txt at 80 and 140, ids named by title, stamps masked). Check 2: TestTabCyclesEveryIDInReadingOrder (stops parent, stage, deps, links, notes; zzzzzz in text and in task:zzzzzz is no stop; Tab wraps, Shift-Tab back; Enter on each kind opens it, Esc back; epic shows CHILDREN (2) and Tab steps them, Enter opens; l still changes section; no ids -> 'no task id here'). Both fail on main (idStops undefined). Gates on rebased tree: go test -race ./huginn/internal/tui/ ok 201.1s; earlier tree go test ./... 32 ok exit 0. Section-Tab tests moved to l/h; messages 'Tab focuses the checks' now 'l focuses the checks'; key bar 'Tab next id', h/l section to ? help below 160 so S screen keeps fitting at 140.
landed 4fb0c92 on main 2026-11-17: hug tui: in a task, Tab steps through every task id shown; Enter opens it…
shipped in v0.7.47 (4fb0c92)
## History
- 2026-11-17T20:45 todo -> doing/active (take)
- 2026-11-17T21:27 doing/active -> doing/waiting (wait)
- 2026-11-18T00:39 doing/waiting -> done (done); was doing_since: 2026-11-17T20:45; waiting_asked_by: nautilus; waiting_for: jhf; waiting_on: check 3: from a waiting row's detail (e.g. jr55s8), Tab to the ticket it waits on, Enter, Esc back, and say whether it is quick (4fb0c92); waiting_since: 2026-11-17T21:27
