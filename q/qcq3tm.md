---
id: qcq3tm
title: "Opening an asked task scrolls to its question; nothing inlined on top"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-20
updated: 2026-11-20
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

## Notes
