---
id: tn8bmn
title: "hug mem: forget and remember report what the store really holds"
state: "done"
type: bug
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzzlq"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T04:27"
done_release: "v0.7.55"
---

## Goal
Every hug mem write reports the store's state after the write: forget exits non-zero unless the memory is inactive afterwards, and remember prints the id the store keeps (after any twin fold), so an agent never acts on a write that did not happen.

## Purpose
hatchling 2026-11-18 daily check (66ex4y, file:lost-forget-repro.txt) on Jørgen's live store: (A) 'hug mem forget xr634s' printed 'forgot' with exit 0, yet xr634s stayed active until a second forget; (B) 'hug mem remember' printed yx7vxv while the store holds erf4tk, and 'show yx7vxv' resolves to erf4tk. Neither reproduces on a HUG_HOME copy, which points at the live store's concurrency (several hooks, capture and MCP servers write it at once) or at the twin fold (5sc5rw). Jørgen's rules: nothing writes silently; concurrent writes wait, never fail.

## Principle
Find the cause first, then fix the cause, not the message. A write's reply is read back from the store under the same lock that wrote it. If a concurrent writer undoes it, the write waits and redoes it; it never reports success it does not have.

## Relevant details
Live store writers today: capture (turn_end, session_end hooks), recall hook, hug mem CLI, 8 hug mcp servers on old binaries (mbdfzs fixes those after one restart). Look at ingest.ForgetMany's lock and the read-modify-write of memories.json; and at remember's id when the fold merges a new memory into an existing twin. wyvern is changing forget's reindex in 4vd1mz: coordinate, don't collide.

## Depends on

## Acceptance
- [x] A test with two concurrent writers (forget and a capture or remember) shows the forget survives: the memory is inactive after both finish (fails first if the race is the cause)
- [ ] ~~remember prints the id the store keeps; when it folds into a twin it says so: 'kept as erf4tk (twin of your text)' (test)~~ (not relevant: crab 03:03Z: remember never folds synchronously (the fold runs in sync); replaced by check 5, printed SID == stored SID after sync on every clone, hatchling, 2026-11-18)
- [x] forget and remember read their result back under the write lock and exit non-zero with a fix line if it is not there (test)
- [x] The cause of (A) and (B) is named in a note, with the evidence
- [x] remember prints the SID the store keeps, and that SID is unchanged after project sync on every clone (test)

## Notes


file:tn8bmn-notes-history.txt (716 bytes, sha256:1618809b)
Cause of A (xr634s forget undone): a lost update between background project sync and forget. Not forget itself. project/sync.go syncFile snapshots the store (LoadNative after MergeImport, outside the lock) and writes the notes file from it. In SyncBranch the separate ingest.MarkSynced runs only after the git commit/push (branch.go:107) and sets every OriginHash from the store as it is THEN. A forget that lands between the snapshot and MarkSynced is stamped as 'equal to the file' while the file still says active:true. On the next pass MergeImport sees file hash != OriginHash, and the local hash == OriginHash, so it is an upstream change, and it writes active:true back. Evidence in file:tn8bmn-notes-history.txt, from muninn-notes in seid-lang: xr634s is active=True in 8a654329e 03:34:49, a42c6334d 03:34:56 and c76aacb8e 03:35:04, while the 5 forgets before it (02:34:40-54Z) are False by a42c6334d. It turns False only in a190f59c3 03:35:57, after the second forget at 02:35:51Z. Why the HUG_HOME copy did not reproduce: no remote there, so no background pass. Cause of B (yx7vxv printed, store holds erf4tk): the SID is shortid.Full(Identity(id, origin))[:6], and the identity is the origin once one is set. remember stores mem_1b78f08288ad0778 with no origin, so its SID is a hash of the id (yx7vxv). The background MarkSynced then sets origin repo:rc-3557a74eab224a4c/mem_1b78f08288ad0778, and assignNativeSIDs re-derives the SID as erf4tk. The index keeps yx7vxv as an alias, which is why show yx7vxv resolves. PLAN: (1) MarkSynced stamps only memories whose state equals the snapshot that was written, so a later edit stays a local edit and the next pass writes it out. (2) remember in a shared project scope sets the repo origin at write time, so the SID it prints is the SID every clone derives, and sync never changes it (keeps TestSharedOriginHasSameSIDOnAnotherMachine). (3) forget and remember read the store back after the write and exit 1 with a fix line if the result is not there. Check 2's twin wording: remember never folds synchronously (fold runs in sync), and reinforce already prints the kept SID, so the test covers reinforce plus SID stability across sync.
file:tn8bmn-failbefore.txt (410 bytes, sha256:8dc2ba71)
Landed 27643306 on main (CI 44229580197). Fix A: ingest.MarkSynced takes the snapshot syncFile wrote the notes file from (SyncStats.snapshot) and stamps origin and OriginHash only on memories still equal to it. A memory changed after the snapshot keeps its old hash, so the next pass sees a local edit and writes it out. Fix B: ingest.Remembering.OriginFor, with project.OriginForScope(scope) from Binding.RememberPlaced, gives a new project or client-project memory its repo origin at write time, so its SID is final. Readback: Binding.ForgetMany goes through forgetHeld, and RememberPlaced through rememberHeld. Both reload the store under LockNative (loadLocked) and fail with 'forget did not hold: SID is active again ... fix: hug mem forget SID' or 'remember did not hold: ... fix: hug mem show SID'. Tests: project TestForgetDuringSyncPushStaysForgotten (check 1) uses the real interleaving through the markSynced seam: SyncBranch snapshots, writes and pushes, the forget lands, then MarkSynced runs, then the next SyncBranch. run TestRememberedSIDSurvivesProjectSync (check 5): remember --project, then hug mem sync, then another clone's Sync, and the printed SID equals the stored one in all three. run TestForgetUndoneByAnotherWriterFails and TestRememberUndoneByAnotherWriterFails (check 3) use the app.WriteLanded seam. Fail-before on origin/main 0a2f06ae, file:tn8bmn-failbefore.txt: 'branch_test.go:286: psq58f is active again after the next pass: the forget was lost' and 'writeheld_test.go:80: remember printed 2d7he6, after sync the store holds 5d9p7y'. Check 3 has no fail-before because its seam is new API; before the fix, forget printed 'forgot' with no readback (run/main.go forgetCmd). -race after rebase, niced, all ok: ingest 3.178s, project 5.615s, app 77.875s, run 35.568s, mcpserver 7.928s, web 7.923s.
landed 2764330 on main 2026-11-18: hug mem: a forget during a project sync push stays forgotten; remember's short…
CI 44229580197 on 27643306: completed success, 6/6 jobs (secrets, bench, test macos-latest, test-windows-cli, test ubuntu-latest, test windows-latest). Live effect needs the next release: the brew hug v0.7.52 still has the race. The one-off scan of reactivated memories is on 66ex4y (4 re-forgotten in hug).
Reflection (dismissed, 2026-11-18, hatchling): The 'concurrency' theory in the ticket was half right: it was a lost update, but the writers were sync's two steps, not two forgets. muninn-notes history (one commit per pass) named the culprit faster than any repro on a copy without a remote could. Lesson: when a write vanishes only on the live store, diff the shared branch's history before suspecting the lock.
shipped in v0.7.55 (2764330)
Reflection 1 dismissed (tb0dra check 5 triage, crab 2026-11-19): G10, a lesson about one's own method, fixed in its own ticket; no instruction misled.
## History
- 2026-11-18T03:38 todo -> doing/active (take)
- 2026-11-18T04:27 doing/active -> done (done); was doing_since: 2026-11-18T03:38
