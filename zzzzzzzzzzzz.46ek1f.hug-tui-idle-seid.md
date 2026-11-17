---
id: 46ek1f
title: "hug tui idle on Seid burns a third of a core"
status: todo
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "44w1ev"
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
