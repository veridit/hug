---
id: g9dg32
title: "Deduce from git: commits on main and release tags mark the ticket"
status: "doing"
type: feature
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "47sxn2"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When a commit whose subject ends with a ticket id, e.g. (16dxsr) or (t709he, 44w1ev), first appears on main (seen at fetch, take or lint), hug adds a line to the ticket: 'landed <sha> on main <date>'. When a release tag (vX.Y.Z) first contains such a commit, it sets done_release on a done ticket, or adds 'shipped in vX.Y.Z'. A merged branch recorded on take (x1t1t1) counts as landed too. Each action is reported to the owner as an observation: what was seen, what hug wrote, and the fix command if it is wrong.

## Purpose
Epic 47sxn2. 0hkaa5, m1c28r and t709he shipped but their tickets did not say so; 58 of 179 main commits in 2 days already name their ticket, so the facts are there.

## Principle
Only certain facts: the id in the subject and the commit reachable from main or the tag. Idempotent: one line per commit per ticket, never duplicated across clones. Never ticks a check and never marks done: that is judgment, left to level 2.

## Relevant details
Scan only commits new since the last scan (store the last seen main and tag in the cache). Id pattern: trailing (id[, id...]) in the subject. Background sync 0hkaa5 is where new main commits arrive. Writes go through the board turn.

## Depends on

## Acceptance
- [ ] a commit (abc123) naming two tickets reaching main adds 'landed abc123' to both tickets once, even after several syncs and from two clones (test)
- [ ] tagging vX.Y.Z adds 'shipped in vX.Y.Z' or sets done_release on each ticket named by commits new in that tag (test)
- [ ] each deduction is reported to the owner as 'observed: ... ; marked: ... ; if wrong: <fix>' (test on text)
- [ ] replayed over hug's main since v0.7.30: every ticket named in a commit gets its landed lines, compared by hand on 5

## Notes
