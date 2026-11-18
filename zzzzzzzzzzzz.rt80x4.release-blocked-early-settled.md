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
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
## History
- 2026-11-18T09:10 todo -> doing/active (take)
