---
id: typq1e
title: "Capture stores no agent status at all (follow-up to ez50rb)"
state: todo
type: bug
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
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
