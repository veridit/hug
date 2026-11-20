---
id: 0xzfsz
title: "Release v0.7.76"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxv"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T05:29"
---

## Goal
v0.7.76 is tagged, release.yml publishes it, and brew upgrade brings hug version v0.7.76 on this machine.

## Purpose
Release nmn6wf: capture retries a hook call cut at 1024 tokens and defers what stays cut.

## Principle
Release from exact-SHA green main only; brew on its own.

## Relevant details
main e98a12ac CI 44456503911 success 8/8.

## Depends on

## Acceptance
- [x] release.yml for v0.7.76 succeeds (run id)
- [x] hug version prints v0.7.76 after brew upgrade

## Notes


check 1: release.yml 44457808705 completed success
check 2: brew upgrade run on its own; hug version prints hug v0.7.76; bin links point at Cellar/hug/0.7.76
## History
- 2026-11-19T05:29 todo -> doing/active (take with p277bp)
- 2026-11-19T05:29 doing/active -> done (done); was doing_since: 2026-11-19T05:29; doing_with: p277bp
