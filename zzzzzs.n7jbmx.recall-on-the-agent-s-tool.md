---
id: n7jbmx
title: "Recall on the agent's tool calls (post_tool_feedback)"
state: "doing"
type: spike
priority: ""
assignee: ["hatchling"]
labels: ["recall"]
created: 2026-11-15
updated: 2026-11-15
parent: "d634kr"
doing_since: "2026-11-18T05:32"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Memories reach the agent mid-turn when what it is doing (files touched, commands run) matches them, with a stricter bar than message recall

## Depends on

## Acceptance
- [x] offline set: tool calls from past jcode sessions (name, input, touched paths) with the memories recall would inject for them, judged blind with a stricter rubric: would the agent act differently, and is it not already in front of it
- [ ] trigger uses CASE paths/tools from stored memory types and when-then fields, not only text similarity
- [ ] hook: hug mem hook jcode-tool reads JCODE_HOOK_TOOL_NAME/TOUCHED_PATHS, prints at most 1-2 memories, under 200 ms p95, silent when nothing passes
- [ ] ship only if judged-useful injections per 100 tool calls rises with irrelevant ones under a stated cap

## Notes

Item 1 done (2026-11-16): offline set real/tool (66 tool calls, 7 sessions) replayed through recall4 on the typed snapshot, judged blind with the strict rubric (judge_tool.py, DeepSeek Flash). default: 30.3 useful and 48.5 noise per 100 calls, 44% of calls get an injection. union (all candidates): 259 useful vs 1741 noise per 100. Conclusion: message recall reused for tool calls is net noise (precision ~38%); the hook must not ship on the default gate. Next (item 2): a trigger keyed on CASE paths/commands from rule-type memories and when-then fields, then rejudge; ship cap proposal: noise <= 10 per 100 calls.

Gate sweep 2026-11-16: norecords1 vs default on 66 calls: useful -12.1 [-18.4,-4.4], noise -21.2 [-37.8,-5.0] per 100; precision ~40% either way. Wide pool precision by type: rule 28%, fact 15%, record 8%, preference 4%. Decision recorded in muninn/docs/RECALL-TOOL.md: do not ship a hook on message-recall gates. Item 2 (CASE paths/commands trigger) is the next experiment; it needs memories that name a path/command, which the when-then extraction (jrj1vq) would supply.


file:n7jbmx-plan.txt (3047 bytes, sha256:506d2708)
## History
- 2026-11-18T05:32 todo -> doing/active (take)
