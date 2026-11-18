---
id: 6ep7y2
title: "Windows CI: a background syncer held the queue 30 s and no pass ran"
state: "doing"
type: feature
priority: ""
assignee: ["wyvern"]
labels: []
created: 2026-11-18
updated: 2026-11-18
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-18T07:20"
---

## Goal
When a test's settled() times out, the pass log and the failure say which lock each syncer waited on and for how long, and the cause of TestBackgroundSyncBurstIsBatched's stall on Windows (CI 44237364634) is found and fixed, with a counting test that fails first.

## Purpose
A stuck queue means commits that never push; on CI it fails the windows-cli job (83 s). The current diagnostics (queued/busy/pending/turn) cannot tell a syncer waiting on the run lock from one settling or a pass that never ends.

## Principle
Measure before changing: add diagnostics, reproduce on a ci/ branch on Windows, then fix with a counting test. Do not paper over it by turning off throttling in tests.

## Relevant details
CI 44237364634 (a13d7c74), test-windows-cli: sync_test.go:223 'background pass still running after 30 s: queued true, busy true, pending false, turn held false'. Pass log: two passes at 04:31:24 and 04:31:28 (the task new), then none during the 20 notes or the 30 s after. queued true = a syncer past takeQueue (it holds the queue): in settle.wait (bounded by SettleMax 10 s) or in flock.Acquire(run). So a pass holding run did not end in ~90 s, while fetch/push have NetTimeout 60 s and log on timeout. takeQueue (mmm8c8, 4336e2b0) cannot spin: two TryAcquire and one Acquire(.probe), which a prober holds only across a non-blocking TryAcquire. Candidates (unmeasured): the pass at below-normal priority (b6jgpn, a201b2e6) starved by the parallel test load; local git without a deadline; deducePass. Earlier, different shapes: 44230245084 (pending true, ubuntu), 44156864971 (ubuntu). 8 of 8 local runs on macOS pass.

## Depends on

## Acceptance
- [ ] settled's failure reports the run lock held or free, and the pass log has a line when a syncer takes the queue and when it takes the run lock, with pid and time
- [ ] the Windows stall is reproduced on a ci/ branch with those diagnostics, and the cause is named on this ticket with the log lines
- [ ] a counting test that fails first on the cause, and passes with the fix; mutation check noted
- [ ] main CI green on all jobs with the fix (run id)

## Notes


