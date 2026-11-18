---
id: ad62aa
title: "cli tests: commit WaitDelay and TestDeduceTwoClonesOneLine fail under load"
state: "doing"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:21"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
huginn/internal/cli tests pass under machine load as they do alone: name the cause of the 'exec: WaitDelay expired before I/O complete' commit errors and the TestDeduceTwoClonesOneLine failure seen at load 300-700, and close it at the cause

## Purpose
Local suites on this shared machine fail on these, so workers can't trust a red run and CI gating slows down

## Principle
No timeout raised, no retry, no skip; find what holds the pipe open or races, like ybbfsa flake 3 (maintenance repack)

## Relevant details
iwazaru 34r1ft run: 2 git commit WaitDelay errors + TestDeduceTwoClonesOneLine at load 300-700; same tests pass alone (ok huginn/internal/cli 413.676s). tigress saw 'WaitDelay expired before I/O complete' in catchup_test.go:68 before 6ecd08fc. Check whether it still happens on main 477696be

## Depends on

## Acceptance
- [ ] Cause named in a note with a log line or reproduction
- [ ] A test or amplified reproduction fails first, then passes -count=50 niced under load after the fix
- [ ] No timeout raised and no retry added

## Notes


## History
- 2026-11-18T20:21 todo -> doing/active (take)
