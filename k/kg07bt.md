---
id: kg07bt
title: "Detail: a calm ACCEPTANCE heading; only the cursor marks the question"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxh"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T16:25"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "crab review"
waiting_since: "2026-11-18T16:56"
waiting_asked_by: "macaque"
---

## Goal
In a task's detail the ACCEPTANCE heading is quiet (no background fill; at most bold or dim, like the other section headings), and the focused asked check / question is not bold plus background plus strong colour at once: it reads as a normal row, and the one blinking underline cursor (accent #00bfff/#006c9c, 530 ms) is the only thing that says 'you are here'.

## Purpose
Jørgen 2026-11-18 14:58 local, answering 0gztsy check 4: the focus was correct, but there is too much visual noise. The ACCEPTANCE heading draws so much attention that it competes with the question line below it, and the question is bold, background-filled and strongly coloured at once. He wants it toned down, with the glowing cursor marking the one place you are, not a visual competition.

## Principle
One motion and one emphasis on screen: the cursor. No bold everywhere, no background fills for emphasis. Stable key-hierarchy colours. Do not change what has focus or which keys act, only how it looks.

## Relevant details
Code: huginn/internal/tui detail.go, stage.go, render.go (asked-check rendering from 41hhgn/0gztsy). Goldens: testdata/detail/asked-check.txt, asked-block.txt at 80 and 140; update with HUGINN_UPDATE_GOLDEN=1 and show the before/after rows in the report. kikazaru is in detail.go for 0ga6m0: take this after 0ga6m0, on top of it.

## Depends on

## Acceptance
- [x] The ACCEPTANCE heading has no background colour and the same weight as the other detail section headings (golden + style assertion).
- [x] The focused asked check renders with no background fill and no bold; the cursor alone marks it (test asserts the SGR codes on that row).
- [ ] Jørgen opens a task asked of him and says it is calm and the cursor is where his answer goes.

## Notes


macaque: branch kg07bt (ci/kg07bt) 5e3890a2 + 3b985709 on main 56e2cc6f (has qpp30y 3d74229a); tz9z14 replayed as a8ee13f5/0a95a06c. Check 1: sectionHeading(focused) = accent bold '▸ HEAD' + dim hint, no fill (all focused headings, crab ok). Before '<1;38;2;11;22;32;48;2;0;191;255>▸ ACCEPTANCE · j/k ...', after '<1;38;2;0;191;255>▸ ACCEPTANCE<0><38;2;135;149;168> · j/k ...'. TestFocusedHeadingIsCalm failed first: 'kg07bt_test.go:81: the focused heading has a background'; now PASS (no 48;, same bold as DESCRIPTION). Golden testdata/detail/calm-focus.txt at 80/140.
Check 2: cursorMark draws the focused asked check (checkStyle colour) and the focused question block (forYou amber) with no fill and no bold; the > is theme.cursor (qpp30y), static, the one underlined cell. Before '<1;48;2;22;54;73>> ? ☐ 2. b<0>', after '<1;4;38;2;0;191;255;4>><0> ? ☐ 2. b'. TestFocusedAskedRowIsCalm failed first: 'kg07bt_test.go:113: aaaaaa: the focused checks row is [bold background 1;48;2;22;54;73]'; now PASS (row starts with theme.cursor '>', then no bold, no 48;, no 4). Focus and keys unchanged (a answers; asked-check/asked-block goldens unchanged). Full ./... 35/35 ok. Report /Users/jhf/.jcode/scratch/kg07bt/REPORT.md
CI 44338340390 on 3b985709 (kg07bt on main 56e2cc6f): completed success, 6/6 jobs, first attempt.
crab review 16:56 local: accepted. 5e3890a2 + 3b985709 (on tz9z14 replay 0a95a06c): heading loses its background fill (bold accent, dim key hint); asked check and question block keep their unfocused colours, no fill, the > drawn in qpp30y's theme.cursor (one cursor style; the answer box's cursor blinks). SGR test failing-first, calm-focus.txt at 80/140, asked goldens unchanged. CI 44338340390 6/6. Lands stacked on the current landing stack.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed 3517e62 on main 2026-11-18: tui: a calm focused heading and asked row; the > is the cursor (kg07bt)
landed 4778ebf on main 2026-11-18: tui: the focused row's > is theme.cursor, the answer box's cursor (kg07bt)
shipped in v0.7.66 (3517e62)
## History
- 2026-11-18T16:25 todo -> doing/active (take)
- 2026-11-18T16:56 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: macaque's session ended (closed 2026-11-19 02:14)
