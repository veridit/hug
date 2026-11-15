---
id: n7jbmx
title: "Recall on the agent's tool calls (post_tool_feedback)"
status: todo
type: spike
priority: ""
assignee: []
labels: ["recall"]
created: 2026-11-15
updated: 2026-11-15
---

## Goal
Memories reach the agent mid-turn when what it is doing (files touched, commands run) matches them, with a stricter bar than message recall

## Depends on

## Acceptance
- [ ] offline set: tool calls from past jcode sessions (name, input, touched paths) with the memories recall would inject for them, judged blind with a stricter rubric: would the agent act differently, and is it not already in front of it
- [ ] trigger uses CASE paths/tools from stored memory types and when-then fields, not only text similarity
- [ ] hook: hug mem hook jcode-tool reads JCODE_HOOK_TOOL_NAME/TOUCHED_PATHS, prints at most 1-2 memories, under 200 ms p95, silent when nothing passes
- [ ] ship only if judged-useful injections per 100 tool calls rises with irrelevant ones under a stated cap

## Notes
