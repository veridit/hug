---
id: 6assqk
title: "Recall: learned 'not when' triggers and a local candidate rule"
state: todo
type: spike
priority: ""
assignee: []
labels: ["recall"]
created: 2026-11-15
updated: 2026-11-15
parent: "d634kr"
group: ""
rank: "zzzzh"
aliases: ["recall-learned-not-when-triggers-and"]
---

## Goal
Recall shows more of the relevant memories search already finds, and stops showing a memory in situations like those where it was judged irrelevant before, all local and free

## Purpose
Jørgen 2026-11-15 (crab session) asked what the LLM filter's big relevance gain consisted of, said recall 'should be local and free' (10:27Z), and asked for 'if x then y' patterns matched by BM25 or similarity (12:36Z). crab's analysis (10:39Z): the gain is almost entirely memories search already finds but today's gates throw away; on short messages the top 20 candidates hold 2.48 relevant memories and recall shows 0.67, and two thirds of the LLM's useful picks sat at ranks 6 to 20. So the problem is choosing, not finding, and this task tries to choose locally.

## Principle
Local and free at recall time: no model call per message (AGENTS.md). Labels for the learned veto come only from earlier judgments, never from the message being scored, so nothing leaks. Ship only under the recall rule: CI-clear gain on the 277 frozen and 147 fresh real messages, judged blind (check 3). A veto may skip a memory in a situation; it never forgets or deletes it.

## Relevant details
Offline 2026-11-15 (notes): the rule adds relevant memories but also noise on long messages (irr +0.30 [-0.01,+0.60]); the learned veto cuts long noise (-0.50 [-0.71,-0.35]) and barely acts on short ones (19% of candidates have history). Labels in production do not exist yet: memory_feedback, forgets and dream vetting give no relevance signal today (memory bgqxqj), and agents essentially never call memory_forget after recall (s6ff5k: 0 of 454 sessions). Warning from s6ff5k check 2 (2026-11-17, offtopic_sim.py): with perfect oracle feedback, per-memory similar-message suppression still cost relevant recall at every threshold, because a memory off-topic for one message is often on-topic for a similar one. Sibling jrj1vq measured the kind rule (kind != record) on the fresh set: long irr -1.98 [-2.44,-1.50]. Script muninn/scripts/learned.py.

## Depends on

## Acceptance
- [ ] candidate rule (top 20 ungated, own project, under 2000 chars, vector >= 0.64) measured against shipped recall on the 277 frozen and 147 fresh messages
- [ ] learned veto: a memory is skipped when its closest earlier judged-irrelevant situation is clearly closer than its closest judged-relevant one; source of labels in production defined (memory_feedback, agent forget, dream vetting)
- [ ] ship only if relevant/msg rises and irrelevant/msg does not, CI-clear, on the fresh set
- [ ] written 'applies when' triggers and recall on tool calls (post_tool_feedback) tracked as follow-ups

## Notes

Offline 2026-11-15, 277 frozen messages, earlier-session history only (55 sessions), blind judge. Long: rule vs current rel +0.48 [+0.32,+0.65], irr +0.30 [-0.01,+0.60]; learned veto vs rule: irr -0.50 [-0.71,-0.35], rel -0.12 [-0.22,-0.05]; learned vs current rel +0.36 [+0.19,+0.53], irr -0.20 [-0.54,+0.09]. Short: rule vs current rel +0.28 [+0.13,+0.41], irr +0.02 [-0.19,+0.25]; veto barely acts (19% of candidates have history). Script muninn/scripts/learned.py.
