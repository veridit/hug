---
id: dswkgj
title: "Release v0.7.92: answering never panics"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-20
updated: 2026-11-20
design_question: "v0.7.92: answering a withdrawn question now says when and why it was withdrawn and gives the command to ask again; a bare answer after the numbered questions answers the design question. Sample run attached. Do the two refusals read well?"
design_for: "jhf"
design_since: "2026-11-20T20:31"
design_asked_by: "crab"
stage: "design"
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

## Notes

file:v0.7.92-answer-sample.txt (2177 bytes, sha256:91951cc3) sample run on v0.7.92: the withdrawn refusal on hpqt99 (board unchanged), a design question answered after a numbered one, and 'no open question' with nothing open

shipped: tag v0.7.92 (e43211fa), release.yml 44801667959 success, brew -> ../Cellar/hug/0.7.92/bin/hug, hug v0.7.92


## History
- 2026-11-20T20:31 todo -> todo/design (ask)
