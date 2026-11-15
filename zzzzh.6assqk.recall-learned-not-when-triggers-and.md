---
id: 6assqk
title: "Recall: learned 'not when' triggers and a local candidate rule"
status: todo
type: spike
priority: ""
assignee: []
labels: ["recall"]
created: 2026-11-15
updated: 2026-11-15
---

## Goal
Recall shows more of the relevant memories search already finds, and stops showing a memory in situations like those where it was judged irrelevant before, all local and free

## Depends on

## Acceptance
- [ ] candidate rule (top 20 ungated, own project, under 2000 chars, vector >= 0.64) measured against shipped recall on the 277 frozen and 147 fresh messages
- [ ] learned veto: a memory is skipped when its closest earlier judged-irrelevant situation is clearly closer than its closest judged-relevant one; source of labels in production defined (memory_feedback, agent forget, dream vetting)
- [ ] ship only if relevant/msg rises and irrelevant/msg does not, CI-clear, on the fresh set
- [ ] written 'applies when' triggers and recall on tool calls (post_tool_feedback) tracked as follow-ups

## Notes
