---
id: awht8w
title: "deliver test: hook ack misses under load"
state: "doing"
type: feature
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:20"
stage: "active"
model: "claude-opus-5-5"
branch: "byj479"
worktree: "/Users/jhf/.jcode/scratch/byj479"
---

## Goal
TestRequestReachesCoordinator passes under machine load as it does alone: name why crab's hook did not acknowledge within 20 s (deliver_test.go:194) at load 95-115, and close it at the cause

## Purpose
A red local cli run on this shared machine must mean a real failure; this flake in the delivery path makes workers rerun and distrust the suite

## Principle
Cause first; no timeout raised, no retry, no skip; a fail-first reproduction under forced starvation, as ad62aa did

## Relevant details
iwazaru byj479 full niced run on fcd1adb1 (scratch/byj479-full.txt): '--- FAIL: TestRequestReachesCoordinator (24.51s) deliver_test.go:194: crab's hook did not acknowledge', load 95-115; alone 2/2 ok (1.786s, 2.072s). Sibling of ad62aa (WaitDelay under starvation). The hook sends on every return path of Acknowledge, so a timeout means it ran >20 s or blocked.

## Depends on

## Acceptance
- [ ] Cause named in a note with a log line or reproduction
- [ ] A test or amplified reproduction fails first, then passes -count=50 niced under load after the fix
- [ ] No timeout raised and no retry added

## Notes


## History
- 2026-11-18T22:20 todo -> doing/active (take)
