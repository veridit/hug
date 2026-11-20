---
id: b8mk9s
title: "Design: how hug stores memories on disk (one file, per day, per memory)"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "5"
created: 2026-11-20
updated: 2026-11-20
parent: "9taa9d"
stage: "design"
design_for: "jhf"
design_since: "2026-11-20T17:08"
design_asked_by: "crab"
design_question: "Six layouts in Details, measured on your 18,615 memories. A day holds 38 KB typically (busiest 2.1 MB); 15% of memories change on a later day. My pick is E: a today folder with one small file per memory, rolled by dream into one JSON Lines file per day. Pick A to F?"
---

## Goal
Jørgen picks how hug stores memories on disk, from one big file to dated JSON Lines files, and how retired ones are cleaned up.

## Purpose
Each remember should write a few hundred bytes, not rewrite an 18.6 MB file.

## Principle
Decide on the consequences first; nothing is built until Jørgen picks.

## Relevant details
TODAY
- All memories are one file on your Mac: ~/Library/Application Support/muninn/memories.json (18.6 MB, 18,615 memories on 2026-11-20). muninn.db (805 MB) is the search index built from it.
- Every remember, forget, dedupe, rescope or import rewrites the whole file (SaveNative: write all 18,615, temp file, rename), under one lock every session waits on. 7 code paths call it (ingest/native.go, importer.go, lock.go, twins.go, app/correction.go, rescope_batch.go, tidy_duplicates.go).
- Retiring sets active: false in the same file. Nothing ever deletes retired memories.
- Memories change after the day they were made: 2,825 of 18,615 (15%) were updated on a later day (strength, retire, rescope). Any dated layout has to handle that.
- Spread over time: 141 days with memories since 2026-06-03. A day holds a median of 38 KB (busiest 2.1 MB, 2026-11-19); a week a median of 351 KB (busiest 3.9 MB).
- The shared hug/memory branch is separate: one memories.jsonl, one line per memory sorted by id (255 KB in this repo). git merges it line by line already. Out of scope here.

THE OPTIONS (all keep muninn.db as the search index, rebuilt from the files)
A. Keep memories.json as is. No work; every write stays 18.6 MB and growing.
B. One append-only JSON Lines file. Each write appends one line; dream rewrites (compacts) it now and then. Small writes, but compaction still rewrites everything.
C. One file per memory (memory/<id>.md). Small writes; 18,615 files, slow for git, Spotlight and backups; the largest migration.
D. One JSON Lines file per day (memory/2026/11/2026-11-20.jsonl, by the day a memory was made). 141 files. A new memory rewrites only today's file (median 38 KB). A change to an old memory rewrites its day's file (up to 2.1 MB), 9x smaller than today.
E. (Jørgen, request 4) A today folder with one small file per memory (memory/today/<id>.json); dream rolls each finished day into its dated JSON Lines file as in D. A change to an old memory is written into today as a newer copy (same id, later 'updated' wins) and folded into its day file by dream. So every write is a few hundred bytes in today/, and cleaning up the current day is deleting files. Cost: dream does more (fold changes, delete old retired lines), and reading must merge today/ over the day files.
F. As E but by week: a current-week folder, past weeks as one JSON Lines file each (24 files, median 351 KB, busiest 3.9 MB). Fewer files, larger rewrites when dream folds changes in.

CLEANING UP RETIRED MEMORIES (any option)
- dream deletes retired memories older than N days (30 proposed, a setting). In D, E and F it only rewrites the day or week files that hold them.

CRAB'S READING
- E gives the smallest writes with few files, and keeps today (the part that changes most) easy to tidy. D is the simplest step from today's code. Day files over week files: median 38 KB versus 351 KB.
- Both D and E keep one rule from today: a memory's id never changes, so the shared branch and imports keep working.

OPEN FOR JØRGEN
- Pick A to F (question on check 5).
- How many days a retired memory is kept (check 6).