Diagnostics implemented in d51a535c (not on main yet): queue/settle/run progress, separate RunHeld, fetch/turn/push and deduction boundaries with pid and time. Counting tests cover 20 passes (80 ordered lines observable inside the callback) and 20 actual queue-only/run-only/idle lock cycles. Mutations: disable logger -> missing log failure; report Queued as RunHeld -> 'queue-only round 0: queued true run held true'. Both restored. Package -race passed before rebase: gitbranch 15.911s, cli 124.792s; rebased package -race rerunning. Planned one ci/6ep7y2 push, exactly 3 full Windows CLI package runs, preserving its parallel tests. Evidence files in scratch/4vd1mz/6ep7y2-*. Clarification: old Busy includes the queue, so it does not prove run was held. A stuck run holder is a hypothesis, not an observed cause. Earlier different-shape failures do not prove the same cause either.
Waiting on CI 44246286559, the single approved diagnostic branch push (ci/6ep7y2, e5303e30). Windows executes the full CLI package exactly three times in this run. The branch-only workflow commit will not land on main. Rebased -race passed: gitbranch 15.414s and cli 111.007s. No acceptance checks ticked, no stall fix claimed. Background watcher 6689510znj writes scratch/4vd1mz/6ep7y2-ci-result.json and will wake wyvern when done.
Approved bounded experiment COMPLETE, no stall reproduced. CI 44246286559, e5303e30, all 6 jobs success. Windows job 134605988981: command 'go test -count=3 -timeout=40m ./huginn/internal/cli/' (raw job log lines 181-182); verbatim result line 187: 'ok  \tgithub.com/veridit/hug/huginn/internal/cli\t659.850s'. Test step 05:45:53Z..05:56:59Z. This is one successful aggregate invocation covering 3 iterations, not 3 separate PASS transcripts. Phase evidence gap: no -v, successful output suppressed and temp phase files not uploaded, so no individual successful phase traces are recoverable. No FAIL/timeout/queue/run phase lines in the 211-line job log. This does not establish a cause or fix. Exact evidence on disk: ~/.jcode/scratch/4vd1mz/6ep7y2-windows-result.md, 6ep7y2-windows-cli.log (SHA256 66f60d0fd01b3f8634d6751712cd3d2266fb3977c67311cf030deec6a189509f), 6ep7y2-ci-all.log and 6ep7y2-ci-result.json. PROPOSAL ONLY, not executed: one further Windows job on the same branch, 3 full package iterations with -v plus cleanup that emits each clone's phase log before temp deletion, same throttle/parallelism/timeouts. Needs crab's renewed approval. Branch-only workflow e5303e30 stays out of main, permanent diagnostics d51a535c separately reviewable. No checks ticked. 4vd1mz remains parked.
Crab ruling 06:00Z applied: no more diagnostic reruns. Diagnostic-only landing candidate a681224e rebased on e9c65c6f; CI workflow has zero diff from main. Cost/privacy review: new phases now log only fixed text, booleans, attempt/durations, timestamp/pid, not full error strings or remote URLs. Bounded local 100-op probe on M1 Max measured phase write 40-48 us, empty pass with 4 phase messages 456893..582962 ns/op vs 275207 ns/op disabled; these are scratch microbenchmarks, NOT Windows or production latency. Raw probes and caveats: scratch/4vd1mz/6ep7y2-landing-review.md and 6ep7y2-cost-*.txt. Added TestSyncTimeoutFailureRetainsPhases: actual child test fails through settled's shared Fatalf path without sleeping 30s, output contains phase/pid/run-held flags after normal TempDir cleanup removed the phase file. Mutation dropping log bytes fails on missing retained phase and pid; restored, focused test passes. Failure retention is test-only; no broader production retention change. Existing timeout/throttle/parallelism unchanged. Latest-base touched -race now running (891854uflf). After green main, tick check 1 only and park awaiting recurrence, not done; then resume 4vd1mz.
Permanent diagnostics LANDED on main as a681224e40256b5ac3e097faab32d4437bc7348d, parent e9c65c6f verified still latest immediately before push. Latest-base -race passed: gitbranch 25.618s, CLI 169.824s (scratch/4vd1mz/6ep7y2-landing-race.txt). Branch-only workflow e5303e30 is not an ancestor and CI workflow has zero diff. Waiting for normal main CI before check 1. Review and retention mutation evidence: scratch/4vd1mz/6ep7y2-landing-review.md. No cause/fix claim, no more diagnostic reruns, check 2/3/4 remain open. After check 1 evidence I will park this task awaiting recurrence, not done, and resume 4vd1mz.
Crab 06:10Z authorizes resuming isolated 4vd1mz while ordinary main CI 44249085050 runs. Watcher 1001424im4 remains owned here. Parked unresolved awaiting recurrence, with check 1 still awaiting green main; no cause or fix established, checks 2/3/4 remain open.
landed a681224 on main 2026-11-18: sync: retain queue, run and pass phase diagnostics (6ep7y2)
Main CI44249085050 red on TestEveryHUGNameIsKnown on all3 OS: fixture HUG_TEST_SYNC_TIMEOUT_REPORT missing from env.Known (my omission). Windows CLI passed. No check ticked. Registering Test-kind variable alongside existing fixtures; env/gitbranch/CLI latest-base race781888fmel before fix-forward push. Raw6ep7y2-main-ci-failed.log and result.json, exact failure preserved in landing-review.md. No stall reproduction or diagnostic rerun.
Fix-forward1dbcf63a1bca8d4433b1acf9e5c4c9f1d021ea7f landed on verified latest parent1541e271. Registered fixture as Test. Latest-base -race passed env4.900s, gitbranch15.397s, CLI116.298s (6ep7y2-env-fix-race.txt). Waiting on ordinary main CI44250806129 via watcher955583g6r9. Review/privacy/cost and exact race lines retained6ep7y2-landing-review.md. No acceptance ticks yet, no cause/fix claim. Parked unresolved awaiting recurrence, resuming approved isolated4vd1mz while normal CI runs.
## History
- 2026-11-18T06:31 todo -> doing/active (take)
- 2026-11-18T07:10 doing/active -> todo/ready (give back; was assignee: wyvern); was doing_since: 2026-11-18T06:31
- 2026-11-18T07:20 todo/ready -> doing/active (take)
