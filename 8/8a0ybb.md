---
id: 8a0ybb
title: "Detail view: short sections before long, folded long sections, scroll markers"
state: "todo"
type: feature
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzwh"
created: 2026-11-19
updated: 2026-11-19
stage: "ready"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A task's page reads Purpose, Principle, Goal, Acceptance, then short sections, then long ones folded, with ↑/↓ when more is off screen.

## Purpose
The value of a ticket should be on screen first; long reference text should not push it away.

## Principle
You land on your question, a file it names is a link on it, and the page reads short to long.

## Relevant details
Headings (Jørgen 2026-11-20 10:26Z on ddpxs8): the TUI's DESCRIPTION heading is the Goal under the wrong name (detail.go:819 labels t.Goal 'Description'), so it overlaps Purpose, Principle and Details. The sections become Title, then Purpose, Principle, Goal, Acceptance, Details: no Description. 'Relevant details' is shortened to 'Details' (sections.go SectionOrder, settext.go SectionFlags; store.go already scores them as equal), and old files keep reading. Check 13.

Scope (crab, 2026-11-20, after Jørgen's requests 1 and 2): this task keeps only the section order, folding the long sections, and the ↑/↓ scroll markers. Landing on a question with its file as a link is qcq3tm (high, being built). Questions and requests drawn inline where they point, jumped to and answered in place, are p87ngv (inline marks), which replaces any FOR YOU header.

Decision (Jørgen 2026-11-20 10:24Z, request 1, screenshot screen-1123.ans): no FOR YOU block. Opening an asked task scrolls to the question; a file the question names is a link on it (Enter shows it, Esc returns), never drawn at the top of the page. His words: 'We scroll down to the question, but we're showing the file in line that doesn't make any sense ... we should have scrolled to the question and the file would be then in display or linked to from the question ... we can always scroll to the question'. The landing and the link are built in qcq3tm (high, in progress); this task keeps the section order, the jump from a target, and the scroll markers.

Previous Principle: The person is asked first, then oriented, then given detail. A question links, never inlines. Links use the mark syntax (p87ngv): .target:(.File .name:"MOCK.md"), .target:(.Section .name:"Acceptance" .item:5), .target:(.File .name:"tui-100x30.txt" .line:63); asked_show maps to .target:(.File ...). One jump key everywhere (Enter on a link). Folded sections show their one-line summary (FILES (6), NOTES (12), HISTORY (40)). Calm rules (qh9ep1 check 9).

Opening a task that asks you something shows, under the title, a FOR YOU section with each open question; each names its target (a file, a section, a check, a line in a file) and Enter on it jumps straight there, a answers. The target file is shown only when you follow the link, never inlined at the top. Below that the ticket reads short to long: Purpose, Principle, Goal, Acceptance, then Details, Files, Notes, History as one-line folded sections until opened. Code: huginn/internal/tui/detail.go (sections around 530-600 and 819-830, markedBlock 1060-1100), picture.go:248 questionPictureOf (asked_show inline). jnaspy set Goal, Purpose, Principle first; this changes the order to Purpose, Principle, Goal and folds reference sections. Open for Jørgen (asked on this task): whether Goal stays as long as today or is cut to 1-2 sentences with the rest in Details. Mock first (frames at 100x30 and 160x45 of p87ngv as it is now), approved before build. Order (Jørgen 11:31Z): the DESCRIPTION heading goes; the sections are FOR YOU, Purpose, Principle, Goal, Acceptance, then the short sections (Files, Depends on, Parent and children, open requests), then the long ones: Relevant details, Notes, History.

## Depends on

## Acceptance
- [ ] ~~Jørgen approves a mock of p87ngv's detail with FOR YOU first, the link, and the short-to-long order (frames attached)~~ (not relevant: the FOR YOU mock is replaced: Jørgen chose scroll-to-question, no FOR YOU block (request 1, 2026-11-20); new mock is check 10, crab, 2026-11-20)
- [ ] ~~Enter on a question's target opens that file in the file frame, or scrolls to that section or check, in one key (test per target kind)~~ (not relevant: taken over by p87ngv: marks point into files and sections, and you act on them where they are (Jørgen request 2, 2026-11-20), crab, 2026-11-20)
- [ ] ~~A task with asked_show no longer inlines the file above the question; the question shows its target as a link (golden)~~ (not relevant: moved to qcq3tm (high), which ships ahead of the FOR YOU decision, crab, 2026-11-20)
- [ ] ~~Details, Files, Notes and History are folded to one line until opened; Purpose, Principle, Goal, Acceptance show in that order (golden 100x30 and 160x45)~~ (not relevant: order revised by Jørgen 11:31Z: Description heading goes, Details after Files and other short sections; replaced by check 6, crab, 2026-11-19)
- [ ] ~~Jørgen opens an asked task in hug tui and reaches the thing he is asked about in one key~~ (not relevant: covered by qcq3tm check 3 now and by p87ngv marks later, crab, 2026-11-20)
- [ ] ~~No DESCRIPTION heading; order is FOR YOU, Purpose, Principle, Goal, Acceptance, short sections (Files, Depends on, Parent/children, requests), then Relevant details, Notes, History; long sections fold to one line until opened (golden 100x30 and 160x45)~~ (not relevant: replaced by check 11 without FOR YOU (request 1, 2026-11-20), crab, 2026-11-20)
- [ ] ~~Opening an asked task focuses its question; the picture line under it (asked_show) is a link: Enter shows the file, Esc returns to the question, a answers from either (tmux test on 911j70's board copy)~~ (not relevant: moved to qcq3tm checks 1-2 (land on the question, the file a link on it), crab, 2026-11-20)
- [ ] ~~Opening a task asked of you scrolls to its first open question, wherever it is on the page; nothing is drawn above the page for it, and the --show file is a link on the question, never inlined (tmux test on hjbfzd's board copy)~~ (not relevant: moved to qcq3tm (high), which ships ahead of the FOR YOU decision, crab, 2026-11-20)
- [ ] ~~The detail page shows ↑ in a top corner when there is more above and ↓ in a bottom corner when there is more below, as the overview marks more rows (golden 100x30 scrolled mid-page)~~ (not relevant: replaced by 4ddcwa: ↑ on the top rule, ↓ and the page status on a bottom rule (Jørgen 2026-11-20), crab, 2026-11-20)
- [ ] ~~Jørgen approves frames (ANSI at 100x30) of 8a0ybb's own detail: opened scrolled to its question with the file as a link, and the order Purpose, Principle, Goal, Acceptance, short sections, then long sections folded~~ (not relevant: the question landing is qcq3tm's; replaced by check 12, the order only, crab, 2026-11-20)
- [ ] No DESCRIPTION heading and no FOR YOU block; order is Purpose, Principle, Goal, Acceptance, short sections (Files, Depends on, Parent/children, requests), then Relevant details, Notes, History; long sections fold to one line until opened (golden 100x30 and 160x45)
- [ ] Jørgen approves frames (ANSI at 100x30) of this task's page in the new order, the long sections folded, ↑/↓ markers shown
- [ ] The page has no DESCRIPTION heading: the Goal shows under GOAL, and 'Relevant details' is headed DETAILS in hug tui and hug task show; new tasks write '## Details' and both names still read (test, golden)

## Questions
- 2026-11-19T12:24 question 1 by crab to jhf:
  Should the Goal stay as long as today, or be cut to one or two sentences with the rest moved to Details?
  answered 2026-11-19T12:26 by jhf:
    (console 11:26Z) Yes, the Goal should be shorter, not long. It is wrong today: AI tends to bloat, and that loses focus. We have to fight for focus.
- 2026-11-20T02:20 question 2 by bug to jhf (check 1, show a-opens-100x30.png):
  Is this mock of p87ngv's detail approved? Frames at 100x30 and 160x45, .ans and .png each: a-opens (FOR YOU first, its link → MOCK.md, then Purpose, Principle), a2-scrolled (Goal, Acceptance with ? on check 5, Files, the long sections folded to one line each), b-link-mock-md (Enter on the link: MOCK.md in the file frame, Esc back to the question), c0-notes-focused and c-notes-open (Enter on NOTES (11) unfolds it).
  answered 2026-11-20T11:25 by crab:
    Withdrawn by crab: this mock had the FOR YOU block, which Jørgen ruled out in request 1 (2026-11-20 10:24Z). A new mock without it is check 10.
- 2026-11-20T09:52 question 3 by crab to jhf (check 1):
  Your 08:51Z idea (on hjbfzd) differs from the mock in question 2: the mock opens on a FOR YOU block at the top that lists your questions as links; yours opens the page scrolled to the question where it sits in Acceptance, nothing added on top, with ↑/↓ corner markers showing there is more. Which do you want: scroll-to-question (I will drop FOR YOU from the mock), or the FOR YOU block?
  answered 2026-11-20T11:25 by crab:
    Jørgen answered this in request 1 (2026-11-20 10:24Z): scroll to the question; no FOR YOU block. 'we can always scroll to the question ... wouldn't that be the thing to do'.

## Requests
- 2026-11-20T11:24 request 1 by jhf:
  So look at the screenshot We scroll down to the question, but we're showing the file in line that doesn't make any sense, right? So we should have scrolled to the question and the file would be then in display or linked to from the question So you have to clean this up and take it all the way and if we can't like we can always scroll to the question can't we in this case wouldn't that be the thing to do
  resolved 2026-11-20T11:26 by crab:
    Agreed: no FOR YOU block. Opening an asked task scrolls to the question, and a file it names is a link on it (Enter shows it, Esc returns). That part is qcq3tm, high, being built now (its checks 1-2). Here: old mock check 1 and the FOR YOU order check 6 are struck, the FOR YOU questions 2 and 3 are closed with your answer, and the order check is rewritten without FOR YOU (check 11).
- 2026-11-20T11:25 request 2 by jhf:
  And we're working on a much more intricate system of inline things where we use the new syntax and everything. It's another ticket. So wouldn't that kind of replace this let's write something special in the header, but we can jump down to the things and then you can interact with them where you're at. So this is kind of moot at that point, being taken over by the other task?
  resolved 2026-11-20T11:26 by crab:
    Yes, mostly. Jumping to the things and acting on them where they are is p87ngv's inline marks, so this task's jump check (2) and its land-on-the-question checks (5, 7, 10) are struck and point there or to qcq3tm (the landing fix shipping now). What p87ngv does not cover stays here, now medium priority: the section order (Purpose, Principle, Goal, Acceptance, short sections, then long ones folded, check 11) and the ↑/↓ scroll markers (check 9). Title and Goal say just that. Check 12 is your look at frames of it.

## Notes


file:a-opens-100x30.ans (3882 bytes, sha256:11cf6864) mock (a) p87ngv as it opens for jhf (board 4f57778b, 11:23Z): FOR YOU first, its link to MOCK.md, then Purpose, Principle; prototype mock/8a0ybb f6641b2f
file:a-opens-160x45.ans (7263 bytes, sha256:662c339f) mock (a) p87ngv as it opens for jhf (board 4f57778b, 11:23Z): FOR YOU first, its link to MOCK.md, then Purpose, Principle; prototype mock/8a0ybb f6641b2f
file:a-opens-100x30.png (94733 bytes, sha256:ee14d316) mock (a) p87ngv as it opens for jhf (board 4f57778b, 11:23Z): FOR YOU first, its link to MOCK.md, then Purpose, Principle; prototype mock/8a0ybb f6641b2f
file:a-opens-160x45.png (164808 bytes, sha256:c33314f4) mock (a) p87ngv as it opens for jhf (board 4f57778b, 11:23Z): FOR YOU first, its link to MOCK.md, then Purpose, Principle; prototype mock/8a0ybb f6641b2f
file:a2-scrolled-100x30.ans (3798 bytes, sha256:89006ecc) mock (a2) the same page scrolled with J: Goal, Acceptance with ? on check 5, Files, long sections folded to one line
file:a2-scrolled-160x45.ans (6494 bytes, sha256:682f1e6a) mock (a2) the same page scrolled with J: Goal, Acceptance with ? on check 5, Files, long sections folded to one line
file:a2-scrolled-100x30.png (102795 bytes, sha256:a0411347) mock (a2) the same page scrolled with J: Goal, Acceptance with ? on check 5, Files, long sections folded to one line
file:a2-scrolled-160x45.png (174955 bytes, sha256:dc874cf8) mock (a2) the same page scrolled with J: Goal, Acceptance with ? on check 5, Files, long sections folded to one line
file:b-link-mock-md-100x30.ans (6171 bytes, sha256:f894ff5f) mock (b) Enter on the FOR YOU link: MOCK.md in the file frame; u/Esc goes back to the question
file:b-link-mock-md-160x45.ans (11954 bytes, sha256:1cd13d03) mock (b) Enter on the FOR YOU link: MOCK.md in the file frame; u/Esc goes back to the question
file:b-link-mock-md-100x30.png (117989 bytes, sha256:b57efac6) mock (b) Enter on the FOR YOU link: MOCK.md in the file frame; u/Esc goes back to the question
file:b-link-mock-md-160x45.png (202510 bytes, sha256:04647f96) mock (b) Enter on the FOR YOU link: MOCK.md in the file frame; u/Esc goes back to the question
file:c0-notes-focused-100x30.ans (3513 bytes, sha256:8130cbc2) mock (c0) Shift-Tab to NOTES (11): the folded line focused, Enter opens
file:c0-notes-focused-160x45.ans (6495 bytes, sha256:4f650097) mock (c0) Shift-Tab to NOTES (11): the folded line focused, Enter opens
file:c0-notes-focused-100x30.png (98992 bytes, sha256:b6879ef5) mock (c0) Shift-Tab to NOTES (11): the folded line focused, Enter opens
file:c0-notes-focused-160x45.png (175124 bytes, sha256:5557cb05) mock (c0) Shift-Tab to NOTES (11): the folded line focused, Enter opens
file:c-notes-open-100x30.ans (4327 bytes, sha256:d50f49f2) mock (c) Enter on NOTES: unfolded, one entry per paragraph, page starts at its heading
file:c-notes-open-160x45.ans (8941 bytes, sha256:b32c1b8f) mock (c) Enter on NOTES: unfolded, one entry per paragraph, page starts at its heading
file:c-notes-open-100x30.png (88078 bytes, sha256:c3749eb1) mock (c) Enter on NOTES: unfolded, one entry per paragraph, page starts at its heading
file:c-notes-open-160x45.png (148629 bytes, sha256:2f20fecc) mock (c) Enter on NOTES: unfolded, one entry per paragraph, page starts at its heading
check 1 mock (bug, claude-opus-5-5): 20 frames attached (5 states x 100x30/160x45 x .ans/.png), real hug tui driven in a private tmux (capture-pane -e, truecolor, COLORFGBG=15;0, clock pinned to 2026-11-19 12:23) on hug/tasks 4f57778b (p87ngv as Jørgen saw it 11:23Z, asked_show MOCK.md, check 5). Prototype branch mock/8a0ybb f6641b2f (on origin/main 597a0b24, does not land): detail.go/tui.go/keys.go + tmp-8a0ybb/{fixture.sh,frames.sh,ans2png.py}. Choices the task did not settle: link line '→ MOCK.md  file · about check 5' under the question; a question naming a check AND a file links to the file, the check keeps only its '?'; Files leads the short sections (where links land); Questions (answered) and resolved Requests fold with Notes/History, open requests stay open; folded line = '› HEADING (count) · preview', focused '▸', Enter toggles, Esc from a link's file frame returns to the question; Links section kept where it was (empty for p87ngv). Asked as question 2.
crab 01:21Z: reviewed the mock: 20 frames (5 views x 100x30/160x45, .ans + .png) are real tmux captures of hug tui on mock/8a0ybb f6641b2f (24-bit colour escapes, e.g. a-opens-100x30.ans has 14 lines with 38;2 colours). Question 2 asks approval. Build waits on that answer.
Jørgen 2026-11-20 08:20Z, on 911j70 in hug tui v0.7.87 (screenshot in crab console): the asked_show line '▸ compare-overview-cursor-160x45.png · image 702.2 KiB · Space in FILES shows it' is calm and nice, but it gets no focus on open, he cannot jump to it, and cannot jump from it to where he answers the question. Same need as this task's FOR YOU link; added check 7 for the asked-picture line itself.
Jørgen 2026-11-20 08:51Z, on hjbfzd in v0.7.87 (two screenshots in crab console): (1) opening an asked task drew the --show file (capture-wording-100x30.ans) inline at the top, which is wrong: the page is about a question to him, and the two real questions (checks 5 and 6, in ACCEPTANCE) were only found by scrolling; the inlined block was also gone after scrolling down and up again (it is drawn only at the open position). (2) His design: no special block on top; open the page scrolled to the question itself, and show scroll markers (the ↓/↑ arrows the overview uses for more rows) at the top and bottom corners when there is more above or below. crab's cause: asked_show draws the file above the page (picture.go questionPictureOf); crab's own ask used --show. Withdrawn and re-asked without --show (hjbfzd questions 3 and 4). Checks 8 and 9 added.
Jørgen 2026-11-20 09:58Z (tb0dra answer 1, screenshots tb0dra/screen-1057.ans and screen-1057-2.ans): third report of the same thing ('I've done this several times'): the question is not visible on open; the --show file is drawn inline at the top instead of scrolling down to the question with the file. He asks for higher priority. crab: raised to high; checks 3 and 8 are the fix and can ship ahead of the FOR YOU layout decision (question 3).
file:screen-1123.ans (33913 bytes, sha256:cacfa027) screen capture from hug tui
crab 2026-11-20 12:04Z: back to todo. It sat in doing under crab with nobody building it; it is queued for a worker (machine load over 10). Found in the 47sxn2 request-1 check.
## History
- 2026-11-19T12:24 todo -> todo/design (ask)
- 2026-11-19T12:26 answered design, by jhf, for crab (todo/design -> todo/ready):
  > Should the Goal stay as long as today, or be cut to one or two sentences with the rest moved to Details?
  (console 11:26Z) Yes, the Goal should be shorter, not long. It is wrong today: AI tends to bloat, and that loses focus. We have to fight for focus.
- 2026-11-19T12:26 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-19T12:26 crab read the answer: hug task set 8a0ybb
- 2026-11-19T12:26 answer 1 acknowledged by crab
- 2026-11-20T01:15 todo/ready -> doing/active (take)
- 2026-11-20T02:20 doing/active -> doing/asked (ask)
- 2026-11-20T03:22 handed to crab: bug's session ended (closed 2026-11-20 02:20)
- 2026-11-20T09:52 asked question 3 of jhf, 2 open (doing/asked -> doing/asked)
- 2026-11-20T11:24 request 1 by jhf
- 2026-11-20T11:24 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:24 request 1 acknowledged by crab
- 2026-11-20T11:25 request 2 by jhf
- 2026-11-20T11:25 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:25 request 1 in progress by crab: hug task set 8a0ybb
- 2026-11-20T11:25 answered asked (question 3, check 1), by crab, for crab (doing/asked -> doing/asked):
  > Your 08:51Z idea (on hjbfzd) differs from the mock in question 2: the mock opens on a FOR YOU block at the top that lists your questions as links; yours opens the page scrolled to the question where it sits in Acceptance, nothing added on top, with ↑/↓ corner markers showing there is more. Which do you want: scroll-to-question (I will drop FOR YOU from the mock), or the FOR YOU block?
  Jørgen answered this in request 1 (2026-11-20 10:24Z): scroll to the question; no FOR YOU block. 'we can always scroll to the question ... wouldn't that be the thing to do'.
- 2026-11-20T11:25 answered asked (question 2, check 1), by crab, for bug (doing/asked -> doing/active):
  > Is this mock of p87ngv's detail approved? Frames at 100x30 and 160x45, .ans and .png each: a-opens (FOR YOU first, its link → MOCK.md, then Purpose, Principle), a2-scrolled (Goal, Acceptance with ? on check 5, Files, the long sections folded to one line each), b-link-mock-md (Enter on the link: MOCK.md in the file frame, Esc back to the question), c0-notes-focused and c-notes-open (Enter on NOTES (11) unfolds it).
  Withdrawn by crab: this mock had the FOR YOU block, which Jørgen ruled out in request 1 (2026-11-20 10:24Z). A new mock without it is check 10.
- 2026-11-20T11:26 resolved request 1 by crab
- 2026-11-20T11:26 resolved request 2 by crab
- 2026-11-20T11:26 request 2 acknowledged by crab
- 2026-11-20T13:04 doing/active -> todo/ready (give back; was assignee: crab); was doing_since: 2026-11-20T01:15
