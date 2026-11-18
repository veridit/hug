---
id: 8zwy6r
title: "Sync merges notes added on both sides instead of stopping"
state: todo
type: bug
priority: "high"
assignee: ["kestrel"]
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
When two clones each add a note (or check a different item, or answer, or add a request) to the same task offline, the next sync keeps both, in time order, and pushes; it stops only when two sides change the same line of the same field differently.

## Purpose
kestrel measured on main a13d7c74 (TestBaselineBothSidesNoteSync): A and B each run 'hug task note ID' offline; A syncs, then B's sync exits 1 with 'sync conflicts: h.ch02nx.noted-twice.md (rebase aborted)', and B's note stays unpushed. Agents note constantly, so two agents on one task hit this routinely. Jørgen's rule: concurrent writes wait, never fail.

## Principle
Merge by meaning, not by text: hug knows a task's sections, so append-only parts (Notes, History, Requests, answers) are merged as the union in time order. Fields that hold one value (state, stage, rank) still stop on a real disagreement, with a fix line. No git union driver for the whole file.

## Relevant details
Today: no .gitattributes; the note goes at the end of ## Notes, so both sides insert at the same line. Fits the rebase path in huginn/gitbranch and the w40cmk catch-up (same resolver). Owner: kestrel, after w40cmk.

## Depends on

## Acceptance
- [ ] Two clones each note the same task offline; both syncs succeed and both notes are in the file in time order (test, fails first on main)
- [ ] Same for a check ticked on each side (different items), an answer and a request (table test)
- [ ] Two sides setting state differently still stop with a fix line naming the field (test)

## Notes
