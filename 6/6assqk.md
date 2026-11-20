---
id: 6assqk
title: "Recall: learned 'not when' triggers and a local candidate rule"
state: "todo"
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
stage: "ready"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Recall shows more of the relevant memories search finds and learns when not to show one, all local and free.

## Purpose
Jørgen 2026-11-15 (crab session) asked what the LLM filter's big relevance gain consisted of, said recall 'should be local and free' (10:27Z), and asked for 'if x then y' patterns matched by BM25 or similarity (12:36Z). crab's analysis (10:39Z): the gain is almost entirely memories search already finds but today's gates throw away; on short messages the top 20 candidates hold 2.48 relevant memories and recall shows 0.67, and two thirds of the LLM's useful picks sat at ranks 6 to 20. So the problem is choosing, not finding, and this task tries to choose locally.

## Principle
Local and free at recall time: no model call per message (AGENTS.md; Jørgen 2026-11-15 10:27Z). Labels for the learned veto come only from earlier judgments, never from the message being scored, so nothing leaks. Ship only under the recall rule: CI-clear gain on the 277 frozen and 147 fresh real messages, judged blind (check 3). A veto only skips a memory in a situation; it never deletes it (as s6ff5k: an off-topic signal never deletes).

## Relevant details
Recall shows more of the relevant memories search already finds, and stops showing a memory in situations like those where it was judged irrelevant before, all local and free

Offline 2026-11-15 (notes): the rule adds relevant memories but also noise on long messages (irr +0.30 [-0.01,+0.60]); the learned veto cuts long noise (-0.50 [-0.71,-0.35]) and barely acts on short ones (19% of candidates have history). Labels in production do not exist yet: memory_feedback, forgets and dream vetting give no relevance signal today (memory bgqxqj), and agents essentially never call memory_forget after recall (s6ff5k: 0 of 454 sessions). Warning from s6ff5k check 2 (2026-11-17, offtopic_sim.py): with perfect oracle feedback, per-memory similar-message suppression still cost relevant recall at every threshold, because a memory off-topic for one message is often on-topic for a similar one. Sibling jrj1vq measured the kind rule (kind != record) on the fresh set: long irr -1.98 [-2.44,-1.50]. Script muninn/scripts/learned.py.

## Depends on

## Acceptance
- [ ] candidate rule (top 20 ungated, own project, under 2000 chars, vector >= 0.64) measured against shipped recall on the 277 frozen and 147 fresh messages
- [ ] learned veto: a memory is skipped when its closest earlier judged-irrelevant situation is clearly closer than its closest judged-relevant one; source of labels in production defined (memory_feedback, agent forget, dream vetting)
- [ ] ship only if relevant/msg rises and irrelevant/msg does not, CI-clear, on the fresh set
- [ ] written 'applies when' triggers and recall on tool calls (post_tool_feedback) tracked as follow-ups

## Notes

Offline 2026-11-15, 277 frozen messages, earlier-session history only (55 sessions), blind judge. Long: rule vs current rel +0.48 [+0.32,+0.65], irr +0.30 [-0.01,+0.60]; learned veto vs rule: irr -0.50 [-0.71,-0.35], rel -0.12 [-0.22,-0.05]; learned vs current rel +0.36 [+0.19,+0.53], irr -0.20 [-0.54,+0.09]. Short: rule vs current rel +0.28 [+0.13,+0.41], irr +0.02 [-0.19,+0.25]; veto barely acts (19% of candidates have history). Script muninn/scripts/learned.py.


