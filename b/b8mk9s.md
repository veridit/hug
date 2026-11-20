---
id: b8mk9s
title: "memories stored one Markdown file per memory, retire = move to _retired/"
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
memories stored one Markdown file per memory, retire = move to _retired/

## Purpose
Jørgen 2026-11-18 (request 1 on this task): the goal is a unified, easy-to-navigate, non-confusing CLI and system. The decision behind it is his of 2026-11-11 (memory j9jdkh, docs/DESIGN.md): one tool named hug for memory, tasks and messages, files on branches as the truth, and memory file-oriented too (one Markdown file per memory) so humans can edit it and git merges stay clean.

## Principle
Consistency through a clean sweep: once we decide hug is the thing, we change everything at once (commands, help, messages, docs, branches, folders, tests, every project's setup), never a mix of old and new names left behind.

## Relevant details
A unified, easy-to-navigate, non-confusing CLI and system: one hug binary with tasks, memories and messages; .hug/<kind>/ on branches hug/<kind>; muninn and huginn kept only as compatibility aliases.

Done: v0.7.0 (2026-11-13) one hug binary, v0.7.1 one hug mcp server, v0.7.3 tasks on branch hug/tasks at .hug/tasks (this repo and seid-lang migrated). Open: memories still live in the SQLite store. On main the sync branch is still muninn-notes (muninn/internal/project/project.go:39); commit c16b7fd6 (2026-11-18, on lane branches, not yet on main) moves project memory to hug/memory through an explicit layout upgrade, and this repo already has a hug/memory branch. Jørgen names it 'memory', not 'memories' (note 2026-11-16). Item 3 (one file per memory, retire = move to _retired/) needs a design talk with him before building. Children: 34r1ft (per-repo memory setting), gav9nw (personal vs shared).

Promoted from item 3 of task:9taa9d.

## Depends on

## Acceptance
- [ ] memories stored one Markdown file per memory, retire = move to _retired/

## Requests
- 2026-11-20T13:11 request 1 by jhf:
  I don't understand the technical consequences of this task. And if we move things to retired, then we should later have a plan to clean it up automatically, right? After some default number of days. And I don't know. So it just says, we're going to do this. It doesn't what do we have today? What are we changing? Why are we making that change? And what are the consequences? This has to be more clear. The purpose is too long.

## Notes


## History
- 2026-11-20T13:11 request 1 by jhf
- 2026-11-20T13:11 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
