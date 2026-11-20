---
id: 6zhyq2
title: "TestViewKeysBlinkAfterV flakes on CI timing (497 ms vs 530 ms)"
state: "doing"
type: bug
priority: ""
assignee: ["bird"]
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-20T00:04"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The view-line blink tests pass on CI regardless of tick timing.

## Purpose
maple 2026-11-19 (yp3c09 CI 44520969132): TestViewKeysBlinkAfterV failed once measuring 497 ms against the expected 530 ms; the test is from a82j3n/r80svs, not yp3c09.

## Principle
Drive the blink by sending cursor blink messages, never by wall-clock sleeps; no retries to hide a flake.

## Relevant details
huginn/internal/tui/a82j3n_test.go, r80svs_test.go: find any time.Sleep or real tick in the blink tests and replace with explicit blink msgs. Same check for TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade (Windows, huginn/internal/cli), seen once in yp3c09 CI 44522707343 attempt 1.

## Depends on

## Acceptance
- [ ] The blink tests use no wall-clock waits (grep in a note) and pass 50 runs in a row niced (-count=50)
- [ ] TestBackgroundSyncSameTaskAppendsConvergeAfterUpgrade failure on Windows is reproduced or explained in a note

## Notes


Check 1 evidence (branch 6zhyq2, blink commit 3342fe76 on 57fce63b). BEFORE, grep -n 'realTick\|time.Since\|time.Now\|BlinkSpeed = time.Millisecond' over a82j3n/r80svs/911j70/qpp30y tests at 57fce63b: a82j3n_test.go:52 'func realTick(t *testing.T, cmd tea.Cmd) (tea.Msg, time.Duration)', :54 'start := time.Now()', :56 'took := time.Since(start)', :111 and :122 'realTick(t, cmd)', r80svs_test.go:140 'msg, took := realTick(t, cmd)', 911j70_test.go:179 and qpp30y_test.go:112 'c.BlinkSpeed = time.Millisecond' (a real 1 ms timer). AFTER: realTick is gone; the two real-tick tests run inside synctest.Test and bubbleTick asserts exactly 530ms on the bubble's clock (a82j3n_test.go:61 'took != 530*time.Millisecond'); vBlinkTick/blinkTick use 'c.BlinkSpeed = 0' (deadline already past, no timer). Mutation: vCursor.BlinkSpeed=497ms in tui.go -> 'a82j3n_test.go:120: the blink tick came after 497ms of the bubble's clock, want 530ms' (reverted). -count=50: 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=50 -run ^(TestViewKeysBlinkAfterV|TestViewLineAfterVBlinksTheChoices|TestNoCursorCellAfterView|TestViewKeysSteadyAfterSecondKey|TestPickedVIsUnderlined|TestViewBlinkGolden|TestViewLineNoColorKeepsCues|TestViewCursorBlinksAfterV|TestViewCursorGolden|TestBoxCursorBlinksUnderline|TestBoxGolden)$ ./huginn/internal/tui/' at 3342fe76 -> 'ok  	github.com/veridit/hug/huginn/internal/tui	42.264s'; again with -v: 'ok  	github.com/veridit/hug/huginn/internal/tui	41.344s', 50 PASS lines for TestViewKeysBlinkAfterV, 50 for TestViewLineAfterVBlinksTheChoices, 550 top-level PASS, 0 FAIL.
## History
- 2026-11-20T00:04 todo -> doing/active (take)
