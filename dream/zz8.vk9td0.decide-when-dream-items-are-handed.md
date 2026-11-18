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


## History
- 2026-11-18T06:31 todo -> doing/active (take)
