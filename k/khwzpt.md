---
id: khwzpt
title: "hug task answer panics when the stage's question is not numbered"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
A bare hug task answer on a design question outside ## Questions answers it or refuses with a fix: line, never panics.

## Purpose
A Go stack trace is the worst answer a person or agent can get.

## Principle
Every refusal names what is wrong and ends with a fix: line; whether the design stage's own question is answered (q=0) or refused is crab's call.

## Relevant details
Found on branch 5vehza (2026-11-20 19:52 local): new task, hug task ask ID --to jhf 'first?' --as piglet; hug task answer ID yes --as jhf; hug task design ID 'second design?' --for jhf; hug task answer ID ok --as jhf -> panic: runtime error: index out of range [0] with length 0 at huginn/internal/task/asks.go:309 (pickQuestion, openNs[0] with no open numbered question). AnswerQuestion (answer.go:245) calls pickQuestion whenever len(Questions) > 0, though the design question lives in design_* fields, not ## Questions.

## Depends on

## Acceptance
- [ ] A test runs the repro and gets either the design question answered or exit 2 with a fix: line, no panic (fails first on main with the panic)

## Notes
