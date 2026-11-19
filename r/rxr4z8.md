---
id: rxr4z8
title: "In lists, Space marks tasks and the footer offers actions for the marked ones"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzm8"
created: 2026-11-17
updated: 2026-11-17
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-18T21:27"
waiting_on: "crab review; Jørgen on check 7"
waiting_since: "2026-11-19T00:44"
waiting_asked_by: "macaque"
---

## Goal
In the overview, list and board, Space toggles a mark on the focused task and steps to the next row. While one or more tasks are marked, the footer becomes an action bar for them, e.g. '2 marked · m move · t take · d done · x drop · w wait · e edit · Esc unmark', and each action applies to every marked task, with one confirm naming the count. Esc clears the marks.

## Purpose
Jørgen 23:42 local, with a screenshot of the overview where Space on 5z4zt4 only said 'Space ticks a check inside the task: Enter opens 5z4zt4': Space should mark the task, then offer context actions like move. Today the actions are scattered single keys that need knowing; marking first shows what can be done to what is marked.

## Principle
Mark, then act, and the screen says what acting will do. Only actions valid for every marked task are offered, and an action hug would refuse shows why with a fix line. Nothing writes without the confirm. Concurrent writes wait, never fail. Subtle colours: a marked row has a quiet tint and a mark in the pie column, never red.

## Relevant details
Today: Space in a list prints 'Space ticks a check inside the task: Enter opens ID' and does nothing (actions.go actionNames maps ' ' to toggle acceptance, only in the detail). Single-key actions live in actions.go (t take, d done, x drop, J/K move, e edit), grab is m (3xcesq). With marks, m grabs the marked tasks as one block and J/K move the block. Inside a task, Space keeps ticking the focused check. Owner: nautilus, after the keys work (e3sevr, qh9ep1, ahejza). Mock the footer in the ticket before building, so Jørgen sees it.

## Depends on

## Acceptance
- [x] Space in overview, list and board marks or unmarks the focused task and moves to the next row; marked rows show a mark and a quiet tint (golden at 80 and 140)
- [x] With N marked, the footer reads 'N marked ·' followed by the actions valid for all of them and 'Esc unmark'; Esc clears every mark (key test)
- [x] t, d, x and w act on every marked task after one confirm naming the count, and a refused task is named with its fix: line while the others go ahead (key test with 3 marked, one refused)
- [x] m grabs the marked tasks as one block; J/K and the drop move them together and keep their order (key test)
- [x] Inside a task Space still ticks the focused check (existing tests stay green)
- [ ] Jørgen confirms on his board after release
- [ ] Jørgen 19:27Z (answer 1): a marked row that also has the cursor shows one combined sign, not > or ◆ alone. Proposed ⋗ (U+22D7, greater-than with dot: the cursor carrying a mark; single width); the alternative ▸ (U+25B8). The mock shows both rows side by side and asks this one question
- [x] Jørgen 19:27Z: the narrow footer (under 70 columns) never shows bare letters: it reads e.g. '3 marked · ? actions · Esc unmark', and ? lists the actions with their words (key test at 60 columns)
- [x] Jørgen 19:27Z: after an action, the tasks that went through are unmarked and the refused ones stay marked (key test, extends check 3)

## Notes

nautilus 23:44, footer mock for Jørgen (not built yet; queued after e3sevr, 0gztsy, qh9ep1, ahejza). 80 columns, 3 marked in NEXT UP, one of them an epic (d not offered: an epic finishes through its children):

    NEXT UP · rank order  [12]
  ◆ ○ 2   000041   feature critical ready     runtime Feature task 41
  ◆ ○ 2   000017   bug     low      ready     docs Bug task 17
  > ○ 1   000009   spike   high     ready     docs Spike task 09
  ◆ ◑ 1/2 a1b2c3   feature high     ready     Launch (epic)
  ...
  3 marked · m move · t take · x drop · w wait · e edit · Esc unmark
  j/k move · Space mark · Enter open · ? help · q quit

 ◆ (dim accent, never red) in gutter column 1 marks a row, with a quiet tint; the cursor row keeps >. t asks once: 'Take 3 tasks? Enter confirm · Esc cancel'. A refused task (e.g. taken by someone else) is named with its fix: line; the others go ahead. Narrow screens drop the action words to keys: '3 marked · m t x w e · Esc'.


