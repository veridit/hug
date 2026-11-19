---
id: dvr16b
title: "Release v0.7.77"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzz8"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "waiting"
doing_since: "2026-11-19T13:42"
doing_with: "p277bp"
waiting_on: "crab: a6xm8n day-later count at 2026-11-20 05:45Z"
waiting_since: "2026-11-19T13:42"
waiting_asked_by: "crab"
---

## Goal
v0.7.77 is tagged, release.yml publishes it, and brew upgrade brings hug version v0.7.77 on this machine.

## Purpose
Release a6xm8n: a hook starts a call only when the time left covers it; a mid-body budget cut is a timeout.

## Principle
Release from exact-SHA green main only; brew on its own.

## Relevant details
main c2b0bbe5 CI 44464516250 success 8/8. CHANGELOG 020b9592, merge 1ce66c8a, release.yml 44465643587.

## Depends on

## Acceptance
- [x] release.yml for v0.7.77 succeeds (run id)
- [x] hug version prints v0.7.77 after brew upgrade
- [ ] crab: capture_runs on the live store shows no 'context deadline exceeded' on turn_end hooks for a day after 2026-11-19T05:37:40Z (a6xm8n; count in a note, scheduled 2026-11-20 05:45Z)

## Notes


check 1: release.yml 44465643587 completed success
check 2: brew upgrade run on its own; hug version prints hug v0.7.77; bin links point at Cellar/hug/0.7.77
## History
- 2026-11-19T06:37 todo -> doing/active (take with p277bp)
- 2026-11-19T06:37 doing/active -> done (done); was doing_since: 2026-11-19T06:37; doing_with: p277bp
- 2026-11-19T13:42 done -> todo/ready (reopen); was done_at: 2026-11-19T06:37
- 2026-11-19T13:42 todo/ready -> doing/active (take with p277bp)
- 2026-11-19T13:42 doing/active -> doing/waiting (wait)
