---
id: pf2d0r
title: "Release v0.7.118: review starts by itself on a worker's last tick"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-27
updated: 2026-11-27
stage: "asked"
doing_with: "rvb3j5"
asked_what: "v0.7.118 is installed. When you write a request or answer on a task, it now goes first to the agent doing that task. If that agent is not running, today you get it twice over: I (the coordinator) get it now, and the agent gets it again when it next starts. Keep that (reply 'both'), or should I only hear that it is waiting, and the agent alone act on it (reply 'agent')?"
asked_for: "jhf"
asked_since: "2026-11-27T08:28"
asked_by: "crab"
asked_check: "1"
doing_since: "2026-11-27T08:28"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Jørgen decides what happens to a request or answer when the agent doing the task is not live.

## Purpose
One release, one question: 5yh7hr shipped alone.

## Principle
One question per shipped release, asked here only; a follow-up task implements the choice if it differs from today.

## Details
Shipped: 5yh7hr (flamingo b762d3ff..1994a83a, CI 45955815297 8/8; main b5e4450c; release.yml 45957504080; brew hug v0.7.118). What changed: a worker's last own tick moves its task to review by the coordinator and tells the coordinator's session one line; requests and answers go first to the agent doing the task; when it is not live, the coordinator gets it now (saying so) AND the agent's next session gets it again. Option A (today): both. Option B: the coordinator is only told it is waiting; the item goes to the agent alone. Open, defaults kept (not asked): a [people] name is never treated as an agent; the reviewer is always the coordinator (no named-reviewer field); the coordinator's own task closes with done.

## Depends on

## Acceptance
- [ ] Jørgen chooses: an item for an agent that is not live goes to both (as today) or waits for that agent alone

## Questions
- 2026-11-27T08:28 question 1 by crab to jhf (check 1):
  v0.7.118 is installed. When you write a request or answer on a task, it now goes first to the agent doing that task. If that agent is not running, today you get it twice over: I (the coordinator) get it now, and the agent gets it again when it next starts. Keep that (reply 'both'), or should I only hear that it is waiting, and the agent alone act on it (reply 'agent')?

## Notes


## History
- 2026-11-27T08:28 todo -> todo/design (ask)
- 2026-11-27T08:28 todo/design -> doing/asked (take with rvb3j5, 1 open question(s) kept); was design_asked_by: crab; design_for: jhf; design_question: v0.7.118 is installed. When you write a request or answer on a task, it now goes first to the agent doing that task. If that agent is not running, today you get it twice over: I (the coordinator) get it now, and the agent gets it again when it next starts. Keep that (reply 'both'), or should I only hear that it is waiting, and the agent alone act on it (reply 'agent')?; design_since: 2026-11-27T08:28