Previous Purpose: Jørgen 2026-11-18 (request 1 on this task): the goal is a unified, easy-to-navigate, non-confusing CLI and system. The decision behind it is his of 2026-11-11 (memory j9jdkh, docs/DESIGN.md): one tool named hug for memory, tasks and messages, files on branches as the truth, and memory file-oriented too (one Markdown file per memory) so humans can edit it and git merges stay clean.
Previous Principle: Consistency through a clean sweep: once we decide hug is the thing, we change everything at once (commands, help, messages, docs, branches, folders, tests, every project's setup), never a mix of old and new names left behind.
Previous Details: A unified, easy-to-navigate, non-confusing CLI and system: one hug binary with tasks, memories and messages; .hug/<kind>/ on branches hug/<kind>; muninn and huginn kept only as compatibility aliases.

Done: v0.7.0 (2026-11-13) one hug binary, v0.7.1 one hug mcp server, v0.7.3 tasks on branch hug/tasks at .hug/tasks (this repo and seid-lang migrated). Open: memories still live in the SQLite store. On main the sync branch is still muninn-notes (muninn/internal/project/project.go:39); commit c16b7fd6 (2026-11-18, on lane branches, not yet on main) moves project memory to hug/memory through an explicit layout upgrade, and this repo already has a hug/memory branch. Jørgen names it 'memory', not 'memories' (note 2026-11-16). Item 3 (one file per memory, retire = move to _retired/) needs a design talk with him before building. Children: 34r1ft (per-repo memory setting), gav9nw (personal vs shared).

Promoted from item 3 of task:9taa9d.

## Depends on

## Acceptance
- [ ] ~~memories stored one Markdown file per memory, retire = move to _retired/~~ (not relevant: the old one-line placeholder; replaced by checks 2-3 (decision) and the build checks filed after it, crab, 2026-11-20)
- [ ] ~~Jørgen decides: one file per memory, or keep memories.json~~ (not relevant: replaced by check 4, which adds append-only JSON Lines as a third option, crab, 2026-11-20)
- [ ] ~~If one file per memory: Jørgen sets how many days a retired file is kept before hug mem dream deletes it (30 proposed)~~ (not relevant: replaced by check 6, which applies to every layout, not only one file per memory, crab, 2026-11-20)
- [ ] ~~Jørgen picks: keep memories.json as is, make it append-only JSON Lines, or one file per memory~~ (not relevant: replaced by check 5, which adds per-day, today-folder and per-week layouts (Jørgen's requests 3 and 4), crab, 2026-11-20)
- [ ] Jørgen picks a storage layout: A (as is), B (one append-only file), C (file per memory), D (file per day), E (today folder, then day files) or F (as E, by week)
- [ ] Jørgen sets how many days a retired memory is kept before hug mem dream deletes it (30 proposed)

## Requests
- 2026-11-20T13:11 request 1 by jhf:
  I don't understand the technical consequences of this task. And if we move things to retired, then we should later have a plan to clean it up automatically, right? After some default number of days. And I don't know. So it just says, we're going to do this. It doesn't what do we have today? What are we changing? Why are we making that change? And what are the consequences? This has to be more clear. The purpose is too long.
  resolved 2026-11-20T13:12 by crab:
    Rewritten from scratch. Details now have four plain parts: TODAY (one 18.6 MB memories.json on your Mac with 18,078 memories, 491 of them retired and never cleaned up; muninn.db is only the search index), WHAT WOULD CHANGE (one Markdown file per memory, retire = move to _retired/), WHY (edit like a task, clean git merges, same shape as tasks) and CONSEQUENCES (a large one-time migration, every write path changes, and an automatic cleanup: dream deletes retired files after 30 days, a setting). Purpose is one sentence. Title and Goal now say it is a decision. Question 2 asks: do it, or keep one file?
- 2026-11-20T13:39 request 2 by jhf:
  So git merge that is the true benefit right Because the JSON-L is pretty efficient today isn't it? I don't think having editing memories manually is that important. It's also easy to edit a huge file and just open and search in it. So it has to be another technical advantage that you see
  resolved 2026-11-20T13:39 by crab:
    You are right: merging is not the gain. The shared branch file is already one line per memory, so git merges it fine today, and hand editing is not a reason. WHY is rewritten with the one real gain: today every remember or forget rewrites the whole 18.6 MB memories.json under a lock all sessions wait on; small per-memory writes fix that. CONSEQUENCES list what speaks against 18,078 files (git and file-system cost, a large migration). And there is a cheaper way to the same gain: keep one file but make it append-only JSON Lines, compacted by dream. Question 2 now asks: keep as is, append-only JSON Lines, or one file per memory.
- 2026-11-20T17:05 request 3 by jhf:
  Did we consider alternatives here such as date folders and then or per date files with each containing JSON lines something like that?
- 2026-11-20T17:06 request 4 by jhf:
  or keeping one today folder with one file per memory and then as we aggregate it goes into a dated JSON line file for that day. Yes, dreaming and removing old things becomes slower. Writing and cleaning up the current day becomes faster. That's also an option or current week and past weeks is also an option.

## Questions
- 2026-11-20T13:12 question 1 by crab to jhf (check 2):
  Store each memory as its own Markdown file (editable, merges cleanly; a large one-time change of 18,078 files), or keep today's single memories.json? TODAY / WHAT WOULD CHANGE / WHY / CONSEQUENCES are in Details.
  answered 2026-11-20T13:39 by crab:
    Withdrawn by crab 2026-11-20: the WHY was rewritten after Jørgen's request 2; question 2 asks the three-way choice.
- 2026-11-20T13:39 question 2 by crab to jhf (check 4):
  The real gain is smaller local writes (today each remember rewrites the whole 18.6 MB file); git merges are already line-based. Keep memories.json, make it append-only JSON Lines (same gain, one file), or one file per memory (same gain, 18,078 files)?
  answered 2026-11-20T17:08 by crab:
    Withdrawn by crab 2026-11-20: Jørgen's requests 3 and 4 added dated layouts; question 3 asks A to F.
- 2026-11-20T17:08 question 3 by crab to jhf (check 5):
  Six layouts in Details, measured on your 18,615 memories. A day holds 38 KB typically (busiest 2.1 MB); 15% of memories change on a later day. My pick is E: a today folder with one small file per memory, rolled by dream into one JSON Lines file per day. Pick A to F?

## Notes


## History
- 2026-11-20T13:11 request 1 by jhf
- 2026-11-20T13:11 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:11 request 1 acknowledged by crab
- 2026-11-20T13:12 request 1 in progress by crab: hug task set b8mk9s
- 2026-11-20T13:12 todo -> todo/design (ask)
- 2026-11-20T13:12 resolved request 1 by crab
- 2026-11-20T13:39 request 2 by jhf
- 2026-11-20T13:39 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:39 request 2 acknowledged by crab
- 2026-11-20T13:39 request 2 in progress by crab: hug task set b8mk9s
- 2026-11-20T13:39 answered design (check 2), by crab, for crab (todo/design -> todo/ready):
  > Store each memory as its own Markdown file (editable, merges cleanly; a large one-time change of 18,078 files), or keep today's single memories.json? TODAY / WHAT WOULD CHANGE / WHY / CONSEQUENCES are in Details.
  Withdrawn by crab 2026-11-20: the WHY was rewritten after Jørgen's request 2; question 2 asks the three-way choice.
- 2026-11-20T13:39 todo/ready -> todo/design (ask)
- 2026-11-20T13:39 resolved request 2 by crab
- 2026-11-20T17:05 request 3 by jhf
- 2026-11-20T17:05 request 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T17:05 request 3 acknowledged by crab
- 2026-11-20T17:06 request 4 by jhf
- 2026-11-20T17:06 request 4 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T17:07 request 4 acknowledged by crab
- 2026-11-20T17:08 request 3 in progress by crab: hug task set b8mk9s
- 2026-11-20T17:08 request 4 in progress by crab: hug task set b8mk9s
- 2026-11-20T17:08 answered design (question 2, check 4), by crab, for crab (todo/design -> todo/ready):
  > The real gain is smaller local writes (today each remember rewrites the whole 18.6 MB file); git merges are already line-based. Keep memories.json, make it append-only JSON Lines (same gain, one file), or one file per memory (same gain, 18,078 files)?
  Withdrawn by crab 2026-11-20: Jørgen's requests 3 and 4 added dated layouts; question 3 asks A to F.
- 2026-11-20T17:08 todo/ready -> todo/design (ask)
