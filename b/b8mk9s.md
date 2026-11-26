---
id: b8mk9s
title: "Design: how hug stores memories, so many agents can write at once"
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
stage: "ready"
---

## Goal
Jørgen picks how hug stores memories, judged on one truth, small writes, many writers at once, one way, and stable links.

## Purpose
Agents writing memories at the same time have lost writes, failed and waited on each other, and a link to a memory must not break.

## Principle
Judge every layout on the same five needs, with the record; pick one way of storing, not several; nothing is built until Jørgen picks.

## Relevant details
YOUR FIVE NEEDS (Jørgen 2026-11-25 09:45): one source of truth; small writes; many agents writing at once; one way of doing it (consistent, easy to test); stable links to memories.

WHAT WENT WRONG WITH MANY WRITERS (the record on your Mac)
- 2026-11-10: two writers at once lost a memory (29 of 30 survived). Patched by a lock: every write now queues (muninn/internal/ingest/lock.go).
- 2026-11-16, seid: 233 of 239 forgets failed with "database is locked" (SQLITE_BUSY). Patched by a second queue, the write turn (muninn/internal/store/turn.go).
- hook.log: 70 captures failed with class=timeout (47 on 11-06, 11 on 11-07, 3 on 11-18, 7 on 11-19, 2 on 11-22); 6 "reindex: database is locked" on 11-04.
- 2026-11-06: duplicate repair changed 74 memory ids (native-repairs.log), so links to those 74 broke.
So the fixes so far are queues. Agents no longer lose writes, but they wait for each other, and each write still rewrites the whole file.

TODAY
- Two stores: memories.json (21.4 MB, 21,949 memories, 538 retired, on 2026-11-25) is the truth; muninn.db (896 MB) is the search index, rebuilt from it, and can fall behind it.
- Every remember, forget or tidy reads and rewrites all 21.4 MB, holding a lock the whole time, and then waits for the index's write turn.
- A memory's link is its 6-letter id (21,948 of 21,949), made from where it came from, so it does not change on edit.

THE OPTIONS, AGAINST YOUR NEEDS
| | One truth | Size of one write | Many writers at once | One way | Stable links |
|---|---|---|---|---|---|
| A as is | json + index | 21.4 MB | all queue on one lock | yes | id |
| B one append-only file | file + index | one line; compaction rewrites all | appends barely wait; compaction stops everyone | two: append, compact | id |
| C one file per memory | files + index | about 1 KB | two different memories never wait for each other | yes | id = file name |
| D one file per day | files + index | today's file, up to 2.1 MB | same-day writers queue | yes | id, but where it lives depends on its date |
| E today folder, then day files | files + index | about 1 KB | today never waits; dream's folding stops everyone | two: today, day files | id |
| F as E, by week | files + index | about 1 KB; folds up to 3.9 MB | as E | two | id |
| G no memories.json: muninn.db is the truth | yes, one store | one row | one writer at a time, short turns | yes | id |
In every option A to F, the index update still waits for the write turn, so the index is the next queue to shorten.

CRAB'S PICK: C
- It is the only file layout where agents writing different memories never wait for each other. A write is one small file, replaced in one step.
- The file name is the id, so a link is a path that never moves; retire moves it to _retired/.
- One way: no folding or compaction step to get right and test.
- Cost: 21,949 small files (Spotlight and Time Machine see many files), and a one-time migration.
- G is the choice if one store matters more than files; it goes against your 2026-11-11 decision that files are the truth (memory j9jdkh).

CLEANING UP RETIRED MEMORIES (any option)
- dream deletes retired memories older than N days (30 proposed, a setting; check 6).

Earlier version, kept as record: the six layouts A-F with day and week sizes measured on 2026-11-20 (141 days; median day 38 KB, busiest 2.1 MB; median week 351 KB, busiest 3.9 MB); 2,825 of 18,615 memories (15%) changed on a later day. Out of scope: the shared hug/memory branch file (memories.jsonl, one line per memory), which git already merges line by line.

## Depends on

