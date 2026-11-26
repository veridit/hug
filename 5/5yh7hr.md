---
id: 5yh7hr
title: "A worker's review request reaches the coordinator's live session"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-26
updated: 2026-11-26
---

## Goal
When a worker runs hug task review ID --by crab, crab's live session is told at once; the ticket is the only progress record.

## Purpose
Jørgen 2026-11-26: crab mirrors each worker's steps as todos to follow them. The worker should tick the ticket's checks, ask for review, and crab hears that one event.

## Principle
One notification per hand-over (review), not per tick. The worker tracks progress only in the ticket (checks, notes); done stays the reviewer's step.

## Details
Today hug delivers live only requests, answers and outcomes (huginn/internal/cli/deliver.go:27, cli.go:1707-1716); hug task review ID --by NAME (6w0gve) records review_by but nothing reaches that agent. Add kind 'review' to Deliver, aimed at review_by (an agent's session or the board coordinator), with the same delivered / not delivered History line. Spawn prompts then say: tick checks with evidence, then hug task review ID --by crab; no todo list.

## Depends on

## Acceptance
- [ ] hug task review ID --by crab hands crab's live session one line naming the task and the reviewer step (test)
- [ ] History records the review as delivered or not delivered with the reason (test)
- [ ] Ticking a check sends nothing (test)

## Notes
