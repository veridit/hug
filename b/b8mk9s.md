---
id: b8mk9s
title: "Design: store each memory as its own file instead of one big JSON file"
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
---

## Goal
Jørgen decides whether hug keeps memories in one JSON file or one Markdown file per memory, and how retired ones are cleaned up.

## Purpose
Memories you can open, edit and diff like task files, with git merging them cleanly.

## Principle
Decide on the consequences first; nothing is built until Jørgen picks.

## Relevant details
TODAY
- All memories are one file on your Mac: ~/Library/Application Support/muninn/memories.json (18.6 MB, 18,078 memories: 17,587 active, 491 retired). muninn.db (805 MB) is a search index built from it.
- To read or fix one memory you use hug mem search / forget; the file is not meant for hand editing.
- Retiring a memory (forget, dedupe) sets active: false in that same file. Retired ones stay forever; nothing cleans them up.
- A shared repo gets one memories.jsonl on its hug/memory branch: one line per memory, so two machines adding memories conflict on the same file.

WHAT WOULD CHANGE
- Each memory becomes one Markdown file, e.g. memory/<id>.md, with its fields at the top and its text below, like a task file.
- Retiring moves the file to memory/_retired/<id>.md instead of setting a flag.
- muninn.db stays the search index, rebuilt from the files.

WHY
- You can open, read and edit a memory in any editor, like a task.
- git merges two machines' new memories cleanly (different files), and a diff shows exactly which memory changed.
- Same shape as tasks (one file per thing), which is the 2026-11-11 decision: one tool, files as the truth.

CONSEQUENCES
- 18,078 small files instead of one: a one-time migration, with a dry-run and a backup first.
- Writing gets a little slower (one file per write); search stays the same (the index).
- Every code path that reads or writes memories.json changes: capture, forget, dedupe, rescope, import, sync. This is a large change.
- Retired files pile up. Proposal: hug mem dream deletes files in _retired/ older than 30 days (a setting); git history keeps them if the folder is on a branch.

OPEN FOR JØRGEN
- Do it, or keep one file?
- If yes: retired files deleted after how many days (30 proposed)?

Previous Purpose: Jørgen 2026-11-18 (request 1 on this task): the goal is a unified, easy-to-navigate, non-confusing CLI and system. The decision behind it is his of 2026-11-11 (memory j9jdkh, docs/DESIGN.md): one tool named hug for memory, tasks and messages, files on branches as the truth, and memory file-oriented too (one Markdown file per memory) so humans can edit it and git merges stay clean.
Previous Principle: Consistency through a clean sweep: once we decide hug is the thing, we change everything at once (commands, help, messages, docs, branches, folders, tests, every project's setup), never a mix of old and new names left behind.
Previous Details: A unified, easy-to-navigate, non-confusing CLI and system: one hug binary with tasks, memories and messages; .hug/<kind>/ on branches hug/<kind>; muninn and huginn kept only as compatibility aliases.

Done: v0.7.0 (2026-11-13) one hug binary, v0.7.1 one hug mcp server, v0.7.3 tasks on branch hug/tasks at .hug/tasks (this repo and seid-lang migrated). Open: memories still live in the SQLite store. On main the sync branch is still muninn-notes (muninn/internal/project/project.go:39); commit c16b7fd6 (2026-11-18, on lane branches, not yet on main) moves project memory to hug/memory through an explicit layout upgrade, and this repo already has a hug/memory branch. Jørgen names it 'memory', not 'memories' (note 2026-11-16). Item 3 (one file per memory, retire = move to _retired/) needs a design talk with him before building. Children: 34r1ft (per-repo memory setting), gav9nw (personal vs shared).

Promoted from item 3 of task:9taa9d.

## Depends on

## Acceptance
- [ ] memories stored one Markdown file per memory, retire = move to _retired/
- [ ] Jørgen decides: one file per memory, or keep memories.json
- [ ] If one file per memory: Jørgen sets how many days a retired file is kept before hug mem dream deletes it (30 proposed)

## Requests
- 2026-11-20T13:11 request 1 by jhf:
  I don't understand the technical consequences of this task. And if we move things to retired, then we should later have a plan to clean it up automatically, right? After some default number of days. And I don't know. So it just says, we're going to do this. It doesn't what do we have today? What are we changing? Why are we making that change? And what are the consequences? This has to be more clear. The purpose is too long.

## Notes


## History
- 2026-11-20T13:11 request 1 by jhf
- 2026-11-20T13:11 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:11 request 1 acknowledged by crab
- 2026-11-20T13:12 request 1 in progress by crab: hug task set b8mk9s
