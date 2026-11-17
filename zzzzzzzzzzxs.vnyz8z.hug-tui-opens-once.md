---
id: vnyz8z
title: "hug tui opens at once from the last snapshot, refreshes behind a spinner"
status: "doing"
type: feature
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "44w1ev"
doing_since: "2026-11-17T16:12"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "jhf: check 4: open and quit hug tui on Seid (median first frame 0.26 s, quit 0.29 s measured on a copy) and say if you agree; checks 1-3 tick on green main CI 44124879024"
waiting_since: "2026-11-17T17:21"
waiting_asked_by: "wyvern"
---

## Goal
hug tui draws a board within 200 ms of start on any board size: it shows the last snapshot it saw (kept in the cache), marks the header with a spinner and 'refreshing', and swaps in the fresh board when the sync lands, keeping the selection. Quitting returns the shell in under 1 s: q never waits for a sync or the watcher. Every wait over 200 ms (start, refresh, quit, a write) shows the spinner, never a frozen screen.

## Purpose
Jørgen 2026-11-17 15:08: 'during startup there is no spinner, it just hangs, and also during close down. It should at least be a spinner, and we should aim at sub-second startup and shutdown.' And: 'do you have to have perfect fidelity when you open it up, or can you open and then adjust? We receive updates while inside the TUI, so the refresh comes in as soon as it lands, with a spinner telling you it is updating in the background.' Measured on Seid: v0.7.41 first frame 8.2-9.6 s, quit 2.9 s; wyvern's stack a4aa5a5 first frame 2.9-6.2 s, quit 0.29 s.

## Principle
Show what is known now, say plainly that it is being refreshed, never act on a stale view: a key that writes while refreshing goes through the store, which reads the file fresh, as today. The spinner is subtle (header, not a modal). Nothing silent: when the refresh changes what is shown, the selection stays on the same task. np2kk2 removes the costs; this ticket stops the user from waiting on whatever cost is left.

## Relevant details
TUI start today: openBoard -> cache.sync -> loadSnapshot -> startWatch, all before the first frame (huginn/internal/tui/tui.go:59, watch.go:68). The TUI already reloads in the background on watch events (apply keeps the selected path), so the fresh board can arrive the same way. The cache can keep the last snapshot (task rows + findings) for a cold-free first frame. First start on a board with no cache: the spinner with 'reading N tasks' instead of a blank screen. Quit: stop the watcher and any running sync without waiting for them; a write in flight finishes (writes never fail) but the screen is released first.

## Depends on

## Acceptance
- [x] the first frame comes from the last snapshot before any sync: a test with a slow sync stub draws the board and the spinner, then the fresh board, selection kept
- [x] q returns while a sync or the watch setup is still running: a test with a blocked sync stub sees the program exit
- [x] a first start with no cache shows the spinner and a count, never a blank screen (golden)
- [ ] on Seid, first frame under 1 s and quit under 1 s, median of 5 with tmp/lanes/tuitime.sh, numbers in the notes; Jørgen opens and quits it and agrees

## Notes

crab 15:09: wyvern does vnyz8z first, then np2kk2 (same start path in tui.go and watch.go, one lane so they don't collide with nautilus). Shutdown measured on Seid: v0.7.41 2.89 s, a4aa5a5 0.29 s.


Design (wyvern 15:31):
(1) runTUI opens the board lazily and passes Options.Initial, the tasks and findings exactly as cache.Open loaded them (board blob only: no walk, no git), plus Options.Refresh, the full sync.
(2) tui.Run builds the model from Initial and sets refreshing. Init returns the refresh as a command; the fresh snapshot arrives as a loadedMsg and goes through applyKeeping (selection kept), as watch reloads already do. With no cache, Initial is empty and the screen says 'reading the board' with the spinner, never blank.
(3) startWatch runs as a command after the first frame, so taskDirs (np2kk2) never delays drawing.
(4) q returns at once: watch and bin closes do not block the exit, and the sync is cancelled through the program context.
(5) Spinner: a dim frame in the header, ticking every 120 ms only while a load has run longer than 200 ms. It is the render.go header line, agreed with nautilus 15:23.
Tests use a Refresh stub that blocks on a channel.
Measured (wyvern 16:12), Seid board copy: git fetch of seid-lang hug/tasks into a scratch repo (3,630 task files, 2520 tasks), so no other hug touches its cache; HUG_SYNC=off; warm cache; tmux 160x45; tuiquit.sh = tmp/lanes/tuitime.sh plus quit timing (q to process gone). 5 interleaved rounds after one warm-up each. main d661273 first frame: 9.01, 8.96, 9.08, 9.17, 9.36 s (median 9.08); quit 0.29, 0.32, 0.30, 0.33, 0.32 s. vnyz8z first frame: 0.24, 0.30, 0.24, 0.27, 0.26 s (median 0.26); quit 0.28, 0.29, 0.31, 0.31, 0.29 s (median 0.29). The vnyz8z first frame shows the whole board: 'HUG TASKS · overview   2520/2520 tasks · ! 5030 findings', IN PROGRESS selected. On the live seid-lang board the first frame was '0/0 tasks': a v0.7.41 hug tui (schema 12) and v0.7.42 (schema 14) rebuild each other's cache.db on every open (read 14,14,12,14 over 10 s), filed as gjmqhn. Cold first start (cache folder moved away): 'reading the board: the first start builds the task cache: 242 of 2520 files', then '2464 of 2520 files', then the board. tmux review test failure fixed: '#aaaaaa Enter' typed before the first board is now answered when it lands (TestJumpTypedBeforeBoardOpensWhenItArrives, fails first; 2 mutations caught).
landed 7705bce on main 2026-11-17: hug tui: draws the cache's last snapshot at once, refreshes behind a spinner…
landed 43cb24c on main 2026-11-17: hug tui: a first start counts the task files it reads ("1234 of 3630 files")…
landed d16fb54 on main 2026-11-17: hug tui: startWatching reads the watch starter on the UI thread, not in its…
Checks 1-3 on main 45cbc0b, CI 44127663573 green on all 5 jobs (secrets, bench, ubuntu with -race, macos, windows); it contains 43cb24c, e582ab4 and the race fix d16fb54. 1: TestStartDrawsSnapshotThenRefreshes; 2: TestQuitDoesNotWaitForRefresh and TestQuitDoesNotWaitForWatchSetup (the latter raced on 43cb24c's ubuntu run 44124879024, fixed by d16fb54); 3: TestFirstStartWithoutCacheSaysSo golden with 'reading the board: ...: 1234 of 3630 files', TestProgressCountsReads. Check 4 waits for Jørgen.
shipped in v0.7.43 (7705bce)
landed ceb9d1a on main 2026-11-17: hug tui test: TestTmuxReload compares the reloaded hug's frame once it has the…
landed f9e5664 on main 2026-11-17: hug tui: the first frame has the terminal's size, read from stdout before the…
shipped in v0.7.44 (ceb9d1a)
shipped in v0.7.45 (f9e5664)
## History
- 2026-11-17T16:12 todo -> doing/active (take)
- 2026-11-17T17:21 doing/active -> doing/waiting (wait)
