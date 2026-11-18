---
id: vk9td0
title: "Decide when dream items are handed out"
state: "doing"
type: spike
priority: "high"
assignee: ["hatchling"]
labels: ["dream"]
created: 2026-11-10
updated: 2026-11-10
doing_since: "2026-11-18T06:31"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Choose session start vs turn end/idle for handing dream items to the agent, with evidence

## Depends on

## Acceptance
- [ ] decision recorded in DESIGN.md with reason

## Notes

Built a first answer (2026-11-12): cue at a break (session start, return after 20 min, wrap-up), at most once per 6 h per session; Response.Break reports which fired. User wants to discuss after trying it.


hatchling 2026-11-18: summary for Jørgen sent to crab, no code. h00cs2 (1151adad) changes when items are made: a waiter runs the pass at least 20 h after the last, once he has been away 30 min and load is below 120 (dreamauto.go QuietFor, DreamEvery, DreamLoadBelow). It does not change the hand-out: still the 11-12 cue on a break turn (start, return after 20 min, wrap-up word), at most once per session per 6 h (recall/dream.go). So the next cue will land on his first message after returning. Live store: 9 items from the one manual pass on 11-12; 5 resolved, 4 handed and all of those resolved, 4 never handed. 27 of 103 sessions cued, the last 11-16 10:57Z. Open for Jørgen: keep the cue on his return, or move it to wrap-up/turn end only.
## History
- 2026-11-18T06:31 todo -> doing/active (take)
