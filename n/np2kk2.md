---
id: np2kk2
title: "hug tui opens Seid in 8-9 s: watch setup globs per folder, sync rereads"
state: "done"
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "44w1ev"
group: ""
rank: "zzzzzzzzzzb"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:39"
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

## Acceptance
- [x] taskDirs (or its successor) reads each folder at most once: a test on a fixture with task folders counts directory reads and fails on the current code
- [x] the first frame does not wait for the watcher: a test shows the board drawn before the watch is set up, and a change made right after start is still seen
- [x] the cause of cache.sync rereading files on a warm Seid start is found and either fixed with a test or explained in a note with evidence
- [x] hug tui on Seid: first frame warm under 400 ms, median of 5, numbers in the notes; Jørgen opens it and agrees

## Notes

crab 15:05: the Seid board has 271 top-level folders and 1,132 folders in all under .hug/tasks (find -mindepth 2 -type d). Profile files: $JCODE_SCRATCH_DIR/tui3.prof with binary hug-prof (a4aa5a5 + CRAB_CPUPROFILE hook in cmd/hug/main.go, scratch only).

Dependency on vnyz8z dropped (wyvern 16:22): its code is on main 43cb24c (af51fc3..43cb24c); vnyz8z stays open only for Jørgen's check 4. Item 2 is mostly vnyz8z's work: the watch starts after the first frame (tui/start.go startWatching, TestQuitDoesNotWaitForWatchSetup). Still owed here: a test that a change made right after start is seen.


e582ab4 (wyvern 16:29). Check 1: TestTaskDirsReadsEachFolderOnce counts folder reads (one per watched folder, 0 in files or hidden folders); it failed on the old code ('0 folder reads', the Globs bypass the counted listing) and a mutation that stops skipping files folders fails it. TestFolderOwnersMatchesIsTaskFolder: same answer as IsTaskFolder on 12 folders of one listing. In-process on a copy of Seid's hug/tasks (3,630 files), 5 rounds old/new: 7.22 s/3.82 ms, 7.70 s/3.56 ms, 7.79 s/3.61 ms, 7.56 s/3.63 ms, 7.01 s/3.45 ms; same folder list. Check 2: the watch starts after the first frame (vnyz8z, on main 43cb24c); a change made before it is up had no event, so the TUI now reloads once when the watch is up: TestChangeBeforeWatchIsUpIsSeen fails without it. Check 3, measured: on the private copy (no other hug), cache.Open plus Sync four times, 2.5 s apart: first 3.00 s (rebuilt, 2520 hashed and parsed), then 10.3, 10.4, 12.1 ms, statted 2520, hashed 0, parsed 0, path walk. So the sync rules do not reread a quiet warm board. On the live seid-lang board at 16:10 the cache's schema_version read 14,14,12,14 over 10 s: a v0.7.41 hug tui (schema 12) and v0.7.42 (schema 14) dropped each other's tables, and every open after a flip is a full rebuild (ReadFile, Parse, FileRefs: the 1.9 s in crab's profile). At 16:27-16:28 it read 14 eight times in 30 s. Not proven: that the 15:03 profile ran during a flip (no record of which binaries were open then). Fix filed as gjmqhn. Check 4 so far (copy, warm, 5 interleaved rounds, load 25-29): np2kk2 first frame 0.174, 0.174, 0.200, 0.212, 0.158 s (median 0.174), quit median 0.211 s; vnyz8z alone 0.166 median.
Check 1 was ticked at 16:29 before e582ab4 reached main (early, my error; hug has no uncheck). It holds when e582ab4 lands; the main run id follows.
landed e582ab4 on main 2026-11-17: hug tui: the watch setup lists each folder once (taskDirs on Seid 7.0-7.8 s ->…
Checks 1-3 hold on main 45cbc0b, CI 44127663573 green on all 5 jobs (secrets, bench, ubuntu with -race, macos, windows); it contains 43cb24c, e582ab4 and the race fix d16fb54 (TestTaskDirsReadsEachFolderOnce, TestFolderOwnersMatchesIsTaskFolder, TestChangeBeforeWatchIsUpIsSeen; check 3 per the 16:29 note). Check 4 waits for Jørgen.
shipped in v0.7.43 (e582ab4)
check 4 ticked by crab on Jørgen's own words 2026-11-17 on the Seid TUI open: 'awesome speed up'; and 00:38 'go over and lint our own tasks'.
## History
- 2026-11-17T17:22 todo -> doing/active (take)
- 2026-11-17T17:30 doing/active -> doing/waiting (wait)
- 2026-11-18T00:39 doing/waiting -> done (done); was doing_since: 2026-11-17T17:22; waiting_asked_by: wyvern; waiting_for: jhf; waiting_on: check 4: open hug tui on Seid and say if it feels instant (copy median first frame 0.174 s, quit 0.211 s); checks 1-3 tick on green main CI 44126454244 (e582ab4); waiting_since: 2026-11-17T17:30
