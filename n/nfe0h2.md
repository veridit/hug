---
id: nfe0h2
title: "Long-message recall noise: design a pre-registered measurement"
state: todo
type: spike
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-19
updated: 2026-11-19
parent: "d634kr"
---

## Goal
A pre-registered, blind-judged measurement decides whether a local situation filter cuts irrelevant recall on long (templated, machine-written) messages without losing relevant recall, on a fresh set no earlier test has seen

## Purpose
Design question, not a build: two separate tests found a CI-clear noise cut on fresh long messages with relevant recall flat. jrj1vq's kind rule (fresh check 2026-11-15, top-20 ungated pool, own project, <2000 chars, vector>=0.60, CASE agrees, kind!=record): long n=64, irr 3.38 -> 1.39, -1.98 [-2.44,-1.50], rel +0.05 [-0.25,+0.32]. 6assqk's candidate rule (bonehound 2026-11-18, check1-table.md on 6assqk): fresh long irr -0.88 [-1.23,-0.48] at 0.64 (-0.88 to -1.30 across 0.60/0.64/0.68), rel flat, while failing its bar on the frozen sets (frozen long irr +0.60 [+0.19,+0.99] at 0.64). Both fresh results share one 64-message long set from 2026-11-15, and the frozen sets disagree, so the lead needs its own pre-registered test on new data before anything is built.

## Principle
Measure before building; nothing ships without the recall rule (muninn/docs/RECALL4.md: CI-clear gain on real replayed messages, confirmed fresh, judged blind). Long messages only; short messages are ta306h. Recall stays local and free: the filter may use kind, vector floor, CASE, length, never a model call per message. PREREG.md (arms, metric, bar, judge, paid-call cap) with its sha in a note and crab's approval before the first paid call. The 277 frozen and the 2026-11-15 147 fresh are development data only.

## Relevant details
Evidence: jrj1vq notes (fresh check 2026-11-15; report $JCODE_SCRATCH_DIR/recall4/real/report-whenthen.md), 6assqk notes and file check1-table.md (scripts $JCODE_SCRATCH_DIR/6assqk/{coverage,arms,table}.py; arms recall4/real/rule{60,64,68}-*.jsonl). Harness: muninn/cmd/recall4, muninn/scripts/run_real.py and judge_real.py (blind DeepSeek Flash rubric). Open design points: which arms (kind rule, rule64, both, a length gate), why frozen long rose while fresh long fell (set composition: fresh long is mostly templated jobs?), and the fresh set (messages after 2026-11-15, in neither earlier set, deduped by text, >= 8 sessions).

## Depends on

## Acceptance
- [ ] PREREG.md written (arms with exact parameters, metric rel/msg and irr/msg on long messages, bar, blind judge, paid-call cap), sha in a note, approved by crab before any paid call
- [ ] fresh long set built from messages after 2026-11-15, deduped by text against the 277 frozen and 147 fresh sets, >= 8 sessions; counts in a note
- [ ] result table with paired session-cluster CIs against shipped recall in a note; verdict against the pre-registered bar

## Notes