## Acceptance
- [ ] ~~memories stored one Markdown file per memory, retire = move to _retired/~~ (not relevant: the old one-line placeholder; replaced by checks 2-3 (decision) and the build checks filed after it, crab, 2026-11-20)
- [ ] ~~Jørgen decides: one file per memory, or keep memories.json~~ (not relevant: replaced by check 4, which adds append-only JSON Lines as a third option, crab, 2026-11-20)
- [ ] ~~If one file per memory: Jørgen sets how many days a retired file is kept before hug mem dream deletes it (30 proposed)~~ (not relevant: replaced by check 6, which applies to every layout, not only one file per memory, crab, 2026-11-20)
- [ ] ~~Jørgen picks: keep memories.json as is, make it append-only JSON Lines, or one file per memory~~ (not relevant: replaced by check 5, which adds per-day, today-folder and per-week layouts (Jørgen's requests 3 and 4), crab, 2026-11-20)
- [ ] ~~Jørgen picks a storage layout: A (as is), B (one append-only file), C (file per memory), D (file per day), E (today folder, then day files) or F (as E, by week)~~ (not relevant: replaced by check 7: Jørgen 2026-11-25 asked that the layouts be judged on his five needs, with G (muninn.db as the one truth) added, crab, crab, 2026-11-25)
- [ ] Jørgen sets how many days a retired memory is kept before hug mem dream deletes it (30 proposed)
- [ ] Jørgen picks a layout, A to G, judged on his five needs (table in Details)

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
  resolved 2026-11-20T17:08 by crab:
    Yes, now considered. Details list six layouts, measured on your memories.json: D is one JSON Lines file per day (141 files, a median day 38 KB, busiest 2.1 MB), so a new memory rewrites only today's file. Details also note what any dated layout must handle: 2,825 memories (15%) change on a later day. Question 3 asks you to pick.
- 2026-11-20T17:06 request 4 by jhf:
  or keeping one today folder with one file per memory and then as we aggregate it goes into a dated JSON line file for that day. Yes, dreaming and removing old things becomes slower. Writing and cleaning up the current day becomes faster. That's also an option or current week and past weeks is also an option.
  resolved 2026-11-20T17:08 by crab:
    Added as option E (today folder with one small file per memory; dream rolls finished days into a dated JSON Lines file, and changes to old memories wait in today as newer copies until dream folds them in) and F (the same by week: 24 files, median 351 KB, busiest 3.9 MB). Title, Goal and Purpose now name the real gain: small writes. Check 5 is the A-F pick, check 6 the retired-days setting for any layout. My pick is E (question 3).
- 2026-11-26T08:20 request 5 by jhf:
  I wonder if we can do something crazy. And by crazy, I mean somehow group the memories so that we get an SQ light database committed with the memories so that instead of having one big SQLite database for all and then you have to scan everything, you can, when things are not changing, just read the SQLite database and it synchronizes with its subdirectory. Bit like a fan-out tree. And then you open a reduced set of SQLite databases and aggregate from them. I'm not sure that even makes sense

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
  answered 2026-11-25T09:45 by jhf:
    I can't really answer that yet. This task was created and it only says we can do each one of the things we can do, but I don't see argument for why one thing is better than another thing, meaning a single source of truth, less I.O. to make edits, easier to coordinate between multiple agents writing at the same time, and so on and so forth. A single way of doing it, so it's more consistent and easier to test, or multiple ways of doing it. So it's like we have lined up all the choices, but we haven't lined up why one choice would give us something that another choice wouldn't. And I think the reasoning is also incorrect. I think I experienced problems of writing memories with multiple different clients at the same time. And I thought there is concurrency issues. We are waiting. Maybe we should have a better file design. And maybe we should have a design where we can link to memories and they're stable, these links.
- 2026-11-25T10:22 question 4 by crab to jhf (check 7):
  Rewritten from your answer: Details now judges each layout on your five needs, starting with the many-writer failures on your Mac (a lost memory on 11-10, 233 of 239 forgets failing on 11-16, 70 capture timeouts). My pick is C, one small file per memory: agents writing different memories never wait for each other, and the id is the file name, so links stay stable. Pick A to G?
  answered 2026-11-26T09:17 by jhf:
    C: one file per memory, in subdirectories so no directory holds too many files, with two databases: memories (many writers, short writes) and a derived index (one writer, built beside and swapped in). Jørgen 2026-11-26 08:16 in chat.

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
- 2026-11-20T17:08 resolved request 3 by crab
- 2026-11-20T17:08 resolved request 4 by crab
- 2026-11-25T09:45 answered design (question 3, check 5), by jhf, for crab (todo/design -> todo/ready):
  > Six layouts in Details, measured on your 18,615 memories. A day holds 38 KB typically (busiest 2.1 MB); 15% of memories change on a later day. My pick is E: a today folder with one small file per memory, rolled by dream into one JSON Lines file per day. Pick A to F?
  I can't really answer that yet. This task was created and it only says we can do each one of the things we can do, but I don't see argument for why one thing is better than another thing, meaning a single source of truth, less I.O. to make edits, easier to coordinate between multiple agents writing at the same time, and so on and so forth. A single way of doing it, so it's more consistent and easier to test, or multiple ways of doing it. So it's like we have lined up all the choices, but we haven't lined up why one choice would give us something that another choice wouldn't. And I think the reasoning is also incorrect. I think I experienced problems of writing memories with multiple different clients at the same time. And I thought there is concurrency issues. We are waiting. Maybe we should have a better file design. And maybe we should have a design where we can link to memories and they're stable, these links.
- 2026-11-25T09:45 answer 3 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
- 2026-11-25T10:22 crab read the answer: hug task set b8mk9s
- 2026-11-25T10:22 todo/ready -> todo/design (ask)
- 2026-11-26T08:20 request 5 by jhf
- 2026-11-26T08:20 request 5 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-26T09:17 answered design (question 4, check 7), by jhf, for crab (todo/design -> todo/ready):
  > Rewritten from your answer: Details now judges each layout on your five needs, starting with the many-writer failures on your Mac (a lost memory on 11-10, 233 of 239 forgets failing on 11-16, 70 capture timeouts). My pick is C, one small file per memory: agents writing different memories never wait for each other, and the id is the file name, so links stay stable. Pick A to G?
  C: one file per memory, in subdirectories so no directory holds too many files, with two databases: memories (many writers, short writes) and a derived index (one writer, built beside and swapped in). Jørgen 2026-11-26 08:16 in chat.
- 2026-11-26T09:17 answer 4 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
