---
id: cy1fqd
title: "TestTmuxReload: 5 s screen waits miss a slow first frame in CI"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
TestTmuxReload waits for the screen it wants on a loaded CI runner: its waitFor calls (reload_tmux_test.go, 'screen never showed ...') wait on the TUI reaching that screen, not on a 5 s clock, the same way hfkt1n fixed the half-drawn frame and adqdfe the reload's git

## Purpose
Four ubuntu CI runs on four branches failed this way in 26 h after hfkt1n; each costs a rerun, and none was a broken screen

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
