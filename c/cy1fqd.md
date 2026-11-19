---
id: cy1fqd
title: "TestTmuxReload: 5 s screen waits miss a slow first frame in CI"
state: "dropped"
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzr"
created: 2026-11-19
updated: 2026-11-19
dropped_reason: "Not reproduced on main (crab 01:11, option a): 0 'screen never showed' in 26 completed ci.yml runs since hfkt1n (2026-11-18 23:00Z, 21 success, 5 failure: TestUpstreamDiff 44417801741, TestTagRowsGolden 44419243717, TestTmuxGolden palette drift 44428150586, TestCompletionInEachShell/fish 44430041895, macOS install tests 44431228949). No local run: TestTmuxReload's loadGuard holds above 5-minute load 120 and the load was 784 ('reload_tmux_test.go:36: load guard: paused before heavy work, 5-minute load 783.79 \u003e 120'). ghycrq does not cover it (only startTmuxServer). If it recurs, iwazaru's CI census catches it."
dropped_at: "2026-11-19T02:12"
---

## Goal
TestTmuxReload waits for the screen it wants on a loaded CI runner: its waitFor calls (reload_tmux_test.go, 'screen never showed ...') wait on the TUI reaching that screen, not on a 5 s clock, the same way hfkt1n fixed the half-drawn frame and adqdfe the reload's git

## Purpose
Four ubuntu CI runs on four branches failed this way on 2026-11-18 (03:41-21:15 UTC); none was a broken screen, each cost a rerun. All four came before hfkt1n landed (bc217cb3, 23:00), which fixed half-drawn and loading frames, not a screen that has not arrived in 5 s: first confirm it still happens on main after bc217cb3

## Principle
Cause first, fail-first under forced starvation; no timeout raised, no retry; check ghycrq (tmux harness deadlines) does not already cover it before starting

## Relevant details
CI census 2026-11-19 (iwazaru, scratch/ci-census.md), ubuntu-latest -race: 44232153907 ci/qh9ep1 eff1c4c1 'reload_tmux_test.go:129: screen never showed "HUG TASKS · list"' while the screen showed 'HUG TASKS · overview   60/60 tasks' (the view switch had not landed); 44236159783 ci/ahejza 1b9da4e6 'reload_tmux_test.go:131: screen never showed "state:doing"'; 44386015920 ci/05r3pg 265c7549 'reload_tmux_test.go:161: screen never showed "COMMAND PALETTE"' with the task detail on screen; 44399621934 ci/rxr4z8 d9a8dcec 'reload_tmux_test.go:124: screen never showed "hug v0.0.1"'. hfkt1n (done, bc217cb3) kept waitFor at 5 s / 15 s. Earlier 'not the same place after the reload' (44132567751, 44133943837, main 2026-11-17) and 'the reload line stayed' (44326448795) are hfkt1n's.

## Depends on

## Acceptance
- [ ] Cause named in a note with a log line or reproduction
- [ ] A test or amplified reproduction fails first, then TestTmuxReload passes -count=20 niced under load after the fix
- [ ] No timeout raised and no retry added

## Notes

crab 01:06: stays todo. Confirm on main after bc217cb3 (hfkt1n) and ghycrq (tmux harness: server start and cleanup deadlines) land, before anyone works it; it may be ghycrq's.

macaque 01:09, before any fix (crab's steps). (1) ghycrq (ci/ghycrq 2f7a91f8, tigress) read: its diff touches only startTmuxServer (ready = the server's socket takes a connection; cleanup = SIGTERM + wait) and adds TestTmuxServerSlowStarts. TestTmuxReload's waitFor (reload_tmux_test.go:111, 5 s / 15 s deadlines on capture-pane) is unchanged by it: ghycrq does not cover cy1fqd. (2) Main 50fa8681 (has hfkt1n bc217cb3, adqdfe bf170d46). CI census since 2026-11-18T23:00Z: 26 completed ci.yml runs, 21 success, 5 failure, 0 'screen never showed' (failures: 44431228949 TestInstallThrowawayHomes etc. macOS, 44430041895 TestCompletionInEachShell/fish, 44428150586 TestTmuxGolden 'golden drift .../palette.txt', 44419243717 TestTagRowsGolden, 44417801741 TestUpstreamDiff). Local repro not run: TestTmuxReload calls loadGuard (tmux_test.go:20, holds while the 5-minute load > 120) and the load is 784 ('reload_tmux_test.go:36: load guard: paused before heavy work, 5-minute load 783.79 > 120'); a 200-run would only wait, so I stopped it. STARVE (starve_test.go) amplifies a test in-process; TestTmuxReload's waits are on a separate hug process under tmux, so STARVE in the test process does not slow the TUI being waited on.


## History
- 2026-11-19T02:12 todo -> dropped (drop)
