---
id: qcq3tm
title: "Opening an asked task scrolls to its question; nothing inlined on top"
state: "doing"
type: feature
priority: "high"
assignee: ["cricket"]
labels: []
group: ""
rank: "zzzzzzzzzzzq"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-20T10:59"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Opening a task that asks you something lands on the question itself, and the file it names is a link, never drawn above the page.

## Purpose
Jørgen reported this three times (hjbfzd 08:51Z, tb0dra 09:58Z, 911j70 08:20Z): the --show file (asked_show) is drawn inline at the top, the real question is only found by scrolling, and the drawn block vanishes after scrolling away and back.

## Principle
Show the question where it lives; follow a link to see a file; nothing is drawn on top of the page for a question.

## Relevant details
Split from 8a0ybb checks 3 and 8 so it ships before the FOR YOU layout decision (8a0ybb question 3). Code: huginn/internal/tui picture.go questionPictureOf (asked_show drawn above the page), detail open position. The question row shows its file as a dim link line ('→ capture-wording-100x30.ans · Enter shows'); Enter opens it in the file frame, Esc returns to the question. Evidence boards: hjbfzd, tb0dra, 911j70 (asked_show set). Scroll markers (↑/↓ corners) stay in 8a0ybb check 9.

## Depends on

## Acceptance
- [ ] Opening a task with an open question to you scrolls so the question is in view and focused, wherever it sits on the page (tmux test, a question below the fold)
- [ ] A question with --show draws no file above the page; the question shows the file as a link line, Enter opens it in the file frame, Esc returns to the question (tmux test)
- [ ] Jørgen opens tb0dra-like asked tasks after the release and lands on the question
- [ ] On p87ngv's board copy (doing/asked, old asked_* stage fields plus ## Questions 11 and 12), pressing a opens the answer box in view under the question it answers, with the cursor in it; typing shows text (tmux test; Jørgen 2026-11-20 10:13Z: a did nothing visible on p87ngv)

## Notes


file:before-tb0dra-100x30.ans (2975 bytes, sha256:d2e372aa) origin/main 4b1a5781, tb0dra as opened: agentsection-proposal.md drawn above the page, the question off screen
file:after-tb0dra-100x30.ans (3927 bytes, sha256:4c747c53) qcq3tm 15667952, tb0dra as opened: lands on check 5's question, file as link line
file:before-hjbfzd-100x30.ans (5337 bytes, sha256:45b53014) origin/main 4b1a5781, hjbfzd as opened: capture-wording-100x30.ans drawn above the page
## History
- 2026-11-20T10:59 todo -> doing/active (take)
