---
id: 8zwy6r
title: "Sync merges notes added on both sides instead of stopping"
state: todo
type: bug
priority: "high"
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzu"
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

crab ruling 04:44: the resolver (union for append-only parts in time order; remote wins on a single-value field changed on both sides, with the local value kept as a note and one printed line per field; a stop only for an unparseable file) is built in w40cmk check 9's catch-up first. 8zwy6r then makes sync use that same resolver.

crab ruling06:15Z for shared resolver now in w40cmk: first-parent replay full parent1 delta once; non-task true conflict remote original + exact local sibling.local-FULLCOMMIT(mode preserved), differing recovery collision stops safely, both paths/commit printed. Same request competing resolutions keep remote canonical and append explicitly labelled local competing-resolution note tied to stable request identity, resolver/time/full verbatim text, summary, no second open request. Untimed appends deterministic remote-before-local once with each side order preserved, no invented timestamps/chronology claim; timed entries actual instants/stable ties. Ordinary offline migration no fetch; only known cross-layout catch-up fetches. Remote task deletion/local edit stops safely naming id/commit/content until tested deletion policy. Implementing under w40cmk before this follow-up wires normal sync.

crab06:26Z shared catch-up translator ruling: if remote base remains layout2 while local independently upgraded3, canonicalize remote base in private checkout with same validated translator, stage coherent full move into first retained replay commit(or explicit migration if none). Retain remote edits, aliases,updated,modes; fail safely on ambiguous/unparseable translation. Reverse direction is not silently treated as already-canonical.

Crab06:46 required ordinary SAME-task shared resolver in w40cmk before check9. Committed35d560e6 on kestrel/w40cmk, no main push. Ordinary foreground and detached background handles now use exact catch-up translator/private replay for all true divergences, including scalar disjoint-line text changes. Same-task A-sync/B-sync/A-sync both clones once and matching HEADs proved by TestOrdinarySyncSameTaskAppendsConvergeAfterUpgrade: --- PASS (3.95s), focused95727423zo exit0. Scalar remote-wins/local-note/report, detached child same-task append, malformed-task full rollback/no premature report also pass. Both scalar note metadata and source commit remain visible, background summary in sync log. Spec huginn/docs/LAYOUT3-CATCHUP.md now describes shared normal sync. This ticket not taken/done, coordinator decides overlap after actual final gates and CI.

Shared ordinary resolver safety correction11c0df2e: two legacy-layout clones must not implicitly migrate during ordinary sync. Failing-first22237123ow reproduced layout3 unexpectedly, corrected with TargetLayout guard; test passes2.95s. Pure disjoint Goal-line test passes4.77s, proves semantic scalar handling cannot be deferred until Git text conflict. Focused287786gtef exit0. Final local gates pending390827vrmb, no main push or ticket completion inferred.
