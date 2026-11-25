---
id: ghycrq
title: "tmux test harness: server start and cleanup deadlines trip under load"
state: "done"
type: bug
priority: "medium"
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzt4"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T03:00"
done_release: "v0.7.75"
---

## Goal
Every tmux-backed TUI test starts and stops its tmux server reliably on a loaded machine: startTmuxServer waits on the server's own readiness and its exit, not on fixed 5 s and 3 s deadlines.

## Purpose
hfkt1n's 50-run campaign under 16 CPU hogs (load ~260) passed 48 of 50 after its fix; both failures were in the harness, not the reload path: 'reload_tmux_test.go:86: foreground tmux server did not become ready' (5 s) and 'tmux_test.go:82: tmux server did not exit after explicit cleanup' (3 s). Every tmux test shares this harness, so each such flake costs a rerun.

## Principle
Find the cause under forced load first (as ad62aa, mq5ger, hfkt1n). A deadline is fine only if what it waits for is real; no raised timeout or retry without a named cause.

## Relevant details
Evidence: /Users/jhf/.jcode/scratch/hfkt1n-ev/REPORT.md (campaign logs). Harness: huginn/internal/tui/tmux_test.go:82, reload_tmux_test.go:86, startTmuxServer. Run niced and bounded (load is often 200+).

## Depends on

## Acceptance
- [x] both failures reproduced under a CPU hog with their verbatim lines and the cause named with file:line
- [x] fixed at the cause: TestTmuxReload and TestTmuxGolden pass -count=50 niced under the same hog with 0 harness failures

## Notes


Cause (tigress, 2026-11-19): startTmuxServer times tmux CLIENT processes, and under load starting any process can take seconds. Readiness (tmux_test.go:82-91 on main 356fb240) runs 'tmux show-options' with a 1 s context (:84) inside a 5 s deadline (:82). Cleanup runs 'tmux kill-server' with a 5 s context (:65) and then gives the server 3 s to exit (:76). If the kill-server client is killed at 5 s, the server never gets SIGTERM, and the 3 s wait fails with 'tmux server did not exit after explicit cleanup' (:79). Measured under 16 hogs (spawn1.log, 251 runs each): /usr/bin/true max 5.142s (17 over 1 s), /bin/echo max 11.561s, tmux -V max 9.731s, tmux show-options max 4.949s. A process already running (gap meter, nice 19, 240 s) never woke more than 174ms late: what is slow is starting a new process, not the server or the CPU. Harness-only probe, 467 start/cleanup cycles with a busy pane (loop2.log): 4x readiness past 5 s ('ready-after=6.5s last-attempt=6.468s attempts=1'), 1x 'kill-server=8.613s ... exit-after=0s orig=KILL-CLIENT-TIMEOUT'. Each one is the old deadline failing on a server that was fine: it answered, and it exited 0s after its kill-server finally ran. tmux's kill-server is kill(getpid(), SIGTERM) (cmd-kill-server.c:58), and server_signal treats SIGTERM the same way (server.c:438). So cleanup can signal the server process directly, and readiness can connect to its socket from the test process: neither needs a new process.
check 1: Both reproduced under 16 hogs, harness-only probe, 467 cycles (ghycrq-ev/loop2.log): 4x readiness past 5 s, e.g. 'PROBE start ready-after=6.5s last-attempt=6.468s attempts=1 ... orig=START-DEADLINE' (= 'did not become ready', tmux_test.go:93 via the 1 s client context at :84 and the 5 s deadline at :82); 1x 'PROBE cleanup kill-server=8.613s err=<nil> out="" exit-after=0s exit=<nil> orig=KILL-CLIENT-TIMEOUT' (= 'did not exit after explicit cleanup', :79: the 5 s context at :65 kills the kill-server client before it signals). Cause: the deadlines time tmux client process starts; under the hog /bin/echo took up to 11.561s and tmux -V up to 9.731s (spawn1.log), while a running process woke at most 174ms late (gap-loop2.log).
crab 00:39: tigress's session ended (status failed) mid-fix. Her uncommitted harness change (tmux_test.go +48/-23, new tmuxserver_test.go failing-first test) is saved as branch wip/ghycrq 57c2f0cb on 356fb240, untested. Evidence in /Users/jhf/.jcode/scratch/ghycrq-ev (spawn1.log, loop2.log). Next owner continues from it.
Fix on ci/ghycrq 2f7a91f8 (7e08da57 failing test, 2f7a91f8 fix + CHANGELOG) on origin/main bf170d46. TestTmuxServerSlowStarts (tmux on PATH that sleeps 6 s before exec): on main's harness 'tmux_test.go:79: tmux server did not exit after explicit cleanup' + 'tmuxserver_test.go:34: foreground tmux server did not become ready' (ghycrq-ev/ff-main.log); with fix '--- PASS: TestTmuxServerSlowStarts (6.05s)' (ff-fix.log). Mutation: a kill-server client back in cleanup -> 'panic: test timed out after 2m0s' (mut-cleanup.log; server never gets SIGTERM). Fix: readiness = net.Dial on the server socket from the test; cleanup = SIGTERM to the server (what kill-server does) then Wait. No deadline added; go test -timeout ends a real hang. CI 44431241435 queued. Hog campaign for check 2 running: ghycrq-ev/c2-sum.txt.
check 2: Fixed harness (fix.test from 2f7a91f8, guard bypassed), niced, one go test process at a time, 60 runs in total: 0 FAIL, 0 harness lines (grep for 'did not become ready|did not exit after|tmux server cleanup|still waiting'). ghycrq-ev/c2-1.log: TestTmuxReload 10, TestTmuxGolden 10. c2-2.log: Reload 10, Golden 10. Both under 16 spinners at nice 0, load '{ 863,88 693,28 530,72 }' and '{ 706,28 746,99 688,38 }'; the hog was then stopped on crab's limit. w-1.log: Reload 10, Golden 10, under 8 spinners at nice 19, at most 15 min, load '{ 650,50 673,23 674,37 }'. Totals: TestTmuxReload 30, TestTmuxGolden 30. Before the fix, the harness alone under the hog failed 5 of 467 cycles (loop2.log). pgrep shows no ghycrq process left. CI 44431241435 on ci/ghycrq 2f7a91f8: success, 9 of 9 jobs.
landed bfe4b2c on main 2026-11-19: tui: failing test, the tmux harness fails a working server when processes start…
landed cd8b87f on main 2026-11-19: tui: the tmux harness waits on its server, not on tmux clients (ghycrq)
shipped in v0.7.75 (bfe4b2c)
crab: landed on main as cd8b87ff (stack tip fbe3a412, CI 44438198799 success 8/8); released in v0.7.75.
Reflection (dismissed, 2026-11-19, crab): nothing (coordinator closing on landed CI evidence)
## History
- 2026-11-19T00:52 todo -> doing/active (take)
- 2026-11-19T02:23 doing/active -> doing/waiting (wait)
- 2026-11-19T03:00 doing/waiting -> done (done, by release v0.7.75: every check holds); was doing_since: 2026-11-19T00:52; waiting_asked_by: tigress; waiting_on: landing: crab restacks ci/ghycrq 2f7a91f8 after z25pb2 (fac02994) lands; waiting_since: 2026-11-19T02:23