bonehound measure-only, no judge calls yet ($JCODE_SCRATCH_DIR/6assqk/coverage.py, output coverage-out.txt). Candidate rule (top 20 ungated pool, own project, <=2000 chars, vector >= 0.64, top 5, session dedup) vs SHIPPED recall (recall4 'default' = bm25x2, identical hits on all 277 frozen messages), cached blind judgments only; an unjudged pick counts as neither. Paired deltas, 95% session-cluster CI (2000 reps, seed 813):
frozen human n=124 (10 sessions): rel 0.50 -> 0.88, +0.38 [+0.22,+0.55]; irr 0.21 -> 0.47, +0.26 [+0.05,+0.56]; unjudged 0.08/msg.
frozen long n=153 (53): rel 0.92 -> 0.99, +0.07 [-0.12,+0.25]; irr 0.96 -> 1.51, +0.55 [+0.19,+0.88]; unjudged 0.09.
fresh human n=83 (6 sessions, under the 8-session bar): rel 0.28 -> 0.18, -0.10 [-0.26,+0.06]; irr 0.33 -> 0.18, -0.14 [-0.50,+0.13]; unjudged 0.05.
fresh long n=64 (62): rel 0.38 -> 0.42, +0.05 [-0.25,+0.32]; irr 3.38 -> 1.62, -1.75 [-2.15,-1.32]; unjudged 0.89 (57 of 188 picks).
Reading so far (not a verdict, 71 picks unjudged): against today's BM25 recall the rule is no clean win. On frozen it adds relevant short-message recall but also irrelevant (irr CI clear of 0 both strata). On fresh long it cuts noise hard, but rel is flat and 57 picks are unjudged. The 2026-11-15 offline numbers were against pre-BM25 recall.
Bar fixed BEFORE judging (crab 16:31): the rule passes check 1's comparison only if irr does not rise on frozen human AND frozen long (upper 95% CI < 0) and rel is not worse on any split (lower CI not entirely below 0). Otherwise check 1 is parked with the full table. Fresh human (6 sessions, below the 8-session bar) is indicative only. Budget: at most 150 DeepSeek Flash calls via judge_real.py, same rubric, cached, blind to arm. Nothing ships.
file:check1-table.md (2000 bytes, sha256:78be2c0b) check 1 table: rule60/64/68 vs shipped bm25x2, every pick judged blind (bonehound 2026-11-18)
check 1 PARKED (bar fixed before judging, crab 16:31: frozen irr upper CI < 0 on human AND long, and rel not worse on any split). Judging: 142 unjudged pairs across rule60/64/68, 27 DeepSeek Flash calls (24 batches, 3 retried), judge_real.py unchanged rubric, cached, blind to arm; 0 unjudged left. Full table attached: file:check1-table.md. Rule vs shipped (bm25x2), delta [95% session-cluster CI]:
rule64 (as specified): frozen human rel +0.47 [+0.30,+0.61] irr +0.36 [+0.06,+0.88]; frozen long rel +0.10 [-0.10,+0.29] irr +0.60 [+0.19,+0.99]; fresh long rel +0.06 [-0.24,+0.34] irr -0.88 [-1.23,-0.48]; fresh human (6 sessions, indicative only) rel -0.07 [-0.26,+0.11] irr -0.12 [-0.50,+0.18].
rule60: frozen irr +0.82 / +0.71, both CI above 0. rule68: frozen irr +0.03 [-0.18,+0.34] / +0.35 [-0.04,+0.72], not below 0; fresh human rel -0.16 [-0.27,-0.01] (indicative).
Verdict: FAIL at all three thresholds. No arm keeps frozen irr from rising, and at 0.64 and 0.60 it rises with the CI clear of 0. The 2026-11-15 gain was against pre-BM25 recall; BM25 v0.7.2 took most of it. What remains: the rule halves fresh-long noise (irr -0.88 to -1.30, CI clear) with rel flat. That is a long-message noise lead (compare jrj1vq's kind rule, long irr -1.98), not this rule. Scripts: $JCODE_SCRATCH_DIR/6assqk/{coverage,arms,table}.py; arms written as recall4/real/rule{60,64,68}-*.jsonl. Nothing shipped.
bonehound gives 6assqk back: check 1 parked with the table above (fails the bar at 0.60/0.64/0.68). Checks 2-4 are not started; the learned veto (check 2) has no production label source yet (memory bgqxqj, s6ff5k).
The fresh-long noise lead (irr -0.88 to -1.30, rel flat) is filed as its own design task nfe0h2 (not a child of this one), with jrj1vq's kind rule (long irr -1.98 [-2.44,-1.50]) as the second piece of evidence. Check 1 stays parked; no rerun (crab 01:02).
## History
- 2026-11-18T17:30 todo -> doing/active (take)
- 2026-11-18T17:35 doing/active -> todo/ready (give back; was assignee: bonehound); was doing_since: 2026-11-18T17:30
