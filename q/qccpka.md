---
id: qccpka
title: "Captures land in personal scope while extract.wide_places is off"
state: "doing"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T02:37"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "crab: trace reported; undo of 9 moves and drop/reword of qccpka are his call"
waiting_since: "2026-11-19T02:40"
waiting_asked_by: "piglet"
---

## Goal
A capture lands in the personal scope only when extract.wide_places is on, as capturePlacement says; with it off, a capture made in a project lands in that project or the folder's scope.

## Purpose
66ex4y daily 2026-11-19: 15 of 562 captures since 2026-11-18T02:31Z are in scope personal, though config.json has no extract.wide_places key (off). 9 of the 15 were hug rules or facts and were rescoped by hand to project:rc-0ca4bf6be5d989c3 (dpvcdm wwqfa8 8z7757 ghtqen y4061x t4q0t0 8e91jw t2tp92 a0yarc). Left in personal: nrd8m1 tg629m 69q162 zq3faj (the others were forgotten as stale or duplicate: 1z7e3x d9afy7).

## Principle
Trace from the stored rows and code, not by guessing. Do not change placement rules beyond what the trace shows is wrong. Recall stays local and free; no paid call to reproduce.

## Relevant details
Sessions (origin capture:jcode:...): hatchling_1794905151586 5 (05:48-05:50Z), nautilus_1794905151412 4 (06:46-06:48Z), calf_1794909717965 4 (08:44Z), puppy_1795000793237 2 (12:01Z), all 2026-11-18, all hug workers. Code: muninn/internal/app/places.go capturePlacement returns PersonalScope only if wide && name is personal/other && personalWritable(); else Placement("", name), whose places are here/client/global. Not yet known: the capture_actions rows for these sids, the session's binding (JCODE_HOME, cwd) at capture time, and which binary ran (brew moved v0.7.73 -> v0.7.74 on 2026-11-19). Daily file: .hug/tasks/6/66ex4y/daily-2026-11-19.txt section 4 and B.

## Depends on

## Acceptance
- [ ] the code path that wrote these 15 rows to personal is named with file:line and the capture_actions rows that show it (note)
- [ ] with wide_places off, a capture whose extractor says personal or other lands in here, never personal (test that fails on today's main if the path is real)

## Notes


file:qccpka-TRACE.md (7366 bytes, sha256:48c9877b)
trace (piglet, read-only): not a capture path. All 15 were captured into project:rc-0ca4bf6be5d989c3 (e.g. dpvcdm: capture_actions run 7460 pos 1, place here, capture.go:437 via places.go:160 with wide off) and moved to personal by q0a8rx's batch rescope at 2026-11-18T19:49:51Z (moves.tsv, crab ruling B; backup tidy/rescope-20261118T194951Z-4179401420 shows them in project just before). The one personal capture_actions row (run 8355 pos 15) is a reinforce taking dpvcdm's existing scope (capture.go:438-446). Consequence: my 66ex4y rescope at 01:26:55Z undid 9 deliberate q0a8rx moves (dpvcdm wwqfa8 8z7757 ghtqen y4061x t4q0t0 8e91jw t2tp92 a0yarc); undo is crab's call. No code bug, so check 2 has nothing to pin. File: qccpka-TRACE.md
crab 01:41Z: ruled. Not a bug: the 15 reached personal by q0a8rx ruling B (batch rescope 2026-11-18T19:49:51Z), which is correct under 1B. piglet's 66ex4y folder rescope of 9 undone by crab: 'hug mem rescope --from scratch/qccpka-undo.tsv' -> 'rescoped 9 to personal', backup tidy/rescope-20261119T014119Z-2196365625. Dropped as not a defect; the side finding (batch rescope writes no memory_fixes row) is filed separately.
## History
- 2026-11-19T02:37 todo -> doing/active (take)
- 2026-11-19T02:40 doing/active -> doing/waiting (wait)
