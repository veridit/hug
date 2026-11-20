---
id: qcq3tm
title: "Opening an asked task scrolls to its question; nothing inlined on top"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzq"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-20T10:59"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "Does opening a task that asks you something now land on the question, with its file as a link line (Enter shows the file, Esc comes back)? Before and after captures at 100x30 are in this task's Files: before-tb0dra-100x30.ans / after-tb0dra-100x30.ans, before-hjbfzd-100x30.ans / after-hjbfzd-100x30.ans and after-hjbfzd-check6-link-100x30.ans. For a on p87ngv: before-p87ngv-a-100x30.ans / after-p87ngv-a-100x30.ans. Built on branch qcq3tm (a5fc5208, CI green); it reaches your board with the next release."
asked_for: "jhf"
asked_since: "2026-11-20T11:54"
asked_by: "cricket"
asked_check: "3"
parent: "fzd0jj"
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
- [x] A question with --show draws no file above the page; the question shows the file as a link line, Enter opens it in the file frame, Esc returns to the question (tmux test)
- [ ] Jørgen opens tb0dra-like asked tasks after the release and lands on the question
- [x] On p87ngv's board copy (doing/asked, old asked_* stage fields plus ## Questions 11 and 12), pressing a opens the answer box in view under the question it answers, with the cursor in it; typing shows text (tmux test; Jørgen 2026-11-20 10:13Z: a did nothing visible on p87ngv)

## Questions
- 2026-11-20T11:54 question 1 by cricket to jhf (check 3):
  Does opening a task that asks you something now land on the question, with its file as a link line (Enter shows the file, Esc comes back)? Before and after captures at 100x30 are in this task's Files: before-tb0dra-100x30.ans / after-tb0dra-100x30.ans, before-hjbfzd-100x30.ans / after-hjbfzd-100x30.ans and after-hjbfzd-check6-link-100x30.ans. For a on p87ngv: before-p87ngv-a-100x30.ans / after-p87ngv-a-100x30.ans. Built on branch qcq3tm (a5fc5208, CI green); it reaches your board with the next release.

## Requests
- 2026-11-20T12:56 request 1 by jhf:
  If you look at the screenshot, this is what it looked like when I open this ticket or this task. Clearly, do you think that matches what the task claims to have done? If not, find five problems or not five, but like find inconsistency and tell me what they are so I can approve the improvement. So I can approve the improvement

## Notes


file:before-tb0dra-100x30.ans (2975 bytes, sha256:d2e372aa) origin/main 4b1a5781, tb0dra as opened: agentsection-proposal.md drawn above the page, the question off screen
file:after-tb0dra-100x30.ans (3927 bytes, sha256:4c747c53) qcq3tm 15667952, tb0dra as opened: lands on check 5's question, file as link line
file:before-hjbfzd-100x30.ans (5337 bytes, sha256:45b53014) origin/main 4b1a5781, hjbfzd as opened: capture-wording-100x30.ans drawn above the page
file:after-hjbfzd-100x30.ans (4057 bytes, sha256:0d34d38d) qcq3tm 15667952, hjbfzd as opened: lands on check 5's question 1
file:after-hjbfzd-check6-link-100x30.ans (3954 bytes, sha256:e68b3d18) qcq3tm 15667952, hjbfzd after j: check 6's question with its link line to capture-wording-100x30.ans
file:before-p87ngv-a-100x30.ans (2483 bytes, sha256:ffee7bf7) origin/main 4b1a5781, p87ngv after a and typing: the box is open (keys line) but out of view
file:after-p87ngv-a-100x30.ans (3668 bytes, sha256:cd9adebd) qcq3tm 15667952, p87ngv after Home, a and typing: the box in view under question 11
check 1: TestTmuxAskedTaskOpensOnItsQuestion (huginn/internal/tui/qcq3tm_tmux_test.go, tb0dra and hjbfzd copies at 100x30). On origin/main 4b1a5781: 'tb0dra opens on its question (check 1), its file not drawn above the page (check 2): never showed ["> ? ☐ 5. after a week of use" ...]'. On qcq3tm a5fc5208: '--- PASS: TestTmuxAskedTaskOpensOnItsQuestion (3.10s)'. CI 44721913557 green (8/8 jobs) on a5fc5208. Captures before/after-tb0dra-100x30.ans, before/after-hjbfzd-100x30.ans.
check 2: Same test: no file drawn above the page, '→ agentsection-proposal.md · Enter shows' under the question, Enter shows it in the file frame ('u/Esc question' in the keys), Esc back to the question; hjbfzd check 6 the same with capture-wording-100x30.ans. Fail first on origin/main 4b1a5781 (line quoted on check 1); '--- PASS' on a5fc5208; CI 44721913557 green. Unit: TestQuestionShowsItsPicture, TestQuestionPictureIsFramed PASS (goldens detail/picture.txt, picture-framed.txt reviewed). Space in FILES unchanged (TestTmuxZhvkmjFilesOnItself PASS).
check 4: TestTmuxAnswerBoxOpensInView on a p87ngv copy (hug/tasks 069928d0: old asked_* fields, question 11 on check 5, question 12 in a block). On origin/main 4b1a5781: 'p87ngv a opens the answer box in view (check 4): never showed ["? ☐ 5. Jørgen approves a mock" "answer:"]', with the keys line 'Enter send · Alt-Enter new line · Esc cancel' (box open, out of view). On a5fc5208: '--- PASS: TestTmuxAnswerBoxOpensInView (2.81s)', 'answer: typed by qcq3tm' on screen. CI 44721913557 green. Captures before/after-p87ngv-a-100x30.ans.
landed 48e3517 on main 2026-11-20: hug tui: an asked task opens on its question; its file is a link line, Enter…
landed 1566795 on main 2026-11-20: hug tui: Esc from a file a question's link showed is named 'question' in the…
landed a5fc520 on main 2026-11-20: tui tests: capture knob QCQ3TM_CAPTURES (no HUG_ name); picture golden names Esc…
shipped in v0.7.89 (48e3517)
crab 2026-11-20 10:54Z: reviewed (actions.go answerKey focusOn+revealAnswerBox; detail.go link line replaces drawn picture, startFocus/openTop no longer special-case pictures; glance.go return to question; picture.go dead code removed), landed a5fc5208 on main as a fast-forward (CI 44721913557 8/8 on that SHA), released in v0.7.89 (20dbb6ca). Release ticket fzd0jj.
file:screen-1255.ans (9067 bytes, sha256:d8f4a13a) screen capture from hug tui
## History
- 2026-11-20T10:59 todo -> doing/active (take)
- 2026-11-20T11:54 doing/active -> doing/asked (ask)
- 2026-11-20T12:55 handed to crab: cricket's session ended (closed 2026-11-20 11:55)
- 2026-11-20T12:56 request 1 by jhf
