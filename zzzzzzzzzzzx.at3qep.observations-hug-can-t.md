---
id: at3qep
title: "Observations hug can't settle become questions to the owner"
status: todo
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "47sxn2"
---

## Goal
Lint observes states that need a person or agent to settle them and turns each into a question on the ticket, addressed to its owner, with the observation and a fix command: (a) landed on main but checks unticked; (b) doing with no note, check or commit from its taker for 2 hours; (c) shipped in a release but not done; (d) a commit on main naming a ticket that is todo; (e) a doing ticket whose recorded branch is gone. Questions reach the agent at its next break (x325pp) and in show/next, and are answered in place (3jtfw9). Answering or fixing clears them.

## Purpose
Epic 47sxn2, level 2. Today Jørgen noticed these by hand: tickets shipped with 0 checks ticked, 5 tickets in doing for one agent, tickets nobody was on.

## Principle
Ask, never guess. Each question names what was observed, from what, and one fix command. One question per ticket at a time, the most important first, so nobody is flooded. Never in the listing beyond the ! marker (11:01 ruling).

## Relevant details
Uses level 1 facts (landed, shipped lines) from the git deduction ticket. Activity = notes, checks, commits naming the ticket, by its taker. Delivery: x325pp nudges, board cue. Answer: 3jtfw9.

## Depends on

## Acceptance
- [ ] each of (a)-(e) on a fixture board produces one question with observation and fix line; fixing it clears it (test per case)
- [ ] the owner's agent gets the question at its next break and in hug task show (test)
- [ ] on hug's own board today, the questions raised match the drift Jørgen saw (0hkaa5, m1c28r, t709he would each have been asked), shown before and after

## Notes
