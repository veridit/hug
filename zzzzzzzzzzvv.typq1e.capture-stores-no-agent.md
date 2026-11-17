---
id: typq1e
title: "Capture stores no agent status at all (follow-up to ez50rb)"
state: "doing"
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T20:41"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Capture never stores a memory that names a swarm agent or its status: from 0.19 and 0.25 such memories per agent transcript (after ez50rb's post-check) to 0, with durable memories unchanged.

## Purpose
ez50rb check 3 says 'never stores agent status'; the shipped post-check (4cf6414) cut it from 0.31/0.42 to 0.19/0.25 per transcript in the blind eval (muninn/docs/CAPTURE4.md), not to 0. The prompt rule that went further lost durable memories and was not shipped. ez50rb was closed with this remainder; this ticket owns it.

## Principle
Recall changes ship only with a measured, blind-judged gain on replayed data (AGENTS.md). Durable memories must not drop. Prefer a deterministic check (agent names from the harness session list) over a prompt rule.

## Relevant details
Eval sets and method: muninn/docs/CAPTURE4.md. Post-check: commit 4cf6414. Agent names: the harness session list, as in ez50rb's agents matcher.

## Depends on

## Acceptance
- [ ] on the CAPTURE4 sets, memories naming an agent are 0 per transcript, durable memories unchanged (blind-judged)
- [ ] the same holds on a fresh set fixed before tuning

## Notes


hatchling 19:44, fresh set fixed before any tuning (check 2): muninn/scripts/capture4_sample.py SEED=1794 EXCLUDE=(CAPTURE4 sample 1 + fresh, 120 sessions) CONTROLS=20 -> 53 items: 40 agent, 13 control (only 13 control sessions left), capture runs 2026-11-06T11:21 .. 2026-11-17T17:18, 0 sessions shared with CAPTURE4. sha256 ccb8f3f5699bcec14c583970cdf802d44b8dc13e6f0dc705fecca0ffbfd4e13a. No output of it read yet.
## History
- 2026-11-17T20:41 todo -> doing/active (take)
