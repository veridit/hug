---
id: m5pbb0
title: "hug tui: a later open question on a task cannot be reached to answer"
state: "doing"
type: bug
priority: ""
assignee: ["blossom"]
labels: []
group: ""
rank: "zzzzzzzzzzzs"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T10:06"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In hug tui every open question on a task can be reached and answered with its own action, whichever is first.

## Purpose
Jørgen 2026-11-19 09:05Z on p87ngv (request 1): 'I can't get the second question here', so he wrote a request instead of answering question 2. p87ngv then held 5 open questions to jhf (1 answered at 09:05Z), 4 struck checks and 6 open ones. q2adr8 check 3 (v0.7.70) promised each open question its own answer action (golden at 100x24 with 3 questions).

## Principle
Reproduce first on a copy of p87ngv's file at 1a5fd862..1026de8f (its state when he wrote the request), at his terminal size if known, then a failing test, then the fix. No guessing at the cause.

## Relevant details
Leads to check: the first question's answer moving focus (stage shows 'question for jhf: <q1>'); a long ticket pushing later questions below the fold; Tab stops after struck checks; questions asked in a burst within one second. Golden from q2adr8: TestSeveralQuestionsEachAnswered.

## Depends on

## Acceptance
- [x] A failing test reproduces it on p87ngv's file as it was at 09:05Z
- [ ] After the fix every open question in that file has a reachable answer action in hug tui (golden at 100x30 and 160x45)
- [ ] Jørgen answers a second question in hug tui after the release

## Notes


check 1: TestOldAnswerLeavesEveryQuestionAnswerable (huginn/internal/tui/m5pbb0_test.go, commit c533de89) replays p87ngv as at hug/tasks bab152ff (10:05:26, after jhf's v0.7.69 TUI answered q1): without the fix both 100x30 and 160x45 fail 'm5pbb0_test.go:52: question 2 not answered (message "usage p87ngv: p87ngv asks nobody: its stage is not design, asked or review\nfix: hug task show p87ngv")'. Store test TestAnswerOpenQuestionWithoutMirror fails the same on todo and doing. Running TUI pid 78631 maps /opt/homebrew/Cellar/hug/0.7.69/bin/hug (vmmap), q2adr8 shipped in v0.7.70.
## History
- 2026-11-19T10:06 todo -> doing/active (take)
