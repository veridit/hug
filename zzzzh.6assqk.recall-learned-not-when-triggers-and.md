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

Offline 2026-11-15, 277 frozen messages, earlier-session history only (55 sessions), blind judge. Long: rule vs current rel +0.48 [+0.32,+0.65], irr +0.30 [-0.01,+0.60]; learned veto vs rule: irr -0.50 [-0.71,-0.35], rel -0.12 [-0.22,-0.05]; learned vs current rel +0.36 [+0.19,+0.53], irr -0.20 [-0.54,+0.09]. Short: rule vs current rel +0.28 [+0.13,+0.41], irr +0.02 [-0.19,+0.25]; veto barely acts (19% of candidates have history). Script muninn/scripts/learned.py.
