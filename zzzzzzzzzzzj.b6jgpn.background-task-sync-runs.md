---
id: b6jgpn
title: "Background task sync runs at full priority; it should be niced"
state: "done"
type: bug
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-17T23:42"
---

## Goal
hug task's detached background sync pass (hug task sync --background-pass, started by gitbranch Background.Nudge) and its git children run niced and with capped GOMAXPROCS, as hug mem's background commands do.

## Purpose
Background work must not spike the machine. muninn's background commands call throttle.Background() (muninn/run/main.go:70); the huginn pass never does: no throttle or Setpriority anywhere under huginn/ or cmd/.

## Principle
Only the detached pass is lowered; interactive hug task commands keep full speed. HUG_FULL_SPEED=1 opts out as in throttle. Children inherit niceness, so lowering the pass process covers its git calls.

## Relevant details
muninn/internal/throttle is internal to muninn/, so huginn cannot import it: move it to a shared package or call it from cmd/hug dispatch for the --background-pass argv. Entry: huginn/internal/cli/cli.go:667 (backgroundPass). Not the cause of the git segfaults in the test loop: those ran under nice -n 19 already.

## Depends on

## Acceptance
- [x] a test starts a background pass and reads its niceness as throttle.Nice (and GOMAXPROCS capped), failing before the change
- [x] HUG_FULL_SPEED=1 leaves the pass at the parent's priority

## Notes


kestrel: landed 24b8a3b..a201b2e. Check 1: TestBackgroundPassRunsNiced spawns real passes and reads the priority each logged; without the throttle.Background() call: 'syncnice_test.go:43: pass 1 ran at "nice 0", want "nice 10" (this test process: "nice 0")'. Check 2: the same test sets HUG_FULL_SPEED=1 in the pass env and requires the parent's priority. -race: cli ok 163.354s (one earlier run failed on 'TestEightWritersMixedCommandsAllLand ... diff --name-status ... signal: segmentation fault', git crashing, filed 3z65k1; another FAIL lost its detail to a tail), throttle ok 3.069s, muninn/run ok 35.363s; 15x TestBackgroundPassRunsNiced+TestBackgroundSync* under -race ok 427.817s.
Reflection (open, 2026-11-17, calf): nothing misled; my own tail -4 hid one failing test's name, which I could not then cite.
## History
- 2026-11-17T23:01 todo -> doing/active (take)
- 2026-11-17T23:42 doing/active -> done (done); was doing_since: 2026-11-17T23:01
