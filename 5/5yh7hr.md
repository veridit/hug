---
id: 5yh7hr
title: "Review starts by itself when a worker ticks its last check"
state: "doing"
type: feature
priority: "high"
assignee: ["flamingo"]
labels: []
group: ""
rank: "0000001"
created: 2026-11-26
updated: 2026-11-26
parent: "erv680"
doing_since: "2026-11-27T06:21"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] ~~hug task review ID --by crab hands crab's live session one line naming the task and the reviewer step (test)~~ (not relevant: replaced by check 4: review starts automatically on the last tick, not by a command the worker must run, crab, 2026-11-26)
- [ ] History records the review as delivered or not delivered with the reason (test)
- [ ] ~~Ticking a check sends nothing (test)~~ (not relevant: replaced by check 7: only the last tick notifies, crab, 2026-11-26)
- [ ] A worker ticking its last own check moves the task to doing/review with review_by the coordinator, without any extra command (test)
- [ ] A task whose assignee is the coordinator does not enter review by itself (test)
- [ ] The coordinator's live session gets one line naming the task when it enters review (test)
- [ ] Ticks before the last send nothing (test)
- [ ] A request or answer on a task an agent is doing is pushed to that agent's live session; the coordinator gets it only when that agent has no live session, and History says which (test)
- [ ] An assigned agent with no live session gets it at its next session start, and the coordinator is told it was not delivered live (test)

## Notes

Jørgen 2026-11-26 22:01: route requests and answers to the assigned agent first. Today Deliver always pushes to the coordinator (huginn/internal/cli/deliver.go:152-176, 'to you as the board coordinator'); the assignee sees it only as a board cue at its next tool call or message (muninn/internal/recall/boardcue.go:861-875 requestFor), so an idle or stopped worker never hears it.


## History
- 2026-11-27T06:21 todo -> doing/active (take)
