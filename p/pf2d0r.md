---
id: pf2d0r
title: "Release v0.7.118: review starts by itself on a worker's last tick"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-27
updated: 2026-11-27
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

## Notes
