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
- [x] Opening a task with an open question to you scrolls so the question is in view and focused, wherever it sits on the page (tmux test, a question below the fold)
- [ ] A question with --show draws no file above the page; the question shows the file as a link line, Enter opens it in the file frame, Esc returns to the question (tmux test)
- [ ] Jørgen opens tb0dra-like asked tasks after the release and lands on the question
- [ ] On p87ngv's board copy (doing/asked, old asked_* stage fields plus ## Questions 11 and 12), pressing a opens the answer box in view under the question it answers, with the cursor in it; typing shows text (tmux test; Jørgen 2026-11-20 10:13Z: a did nothing visible on p87ngv)

## Notes


file:before-tb0dra-100x30.ans (2975 bytes, sha256:d2e372aa) origin/main 4b1a5781, tb0dra as opened: agentsection-proposal.md drawn above the page, the question off screen
file:after-tb0dra-100x30.ans (3927 bytes, sha256:4c747c53) qcq3tm 15667952, tb0dra as opened: lands on check 5's question, file as link line
file:before-hjbfzd-100x30.ans (5337 bytes, sha256:45b53014) origin/main 4b1a5781, hjbfzd as opened: capture-wording-100x30.ans drawn above the page
file:after-hjbfzd-100x30.ans (4057 bytes, sha256:0d34d38d) qcq3tm 15667952, hjbfzd as opened: lands on check 5's question 1
file:after-hjbfzd-check6-link-100x30.ans (3954 bytes, sha256:e68b3d18) qcq3tm 15667952, hjbfzd after j: check 6's question with its link line to capture-wording-100x30.ans
file:before-p87ngv-a-100x30.ans (2483 bytes, sha256:ffee7bf7) origin/main 4b1a5781, p87ngv after a and typing: the box is open (keys line) but out of view
file:after-p87ngv-a-100x30.ans (3668 bytes, sha256:cd9adebd) qcq3tm 15667952, p87ngv after Home, a and typing: the box in view under question 11
check 1: TestTmuxAskedTaskOpensOnItsQuestion (huginn/internal/tui/qcq3tm_tmux_test.go, tb0dra and hjbfzd copies at 100x30). On origin/main 4b1a5781: 'tb0dra opens on its question (check 1), its file not drawn above the page (check 2): never showed ["> ? ☐ 5. after a week of use" ...]'. On qcq3tm a5fc5208: '--- PASS: TestTmuxAskedTaskOpensOnItsQuestion (3.10s)'. CI 44721913557 green (8/8 jobs) on a5fc5208. Captures before/after-tb0dra-100x30.ans, before/after-hjbfzd-100x30.ans.
## History
- 2026-11-20T10:59 todo -> doing/active (take)
