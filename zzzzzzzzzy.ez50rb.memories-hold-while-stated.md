---
id: ez50rb
title: "Memories hold while a stated condition holds, and name no agent"
status: "doing"
type: feature
priority: "high"
assignee: ["guppy"]
labels: []
created: 2026-11-16
updated: 2026-11-16
parent: "d634kr"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A memory that is only true for a while says so when it is written: 'holds until <date>', 'holds while task:ID is open', 'holds while commit/branch X is unmerged'. When the condition ends, the memory stops being recalled and goes to a dream item for an agent to retire or keep. Memories never name an ephemeral agent; they name the ticket, commit, file or model instead.

## Purpose
Jørgen 2026-11-16: memories naming agents are bad memories; 'if you know it at the time, write the condition or when it expires ... if that's no longer valid, it will be gone'. Evidence the same day: orangutan scanned 10,107 seid memories, found 239 naming 97 stopped agents (including words like tiger and swan that needed hand checks) and bulk-deleted them; 233 of the first 239 forgets failed under concurrent writes. bx26kc (2026-11-12) already said capture should not store short-lived agent status.

## Principle
State the condition at write time, the same way jrj1vq states when/then: the storer knows it then, nobody knows it later. Conditions are checkable by hug without a model call: a date, a task's status, a git ref. When one ends, the memory is hidden from recall and offered once as a dream item (agent-judged, never deleted automatically, per 156x77). Capture rewrites agent names to what they refer to (the ticket the agent was on), and drops pure agent status. No condition means the memory holds until judged otherwise, as today.

## Relevant details
Builds on the when-then format (jrj1vq, same epic d634kr): this is one more field, holds_while/until. Capture prompt: muninn/internal/extract/extract.go:239. Recall filter: muninn/internal/recall. Old memories naming agents: dream items, one at a time, not a regex purge. Ship recall changes only with a measured, CI-clear gain on replayed messages (AGENTS.md).

## Depends on

## Acceptance
- [ ] memory_remember and hug mem remember accept a condition (until DATE, while task:ID open, while REF unmerged) and store it (test)
- [ ] a memory whose condition has ended is not recalled and appears once as a dream item; nothing is deleted automatically (test)
- [ ] capture on a transcript that names a running agent writes the ticket or file instead and never stores agent status (replayed sample, counted)
- [ ] on the seid store, memories naming stopped agents are offered as dream items with the ticket they referred to (count reported)

## Notes

guppy 2026-11-16 20:22, prework (read-only, no store writes), script $JCODE_SCRATCH_DIR/ez50rb/agents2.py: 124 agent names from jcode session files; of 29,117 active memories in the local store, 2,391 mention a name somewhere, 197 name it in agent context (v2 matcher), 58 of those in seid scopes. v1 matcher hand-checked on a seed-7 sample: 24/50 agent refs (misses: 'Dragon 1' design packets, 'bug' the noun). v2 written after that, checked on a NEW seed-11 sample: 48/50 agent refs (2 false: 'independent bug scouting', 'compiler lane bug'). Design waiting for crab: tmp/lanes/ez50rb-design.md. jrj1vq is not built (todo), so holds is its own field.

guppy 21:02 ci/ez50rb 787e83c (rebased on 71a6de5). Regression, recall4-main (origin/main) vs recall4-holds, default variant, frozen snapshots: development 200 vs 200 messages, 253 hits, differing 0; heldout 77 vs 77, 131 hits, differing 0; fresh 147 vs 147, 290 vs 290 hits, differing 0 ($JCODE_SCRATCH_DIR/recall4/ez50rb/{regress,fresh}.py). Agent matcher, Go, on a read-only copy of the local store: 124 names, 29,118 memories, 291 name an agent (87 seid). Tuning: first Go version found 302; hand sample seed 23 showed noun false matches (bug hunt, Badger the DB, dragon-task, 'No bug. The architect'); fixed; seed 31 sample showed over-tightening lost true hits (DM fox only, coordinator is fox), loosened. Then a NEW seed-41 sample: 50/50 agent refs ($JCODE_SCRATCH_DIR/ez50rb/go-sample50-seed41.txt). Names are reused across months, so each memory is matched to the session of that name alive at its write time (agents.Meant); a memory naming a running agent (session written < 24 h) waits. Counts: 132 name a running agent, 159 a stopped one (73 seid), 119 of those 159 get a ticket hint from that session (65 seid); 5 have no session at or before write time. Full scan 3.5 s, so the dream scan is incremental (meta agent_scan_ns). Native store has 0 memories with a recorded branch today, so no branch items will appear yet. Capture (check 3) NOT done: no prompt change, no post-check; the ruling's gate needs a blind-judged gain on 40 transcripts, not run. Tests: holds, agents, app, recall, store, index, ingest, mcpserver, run, extract all ok; vet ./muninn/... clean.
