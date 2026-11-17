---
id: np2kk2
title: "hug tui opens Seid in 8-9 s: watch setup globs per folder, sync rereads"
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
hug tui on the Seid board (2,519 tasks, 1,132 attachment folders, 79 MB) draws its first frame in under 400 ms warm, as every other interaction (Jørgen's target 2026-11-16), and the watcher is set up without delaying that frame.

## Purpose
Jørgen 2026-11-17 15:03: 'still very very slow to open hug tui on seid, around eight or nine seconds'. Measured by crab, interleaved, load 12-17: v0.7.41 first frame 8.2 s and 9.6 s; wyvern's land stack a4aa5a5 6.0 s and 6.2 s interleaved, then 5.5, 3.0, 2.9 s alone. 44w1ev check 4 measured a synthetic board without attachment folders, so it missed this.

## Principle
Measure, then remove the largest cost; a test that counts the work (directory reads, file reads) fails first, no time budget in normal tests. Correctness first: the watcher must still see every task folder and attachment change.

## Relevant details
CPU profile of a4aa5a5 + a local pprof hook, hug tui on Seid, ~5 s sampled: (1) tui.startWatch -> taskDirs 2.0-2.07 s (41%): WalkDir over the whole board, and task.IsTaskFolder (huginn/internal/task/files.go:37) runs two filepath.Glob per directory, each of which reads the whole parent folder: folders x entries readdirs (Readdirnames 1.13 s, syscall.Open 1.04 s). Fix shape: one ReadDir per folder and decide task folders from that listing (the same trick as loadSnapshot's 44w1ev folder list), and start the watch after the first frame. (2) cache.sync 1.85-1.95 s on starts that should be warm: sharefile.ReadFile 1.05 s, task.Parse 0.45 s, FileRefs 0.39 s, so sync rereads files it should skip. Find out why (racy-clean window? a writer on the Seid board? schema rebuild when binaries alternate?) and record it. hug task ls warm on the same board: 0.23 s on a4aa5a5, 2.45 s on v0.7.41. hug task lint: 7.1 s -> 0.55 s. Timing script: tmp/lanes/tuitime.sh BIN DIR (crab's machine).

## Depends on

- vnyz8z
## Acceptance
- [ ] taskDirs (or its successor) reads each folder at most once: a test on a fixture with task folders counts directory reads and fails on the current code
- [ ] the first frame does not wait for the watcher: a test shows the board drawn before the watch is set up, and a change made right after start is still seen
- [ ] the cause of cache.sync rereading files on a warm Seid start is found and either fixed with a test or explained in a note with evidence
- [ ] hug tui on Seid: first frame warm under 400 ms, median of 5, numbers in the notes; Jørgen opens it and agrees

## Notes

crab 15:05: the Seid board has 271 top-level folders and 1,132 folders in all under .hug/tasks (find -mindepth 2 -type d). Profile files: $JCODE_SCRATCH_DIR/tui3.prof with binary hug-prof (a4aa5a5 + CRAB_CPUPROFILE hook in cmd/hug/main.go, scratch only).
