---
id: rjk44w
title: "Memory fixes leave a trail: who forgot what, after which recall, and why"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzyx"
created: 2026-11-18
updated: 2026-11-18
parent: "d634kr"
---

## Goal
Every hug mem forget and memory_remember that corrects a note records, locally, the session, the harness, whether that id was shown in a recall block in the same session and how many turns earlier, and an optional reason (--reason / MCP reason). hug mem stats --fixes prints, per day and per instance, recall blocks shown, fixes of a shown id, fixes the owner asked for versus unprompted, and forgets later undone or contradicted.

## Purpose
s6ff5k item 1 had to be measured by regex over four jcode instances' transcripts, and the answer is still soft: the injections table holds 84 jcode sessions since 2026-11-16 12:00 but only 35 of their transcripts contain the block (the rest are absent or rewritten), and nothing says why a memory was retired. Of the 5 working sessions that forgot a shown id since then, in 3 (orangutan, spider, rabbit) the owner had told the agent to ('Okay, if you see a recalled note that is obsolete, you have to remove it'; 'Doesn't that instruction tell you to?'; 'Remove obsolete memory'). Without a trail we cannot tell header-driven fixes from owner-driven ones, or see a wrong forget at all (H2 made 28 forgets of notes that still held, found only by a blind judge in a replay).

## Principle
Local and free: a row in muninn.db per fix, no model call, no change to what recall shows or ranks. Recording never blocks the write and never asks a question. The reason is optional; absent is recorded as absent. The stats read the trail, not transcripts.

## Relevant details
Today: units has active/updated_ns only; holds_ended has reason but only for holds; injections (session, unit_id, at_ns) already ties ids to sessions, so 'shown before' is a join. Scripts and counts: /Users/jhf/.jcode/scratch/s6ff5k-feedback/forget_stats3.py, trigger.py, REPORT.md cluster C1. 173 units went inactive since 2026-11-16 12:00 with no record of which command, session or reason.

## Depends on

## Acceptance
- [ ] hug mem forget ID --reason TEXT and MCP memory_forget reason are stored with session, harness, time and shown-in-session (test with a recall block injected first)
- [ ] memory_remember naming an old id (supersedes or reinforce) records the same row (test)
- [ ] hug mem stats --fixes prints recall blocks shown, fixes of a shown id, and fixes without a shown id per day and instance; a golden over a fixture store
- [ ] A forget whose id is remembered again or reinforced within 7 days is counted as undone in the stats (test)
- [ ] No recall output changes: the recall4 frozen replay is byte-identical before and after

## Notes
