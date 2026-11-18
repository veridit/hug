---
id: zhvkmj
title: "Status colors: rows and sections share one subtle scheme; done shows a count"
state: "doing"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzzzz6h"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "active"
---

## Goal
Each status (todo, doing, blocked, done, dropped) has one subtle color, used in the status word, its section header (NEXT UP, IN PROGRESS, BLOCKED, RECENTLY DONE) and the overview count line, so a glance tells the state. The progress column shows a fraction only while work is partial: a done task shows its check count in the done color (● 5), not 5/5.

## Purpose
Jørgen 2026-11-17 on the hug overview (v0.7.37): 5/5 on done rows says nothing, since lint already requires every check ticked before done; he wants the plain count in green. He also wants todo, doing, blocked and done to have their own colors, reflected subtly in each section's header.

## Principle
Subtle, not loud: tint, not background blocks. Reuse the colors the count line already has (todo, doing, blocked red, done green) so there is one scheme. Red stays for problems only (blocked counts as one). Works in no-color mode by text alone. Proposed, for Jørgen to confirm in review: done shows ● N; todo with nothing ticked shows the check count dim (○ 3), no 0/3; a fraction or pie appears only when some but not all checks are ticked; dropped shows no progress.

## Relevant details
Progress pie came with d8yx7y (v0.7.33). Count line colors: huginn/internal/tui render. Section headers are all the same blue today. Coordinate with 102bc0 (rows fit one line) in the same render code.

## Depends on

