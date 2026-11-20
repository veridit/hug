---
id: xekn3z
title: "Release v0.7.88"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzo"
created: 2026-11-20
updated: 2026-11-20
stage: "active"
doing_since: "2026-11-20T10:45"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
v0.7.88 is published, installs via brew, and Jørgen says whether his board stays clean after a release.

## Purpose
Release-as-group trial (75mxf3): one ticket groups what ships and asks one question per user-facing change.

## Principle
Shipped tickets are children; one ask per user-facing change.

## Relevant details
main 1e4a3386 (CI 44710823480, 8/8), CHANGELOG 79a9ae52, merge 489684a6, release.yml 44712289344 success. Ships 72n8m4 (done refuses a task with an open question; the board cue stops offering it) and yf39ya checks 7-9 (todo task with an open question is quiet; a shipped task's agent-owned check is handed to its owner; a shipped task waiting on a dependency is quiet until it holds).

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.88 with links into the new Cellar

## Questions
- 2026-11-20T10:45 question 1 by crab to jhf:
  72n8m4 + yf39ya: after this release, does your overview stay free of 'shipped, but not done', with what waits on you in ASKED and not in IN PROGRESS? (The 2 broken left are the quoted-path false positive, 7j706s.)
  answered 2026-11-20T11:42 by jhf:
    I think it does. It would have been simpler if you took some screenshots and added them to the files list of this one so I can just look at what you meant

## Notes

check 1: release.yml 44712289344 success; brew upgrade: hug v0.7.88, hug/muninn/huginn -> ../Cellar/hug/0.7.88/bin/*


check 1: release.yml 44712289344 success; brew upgrade: hug v0.7.88, hug/muninn/huginn -> ../Cellar/hug/0.7.88/bin/*
Jørgen 2026-11-20 10:43Z (answer 1): 'I think it does. It would have been simpler if you took some screenshots and added them to the files list of this one so I can just look at what you meant.' Lesson (crab): every release ask attaches ANSI frames (before and after) of what it asks about to the release ticket.
## History
- 2026-11-20T10:45 todo -> todo/design (ask)
- 2026-11-20T10:45 todo/design -> doing/asked (take, 1 open question(s) kept); was design_asked_by: crab; design_for: jhf; design_question: 72n8m4 + yf39ya: after this release, does your overview stay free of 'shipped, but not done', with what waits on you in ASKED and not in IN PROGRESS? (The 2 broken left are the quoted-path false positive, 7j706s.); design_since: 2026-11-20T10:45
- 2026-11-20T11:42 answered asked, by jhf, for crab (doing/asked -> doing/active):
  > 72n8m4 + yf39ya: after this release, does your overview stay free of 'shipped, but not done', with what waits on you in ASKED and not in IN PROGRESS? (The 2 broken left are the quoted-path false positive, 7j706s.)
  I think it does. It would have been simpler if you took some screenshots and added them to the files list of this one so I can just look at what you meant
- 2026-11-20T11:42 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:43 answer 1 acknowledged by crab
- 2026-11-20T11:44 crab read the answer: hug task note xekn3z "Jørgen 2026-11-20 10:43Z (answer 1): 'I think it does. It would have been simpler if you took some screenshots and adde…"
