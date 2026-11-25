---
id: dswkgj
title: "Release v0.7.92: answering never panics"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzxe"
created: 2026-11-20
updated: 2026-11-20
stage: "ready"
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

## Questions
- 2026-11-20T20:31 question 1 by crab to jhf (check 1):
  v0.7.92: answering a withdrawn question now says when and why it was withdrawn and gives the command to ask again; a bare answer after the numbered questions answers the design question. Sample run attached. Do the two refusals read well?
  answered 2026-11-25T09:34 by jhf:
    I think the responses are sensible enough. I do question the CLI. When you have positional arguments, I think it's quite brittle. That's the only thing I notice.

## Notes

file:v0.7.92-answer-sample.txt (2177 bytes, sha256:91951cc3) sample run on v0.7.92: the withdrawn refusal on hpqt99 (board unchanged), a design question answered after a numbered one, and 'no open question' with nothing open

shipped: tag v0.7.92 (e43211fa), release.yml 44801667959 success, brew -> ../Cellar/hug/0.7.92/bin/hug, hug v0.7.92


## History
- 2026-11-20T20:31 todo -> todo/design (ask)
- 2026-11-25T09:34 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.92: answering a withdrawn question now says when and why it was withdrawn and gives the command to ask again; a bare answer after the numbered questions answers the design question. Sample run attached. Do the two refusals read well?
  I think the responses are sensible enough. I do question the CLI. When you have positional arguments, I think it's quite brittle. That's the only thing I notice.
- 2026-11-25T09:34 answer 1 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
