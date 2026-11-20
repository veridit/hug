---
id: 3jnfsm
title: "Release v0.7.78"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzy1"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T08:07"
---

## Goal
v0.7.78 is tagged, release.yml publishes it, and brew upgrade brings hug version v0.7.78 on this machine.

## Purpose
Release j13xs0: rescope records each move in memory_fixes; stats --fixes lists them.

## Principle
Release from exact-SHA green main only; brew on its own.

## Relevant details
main 8c0093d0 CI 44470335475 success 8/8. CHANGELOG 15df49d1, merge c2f08fa5, release.yml 44476285935.

## Depends on

## Acceptance
- [x] release.yml for v0.7.78 succeeds (run id)
- [x] hug version prints v0.7.78 after brew upgrade

## Notes


check 1: release.yml 44476285935 completed success
check 2: brew upgrade run on its own; hug version prints hug v0.7.78; bin links point at Cellar/hug/0.7.78
## History
- 2026-11-19T08:07 todo -> doing/active (take with p277bp)
- 2026-11-19T08:07 doing/active -> done (done); was doing_since: 2026-11-19T08:07; doing_with: p277bp
