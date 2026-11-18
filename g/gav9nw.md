---
id: gav9nw
title: "Design: personal vs shared memories"
state: todo
type: spike
priority: "low"
assignee: []
labels: ["design"]
created: 2026-11-10
updated: 2026-11-10
parent: "9taa9d"
group: "design"
rank: "zy"
aliases: ["design-personal-vs-shared-memories"]
---

## Goal
Decide how muninn tells personal memories from shared ones

## Purpose
Jørgen asked, in an earlier crab session (memories 5zwpx7, 1c49sh), to revisit how muninn judges whether a memory is personal or not and to put it on the to-do list; it is open question 2 in docs/DESIGN.md (2026-11-11). It matters because only shared project memories may be pushed to a repo branch, while personal ones never go into a repo (hbzc5e), and he wants an in-between team-shared scope as well (873tqd).

## Principle
A design decision first, written in DESIGN.md, no code in this task. Start from the scopes that exist today (personal, global, project, client) and the rules already decided (below); do not reopen them unless the decision needs it. Jørgen makes the call; the task brings him options with their consequences.

## Relevant details
Decided so far: new Engram memories with no project default to personal, and global is reserved for working rules shared with all non-client views (s755vd; 9s4v6g: 9 personal memories moved to personal, global kept 2 rules). Placement rule 2026-11-06 (s5tse0): a session sees only its top-level (global or client root) memories plus its own project's. Related: 34r1ft (per-repo setting whether project memories go on a branch). Epic 9taa9d moves memory to one file per memory, so the decision should hold for that layout too.

## Depends on

## Acceptance
- [ ] decision in DESIGN.md

## Notes