## Acceptance
- [x] done rows show ● N in the done color, never N/N (golden)
- [x] each section header and the status word use that status's color; the count line uses the same colors (golden, 140x40 color)
- [x] no-color goldens still distinguish everything by text
- [ ] Jørgen confirms the look on his board after release
- [ ] On zhvkmj itself (2 .ans captures), Tab or ]/[ from the top of the detail reaches the Files section, j/k picks a capture and Space shows it in the embedded viewer; a tmux test on a copy of hug's board drives exactly that and fails first if it cannot (Jørgen 2026-11-18 19:52: he could not put focus in Files)
- [ ] Depends is not a problem colour: a task waiting on another task's checks reads in a calm colour of its own (not red, not amber), in the count line, the DEPENDS section heading and the row; red stays only for broken lint (TestOneColourPerMeaning updated, golden at 140 colour)

## Requests
- 2026-11-18T19:52 request 1 by jhf:
  I would love something is off when I open this I see one of the colors overview screens but unfortunately for me I can't navigate and put focus into the files section and thus view with the embedded viewer using space on purpose and I think I would like zooming as well by all means there and the color for dependency is red I don't like that that seems like dependencies is a problem it isn't
  resolved 2026-11-18T19:53 by crab:
    Added check 5 (Files reachable with Tab/] and Space shows a capture, tested on this very task, since you couldn't) and check 6 (depends gets its own calm colour, not red; red only for broken lint). Zoom is in 33v71x (+/- zoom, j/k scroll, o opens Preview). Check 4 (does it read as one scheme) waits until 5 and 6 ship.

## Notes

Jørgen 2026-11-17 08:42, on NEXT UP: the pie and fraction are useful on todo tasks that were started and paused (h176dp 4/5, c3hszg 3/4, n7jbmx 1/4, 0sw0dd 1/5): they show resumable work. Keep the fraction on todo, doing and blocked whenever some checks are ticked. Only done drops it, to ● N. Todo with nothing ticked: dim ○ N.

vole: on ci/zhvkmj (run 44058841019). check 1: progress.go progressText; TestProgressCellByStatus: done 5/5 -> '● 5' done colour, todo 0/3 -> '○ 3' dim, todo 4/5 '◕ 4/5' yellow, doing 1/4 '◔ 1/4', dropped blank (per Jørgen's 08:42 note: fraction stays on todo/doing/blocked whenever some are ticked). Goldens counted: before 58x '● 2/2' + 2x '● 12/12', after 60x '● N', 0x '● N/M', 0x '○ 0/'. check 2: sectionStyle: IN PROGRESS doing, NEXT UP todo, BLOCKED red, RECENTLY DONE green, LINT warning; TestSectionTitlesShareStatusColours (truecolor SGR prefix of each title = its status style; done row status word in done colour). The tmux color goldens store plain capture text, so colour is asserted by this test, not by the golden. check 3: TestNoColorReadsByText (no SGR, '● 12 ' and '○ 12 ' and status words present, no '12/12'); no-color goldens regenerated. check 4 (Jørgen confirms after release) open.

landed on main c6e0bff..c818f34 (ff). CI run 44066450337 (ff3c2a3, the same code; only crab's v0.7.40 CHANGELOG commit came in after): ubuntu, macos, windows and bench success. Checks 1-3 ticked with the evidence above. Check 4 (Jørgen confirms the look on his board after release) stays open, so the task stays doing until then.


file:screen-1821.ans (10206 bytes, sha256:e77117eb) screen capture from hug tui
crab ruling 02:52 local on Jørgen's 02:32 answer ('it bugs me that waiting and lint are two different issues [in one colour]... should lint be more red, like an error, or warning, or should waiting be less alarmist'): both. Today both use warning #ffb454. From his rules (red only for problems; subtle; ASKED is what he must address, WAITING is where inactive work parks): broken lint is a problem, so red (the blocked red #ff8585/#ae2525), and the ! mark and LINT section with it; advice-only lint is dim. Waiting on a thing (CI, a date) is calm: dim, no orange. Amber (#ffb454) is kept for one meaning only: something waits on YOU (asked of you, ? N for you, the ASKED section). Re-ask check 4 with a capture after the change.
nautilus: built per crab 02:52/03:01 on nautilus/zhvkmj, CI on ci/zhvkmj. Meaning -> colour table: e3sevr's header-spec.md 'Colours: one meaning each' (attached sha256:b44592e4); render.go forYouColour/problemColour/typeColours/priorityColours. TestOneColourPerMeaning (zhvkmj_colour_test.go) fails on main: 'dark: priority high #ffb454 is close to waits on you #ffb454', 'light: type bug #ae2525 is close to a problem #ae2525', '"WAITING" is not in its colour', '"LINT · tasks" is not in its colour', 'the wait on CI is not calm', 'amber on a line that waits on no one' (6 rows).
file:colours-overview.ans (6580 bytes, sha256:27713b8f) after 28791dea: one colour per meaning, test board at 120x32, user jhf
landed 6012b15e..28791dea (32e62173 rebased over the v0.7.53 CHANGELOG; code identical). CI 44221245136 on 32e62173: 6/6 success; tmux colour goldens did not drift. -race ./huginn/internal/tui ok 113.9s; tui suite ok 56.2s. TestOneColourPerMeaning passes; on main it failed with 15 lines (verbatim in the note above).
landed 28791de on main 2026-11-18: tui: one colour per meaning: amber only for what waits on you, red only for what…
shipped in v0.7.54 (28791de)
crab 19:54 local, request 1 (Jørgen 19:52): three points. (1) Files not reachable on this task: new check 5, reproduce on zhvkmj itself first. (2) zoom: that is 33v71x (+/- zoom, j/k scroll, o opens Preview), filed 19:48 from his 911j70 request. (3) red for depends reads like a problem, which it is not: new check 6, depends gets its own calm colour, red only for broken lint. That overrides crab's 02:52 ruling that grouped depends with problems.
## History
- 2026-11-17T14:30 doing -> doing/waiting (wait)
- 2026-11-18T02:05 doing/waiting -> doing/asked (hug upgrade, 41hhgn: a question to a person is its own stage)
- 2026-11-18T02:32 answered asked (check 4), by jhf, for nautilus (doing/asked -> doing/active):
  > do the status colours read as one subtle scheme? The capture shows the overview on this board (IN PROGRESS, NEXT UP, RECENTLY DONE, done rows with a green count)
  It bugs me that weighting and lint are two different issues. I'm wondering if lint should be more red, like an error, or if warning, or if whiting should be less alarmist
- 2026-11-18T02:50 doing/active -> doing/waiting (wait)
- 2026-11-18T02:50 nautilus read the answer: hug task wait zhvkmj "crab's ruling on Jørgen's 02:xx answer: lint vs waiting colour (red/error for lint, or a calmer waiting)"
- 2026-11-18T03:18 doing/waiting -> doing/asked (ask)
- 2026-11-18T12:56 doing/asked -> doing/asked (settle: one stage, asked); was waiting_asked_by: nautilus; waiting_on: crab's ruling on Jørgen's 02:xx answer: lint vs waiting colour (red/error for lint, or a calmer waiting); waiting_since: 2026-11-18T02:50
- 2026-11-18T19:52 request 1 by jhf
- 2026-11-18T19:52 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T19:52 request 1 acknowledged by crab
- 2026-11-18T19:53 request 1 in progress by crab: hug task set zhvkmj
- 2026-11-18T19:53 resolved request 1 by crab
- 2026-11-18T19:57 answered asked (check 4), by jhf, for nautilus (doing/asked -> doing/active):
  > after the release: amber now means only 'waits on you' (ASKED, ? N for you), red only 'wrong' (broken lint, depends), waiting on CI and advice are dim. Does it read as one subtle scheme?
  Well, it looks good enough, it's too much color, and I told you already, I told you to adjust this. The pen shouldn't be red, it's too intense in the currently selected row. I have another ticket making it more like a spreadsheet, more subtle, but yeah, it's good enough for this ticket, but there will be follow-up tickets
