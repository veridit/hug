---
id: hpqt99
title: "Design: imported data pass"
state: todo
type: spike
priority: "low"
assignee: []
labels: ["design"]
group: "design"
rank: "zz"
created: 2026-11-10
updated: 2026-11-10
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

## Notes
