---
id: kg07bt
title: "Detail: a calm ACCEPTANCE heading; only the cursor marks the question"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzuv"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T16:25"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In a task's page, the focused row is marked by a pulsing > with no underline, the one moving thing on screen.

## Purpose
Jørgen 2026-11-18 14:58 local, answering 0gztsy check 4: the focus was correct, but there is too much visual noise. The ACCEPTANCE heading draws so much attention that it competes with the question line below it, and the question is bold, background-filled and strongly coloured at once. He wants it toned down, with the glowing cursor marking the one place you are, not a visual competition.

## Principle
One motion and one emphasis on screen: the cursor. No bold everywhere, no background fills for emphasis. Stable key-hierarchy colours. Do not change what has focus or which keys act, only how it looks.

## Relevant details
Change (Jørgen 2026-11-20 10:33Z, request 1, screenshot screen-1132.ans): today the focused row's > is drawn in theme.cursor, underlined and static (since v0.7.66). An underlined character reads as a key you can press (the view line underlines keys), and > is a key: in a page it goes to the bottom (tui.go scroll: '>', 'end' -> GotoBottom; '<' -> GotoTop). So the > loses the underline and pulses or blinks in the accent at the box cursor's 530 ms instead, as a cursor does. When the answer box is open, its cursor is the one that moves and the > stays still, so only one thing moves at a time (qpp30y). Same rule as 911j70: the current cell gets >, no underline. His words: 'the cursor underlines the greater than that looks like I can utilize a shortcut ... I think there has to be the greater than pulsating like it is a cursor, which brings me to focus there.' He also expects > and < to go to the bottom and top in this view; check that they do.

In a task's detail the ACCEPTANCE heading is quiet (no background fill; at most bold or dim, like the other section headings), and the focused asked check / question is not bold plus background plus strong colour at once: it reads as a normal row, and the one blinking underline cursor (accent #00bfff/#006c9c, 530 ms) is the only thing that says 'you are here'.

Code: huginn/internal/tui detail.go, stage.go, render.go (asked-check rendering from 41hhgn/0gztsy). Goldens: testdata/detail/asked-check.txt, asked-block.txt at 80 and 140; update with HUGINN_UPDATE_GOLDEN=1 and show the before/after rows in the report. kikazaru is in detail.go for 0ga6m0: take this after 0ga6m0, on top of it.

## Depends on

## Acceptance
- [x] The ACCEPTANCE heading has no background colour and the same weight as the other detail section headings (golden + style assertion).
- [x] The focused asked check renders with no background fill and no bold; the cursor alone marks it (test asserts the SGR codes on that row).
- [ ] ~~Jørgen opens a task asked of him and says it is calm and the cursor is where his answer goes.~~ (not relevant: replaced by check 6: Jørgen saw the static underlined > and asked for a pulsing one (request 1, 2026-11-20), crab, 2026-11-20)
- [ ] The focused row's > has no underline and blinks or pulses in the accent at 530 ms; with the answer box open the > is still and only the box cursor moves (test on the SGR codes per tick, golden)
- [ ] In a task's page, > and End go to the bottom and < and Home to the top, also while a check is focused (tmux test)
- [ ] Jørgen opens a task asked of him and the pulsing > draws his eye to the question

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 3):
  v0.7.66 made the ACCEPTANCE heading calm and the cursor the only mark on the focused question. Open a task that asks you something: is it calm, and is the cursor where your answer goes?
  answered 2026-11-20T11:33 by jhf:
    So this ticket is accepted and my feedback that you got from this ticket can be new tickets.

## Requests
- 2026-11-20T11:33 request 1 by jhf:
  Look at this line. The cursor marks the question. Well, right now, the cursor underlines the greater than that looks like I can utilize a shortcut, which I can't because it will jump up and down. I hope it goes to top and bottom, which it should in this view. So we have to do something else. I think there has to be the greater than pulsating like it is a cursor, which brings me to focus there.
  resolved 2026-11-20T11:33 by crab:
    Changed the Goal: the focused row is marked by a > that pulses like a cursor, with no underline, so it no longer reads as a key. Check 4 builds it (only one thing moves: while the answer box is open its cursor moves and the > stays still). Check 5 makes sure > and < go to the bottom and top of the page in this view, as you expect. Check 6 is your look after release; old check 3 is struck. Queued in the TUI lane after cricket (qcq3tm).
