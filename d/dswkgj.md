---
id: dswkgj
title: "Release v0.7.92: answering never panics"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Jørgen confirms v0.7.92: hug task answer refuses a withdrawn question and answers a design question, never panicking.

## Purpose
A Go stack trace is the worst reply a person or agent can get, and over MCP it killed the hug mcp server.

## Principle
One question per shipped task; a sample run is attached so it can be judged without installing.

## Relevant details
Shipped: 5vehza (withdrawn question: exit 2, 'question N on ID was withdrawn at AT (REASON)', fix: hug task ask ...) and khwzpt (a bare answer after the numbered questions are done answers the design question; with nothing open: 'no open question on ID', fix: hug task show ID). Commits ea218b3d, 13ce8145; CI 44799862741 8/8. Before, both panicked: asks.go:317 nil pointer, asks.go:309 index out of range.

## Depends on

## Acceptance
- [ ] Jørgen confirms the two refusals read well

## Notes

file:v0.7.92-answer-sample.txt (2177 bytes, sha256:91951cc3) sample run on v0.7.92: the withdrawn refusal on hpqt99 (board unchanged), a design question answered after a numbered one, and 'no open question' with nothing open
