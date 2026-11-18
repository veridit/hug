---
id: 1rca5g
title: "Flaky on Linux: background throttle test reads nice=0 instead of 10"
state: "done"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzz9"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "ci/1rca5g"
worktree: "/Users/jhf/.jcode/scratch/guppy-own"
done_at: "2026-11-17T16:08"
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
- [x] the cause is named with evidence (which thread was niced, e.g. /proc/self/task/*/stat before and after)
- [x] background work is niced on every thread hug uses for it (test reads every /proc/self/task entry), and the test passes -count=50 on ubuntu CI (run id)

## Notes


hatchling 13:39: cause, with evidence: Linux niceness is per thread; Background() called setpriority(PRIO_PROCESS, 0), which lowers only the calling thread. New test TestBackgroundNicesEveryThread (threads_linux_test.go) holds 4 locked threads before and 4 after Background and reads every /proc/self/task/*/stat field 19. Before the fix, in golang:1.26-alpine (docker, 2 CPUs): '11 of 12 threads not niced to 10 after Background: [tid=1153 nice=0 ...]'. The old test read nice of whichever thread its goroutine landed on, so it passed or failed by scheduling (the CI flake). Fix: os_linux.go lowerThreads() renices each /proc/self/task tid (repeat passes until stable); new threads inherit from their creator. After: -count=50 of TestBackground* green in the same container. ci/1rca5g 650bfb5+.
hatchling 13:42: ubuntu CI probe run 44094839981 (probe/1rca5g = e420c3a + probe workflow, not for main): '50 --- PASS: TestBackgroundLowersPriorityAndProcs', '50 --- PASS: TestBackgroundNicesEveryThread', then -race -count=50 'ok ... 153.149s'. ci/1rca5g e420c3a CI 44094805524 running.
hatchling 15:08: landed on main by fast-forward 4ecfa25..b8f9000 (zd3x24 95bcb99, 1rca5g a8d2f90, tpnd3n b8f9000). Pre-push CI 44107906191 green on ubuntu, macos, windows; main CI 44110812902 started.
## History
- 2026-11-17T14:38 todo -> doing/active (take)
- 2026-11-17T14:45 doing/active -> doing/waiting (wait)
- 2026-11-17T16:08 doing/waiting -> done (done); was doing_since: 2026-11-17T14:38; waiting_asked_by: hatchling; waiting_on: its landing turn (ci/1rca5g e420c3a, CI 44094805524); waiting_since: 2026-11-17T14:45
