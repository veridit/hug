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
doing_since: "2026-11-18T06:18"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] Space in overview, list and board marks or unmarks the focused task and moves to the next row; marked rows show a mark and a quiet tint (golden at 80 and 140)
- [ ] With N marked, the footer reads 'N marked ·' followed by the actions valid for all of them and 'Esc unmark'; Esc clears every mark (key test)
- [ ] t, d, x and w act on every marked task after one confirm naming the count, and a refused task is named with its fix: line while the others go ahead (key test with 3 marked, one refused)
- [ ] m grabs the marked tasks as one block; J/K and the drop move them together and keep their order (key test)
- [ ] Inside a task Space still ticks the focused check (existing tests stay green)
- [ ] Jørgen confirms on his board after release
- [ ] Jørgen 19:27Z (answer 1): a marked row that also has the cursor shows one combined sign, not > or ◆ alone. Proposed ⋗ (U+22D7, greater-than with dot: the cursor carrying a mark; single width); the alternative ▸ (U+25B8). The mock shows both rows side by side and asks this one question
- [ ] Jørgen 19:27Z: the narrow footer (under 70 columns) never shows bare letters: it reads e.g. '3 marked · ? actions · Esc unmark', and ? lists the actions with their words (key test at 60 columns)
- [ ] Jørgen 19:27Z: after an action, the tasks that went through are unmarked and the refused ones stay marked (key test, extends check 3)

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
