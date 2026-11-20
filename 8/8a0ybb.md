---
id: 8a0ybb
title: "Detail view: the question first, links that jump, short sections before long"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzyv"
created: 2026-11-19
updated: 2026-11-19
stage: "asked"
doing_since: "2026-11-20T01:15"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "Is this mock of p87ngv's detail approved? Frames at 100x30 and 160x45, .ans and .png each: a-opens (FOR YOU first, its link → MOCK.md, then Purpose, Principle), a2-scrolled (Goal, Acceptance with ? on check 5, Files, the long sections folded to one line each), b-link-mock-md (Enter on the link: MOCK.md in the file frame, Esc back to the question), c0-notes-focused and c-notes-open (Enter on NOTES (11) unfolds it)."
asked_for: "jhf"
asked_since: "2026-11-20T02:20"
asked_by: "bug"
asked_check: "1"
asked_show: "a-opens-100x30.png"
---

## Goal
An asked task opens on its question, which jumps to its target in one key, and reads short to long.

## Purpose
Jørgen 2026-11-19 11:23Z (crab console, screenshot of p87ngv in hug tui v0.7.81): the page opened on MOCK.md inlined at the top (asked_show) with no question above it, so it looked like content from later in the ticket; the question was only found by scrolling; the long description came before purpose and principle ('where am I getting the value for the money'). 11:24Z (second screenshot, check 5 with its question and 'answer'): the question names MOCK.md but offers no way to jump to it; he had to press down many times to reach FILES and then the file.

## Principle
The person is asked first, then oriented, then given detail. A question links, never inlines. Links use the mark syntax (p87ngv): .target:(.File .name:"MOCK.md"), .target:(.Section .name:"Acceptance" .item:5), .target:(.File .name:"tui-100x30.txt" .line:63); asked_show maps to .target:(.File ...). One jump key everywhere (Enter on a link). Folded sections show their one-line summary (FILES (6), NOTES (12), HISTORY (40)). Calm rules (qh9ep1 check 9).

## Relevant details
Opening a task that asks you something shows, under the title, a FOR YOU section with each open question; each names its target (a file, a section, a check, a line in a file) and Enter on it jumps straight there, a answers. The target file is shown only when you follow the link, never inlined at the top. Below that the ticket reads short to long: Purpose, Principle, Goal, Acceptance, then Details, Files, Notes, History as one-line folded sections until opened. Code: huginn/internal/tui/detail.go (sections around 530-600 and 819-830, markedBlock 1060-1100), picture.go:248 questionPictureOf (asked_show inline). jnaspy set Goal, Purpose, Principle first; this changes the order to Purpose, Principle, Goal and folds reference sections. Open for Jørgen (asked on this task): whether Goal stays as long as today or is cut to 1-2 sentences with the rest in Details. Mock first (frames at 100x30 and 160x45 of p87ngv as it is now), approved before build. Order (Jørgen 11:31Z): the DESCRIPTION heading goes; the sections are FOR YOU, Purpose, Principle, Goal, Acceptance, then the short sections (Files, Depends on, Parent and children, open requests), then the long ones: Relevant details, Notes, History.

## Depends on

## Acceptance
- [ ] Jørgen approves a mock of p87ngv's detail with FOR YOU first, the link, and the short-to-long order (frames attached)
- [ ] Enter on a question's target opens that file in the file frame, or scrolls to that section or check, in one key (test per target kind)
- [ ] A task with asked_show no longer inlines the file above the question; the question shows its target as a link (golden)
- [ ] ~~Details, Files, Notes and History are folded to one line until opened; Purpose, Principle, Goal, Acceptance show in that order (golden 100x30 and 160x45)~~ (not relevant: order revised by Jørgen 11:31Z: Description heading goes, Details after Files and other short sections; replaced by check 6, crab, 2026-11-19)
- [ ] Jørgen opens an asked task in hug tui and reaches the thing he is asked about in one key
- [ ] No DESCRIPTION heading; order is FOR YOU, Purpose, Principle, Goal, Acceptance, short sections (Files, Depends on, Parent/children, requests), then Relevant details, Notes, History; long sections fold to one line until opened (golden 100x30 and 160x45)
- [ ] Opening an asked task focuses its question; the picture line under it (asked_show) is a link: Enter shows the file, Esc returns to the question, a answers from either (tmux test on 911j70's board copy)
- [ ] Opening a task asked of you scrolls to its first open question, wherever it is on the page; nothing is drawn above the page for it, and the --show file is a link on the question, never inlined (tmux test on hjbfzd's board copy)
- [ ] The detail page shows ↑ in a top corner when there is more above and ↓ in a bottom corner when there is more below, as the overview marks more rows (golden 100x30 scrolled mid-page)

## Questions
- 2026-11-19T12:24 question 1 by crab to jhf:
  Should the Goal stay as long as today, or be cut to one or two sentences with the rest moved to Details?
  answered 2026-11-19T12:26 by jhf:
    (console 11:26Z) Yes, the Goal should be shorter, not long. It is wrong today: AI tends to bloat, and that loses focus. We have to fight for focus.
- 2026-11-20T02:20 question 2 by bug to jhf (check 1, show a-opens-100x30.png):
  Is this mock of p87ngv's detail approved? Frames at 100x30 and 160x45, .ans and .png each: a-opens (FOR YOU first, its link → MOCK.md, then Purpose, Principle), a2-scrolled (Goal, Acceptance with ? on check 5, Files, the long sections folded to one line each), b-link-mock-md (Enter on the link: MOCK.md in the file frame, Esc back to the question), c0-notes-focused and c-notes-open (Enter on NOTES (11) unfolds it).
- 2026-11-20T09:52 question 3 by crab to jhf (check 1):
  Your 08:51Z idea (on hjbfzd) differs from the mock in question 2: the mock opens on a FOR YOU block at the top that lists your questions as links; yours opens the page scrolled to the question where it sits in Acceptance, nothing added on top, with ↑/↓ corner markers showing there is more. Which do you want: scroll-to-question (I will drop FOR YOU from the mock), or the FOR YOU block?

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
