---
id: hfkt1n
title: "TestTmuxReload: 'reload line stayed' under load"
state: "done"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzzl"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T01:10"
done_release: "v0.7.73"
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
- [x] A reproduction under a CPU hog fails first, then passes -count=50 niced under the same hog after the fix
- [x] No sleep added and no timeout raised unless the cause is a real wait, said in the note

## Notes


Cause (tigress): a capture of a half-drawn frame. After Esc, hug repaints the whole screen in one frame, top line first (bubbletea standard_renderer flush: lines in order, the key bar last). Under load the tmux server reads that output in pieces, and capture-pane can run between them. The test's wait (reload_tmux_test.go:207-211) stops at the first capture whose line 1 has no '›'. Then line 1 is the list, but rows below are still the detail's, including 'reloaded hug v0.0.2' and the detail key bar. The assertion at :215 reads that partial frame. Reproduced without hug changes: SIGSTOP the tmux server right after Esc, queue capture-pane, SIGCONT. Probe5 got 1 torn frame in 125 captures, with line 1 'HUG · repo · list   15/60 tasks · ! 2 broken · 5 advice', rows 21-37 blank but row 33 '                                                 xture.' (detail text not yet overwritten), row 38 'reloaded hug v0.0.2', and row 40 'j/k scroll · edit · done ...', the shape of duckling's hjbfzd-tui3.log. Two things show this is not a hug bug: the next capture is whole, and stopping hug instead (probe4, 114 captures) gave 0 torn frames. The wait is real (for the frame to finish arriving), so the fix is the condition, not the time. A related load flake in the same test: :195 'not the same place after the reload' (hog2.log 3 of 4 FAIL; hog3 2 of 14), where the 'after' capture's line 1 has '· ⠸ refreshing' while the reloaded hug's first load runs past spinAfter (200 ms).
check 1: Cause in the note of 2026-11-18: TestTmuxReload's wait after Esc accepts a half-drawn frame. Reproduced: SIGSTOP the tmux server after Esc, queue capture-pane, SIGCONT; 1 torn capture in 125 (hfkt1n-ev/srv1.log), line 1 the list header, row 38 'reloaded hug v0.0.2', row 40 'j/k scroll · edit · done ...', the shape of hjbfzd-tui3.log. Stopping hug instead: 0 torn in 114 (stop1.log, hog4.log). Both frames are now testdata in reloadflake d22f1e9b (testdata/reload/torn-*.txt).
Fix on branch reloadflake: 784b50e5 (failing test) + 80410278 (fix, CHANGELOG). Test-only; tui.go/render.go untouched. Two causes, both real waits made explicit, no sleep added, no timeout raised: (1) wait after Esc now wants the list's key bar 'j/k move' as the last line too (half-drawn frame, hjbfzd :216); (2) before/offered/after wait for no 'refreshing' in line 1 (a load running past spinAfter; pre-fix hog run prefix-hog.log: 2x ':203 not the same place after the reload'; fix v1 run1/fixed-1.log: ':155 line 1 changed before R' with '⠋ refreshing'). TestReloadFramesSettle at 784b50e5: 'after-refreshing.txt: true, want false', 'torn-hjbfzd.txt: true, want false', 'torn-after-esc.txt: true, want false', 'offered-refreshing.txt: true, want false'; at 80410278: ok. Mutations: dropping the refreshing clause fails after-refreshing.txt; neutralising 'j/k move' fails both torn frames. Fixed TestTmuxReload under 16 hogs (loadavg '{ 256,28 271,47 269,33 }'), guard bypassed, -count=50 niced in 5 chunks: 'PASS 48 FAIL 2'. 0 reload-path failures; the 2 are the shared harness under that load: 'reload_tmux_test.go:86: foreground tmux server did not become ready' (startTmuxServer 5 s ready deadline) and 'tmux_test.go:82: tmux server did not exit after explicit cleanup' (3 s), both outside the reload path, at twice loadGuard's 120. Pre-fix also showed ':141 screen never showed "HUG · repo · list ›"' (keys jj+Enter not applied in 5 s), not seen in the 50. Evidence: ~/.jcode/scratch/hfkt1n-ev/REPORT.md.
check 3: 80410278 adds no sleep and raises no timeout: git diff 4e287c1a..80410278 changes only reload_tmux_test.go, testdata/reload/*, CHANGELOG.md. The new conditions are real waits under the existing deadlines (waitFor keeps 5 s / 15 s): for the whole frame (list key bar 'j/k move') and for the load to end (no 'refreshing'), said in the note.
Pushed once: hfkt1n and ci/hfkt1n at a943b528 (1340c90a failing test, a943b528 fix), rebased on origin/main 356fb240 (v0.7.72). CI 44422776917: 'completed success', all 8 jobs success (test ubuntu/macos/windows, windows-cli shards, bench, secrets). After the rebase, 1340c90a: TestReloadFramesSettle FAIL; a943b528: 'ok github.com/veridit/hug/huginn/internal/tui 75.874s' (-short). Check 2 left open for crab: the hog -count=50 was 'PASS 48 FAIL 2', and both failures are the tmux server start/stop deadlines in startTmuxServer, not the reload path.
check 2: crab 23:52 ruling: met for the reload path. Pre-fix 5/8 under 16 hogs; post-fix 48/50 with 0 reload-path failures; the 2 failures are tmux harness deadlines (tmux_test.go:82, reload_tmux_test.go:86), split into their own task. CI 44422776917 8/8.
landed 28de94c on main 2026-11-19: tui: failing test, TestTmuxReload's waits accept half-drawn and loading frames…
landed bc217cb on main 2026-11-19: tui: TestTmuxReload compares only whole frames with no load running (hfkt1n)
shipped in v0.7.73 (28de94c)
## History
- 2026-11-18T22:38 todo -> doing/active (take)
- 2026-11-19T00:52 doing/active -> doing/waiting (wait)
- 2026-11-19T01:10 doing/waiting -> done (done, by release v0.7.73: every check holds); was doing_since: 2026-11-18T22:38; waiting_asked_by: tigress; waiting_on: landing: stacked as ci/hfkt1n-rebased bc217cb3 on rxr4z8; crab says when it lands; waiting_since: 2026-11-19T00:52
