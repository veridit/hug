---
id: kf00pd
title: "Release v0.7.62: default sort row"
state: "done"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzth"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T13:15"
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
- [x] Tag release workflow succeeds before targeted brew upgrade, hug version prints v0.7.62

## Notes

Tag v0.7.62 on e4206e21 (CHANGELOG commit: only the nch9d4 entry gets the heading, 0 removed lines vs v0.7.61, no conflict markers), release merge 77d25855 on main. Content: 6aee8e30 (CI 44295351540 all 6 green on exact SHA) and 65ef4d07 dim-c follow-up (CI 44298750806 all 6 green on exact SHA). Release workflow running; install only after success.


Tag v0.7.62 on e4206e21 (CHANGELOG commit: heading only over the nch9d4 entry, 0 removed lines vs v0.7.61, no conflict markers), release merge 77d25855 on main. Content: 6aee8e30 (CI 44295351540 all 6 green on exact SHA) and 65ef4d07 dim-c (CI 44298750806 all 6 green on exact SHA). Install only after release.yml success.
release.yml run 44301186296 success on tag v0.7.62 (e4206e21) before install; brew upgrade veridit/tap/hug then hug version: 'hug v0.7.62'. No --machine migration.
## History
- 2026-11-18T13:09 todo -> doing/active (take)
- 2026-11-18T13:09 doing/active -> doing/waiting (wait)
- 2026-11-18T13:15 doing/waiting -> done (done); was doing_since: 2026-11-18T13:09; waiting_asked_by: crab; waiting_on: release.yml for v0.7.62; waiting_since: 2026-11-18T13:09