file:footer-mock.md (3515 bytes, sha256:3d56be16) nautilus 05:20: the footer and marks mock at 80/140/narrow, drawn from main 52ff46f9; three questions for Jørgen at the end
User testing 2026-11-18 11:06Z: wants to mark several tasks with structural problems, then send one request applying to that selection so an agent repairs them. Current Space says ticks a check inside task and does not provide requested row marking. Grid/header visual redesign must show marking distinctly from row/cell focus. Bulk-request workflow is required alongside single-task r, not merely unrelated bulk state changes.
crab 20:29 local, Jørgen's answer 1 (19:27Z) settled: (1) ◆ is the mark (dim accent, never red); a marked row under the cursor needs one combined sign: check 7. (2) unmark what went through, keep refused marked: check 9. (3) Space on a heading folds, Space on a task marks; Jørgen may later prefer x ('X marks the spot') but Space stays for now. (4) narrow footer: '? actions' instead of bare letters: check 8. Process: he wants one question per thing to answer, not three in one ask: filed as its own task; until it lands, ask one question at a time (one hug task ask per question, --check N where it fits). Build order: after kikazaru's 05r3pg (palette also owns the bottom line).
crab 21:07 local: back to todo for a builder (nautilus's session ended); answers settled in checks 7-9; build after kikazaru's 05r3pg (shares the bottom line).
file:cursor-on-mark.md (1455 bytes, sha256:42f010d6) check 7: ⋗ and ▸ side by side, cursor on a marked row
crab 21:31 local, ruling on check 1's 'quiet tint': Jørgen's calm rules (qh9ep1 check 9: one cue per meaning, no background fills) supersede it. A marked row shows ◆ in the gutter (dim accent) and nothing else: no tint, no recoloured ID. Check 1 is read that way; the golden proves no 48;2 background on a marked row.
crab 20:44 ruling: with 2+ marked, t is not offered; pressed, it says 'take one at a time: mark one task to take it'. t on 1 marked takes it after one confirm. Once byj479 (hug task take B --with A, iwazaru) lands, t on N marked becomes 'take the first, the rest with it' after one confirm naming them; t's marked path is kept in one function (takeMarked) so that change is small. Also crab 20:31: the mark is ◆ in the gutter only, no tint, no recoloured id (calm rule); check 1 reads that way.
macaque: built on main 8a45b0eb (05r3pg landed), commits bafad978 + d9a8dcec on branch rxr4z8 / ci/rxr4z8. Fail-first: 12 new tests in marks_test.go failed against a stub marks.go (isMarked false), e.g. 'footer with 2 marked: "Space ticks a check inside the task: Enter opens sbehks"' and 'block not gathered'. Two old tests asserted that old Space message and now assert Space marks: group_test.go TestSpaceFoldsAHeading, invariants_test.go TestErrorAndTerminalInputSafety. Golden: testdata/marks/marked.txt (list and overview at 80, 140, 60). Other goldens changed only by 'Space mark' in the key bar and the Space row in help.
check 1: TestSpaceMarksAndSteps (list, overview, board): Space marks, steps to the next row, stays on the last, Space again unmarks, 0 commits. TestMarkIsTheGutterOnly (truecolor): gutter ◆, no 48;2 on the marked row, cells after the gutter byte-identical marked and unmarked (crab 20:31 ruling: no tint). Golden testdata/marks/marked.txt at 80, 140, 60 (list and overview). f385c3a0
check 2: TestMarkedFooterAndEsc: footer '2 marked · move · edit · Esc unmark' for 2 todo (drop too; done, wait, take not offered); key bar 'Space mark'; Esc clears all, says 'unmarked 2; nothing changed'. TestMarkedCountInTheHeading: NEXT UP heading '· 2 marked'. f385c3a0
check 3: TestDoneMarkedOneRefused: 3 doing marked, d confirms 'Done 3 tasks: A, B, C', Enter: 2 done (2 commits), B refused 'marked done 2 of 3 · B refused: unchecked-acceptance…' with 'fix: hug task check B 1'. TestDropMarked: x asks the reason once, 'Drop 2 tasks: …: not needed', 2 dropped, 2 commits. TestWaitMarked: w asks once, both waiting. t: one marked task after one confirm (TestTakeMarked); 2+ marked not offered and says 'take one at a time: mark one task to take it' (crab 20:44, n2hq8p; byj479 swaps it for take-with). f385c3a0
check 4: TestGrabMarkedBlock: m with A and C marked gathers A C at A's row (A C B D E F), j j moves the block (B D A C E F), Enter writes that order on disk, marks cleared; Esc after k puts every row back. f385c3a0
check 5: Detail tests unchanged and green: go test ./huginn/internal/tui/ ok 115.216s on f385c3a0, incl. TestAcceptanceSelectionClampedAcrossTasks (Space in a task ticks) and TestErrorAndTerminalInputSafety (now asserts Space in a list marks, never ticks).
check 8: TestNarrowMarkedFooter at 60: footer exactly '3 marked · ? actions · Esc unmark'; ? leads with '3 MARKED' and lists 'move the marked as one block…', 'drop every marked task…', 'unmark all'. f385c3a0
check 9: TestDoneMarkedOneRefused: after d, only the refused task stays marked (marks == [B]); TestDropMarked, TestWaitMarked, TestTakeMarked: all went through, marks empty. f385c3a0
f385c3a0, CI 44402446974 success 8/8. Report /Users/jhf/.jcode/scratch/rxr4z8/REPORT.md. Next: rebase after hjbfzd + byj479 land (t on N marked: Take + TakeWith; drop captureKey confirm case).
Rebased on main 4e287c1a (hjbfzd + byj479 in): 33616641, CI 44421843248 success 8/8. t on several marked: one confirm 'Take 2 tasks: A, B, B with A', then Take(A) + TakeWith(B, A, me), one commit each ('hug task take B --with A'); if A is refused the rest are not taken and all stay marked. Fail-first: TestTakeMarked 'footer with 2 todo marked does not offer take: "2 marked · move · x drop · edit · Esc unmark"'. New TestTakeMarkedFirstRefused. captureKey confirm case dropped (hjbfzd removed it). Local: go test -short ./huginn/internal/tui/ ok 71.214s (load ~230 held the tmux tests; the two 140x40 tmux help goldens are CI 44419243717's capture, diff = the Space row only).
check 3: Updated on 33616641 for byj479: TestTakeMarked takes 2 marked todo with one confirm naming them, A taken, B doing_with A, 2 commits, 'took 2 of 2'; TestTakeMarkedFirstRefused: A refused busy, B 'not taken: A was not', both stay marked, fix: hug task done F. d/x/w as before (TestDoneMarkedOneRefused, TestDropMarked, TestWaitMarked). CI 44421843248 8/8.
landed 4b49e10 on main 2026-11-19: tui: Space marks tasks in lists; the footer acts on the marked (rxr4z8)
landed 1614eb4 on main 2026-11-19: tui: goldens for Space mark on main's palette and asked screens (rxr4z8)
landed da04490 on main 2026-11-19: tui: the list key bar says h/l pane, so 140 columns keep the version (rxr4z8)
landed 0e577a0 on main 2026-11-19: tui: t on several marked takes the first and the rest with it (rxr4z8, byj479)
landed f7235d4 on main 2026-11-19: tui: marked and help goldens on main's TAGS column (rxr4z8)
landed 9b56535 on main 2026-11-19: tui: tags, taken-with and 140 tmux help goldens with Space mark (rxr4z8)
shipped in v0.7.73 (4b49e10)
## History
- 2026-11-18T06:18 todo -> doing/active (take)
- 2026-11-18T06:19 doing/active -> doing/asked (ask)
- 2026-11-18T12:56 doing/asked -> doing/asked (settle: one stage, asked); was waiting_asked_by: nautilus; waiting_on: Jørgen's answer on footer-mock.md (asked 05:19), then build; waiting_since: 2026-11-18T06:19
- 2026-11-18T20:27 answered asked, by jhf, for nautilus (doing/asked -> doing/active):
  > Before I build marks: is footer-mock.md (in FILES) what you want? Three choices in it: (1) ◆ as the mark, or another sign; (2) after an action, unmark the tasks that went through and keep the refused ones marked, or keep every mark; (3) Space on a grouped heading keeps folding, Space on a task marks.
  1. If you ask three questions, I should have three questions to answer in the ticket. This makes no sense. Let's structure this better. Make a new ticket if you need to. Now for the responses. Yes, the sigil for the mark is okay, but we need to have a combined sigil for when there is mark and focus. Is there a Unicode that can assist us that has something and >. I agree. If something is done, unmark everything that was processed and keep the ones that weren't. I agree. Let's keep the folding and the for grouped headings and space on task marks for now, unless I find something better to do. I know sometimes you can use X marks the spot too. That's a common enough, but uh maybe space is the easy one. And the short view. I think you should rather see like a question mark for guidance or CLI or help or something rather than showing just the letters without any explanation
- 2026-11-18T20:27 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:27 answer 1 acknowledged by crab
- 2026-11-18T20:29 answer 1 in progress by crab: hug task set rxr4z8
- 2026-11-18T21:07 doing/active -> todo/ready (give back; was assignee: crab); was doing_since: 2026-11-18T06:18
- 2026-11-18T21:27 todo/ready -> doing/active (take)
- 2026-11-18T21:30 doing/active -> doing/asked (ask)
- 2026-11-18T22:45 doing/asked -> doing/waiting (wait); was asked_by: macaque; asked_check: 7; asked_for: jhf; asked_show: cursor-on-mark.md; asked_since: 2026-11-18T21:30; asked_what: When the cursor is on a marked row, which sign should it show: A ⋗ or B ▸? The file draws both in your terminal.
- 2026-11-19T04:01 handed to crab: macaque's session ended (closed 2026-11-19 02:14)
