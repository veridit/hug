---
id: 5vehza
title: "hug task answer panics on a withdrawn question"
state: "done"
type: bug
priority: "high"
assignee: ["dove"]
labels: []
group: ""
rank: "zzzzzzzzzzzg"
created: 2026-11-20
updated: 2026-11-20
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "dswkgj"
done_at: "2026-11-20T20:23"
done_release: "v0.7.92"
---

## Goal
Answering a withdrawn question says it was withdrawn, when and why, with a fix: line, instead of crashing.

## Purpose
A crash with a Go stack trace is the worst answer a person or agent can get.

## Principle
Every refusal names what is wrong and ends with a fix: line.

## Relevant details
Repro on v0.7.90, 2026-11-20 12:44Z: hug task answer hpqt99 1 --as crab "..." -> panic: runtime error: invalid memory address or nil pointer dereference at huginn/internal/task/asks.go:317 (pickQuestion). Question 1 on hpqt99 has withdrawn {at 2026-11-18T23:32, text 'doing/asked -> doing/active'} and no answer; q.Open() is false, so the code reads q.Answer.By on a nil Answer. Fix: when q.Withdrawn != nil, refuse with 'question N on ID was withdrawn at AT (REASON)' and fix: hug task ask ID --to NAME "..." to ask again; also guard Answer nil in the answered branch.

## Depends on

## Acceptance
- [x] A test answers a withdrawn question and gets exit 2 with 'was withdrawn' and a fix: line, no panic (fails first on main with the panic)
- [x] No other read of q.Answer in huginn/internal/task assumes it is set when Open() is false (grep listed in a note)

## Notes


q.Answer reads in huginn/ (task, cli, mcp, tui; non-test), branch 5vehza: task/asks.go:54 Open() nil-compare only, safe; asks.go:86 nil-compare, safe; asks.go:93 assignment, safe; asks.go:104 nil-compare, safe; asks.go:120-121 q.Answer.Text guarded by 'if q.Answer != nil' at 120; asks.go:317-318 q.Answer.By/At now guarded by 'case q.Answer != nil' (was '!q.Open()', the 5vehza panic; withdrawn is its own case at 319-322); cli/askquestions.go:30-32 By/At/Text guarded by 'if q.Answer != nil' at 30; cli/askquestions.go:38 nil-compare, safe; tui/detail.go:1242 'case q.Answer != nil', safe; tui/detail.go:1253 nil-compare, safe; tui/detail.go:1259 'if a := q.Answer; a != nil', safe. mcp: no q.Answer reads (task_update answer runs cli Execute -> AnswerQuestion -> pickQuestion; TestAnswerWithdrawnQuestionOverMCP covers it). Withdrawn.At reads: tui/detail.go:1247 under 'case q.Withdrawn != nil' at 1244; asks.go:322 under 'case q.Withdrawn != nil'. Task-level t.Answer (a different field): answer.go:184 and store.go:776 guarded by t.Answer != nil / AnswerFor. grep: grep -rnE '\b[a-zA-Z_]+\.Answer\b' huginn/internal/{task,cli,mcp,tui} --include=*.go | grep -v _test.go
check 1: TestAnswerWithdrawnQuestionRefused (huginn/internal/cli/askwithdrawn_test.go:40) and TestAnswerWithdrawnQuestionOverMCP (huginn/internal/mcp/askwithdrawn_test.go:18). On origin/main 00a06a61 both failed with 'panic: runtime error: invalid memory address or nil pointer dereference' at huginn/internal/task/asks.go:317 (pickQuestion); over MCP the panic killed the server. Fix ea218b3d: exit 2, 'question 1 on ID was withdrawn at AT (doing/asked -> doing/active)', ends 'fix: hug task ask ID --to jhf --check 1 "Is 2kb574 a leak?"'. Mutants (asks.go restored, shasum 6af8b222 before and after): withdrawn case dropped -> both tests panic; fix line hug task show -> both fail; reason dropped -> cli test fails. Branch 5vehza at 13ce8145da04e1eeeb9fc002145e26741335bc7c, CI run 44799862741: all 8 jobs success.
check 2: q.Answer reads listed in the note, file:line each: all 13 non-test reads in task/cli/tui are nil-compares or guarded by an Answer != nil check; mcp has none (it goes through AnswerQuestion). The only unguarded one, asks.go:317 (the '!q.Open()' branch), is now 'case q.Answer != nil' in ea218b3d.
landed ea218b3 on main 2026-11-20: hug task answer refuses a withdrawn question instead of panicking (5vehza)
shipped in v0.7.92 (ea218b3)
Reflection (open, 2026-11-20, crab): nothing: the task named the line and the fix
## History
- 2026-11-20T19:50 todo -> doing/active (take)
- 2026-11-20T20:23 doing/active -> done (done, by release v0.7.92: every check holds); was doing_since: 2026-11-20T19:50
