---
id: 4bwcgp
title: "Detail: Tab reaches a task's links (deps, waited-on ticket, parent); Enter opens"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T20:45"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] in a detail whose waiting_on names a task id, Tab moves to that id, Enter opens it, Esc returns to the first detail (test + golden)
- [ ] dependencies, parent, children and task:ID refs are all on the same Tab cycle, in reading order; an id that resolves to no task is not a stop (test)
- [ ] Jørgen tabs from a waiting row's detail to the ticket it waits on and back, and says it is quick

## Notes


## History
- 2026-11-17T20:45 todo -> doing/active (take)
