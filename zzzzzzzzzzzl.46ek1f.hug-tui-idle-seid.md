---
id: 46ek1f
title: "hug tui on Seid: one reload per commit; a frame's work does not grow with the board"
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
An open hug tui on Seid's board (2,520 tasks) spends little CPU per board change and per keypress: one load per commit, and a frame whose work does not grow with the number of tasks.

## Purpose
crab 17:44, measured right after Jørgen praised the v0.7.43 open speed: a v0.7.43 hug tui opened on seid-lang and left alone took 19.2 s CPU over 60 s (ps time 0:01.70 -> 0:20.90), with 1 commit on the board in 10 minutes. Jørgen's v0.7.41 TUI on Seid (pid 227) had used 58:49 CPU in 2 h 37 min. Machine load was 135-185; an idle TUI should not add to it.

## Principle
Count, never time: a test counts loads per commit, and another counts per-task work per frame on a large board; each fails first. Per-snapshot facts (stages, pane membership, progress width, attachment lists) are computed once per load, not once per frame.

## Relevant details
Re-scoped 2026-11-17 18:44 with crab: the idle premise did not reproduce (idle 0.18-0.31 s/60 s on every release; crab's 19.2 s was a first start rebuilding cache-16.db at 17:42). Measured per change: one hug task note costs 16.0 s on v0.7.41, 0.6-0.8 s on v0.7.42, 1.0-1.1 s on v0.7.44; each commit gives 2 loads (task file event, then ref event about 300 ms later); View is about 40 ms per frame (overviewText 1.21 s of 1.33 s over 32 frames; task.Stage via StageNames allocates per row).

## Depends on

## Acceptance
- [ ] a test runs the TUI model idle for N watch periods on an unchanged board and counts reloads and ticks: zero reloads, spinner ticks stop once the refresh lands; fails on main
- [ ] on Seid, an idle hug tui uses under 1 s CPU per minute (ps time over 60 s), numbers in the notes
- [ ] a test commits once on a watched board and counts loads: exactly 1 per commit (a hug task note writes the file and moves the ref); fails on main
- [ ] a test renders the overview of a 2,000-task board and counts per-task stage computations per View: none once the snapshot is loaded; fails on main
- [ ] on the Seid copy, the CPU of one hug task note in an open TUI (ps time) before and after, in the notes

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
