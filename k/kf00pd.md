---
id: kf00pd
title: "Release v0.7.62: default sort row"
state: "doing"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:09"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
v0.7.62 ships nch9d4 (sort picker r rank row, now/new lines) from exact all-green 6aee8e30, published before targeted install.

## Purpose
Jørgen is testing releases live this morning; his default-sort feedback is on main with all 6 CI jobs green.

## Principle
Release only new content with exact-SHA all-job CI green; release.yml success before brew upgrade; no --machine migration.

## Depends on

## Acceptance
- [x] CHANGELOG v0.7.62 heading covers only nch9d4, no conflict markers, released history unchanged
- [ ] Tag release workflow succeeds before targeted brew upgrade, hug version prints v0.7.62

## Notes

Tag v0.7.62 on e4206e21 (CHANGELOG commit: only the nch9d4 entry gets the heading, 0 removed lines vs v0.7.61, no conflict markers), release merge 77d25855 on main. Content: 6aee8e30 (CI 44295351540 all 6 green on exact SHA) and 65ef4d07 dim-c follow-up (CI 44298750806 all 6 green on exact SHA). Release workflow running; install only after success.


## History
- 2026-11-18T13:09 todo -> doing/active (take)
