---
id: jffwyv
title: "Release v0.7.106: a task with two Details sections shows both"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzz"
created: 2026-11-22
updated: 2026-11-22
stage: "active"
doing_since: "2026-11-25T13:00"
doing_with: "hktdgs"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
v0.7.106 never hides text when an old hug added a second Relevant details section.

## Purpose
One release ticket per release. v0.7.106 = main 91927791, landed 668e6984 on exact-SHA CI 8/8 (run 45022930847).

## Principle
Nothing for Jørgen to try: no file on the live board has both headings; chxk50 asks about restarting the old processes.

## Details
Shipped hxqqdp checks 1-2: show, --json, MCP and the TUI show both texts (Details first); lint old-layout with fix: hug upgrade; hug upgrade merges them. Cache schema 32. After brew upgrade at 12:52Z: hug task lint 'no findings'.

## Depends on

## Acceptance
- [x] Nothing to try; Jørgen says close, or names what to check

## Questions
- 2026-11-22T13:52 question 1 by crab to jhf (check 1):
  v0.7.106 is installed. Nothing to try: it only matters if an old hug process writes Details (see chxk50). Close this ticket, or is there something you want checked?
  answered 2026-11-25T10:47 by jhf:
    Close it

## Notes

crab 12:53Z: shipped v0.7.106 (main 91927791, landed 668e6984, CI run 45022930847 8/8). release.yml success; brew-verified: Cellar/hug/0.7.106, hug version = v0.7.106. hug task lint: no findings.


check 1: Jørgen 2026-11-25: 'Close it'
## History
- 2026-11-22T13:52 todo -> todo/design (ask)
- 2026-11-25T10:47 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.106 is installed. Nothing to try: it only matters if an old hug process writes Details (see chxk50). Close this ticket, or is there something you want checked?
  Close it
- 2026-11-25T10:47 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-25T13:00 todo/ready -> doing/active (take with hktdgs)
- 2026-11-25T13:00 crab read the answer: hug task take jffwyv
