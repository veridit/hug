---
id: yphbra
title: "Asking a person about a todo task: ask works without taking it"
state: "doing"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:54"
stage: "active"
model: "claude-opus-5-5"
branch: "askfree"
worktree: "/Users/jhf/.jcode/scratch/askfree"
---

## Goal
Anyone can ask a person a question on a todo or blocked task (hug task ask ID --to NAME [--check N] [--show FILE], MCP task_update ask), and it shows in that person's ASKED with an answer action, without the asker taking the task into doing and without lifting its dependencies.

## Purpose
piglet 2026-11-18 on ec6z6j: to ask Jørgen one design question on check 1 of a todo task blocked by rxr4z8, take refused ('blocked by rxr4z8') and ask refused ('todo -> doing/asked is not a move a command makes'); the workaround was to remove the dependency, take, ask, and re-add it. A design question must not need the work to be claimable.

## Principle
Asking is not claiming: the task's state, assignee and dependencies stay as they were; the question carries who asked. One question per ask (q2adr8). Answering returns the answer to the asker, not to a task holder. Do not touch busy.go (byj479, iwazaru). Do not change the ## Questions format without iwazaru.

## Relevant details
Ask was rebuilt by iwazaru in q2adr8 (## Questions, d5c66a1f/fbf06637). Today ask goes through the doing -> asked move; todo has no such move. Take checks dependencies (busy.go is off limits). Paths: huginn/internal/task (ask), internal/cli, internal/mcp, tui ASKED.

## Depends on

## Acceptance
- [ ] hug task ask on a todo task that depends on an unfinished task records the question, leaves state todo, assignee and the dependency unchanged, and exits 0 (CLI test)
- [ ] the asked person sees it in ASKED with its answer action, and answering delivers the answer to the asker (test)
- [ ] MCP task_update ask does the same on a todo task (test)
- [ ] ask on a doing task behaves as today (existing tests stay green)

## Notes


## History
- 2026-11-18T21:54 todo -> doing/active (take)
