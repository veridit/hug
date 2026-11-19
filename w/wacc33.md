---
id: wacc33
title: "TUI tests leave tmux servers running after they finish"
state: "doing"
type: bug
priority: ""
assignee: ["bat"]
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T21:05"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every tmux server a test starts is gone when the test ends, pass or fail.

## Purpose
crab 12:01Z: found 3 orphaned tmux servers (parent launchd): huginn-reload-90820 (since 19 Nov 02:14, running a v0.0.2 'hug tui' in a scratch repo), huginn-golden-99210 (since 17 Nov 22:45), huginn-golden-62963 (since 19 Nov 01:11); killed. They hold memory and a TUI process on a shared machine.

## Principle
t.Cleanup kills the server even on t.Fatal or a timeout; a test fails if a server it started is still alive at cleanup.

## Relevant details
Grep huginn/internal/tui for tmux -D -L huginn-golden / huginn-reload; check whether killed tests (go test -timeout, CI cancel) skip cleanup, and whether -D detached servers survive their parent.

## Depends on

## Acceptance
- [x] A test that fails or times out mid-run leaves no tmux server (test that runs a failing child and checks pgrep)
- [x] After a full niced go test ./... no huginn-* tmux server remains (note with pgrep output)

## Notes


check 1: fix 0dd8da47 on branch wacc33: TestTmuxServerGoneWhenTestEnds (huginn/internal/tui/tmuxorphan_test.go) runs a child test that starts a server via startTmuxServer and ends by t.Fatal, subtest panic, -test.timeout=15s, SIGKILL. On origin/main 3c81735d: '--- FAIL: TestTmuxServerGoneWhenTestEnds/panic (10.38s) tmuxorphan_test.go:185: panic: tmux server 19447 (-L huginn-wacc33-19446) still runs 10s after its test process ended (exit status 2)', same for kill (signal: killed) and timeout (exit status 2); fatal passed. After fix: PASS fatal 0.49s, kill 0.38s, panic 0.47s, timeout 15.29s; TestTmuxOrphanReaped PASS (next run SIGTERMs a huginn-* server whose owner pid is gone). Fix: watchdog sh holds the server and kills it when the test's stdin pipe closes; cleanup is t.Cleanup too; TestMain reaps orphans (internal/tmuxtest).
check 2: 62889dc4 (branch wacc33): GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./... exit=0 at 2026-11-19T20:58:12Z, huginn/internal/tui ok 154.079s, muninn/internal/tui ok 19.251s, internal/tmuxtest ok. Then pgrep -fl 'tmux -D -L huginn' printed nothing, exit=1 (log: $JCODE_SCRATCH_DIR/wacc33/fullrun2.txt).
pgrep after full niced run on 62889dc4 (2026-11-19T20:58:12Z): `pgrep -fl 'tmux -D -L huginn'` -> empty output, exit 1. The first full run on 0dd8da47 also left none (empty, exit 1) but failed TestEveryHUGNameIsKnown (HUG_TEST_TMUX_CHILD not on env.Known), fixed in 62889dc4; CI 44612421566 on 0dd8da47 cancelled for that.
CI 44615073708 on c1478721 (branch ci/wacc33): success, 8/8 jobs (secrets, bench, test ubuntu/macos/windows, test-windows-cli sync/rest/shards). c1478721 = 62889dc4 + TestLabelOwner skips the signal-0 liveness check on Windows (CI 44613272670 on 62889dc4: windows 'tmuxtest_test.go:40: Label "huginn-golden-8488": owner 8488 is not this live process'; ubuntu and macos success). Fix commits: 0dd8da47, 62889dc4, c1478721 on wacc33, not merged to main.
## History
- 2026-11-19T21:05 todo -> doing/active (take)
