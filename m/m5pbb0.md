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
- [x] After the fix every open question in that file has a reachable answer action in hug tui (golden at 100x30 and 160x45)
- [ ] Jørgen answers a second question in hug tui after the release

## Notes


check 1: TestOldAnswerLeavesEveryQuestionAnswerable (huginn/internal/tui/m5pbb0_test.go, commit c533de89) replays p87ngv as at hug/tasks bab152ff (10:05:26, after jhf's v0.7.69 TUI answered q1): without the fix both 100x30 and 160x45 fail 'm5pbb0_test.go:52: question 2 not answered (message "usage p87ngv: p87ngv asks nobody: its stage is not design, asked or review\nfix: hug task show p87ngv")'. Store test TestAnswerOpenQuestionWithoutMirror fails the same on todo and doing. Running TUI pid 78631 maps /opt/homebrew/Cellar/hug/0.7.69/bin/hug (vmmap), q2adr8 shipped in v0.7.70.
check 2: With fix 42b71005 (+dd6e5cf9): TestOldAnswerLeavesEveryQuestionAnswerable (p87ngv@bab152ff, 3 open) and TestFiveQuestionsEachReachable (p87ngv@19977d7d, 5 open) pass at 100x30 and 160x45: answerActions == open count, Tab reaches each block, a on question 2 writes '  answered 2026-11-19T10:06 by jhf:' under it alone. Goldens testdata/detail/p87ngv-{old-answer,five-questions}-{100x30,160x45}.txt. Local go test ./... exit 0 (40 ok). CI run 44500113493 on ci/m5pbb0 success (ubuntu, macos, windows).
Fix on branch m5pbb0 (c533de89 tests, 42b71005 fix, dd6e5cf9 CRLF test), CI 44500113493 green. Not merged/released. Jørgen's TUI pid 78631 is v0.7.69 (vmmap); restart it. Report /Users/jhf/.jcode/scratch/m5pbb0/REPORT.md
crab 10:02Z: reviewed (answer.go question() fallback on OpenQuestions, askStage; tests c533de89 fail first; goldens 100x30/160x45). Main had not moved since branch: fast-forwarded main to dd6e5cf9 (CI 44500113493 8/8 at that exact SHA). Released v0.7.80 (CHANGELOG b2c9d5e4, merge 41dc277d). Check 3 is Jørgen's: restart hug tui (pid 78631 was v0.7.69), answer a second question.
## History
- 2026-11-19T10:06 todo -> doing/active (take)
