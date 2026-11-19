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
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
## History
- 2026-11-19T02:37 todo -> doing/active (take)
