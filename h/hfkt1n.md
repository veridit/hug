---
id: hfkt1n
title: "TestTmuxReload: 'reload line stayed' under load"
state: "doing"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzzr"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:38"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
TestTmuxReload passes under machine load as it does alone: name why 'the reload line stayed after leaving the task' fires at load ~120 and close it at the cause

## Purpose
A red TUI run at load ~120 (duckling's full TUI run, ~/.jcode/scratch/hjbfzd-tui3.log) costs a rerun and trust in local suites; it passes alone and in CI

## Principle
Cause before fix; no sleep and no longer timeout unless the cause is a real wait; keep to the test and the reload path; DM macaque before touching tui.go/render.go (tz9z14, rxr4z8)

## Relevant details
Verbatim: 'reload_tmux_test.go:216: the reload line stayed after leaving the task:' with line 1 'HUG · repo · list   15/60 tasks · ! 2 broken · 5 advice · ⠹ refreshing' (the list) but lower lines still the detail's (checks, NOTES, HISTORY, 'reloaded hug v0.0.2', key bar 'j/k scroll · edit · done · wait · x drop ...'). The wait at reload_tmux_test.go:207-211 stops as soon as line 1 has no ›.

## Depends on

## Acceptance
- [x] The cause is named in a note with the failing capture or a reproduction
- [ ] A reproduction under a CPU hog fails first, then passes -count=50 niced under the same hog after the fix
- [ ] No sleep added and no timeout raised unless the cause is a real wait, said in the note

## Notes


Cause (tigress): a capture of a half-drawn frame. After Esc, hug repaints the whole screen in one frame, top line first (bubbletea standard_renderer flush: lines in order, the key bar last). Under load the tmux server reads that output in pieces, and capture-pane can run between them. The test's wait (reload_tmux_test.go:207-211) stops at the first capture whose line 1 has no '›'. Then line 1 is the list, but rows below are still the detail's, including 'reloaded hug v0.0.2' and the detail key bar. The assertion at :215 reads that partial frame. Reproduced without hug changes: SIGSTOP the tmux server right after Esc, queue capture-pane, SIGCONT. Probe5 got 1 torn frame in 125 captures, with line 1 'HUG · repo · list   15/60 tasks · ! 2 broken · 5 advice', rows 21-37 blank but row 33 '                                                 xture.' (detail text not yet overwritten), row 38 'reloaded hug v0.0.2', and row 40 'j/k scroll · edit · done ...', the shape of duckling's hjbfzd-tui3.log. Two things show this is not a hug bug: the next capture is whole, and stopping hug instead (probe4, 114 captures) gave 0 torn frames. The wait is real (for the frame to finish arriving), so the fix is the condition, not the time. A related load flake in the same test: :195 'not the same place after the reload' (hog2.log 3 of 4 FAIL; hog3 2 of 14), where the 'after' capture's line 1 has '· ⠸ refreshing' while the reloaded hug's first load runs past spinAfter (200 ms).
check 1: Cause in the note of 2026-11-18: TestTmuxReload's wait after Esc accepts a half-drawn frame. Reproduced: SIGSTOP the tmux server after Esc, queue capture-pane, SIGCONT; 1 torn capture in 125 (hfkt1n-ev/srv1.log), line 1 the list header, row 38 'reloaded hug v0.0.2', row 40 'j/k scroll · edit · done ...', the shape of hjbfzd-tui3.log. Stopping hug instead: 0 torn in 114 (stop1.log, hog4.log). Both frames are now testdata in reloadflake d22f1e9b (testdata/reload/torn-*.txt).
## History
- 2026-11-18T22:38 todo -> doing/active (take)
