---
id: 8zwy6r
title: "Sync merges notes added on both sides instead of stopping"
state: "done"
type: bug
priority: "high"
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzma"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T18:55"
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
- [x] Two clones each note the same task offline; both syncs succeed and both notes are in the file in time order (test, fails first on main)
- [x] Same for a check ticked on each side (different items), an answer and a request (table test)
- [x] Two sides setting state differently still stop with a fix line naming the field (test)
- [x] Two sides setting state differently: the sync succeeds, the board's state is kept whole (the losing state's node fields go to History), this clone's value is kept in a note, and one 'sync: ID state:' line is printed (test; crab 04:44 ruling)

## Notes

crab ruling 04:44: the resolver (union for append-only parts in time order; remote wins on a single-value field changed on both sides, with the local value kept as a note and one printed line per field; a stop only for an unparseable file) is built in w40cmk check 9's catch-up first. 8zwy6r then makes sync use that same resolver.

crab ruling06:15Z for shared resolver now in w40cmk: first-parent replay full parent1 delta once; non-task true conflict remote original + exact local sibling.local-FULLCOMMIT(mode preserved), differing recovery collision stops safely, both paths/commit printed. Same request competing resolutions keep remote canonical and append explicitly labelled local competing-resolution note tied to stable request identity, resolver/time/full verbatim text, summary, no second open request. Untimed appends deterministic remote-before-local once with each side order preserved, no invented timestamps/chronology claim; timed entries actual instants/stable ties. Ordinary offline migration no fetch; only known cross-layout catch-up fetches. Remote task deletion/local edit stops safely naming id/commit/content until tested deletion policy. Implementing under w40cmk before this follow-up wires normal sync.

crab06:26Z shared catch-up translator ruling: if remote base remains layout2 while local independently upgraded3, canonicalize remote base in private checkout with same validated translator, stage coherent full move into first retained replay commit(or explicit migration if none). Retain remote edits, aliases,updated,modes; fail safely on ambiguous/unparseable translation. Reverse direction is not silently treated as already-canonical.

Crab06:46 required ordinary SAME-task shared resolver in w40cmk before check9. Committed35d560e6 on kestrel/w40cmk, no main push. Ordinary foreground and detached background handles now use exact catch-up translator/private replay for all true divergences, including scalar disjoint-line text changes. Same-task A-sync/B-sync/A-sync both clones once and matching HEADs proved by TestOrdinarySyncSameTaskAppendsConvergeAfterUpgrade: --- PASS (3.95s), focused95727423zo exit0. Scalar remote-wins/local-note/report, detached child same-task append, malformed-task full rollback/no premature report also pass. Both scalar note metadata and source commit remain visible, background summary in sync log. Spec huginn/docs/LAYOUT3-CATCHUP.md now describes shared normal sync. This ticket not taken/done, coordinator decides overlap after actual final gates and CI.

Shared ordinary resolver safety correction11c0df2e: two legacy-layout clones must not implicitly migrate during ordinary sync. Failing-first22237123ow reproduced layout3 unexpectedly, corrected with TargetLayout guard; test passes2.95s. Pure disjoint Goal-line test passes4.77s, proves semantic scalar handling cannot be deferred until Git text conflict. Focused287786gtef exit0. Final local gates pending390827vrmb, no main push or ticket completion inferred.


macaque 4504654a (branch 8zwy6r + ci/8zwy6r, on origin/main c84d3273). Finding: the shared semantic replay already on main (798d91c8 'Use shared semantic replay for ordinary task-board sync', kestrel's w40cmk work) makes ordinary sync merge same-task appends; no gitbranch change needed, so no overlap with tigress's fcf4fg. Check 1: TestSyncMergesBothSidesAppends/note_on_each_side (cli, plain fresh board, no layout upgrade): on a13d7c74 (kestrel's baseline) it fails 'sync in .../two: exit 1 / sync conflicts: h.88px62.shared-task.md (rebase aborted)'; on c84d3273 and 4504654a '--- PASS' (A, B, A sync all exit 0, each note once, A before B, HEADs equal).
Check 2: same table test, PASS on 4504654a: 'a_different_check_ticked_on_each_side' (- [x] one, - [ ] two, - [x] three), 'an_answer_against_a_note' (ask, then A notes while B answers), 'a_request_on_each_side'. Each: 3 syncs exit 0, entries once in order, clones converge.
Check 3 per crab 04:44 (supersedes the acceptance's 'still stop'): state set on both sides (A doing, B drop) syncs exit 0, prints one 'sync: ID state: kept the board's doing; this clone's dropped is in a note on ID', the board keeps doing, the local value is in a note. Fixed a gap: the losing drop's dropped_reason/dropped_at stayed in the front matter and linted broken [stray-node-field] on the board's doing task. task/merge.go strayToHistory moves them to History at the local commit's time ('- 2026-11-18T05:10 sync kept the board's state doing; this clone's dropped_at: ..., dropped_reason: ...'). Fail first: TestMergeStateBothSidesMovesLosingNodeFields 'merged task lints : node field dropped_reason belongs to dropped, and this task is doing/active'; e2e TestSyncBothSidesStateRemoteWinsAndSays with merge.go reverted '8zwy6r_sync_test.go:130: ... the merged task lints broken'. Now PASS; all 19 TestMerge* pass; the 6 existing TestOrdinarySync*/TestBackgroundSyncSameTask* pass. Full ./... 35/35 ok (cli 262.9s).
CI 44343714260 on 4504654a: completed success, 6/6 jobs, first attempt. Report /Users/jhf/.jcode/scratch/8zwy6r/REPORT.md
crab review 17:29 local: accepted. 4504654a on c84d3273: main already unions both sides' appends (798d91c8); test reproduces kestrel's a13d7c74 failure verbatim and passes now; real gap fixed in task/merge.go strayToHistory (the losing drop's node fields go to History at the local commit time, same bytes on both clones). CI 44343714260 6/6. Check 3 text follows crab's 04:44 ruling: editing it to match.
check 4: TestSyncBothSidesStateRemoteWinsAndSays PASS on 4504654a: exit 0, one 'sync: ID state: kept the board's doing; this clone's dropped is in a note on ID', state doing, no [stray-node-field]; TestMergeStateBothSidesMovesLosingNodeFields PASS; CI 44343714260 6/6
Item 3 cannot be struck: 'item 3 is ticked: it was done, so it is not struck out' (hug task strike refuses a ticked item and there is no uncheck). Its wording ('still stop with a fix line') is superseded by crab's 04:44 ruling; item 4 states the real behaviour and is ticked with evidence.
landed 518ecc1 on main 2026-11-18: sync: both sides' appends tested on a plain board; a kept state stays whole…
shipped in v0.7.67 (518ecc1)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T16:57 todo -> doing/active (take)
- 2026-11-18T17:27 doing/active -> doing/waiting (wait)
- 2026-11-18T18:55 doing/waiting -> done (done); was doing_since: 2026-11-18T16:57; waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-18T17:27
