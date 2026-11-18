---
id: 9ahvgt
title: "stale-owner lint misses owners whose sessions are gone"
state: "doing"
type: bug
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:44"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task lint reports stale-owner for every doing task whose assignee has had no live harness session for over a day, including workers spawned in other jcode homes and names with no session file at all.

## Purpose
retriever 2026-11-18 12:34Z: lint --code stale-owner said 'no findings' while 26+ doing tasks belonged to gone sessions: hatchling 6, nautilus 11 (journal last 07:46), kestrel 8 (no session file), wyvern 1. Requests and answers then reach nobody.

## Principle
A finding must reflect what hug can actually see; when it cannot see sessions (unreadable home) it says so once, rather than staying silent.

## Relevant details
Code: huginn/internal/task/owner.go:95 builds the finding; it reads ~/.jcode/sessions only when readable. Check: does it match assignee names to session names (session_<name>_<ts>), and how does it treat a name with no session file (kestrel)? Scratch: /Users/jhf/.jcode/scratch/board-orphans/REPORT.md.

## Depends on

## Acceptance
- [ ] On hug's board, lint reports stale-owner for nautilus, kestrel and hatchling tasks; a test with a gone owner and with an owner that has no session file fails first on main.
- [ ] A live worker's task is not reported (test).

## Notes


## History
- 2026-11-18T13:44 todo -> doing/active (take)
