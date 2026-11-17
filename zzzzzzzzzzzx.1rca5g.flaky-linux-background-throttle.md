---
id: 1rca5g
title: "Flaky on Linux: background throttle test reads nice=0 instead of 10"
status: "doing"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T14:38"
stage: "active"
model: "claude-opus-5-5"
branch: "ci/1rca5g"
worktree: "/Users/jhf/.jcode/scratch/guppy-own"
---

## Goal
TestBackgroundLowersPriorityAndProcs passes reliably on ubuntu: the cause of reading 'nice=0' after Background() is found and fixed, with a test that fails before the fix.

## Purpose
Main CI run 44092559068 on d198e01, ubuntu: 'throttle_test.go:48: Background: got "nice=0 procs=1", want "nice=10 procs=1"'. The same commit was green on ci/6ephqk-rows (run 44090100603). It blocked the v0.7.41 release.

## Principle
Find the cause: on Linux setpriority(PRIO_PROCESS, 0) applies to the calling thread, not the whole process, so a goroutine on another OS thread can read nice=0. Fix hug if hug's throttling is per thread (then background work is not actually niced), not just the test.

## Relevant details
muninn/internal/throttle/throttle_test.go:48. Linux nice is per thread (tid); Go schedules goroutines across threads. If Background() renices only one thread, hug's background jobs run un-niced on other threads, which breaks the machine-load rule (nice -n 19 on the shared machine).

## Depends on

## Acceptance
- [ ] the cause is named with evidence (which thread was niced, e.g. /proc/self/task/*/stat before and after)
- [ ] background work is niced on every thread hug uses for it (test reads every /proc/self/task entry), and the test passes -count=50 on ubuntu CI (run id)

## Notes


## History
- 2026-11-17T14:38 todo -> doing/active (take)
