---
id: mmm8c8
title: "Flaky TestSettledWaitsForSlowChild since the 100 ms settle"
state: todo
type: bug
priority: "high"
assignee: ["wyvern"]
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
TestSettledWaitsForSlowChild passes on every run on every OS, because the code it tests waits for the pass to push by a signal, not by time; the test counts events, never sleeps.

## Purpose
nautilus 2026-11-18 04:25 local: macOS CI failed once with huginn/internal/cli sync_test.go:184 'settled returned before the pass pushed', then passed on rerun. 0sbvnq (1c40426d, wyvern) cut the sync tests' settle from 1.5 s to 100 ms; a test that depends on a settle time is a timing test. Jørgen's rule: never skip a test or raise a limit to hide a flake.

## Principle
Fix the cause: settled must return only after the pass it waits for has pushed, by a signal the pass writes. Do not raise the settle back, do not retry the test, do not skip it.

## Relevant details
Test: huginn/internal/cli/sync_test.go:184 TestSettledWaitsForSlowChild; settle from HUG_TEST_SETTLE (0sbvnq). Prove the fix with a counting stress run (-count=200 under load, or a forced child delay via HUG_TEST_CHILD_DELAY) that fails first on main.

## Depends on

## Acceptance
- [ ] A stress run that fails first on main (N failures of M runs, quoted) passes M of M after the fix
- [ ] No sleep or settle time decides the test's outcome (code review note naming the signal)

## Notes
