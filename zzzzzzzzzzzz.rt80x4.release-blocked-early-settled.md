---
id: rt80x4
title: "Release blocked by early settled sync return"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
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
