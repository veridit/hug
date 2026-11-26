---
id: 5yh7hr
title: "A worker's review request reaches the coordinator's live session"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "0000001"
created: 2026-11-26
updated: 2026-11-26
---

## Goal
When a worker ticks its last own check, hug moves the task to review by the coordinator and tells that live session; nobody has to remember.

## Purpose
Jørgen 2026-11-26: crab mirrors each worker's steps as todos to follow them. The worker should tick the ticket's checks, ask for review, and crab hears that one event.

## Principle
Automatic, like after-release asking (afterrelease.go): the last worker tick triggers it, not a command the worker must remember. Reviewer is never the worker: the board coordinator by default, a named reviewer if set. One notification per hand-over; ticks before the last send nothing. done stays the reviewer's step.

## Details
Today hug delivers live only requests, answers and outcomes (huginn/internal/cli/deliver.go:27, cli.go:1707-1716); hug task review ID --by NAME (6w0gve) records review_by but nothing reaches that agent, and nothing triggers it. After-release asking (huginn/internal/task/afterrelease.go personChecks) already advances a task when only a person's checks remain; review is the same shape for the step before. Rule: a doing task whose open checks are all a person's or none, ticked by its assignee, enters doing/review with review_by = the board coordinator (hug task coordinator), unless the ticker is the coordinator. Add kind 'review' to Deliver, same delivered / not delivered History line. The reviewer's answer 'looks right' lets the coordinator land, release and then the after-release ask runs as today. hug task coordinator is unset on this board today: set it (crab) as part of this.

## Depends on

## Acceptance
- [ ] hug task review ID --by crab hands crab's live session one line naming the task and the reviewer step (test)
- [ ] History records the review as delivered or not delivered with the reason (test)
- [ ] Ticking a check sends nothing (test)
- [ ] A worker ticking its last own check moves the task to doing/review with review_by the coordinator, without any extra command (test)
- [ ] A task whose assignee is the coordinator does not enter review by itself (test)

## Notes