- 2026-11-20T11:33 request 2 by jhf:
  Or you can continue working on the ticket, feel free. But I think this one has shipped

## Notes


macaque: branch kg07bt (ci/kg07bt) 5e3890a2 + 3b985709 on main 56e2cc6f (has qpp30y 3d74229a); tz9z14 replayed as a8ee13f5/0a95a06c. Check 1: sectionHeading(focused) = accent bold '▸ HEAD' + dim hint, no fill (all focused headings, crab ok). Before '<1;38;2;11;22;32;48;2;0;191;255>▸ ACCEPTANCE · j/k ...', after '<1;38;2;0;191;255>▸ ACCEPTANCE<0><38;2;135;149;168> · j/k ...'. TestFocusedHeadingIsCalm failed first: 'kg07bt_test.go:81: the focused heading has a background'; now PASS (no 48;, same bold as DESCRIPTION). Golden testdata/detail/calm-focus.txt at 80/140.
Check 2: cursorMark draws the focused asked check (checkStyle colour) and the focused question block (forYou amber) with no fill and no bold; the > is theme.cursor (qpp30y), static, the one underlined cell. Before '<1;48;2;22;54;73>> ? ☐ 2. b<0>', after '<1;4;38;2;0;191;255;4>><0> ? ☐ 2. b'. TestFocusedAskedRowIsCalm failed first: 'kg07bt_test.go:113: aaaaaa: the focused checks row is [bold background 1;48;2;22;54;73]'; now PASS (row starts with theme.cursor '>', then no bold, no 48;, no 4). Focus and keys unchanged (a answers; asked-check/asked-block goldens unchanged). Full ./... 35/35 ok. Report /Users/jhf/.jcode/scratch/kg07bt/REPORT.md
CI 44338340390 on 3b985709 (kg07bt on main 56e2cc6f): completed success, 6/6 jobs, first attempt.
crab review 16:56 local: accepted. 5e3890a2 + 3b985709 (on tz9z14 replay 0a95a06c): heading loses its background fill (bold accent, dim key hint); asked check and question block keep their unfocused colours, no fill, the > drawn in qpp30y's theme.cursor (one cursor style; the answer box's cursor blinks). SGR test failing-first, calm-focus.txt at 80/140, asked goldens unchanged. CI 44338340390 6/6. Lands stacked on the current landing stack.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed 3517e62 on main 2026-11-18: tui: a calm focused heading and asked row; the > is the cursor (kg07bt)
landed 4778ebf on main 2026-11-18: tui: the focused row's > is theme.cursor, the answer box's cursor (kg07bt)
shipped in v0.7.66 (3517e62)
file:screen-1132.ans (26114 bytes, sha256:449dc9ba) screen capture from hug tui
## History
- 2026-11-18T16:25 todo -> doing/active (take)
- 2026-11-18T16:56 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: macaque's session ended (closed 2026-11-19 02:14)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: macaque; waiting_on: crab review; waiting_since: 2026-11-18T16:56
- 2026-11-20T11:33 request 1 by jhf
- 2026-11-20T11:33 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:33 request 1 acknowledged by crab
- 2026-11-20T11:33 answered asked (check 3), by jhf, for crab (doing/asked -> doing/active):
  > v0.7.66 made the ACCEPTANCE heading calm and the cursor the only mark on the focused question. Open a task that asks you something: is it calm, and is the cursor where your answer goes?
  So this ticket is accepted and my feedback that you got from this ticket can be new tickets.
- 2026-11-20T11:33 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:33 request 2 by jhf
- 2026-11-20T11:33 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:33 crab read the answer: hug task set kg07bt
- 2026-11-20T11:33 request 1 in progress by crab: hug task set kg07bt
- 2026-11-20T11:33 resolved request 1 by crab
- 2026-11-20T11:34 answer 1 acknowledged by crab
