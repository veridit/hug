---
id: gn7tsm
title: "Release v0.7.79"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzztr"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T08:29"
---

## Goal
v0.7.79 is tagged, release.yml publishes it, and brew upgrade brings hug version v0.7.79 on this machine.

## Purpose
Release b5jks7: search matches concept words in one read; capture planning 115.8 -> 66.3 s on replay, output byte-identical.

## Principle
Release from exact-SHA green main only; brew on its own.

## Relevant details
main 75cf6431 CI 44478091776 success 8/8.

## Depends on

## Acceptance
- [x] release.yml for v0.7.79 succeeds (run id)
- [x] hug version prints v0.7.79 after brew upgrade

## Notes


check 1: release.yml 44479244769 completed success
check 2: brew upgrade run on its own at 07:29:19Z; hug version prints hug v0.7.79; bin links point at Cellar/hug/0.7.79
## History
- 2026-11-19T08:29 todo -> doing/active (take with p277bp)
- 2026-11-19T08:29 doing/active -> done (done); was doing_since: 2026-11-19T08:29; doing_with: p277bp
