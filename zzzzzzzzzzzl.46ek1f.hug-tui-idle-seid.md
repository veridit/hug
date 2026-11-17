---
id: 46ek1f
title: "hug tui idle on Seid burns a third of a core"
status: "doing"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "44w1ev"
doing_since: "2026-11-17T18:53"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
An idle hug tui (no keys, board unchanged) uses under 1% CPU on Seid's board, as on a small board.

## Purpose
crab 17:44, measured right after Jørgen praised the v0.7.43 open speed: a v0.7.43 hug tui opened on seid-lang and left alone took 19.2 s CPU over 60 s (ps time 0:01.70 -> 0:20.90), with 1 commit on the board in 10 minutes. Jørgen's v0.7.41 TUI on Seid (pid 227) had used 58:49 CPU in 2 h 37 min. Machine load was 135-185; an idle TUI should not add to it.

## Principle
Find what wakes it (a spinner tick that never stops, a reload loop from its own cache writes, a poll) with a CPU profile; a test counts reloads or ticks over an idle period and fails first. No timing budget.

## Relevant details
Likely candidates: the vnyz8z refresh spinner tick keeps running after the refresh lands; the watcher sees the cache's own writes (.git/huginn/cache-N.db) or the board turn's files and reloads in a loop; the old poll mode. Profile: CRAB_CPUPROFILE-style hook as in np2kk2's note, or go tool pprof against a test.

## Depends on

## Acceptance
- [ ] a test runs the TUI model idle for N watch periods on an unchanged board and counts reloads and ticks: zero reloads, spinner ticks stop once the refresh lands; fails on main
- [ ] on Seid, an idle hug tui uses under 1 s CPU per minute (ps time over 60 s), numbers in the notes

## Notes


Measured 2026-11-17 18:00-18:33 (Seid copy at ~/.jcode/scratch/seidcopy, 2,520 tasks, warm cache; ps time):
- Idle with the board unchanged is already under the goal: main build 0.31 s/60 s; v0.7.43 0.22 s/60 s; v0.7.42 0.08 s/30 s; v0.7.41 0.18 s/60 s. Views counted with a scratch counter: 3 frames in 19 s idle; the spinner stops once the refresh lands.
- The cost is per board change. One hug task note: v0.7.41 16.0 s (24.32 -> 40.33), v0.7.42 0.58-0.78 s, v0.7.44 1.0-1.1 s (1.68 -> 2.82 -> 3.74). Scratch event log: each commit gives 2 reloads, the file event and then the ref event about 300 ms later (debounce is 200 ms); each reload's loadSnapshot takes 150-250 ms.
- Profile of 110 s with 2 notes (3.43 s of samples): View 1.33 s (overviewText 1.21 s: overviewRows 0.49, taskRowMark 0.31, paneIndices 0.14; task.Stage 0.38 s, mostly StageNames allocating); loadSnapshot 0.70 s (taskFiles WalkDir 0.57 s). The cache sync is 21-99 ms and reads 0-1 files.
- crab's 19.2 s in 60 s (17:44) matches a first start, not idle: cache-16.db on live Seid was created at 17:42:10, so that TUI rebuilt the cache for the new schema. Fresh copy with no cache on v0.7.44: 0:01.85 -> 0:04.76 in the first 10 s, then 0.03 s per 10 s.
- Live: Jørgen's pid 99030 (v0.7.43) used 12.5 s over 20.4 min (1:06 -> 1:18.55) while about 10 board commits landed; 0.21 s in the next 60 s.
Next: the per-change cost (2 loads per commit; taskFiles walks every attachment folder on every load; a frame walks all 2,520 tasks), with a test that counts loads per commit.
## History
- 2026-11-17T18:53 todo -> doing/active (take)
