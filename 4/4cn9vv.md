---
id: 4cn9vv
title: "Release v0.7.80"
state: "todo"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzyh"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "ready"
---

## Goal
v0.7.80 (m5pbb0) is published and installs via brew upgrade

## Purpose
Hourly release

## Principle
Release procedure: exact-SHA green main, CHANGELOG cut, tag, release.yml, brew verified

## Relevant details
main dd6e5cf9 (CI 44500113493 8/8), CHANGELOG b2c9d5e4, merge 41dc277d, release.yml 44501832424

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.80 with links into the new Cellar

## Questions
- 2026-11-19T12:21 question 1 by crab to jhf:
  m5pbb0: in hug tui (v0.7.80 or later), on a task with several open questions, can you reach and answer the second one?
  answered 2026-11-20T11:06 by jhf:
    Yes, I've done it. It worked. Nice. The only thing I wonder about is where are the questions? Because when I view the file, I can't see the questions in the markdown. But so where are the physically? You can just answer that in the console and you can close this ticket.

## Notes

check 1: release.yml 44501832424 success; brew upgrade: hug version 0.7.80, links into Cellar/hug/0.7.80

check 1: release.yml 44501832424 success; brew upgrade: hug version v0.7.80, hug/muninn/huginn link into Cellar/hug/0.7.80


check 1: release.yml 44501832424 success; brew upgrade: hug version v0.7.80, hug/muninn/huginn link into Cellar/hug/0.7.80
Reflection (unanswered, 2026-11-19, crab)
## History
- 2026-11-19T11:10 todo -> doing/active (take with p277bp)
- 2026-11-19T11:10 doing/active -> done (done); was doing_since: 2026-11-19T11:10; doing_with: p277bp
- 2026-11-19T12:21 done -> todo/ready (reopen); was done_at: 2026-11-19T11:10
- 2026-11-19T12:21 todo/ready -> todo/design (ask)
- 2026-11-20T11:06 answered design, by jhf, for crab (todo/design -> todo/ready):
  > m5pbb0: in hug tui (v0.7.80 or later), on a task with several open questions, can you reach and answer the second one?
  Yes, I've done it. It worked. Nice. The only thing I wonder about is where are the questions? Because when I view the file, I can't see the questions in the markdown. But so where are the physically? You can just answer that in the console and you can close this ticket.
- 2026-11-20T11:06 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
