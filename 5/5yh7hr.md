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
- [x] History records the review as delivered or not delivered with the reason (test)
- [ ] ~~Ticking a check sends nothing (test)~~ (not relevant: replaced by check 7: only the last tick notifies, crab, 2026-11-26)
- [x] A worker ticking its last own check moves the task to doing/review with review_by the coordinator, without any extra command (test)
- [x] A task whose assignee is the coordinator does not enter review by itself (test)
- [x] The coordinator's live session gets one line naming the task when it enters review (test)
- [x] Ticks before the last send nothing (test)
- [x] A request or answer on a task an agent is doing is pushed to that agent's live session; the coordinator gets it only when that agent has no live session, and History says which (test)
- [x] An assigned agent with no live session gets it at its next session start, and the coordinator is told it was not delivered live (test)

## Notes

Jørgen 2026-11-26 22:01: route requests and answers to the assigned agent first. Today Deliver always pushes to the coordinator (huginn/internal/cli/deliver.go:152-176, 'to you as the board coordinator'); the assignee sees it only as a board cue at its next tool call or message (muninn/internal/recall/boardcue.go:861-875 requestFor), so an idle or stopped worker never hears it.


mutants (12, all killed; script $JCODE_SCRATCH_DIR/mutate.py, run niced, targeted tests in task, cli, muninn/run): M1 ReviewDue ignores own open checks KILLED (TestReviewDue, TestLastTickStartsReview, TestStepOverATasksLife); M2 coordinator may review own task KILLED (TestReviewDue, TestCoordinatorsTaskNoReview); M3 any ticker starts review KILLED (TestReviewDue, TestCoordinatorsTaskNoReview, TestShippedPersonCheckIsAsked); M4 review not delivered after tick KILLED (TestLastTickStartsReview, TestReviewNotDelivered); M5 doer skipped, always coordinator KILLED (TestRequestGoesToDoerFirst, TestDoerGetsItAtNextSession); M6 coordinator told even when doer live: first SURVIVED (equivalent: coord reset was dead code, removed), re-targeted at the guard, KILLED (TestRequestGoesToDoerFirst asserts no crab line, no crab pending); M7 missed line dropped KILLED (TestRequestGoesToDoerFirst, TestDoerGetsItAtNextSession); M8 OwedTo doer owed only while pending KILLED (TestOwedTo, TestDoerGetsItAtNextSession); M9 OwedTo ignores delivered-to-name KILLED (TestOwedTo); M10 StepOf hand-over removed KILLED (TestStepHandsToReview, TestStepChangesAskedIsNoApproval); M11 cue repeats delivered review KILLED (TestBoardCueSkipsDeliveredReview); M12 shipped task enters review: first SURVIVED, TestStartReview added, KILLED.
check 2: evidence: TestReviewNotDelivered (huginn/internal/cli/autoreview_test.go) asserts History 'review 1 not delivered to crab: crab's jcode session SID is not live in a jcode daemon' and 'no jcode session named crab on this machine', and 'review 1 delivered to crab (SID)' at crab's next session; TestLastTickStartsReview asserts 'review 1 delivered to crab (SID)'. GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -race ./huginn/internal/cli/ -> 'ok  github.com/veridit/hug/huginn/internal/cli 274.495s' (commit e122d180)
check 4: evidence: TestLastTickStartsReview: wolf's check 2 (its last own, check 3 is jhf's) gives stage review, review.by crab, History 'doing/active -> doing/review (review: wolf ticked the last own check)', no other command; TestStartReview (task) and TestReviewDue (13 cases). go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'; ./huginn/internal/task/ -> 'ok  github.com/veridit/hug/huginn/internal/task 31.137s'
check 5: evidence: TestCoordinatorsTaskNoReview: crab's last tick on its own task leaves stage active, next 'hug task done ID'; a tick by otter on seal's task and a board with no coordinator stay active; TestReviewDue cases 'the coordinator ticks its own' and 'the coordinator does it too'. go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'
check 6: evidence: TestLastTickStartsReview: the fake jcode daemon gets exactly 1 message, to crab's live session, one line: 'hug: wolf ticked the last own check of ID (Review by itself) on board ...: it is in review by you. Read it (hug task show ID), then answer: hug task answer ID "looks right" (or what to change) [hug ID review 1 ...]'; review_receipt 'pending, delivered to crab (SID)'. go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'
check 7: evidence: TestLastTickStartsReview: after wolf's first tick the daemon has 0 messages, no review_receipt, stage active, no review_by; after the hand-over a tick by jhf in review adds none (still 1). DeliveryFlow claim 'earlier ticks tell nobody' checked in TestDeliveryInstructionsMatchCode. go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'
check 8: evidence: TestRequestGoesToDoerFirst: wolf live -> receipt 'pending, delivered to wolf (SID)', message 'to you as the agent doing it', History 'request 1 delivered to wolf', nothing to crab or crab's pending; wolf not live -> receipt 'pending, delivered to crab (SID); not delivered live to wolf, who is doing it: REASON', History has both 'request 2 not delivered to wolf: REASON' and 'request 2 delivered to crab'; an answer goes to wolf too; a todo task's request to crab. go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'
check 9: evidence: TestDoerGetsItAtNextSession: no wolf session -> crab's message has 'Not delivered live to wolf, who is doing it (no jcode session named wolf on this machine): hug hands it to wolf's next session when it starts.'; wolf's pending holds 1 item; after crab's ack, DeliverPending at wolf's next session delivers 'ID request 1' once ('to you as the agent doing it'), History 'request 1 delivered to wolf (SID)'; a request resolved before wolf starts is not handed over. TestOwedTo covers the rule. go test -race ./huginn/internal/cli/ -> 'ok ... 274.495s'
## History
- 2026-11-27T06:21 todo -> doing/active (take)
