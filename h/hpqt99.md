---
id: hpqt99
title: "Design: imported data pass"
state: "doing"
type: spike
priority: "low"
assignee: ["piglet"]
labels: ["design"]
group: "design"
rank: "zz"
created: 2026-11-10
updated: 2026-11-10
doing_since: "2026-11-18T23:18"
stage: "asked"
model: "claude-opus-5-5"
branch: "hpqt99"
worktree: "/Users/jhf/.jcode/scratch/hpqt99"
asked_by: "piglet"
asked_check: "1"
asked_what: "Imported data (docs/DESIGN.md, 'Imported data', branch hpqt99 9c8fb6a1). Recommend both: (1B) an import places a memory by the folder it was written in, so a memory written under a client folder lands in that client's scope (narrowing only; one leak today: 2kb574, written in ~/Projects/frogs, sits in global). (2B) import each jcode instance's memories once (17,089 rows; jcode files frozen since 2026-11-05), gated by a dry-run whose per-scope counts must match the live read, then stop reading jcode files live. Approve 1B+2B, or name another option (1A/1C, 2A/2C)?"
asked_for: "jhf"
asked_since: "2026-11-18T23:30"
---

## Goal
Decide client provenance for imports and live-read vs import of jcode memories

## Purpose
Jørgen 2026-11-05 17:00Z (crab session), in the same breath as personal vs shared (gav9nw): 'Imported data. Okay, we have to do that again later and write into the to-dos' (memories 5zwpx7, 1c49sh). Two points are open. (1) Imported data (e.g. Engram) does not record client origin, so it becomes client-only only through the explicit cross-client admin rescope (fqdv6k, 2026-11-06; muninn/docs/ROADMAP.md). (2) jcode memories are both read live and imported, and Jørgen wants muninn to own all memories eventually: import jcode memories once, stop jcode writing, stop reading jcode files live, so forget and sync work on one source of truth (b97h8c).

## Principle
A design decision first, written down, no code in this task. Keep the client boundary that shipped 2026-11-05: imports and sync can never bypass it, and harness files stay read-only (ROADMAP.md). The direction is Jørgen's (one source of truth in muninn); the task decides how and when to get there, and he makes the call.

## Relevant details
Today: hug setup turns jcode's own memory off per instance and routes saves to muninn's MCP, while jcode's existing memory files stay and are indexed read-only (muninn/internal/harness/jcode_takeover.go). A jcode memory both read live and imported shares one short id; recall shows it once and forget removes both (1tz1yn). Only admin 'rescope <id> --to <scope> --cross-client' moves memories across client boundaries. The acceptance says 'decision in DESIGN.md': the hug design is now docs/DESIGN.md (huginn/DESIGN.md is the older tasks design).

## Depends on

## Acceptance
- [ ] decision in DESIGN.md

## Questions
- 2026-11-18T23:30 question 1 by piglet to jhf (check 1):
  Imported data (docs/DESIGN.md, 'Imported data', branch hpqt99 9c8fb6a1). Recommend both: (1B) an import places a memory by the folder it was written in, so a memory written under a client folder lands in that client's scope (narrowing only; one leak today: 2kb574, written in ~/Projects/frogs, sits in global). (2B) import each jcode instance's memories once (17,089 rows; jcode files frozen since 2026-11-05), gated by a dry-run whose per-scope counts must match the live read, then stop reading jcode files live. Approve 1B+2B, or name another option (1A/1C, 2A/2C)?

## Notes


Design pushed: 9c8fb6a1 on hpqt99 + ci/hpqt99 (docs/DESIGN.md, 'Imported data'). Read-only counts on this machine 2026-11-18 22:25Z: 17,089 live jcode rows / 128 files (none newer than 2026-11-05 22:04), 14,258 native; 0 native rows from 'import jcode'; 25 tidy keepers share a sid with a live row; 8,758 jcode rows under client roots (brovs 5,992, ssb 2,766); engram 3,703 obs, 1,526 written under a client root, 1,525 in client scope, 1 leak (2kb574, frogs -> global). Recommend 1B + 2B; asked jhf on check 1.
## History
- 2026-11-18T23:18 todo -> doing/active (take)
- 2026-11-18T23:30 doing/active -> doing/asked (ask)
