---
id: yphbra
title: "Asking a person about a todo task: ask works without taking it"
state: "done"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzs"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "askfree"
worktree: "/Users/jhf/.jcode/scratch/askfree"
done_at: "2026-11-18T23:24"
done_release: "v0.7.71"
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
- [x] hug task ask on a todo task that depends on an unfinished task records the question, leaves state todo, assignee and the dependency unchanged, and exits 0 (CLI test)
- [x] the asked person sees it in ASKED with its answer action, and answering delivers the answer to the asker (test)
- [x] MCP task_update ask does the same on a todo task (test)
- [x] ask on a doing task behaves as today (existing tests stay green)

## Notes


Design, crab rulings 21:59 local. (1) ask on a todo task appends a numbered question to iwazaru's ## Questions (same writer and format); state, assignee and deps unchanged; design_* mirror the oldest open question; the stage derives as before (depends stays depends). (2) answer/answer N/tick by the person asked answers it, for the asker. (3) DISTINCTION for Jørgen to see and overrule: a question asked with hug task ask is in the person's ASKED at once, even while a dependency outranks design (the asker chose to ask now); a hug task design question keeps the 11-17 order (deferred, depends, design) and waits for its dependency. (4) withdrawOpen leaves a todo task's open questions open. (5) MCP ask goes through the same path. Take (ruling A): open questions come along, so the task becomes doing/asked with asked_* mirroring them, and nothing is withdrawn. --after on a todo task is refused (it parks the asker's own doing task). busy.go untouched.
check 1: b51a2cda (tests 4cb68eb0 failing first on 8a45b0eb: every ask on a todo task refused 'todo -> doing/asked is not a move a command makes'); cli asktodo_test.go TestAskOnBlockedTodoTask: state todo, stage depends, assignee and dep kept, question 1 in ## Questions, exit 0; mutants M1 (withdraw guard) and M7 (askable) fail it
check 2: b51a2cda (tests 4cb68eb0 failing first on 8a45b0eb: every ask on a todo task refused 'todo -> doing/asked is not a move a command makes'); TestAnswerOnTodoTaskGoesToAsker (answer 2 and tick on check 1, 'answered design (question N, check N), by jhf, for piglet', show answer.for piglet) and tui TestAskedTodoQuestionInAsked (in jhf's ASKED while blocked, a answers it; an outranked hug task design question is not in ASKED); mutants M3, M4, M5, M6 each fail
check 3: b51a2cda (tests 4cb68eb0 failing first on 8a45b0eb: every ask on a todo task refused 'todo -> doing/asked is not a move a command makes'); mcp asktodo_test.go TestAskTodoTaskOverMCP: task_update ask on a blocked todo task by a non-holder, stays todo/depends, answer for piglet; M7 fails it
check 4: full huginn suite niced -count=1 on b51a2cda: 11 ok, 0 FAIL (cli 304.871s, tui 168.284s); existing ask/answer/q2adr8 tests unchanged
landed 89ce012 on main 2026-11-18: task, cli, mcp, tui: failing tests, ask on a todo task without taking it; take…
landed 6448e7d on main 2026-11-18: task, cli, mcp, tui: hug task ask on a todo task without taking it; take keeps…
shipped in v0.7.71 (89ce012)
## History
- 2026-11-18T21:54 todo -> doing/active (take)
- 2026-11-18T22:23 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> done (done, by release v0.7.71: every check holds); was doing_since: 2026-11-18T21:54; waiting_asked_by: piglet; waiting_on: CI run 44400965600 on b51a2cda (ci/yphbra), then crab's review and landing; waiting_since: 2026-11-18T22:23
