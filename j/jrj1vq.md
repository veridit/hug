---
id: jrj1vq
title: "When-then memories: triggers stated at write time"
state: todo
type: feature
priority: ""
assignee: []
labels: ["recall", "memory-format"]
created: 2026-11-15
updated: 2026-11-15
parent: "d634kr"
group: ""
rank: "zzzzq"
aliases: ["when-then-memories-triggers-stated-at"]
---

## Goal
Memories carry an explicit trigger so recall injects them when the situation matches, not when the words do

## Purpose
Jørgen 2026-11-15 (crab session) wants recall to know that 'in a case so-and-so you need to do so-and-so, not just a keyword match on anything' (10:28Z), and named the when-then pattern (12:44Z): the storer has to specify when-then in the memory, with variants plain ('this happened'), when-then, when-not-then and case-when-then (memory xm0z3b). After the first extraction test he said (12:52Z): 'logic is important, and if you know that something condition hinges on something else ... this is a sensible way. Let's explore this.' Today a memory fires because its own words resemble the message, so a note about a ticket's history fires on every review of that ticket (c3hszg fresh check).

## Principle
The storer states the trigger at write time, when it knows the situation (Jørgen: 'the storer has to specify'). Match the situation, not the note: CASE by exact filters, WHEN by BM25 and vectors, THEN is only what is shown. Plain memories stay valid. Paid model calls only at write time (capture, dreaming); recall stays local and free. Ship only under the recall rule, on the frozen and fresh sets, judged blind (check 4).

## Relevant details
Measured 2026-11-15 (notes): WHEN as a vector did not beat the memory text (AUC 0.608 vs 0.644); BM25 on WHEN was the best single signal on long messages (0.676); WHEN NOT as written is useless (0.503). KIND is the useful structure (relevant share rule 0.44, preference 0.47, fact 0.45, record 0.17). Fresh set: kind rule cuts long-message noise 3.38 -> 1.39 irrelevant per message (-1.98 [-2.44,-1.50]) with relevant unchanged; short messages not significant, mostly one session. Already in the store: inferred_types table and a mtype column (live: 246 rule, 1352 fact, 35 preference, 18 record of 30,051 rows; the rest untyped); skip_records_above 600 is the default (config.go:53). Siblings reuse this: ez50rb (holds conditions, done) and n7jbmx (tool-call anchors take CASE paths once this lands). Reports: $JCODE_SCRATCH_DIR/recall4/real/report-whenthen.md; scripts muninn/scripts/{extract,eval}_whenthen.py.

## Depends on

## Acceptance
- [ ] memory format: optional case (scope, paths, tool), when (situation text), when_not (suppressing situations), then (the note); plain memories stay valid
- [ ] memory_remember and capture ask the storer for when/then when the memory is a rule or pattern; dreaming proposes when/then for existing rules, agent-judged
- [ ] recall matches the current situation (message, and later tool calls) against when/when_not with BM25 + vector, separately from the note text
- [ ] measured on the frozen and fresh replay sets against shipped recall before becoming the default

## Notes

Extraction test 2026-11-15: DeepSeek Flash wrote kind/case/when/when_not/then for 2,932 memories (366 calls, 1.09M prompt + 0.50M completion tokens), seeing only memory text, scope and date. On 277 frozen messages, blind judge: (1) matching the situation against WHEN is NOT better than against the memory text: within-message AUC sem_when 0.608 vs sem_text 0.644; BM25 on WHEN 0.675 (best single signal on long, 0.676 vs 0.613). (2) WHEN NOT as written is useless: AUC 0.503. (3) KIND is the useful structure: relevant share rule 0.44, preference 0.47, fact 0.45, record 0.17 (n=1372). Text rule minus records vs current recall, all: rel +0.16 [+0.06,+0.29], irr -0.32 [-0.50,-0.15]; at vector >= 0.60 with case: rel +0.27 [+0.13,+0.42], irr -0.20 [-0.41,-0.01]. Report: $JCODE_SCRATCH_DIR/recall4/real/report-whenthen.md; scripts muninn/scripts/{extract,eval}_whenthen.py

Fresh check 2026-11-15 (147 post-freeze messages, frozen 5 Nov snapshot, blind judge), kind rule = top-20 ungated pool, own project, <2000 chars, vector>=0.60, CASE agrees, kind!=record, vs shipped v0.7.2. Long n=64: irr 3.38 -> 1.39, -1.98 [-2.44,-1.50]; rel 0.38 -> 0.42, +0.05 [-0.25,+0.32]. Short n=83 (6 sessions): rel +0.04 [-0.22,+0.38], irr +0.18 [-0.46,+0.79], not significant; 26 of the 33 added irrelevant picks come from one session (session_lion_179447205, hours logging). Verdict: CI-clear noise cut on long messages, no evidence either way on short. Next: ship kind for long messages first, or gate short ones by a stricter vector floor.
