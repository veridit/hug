---
id: rt80x4
title: "Release blocked by early settled sync return"
state: "doing"
type: feature
priority: ""
assignee: ["wyvern"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T09:10"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "CI run 44266307354, single ordinary CI-only cross-platform gate on00df677f; wyvern owns watcher130570yi3u, main authorization retained by crab"
waiting_since: "2026-11-18T09:22"
waiting_asked_by: "wyvern"
---

## Goal
Prove and fix the early-settled sync recurrence in release CI44264072826 without weakening test or sync guarantees.

## Purpose
v0.7.59 is blocked by TestSettledWaitsForSlowChild returning before its note reaches the remote. Preserve failure phases and remove the demonstrated state-observation or wakeup defect.

## Principle
Read exact release6d994e71 first, deterministic counting/interleaving fails-first and mutation evidence, fix forward only. Do not assume prior stall cause, raise waits, skip tests, rerun release to fish green, rewrite tags, or edit calf files without exact boundary ACK.

## Relevant details
Exact release44264072826 TestSettledWaitsForSlowChild sync_test.go193 reports remote head hug task new atj0m4. Narrow source/test scope proposed to crab and calf before edits. Existing6ep7y2 cause remains unresolved.

## Depends on

## Acceptance
- [ ] Actual early-return failure output retains phase/state logs before cleanup, proven by deterministic failure-output regression.
- [ ] Deterministic counting/interleaving test fails on exact release source, passes with scoped fix, and fails under the corresponding mutation.
- [ ] Latest-base touched race and ordinary approved CI are green, with exact source SHA and compact evidence packet.

## Notes


Initial evidence/plan scratch/rt80x4/initial-plan.md sent to crab+calf08:11:20Z BEFORE any source edits. Exact release6d994e71d0b5e43069be2c9ccb701a5f39c0a537 CI44264072826 job134664162174 Test go test ./..., TestSettledWaitsForSlowChild7.48s, cli29.662s. Full raw failure/JSON in scratch/4vd1mz/release-recurrence-failed.log and release-recurrence-ci.json. Early-return assertion lacks phase/state log while timeout helper retains it. Separate-sample Busy(queue then run) followed by Pending admits candidate handoff-observation gap, HYPOTHESIS ONLY, not proven actual releasecause. Proposed writes exactly4paths: gitbranch/background.go,new background_idle_test.go, cli/sync_test.go,sync_timeout_output_test.go. Need crab approval+actual calf boundary ACK before edits. No waiting-limit/throttle/parallelism changes,newworkers,release reruns/tags/live writes. Final-layoutbenchmarkpaused.
Actual calf four-path ACK08:11/08:12 and crab scope/fix approvals08:11:50/08:14:47 recorded. Narrow commit00df677fdeca8a9df61864c024e8094a1c45d1da on latestmainbaseecf6a268, NOT PUSHED. Deterministic actualBusy/realRun gated with existing Settle.Probe:10/10 fails-first BEFORE production edit, package0.022s, busybeforefalse/pendingafterfalse/queueheldtrue/runheldfalse, passcount0before1after. Proves admissible falseidle, NOT historicalreleasecause. Fix Pending FIRST inBusy; Queued/Nudge5sexpiry unchanged. ZEROcurrentproductionBusy callsites, boundedsettledtestwaiter/diagnostics only; expiredorphan nowvisibleBusytrue intentionally, no newdeadline. ActualNudge missing-executable failedStart test proves retryattempt andpendingcleanup. Four focused tests3xPASS (gitbranch0.022s/cli3.197s). Three compilingmutants killed:removePendingBusy0.009sFAIL, NudgeusesBusy0.008sFAIL, stripactualearlyphase/stateoutput1.097sFAIL, ALL RESTORED. Actualassertion helper nowretainsstate+phase beforecleanup, realFatalfsubprocess proves ordinaryoutputandtempdeletion. Final exact touchedrace9575492wnl running. Evidence scratch/rt80x4/observed-design-and-audit.md, final.diff, raw logs. No mainauthorization/ticks/release/tag/newenv/waits/throttle/parallelism changes; finalbenchmarkpaused.
Actual exact-source race9575492wnl exit0113.7s: gitbranch14.804s, cli109.339s, env4.711s, repocheck1.260s. Candidate00df677fdeca8a9df61864c024e8094a1c45d1da clean latestbaseecf6a268, source and rawlines scratch/rt80x4/final-race-source.txt/final-touched-race.txt. Crab08:21 authorizes ONEfreshCI-only branch afterthese gates. Fetchedbaseunchanged, diffcheck/ancestry/cleantree guards passed. Push099606dl01 created ci/rt80x4-20261118, remoteexact00df verified. OrdinaryCI44266307354 push-event in_progress, exactheadconfirmed, ownwatcher130570yi3u toterminal, no reruns. No main/CHANGELOG/release/tag/newenv/waitchanges. HistoricalreleasecauseNOTestablished, benchmarkpaused. Packet scratch/rt80x4/review-packet.md. No acceptance ticks until approvedgreenmain.
## History
- 2026-11-18T09:10 todo -> doing/active (take)
- 2026-11-18T09:22 doing/active -> doing/waiting (wait)
