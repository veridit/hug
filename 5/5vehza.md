---
id: 5vehza
title: "hug task answer panics on a withdrawn question"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-20
updated: 2026-11-20
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
- [ ] A test answers a withdrawn question and gets exit 2 with 'was withdrawn' and a fix: line, no panic (fails first on main with the panic)
- [ ] No other read of q.Answer in huginn/internal/task assumes it is set when Open() is false (grep listed in a note)

## Notes
