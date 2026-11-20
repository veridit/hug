---
id: r80svs
title: "View line: bold marks keys, underline marks state, colour marks the key step"
state: "done"
type: feature
priority: ""
assignee: ["rose"]
labels: []
group: ""
rank: "zzzzzzzzzzyq"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T12:51"
parent: "5zs05h"
---

## Goal
The view line gives each cue one meaning. Bold: a key you can press (v, o, l, b, !). Colour: the step of the key sequence (v in the accent, o l b ! in violet), which shows that v opens the others. Steady underline under the whole word: the active view. Steady underline under one letter: the key you picked (v while it waits). Blinking underline: the keys you are choosing between now (o l b ! after v). The rest of the line stays dim; no fills.

## Purpose
Jørgen 2026-11-19 11:09Z (crab console, screenshot of v0.7.81's view line): bold, colour and underline are used together, it is too much, and there is no way to know which view is active. He proposed: bold or underline for possible keys, not both; with bold for keys, a fixed underline for what is selected, a blinking underline for what you are picking between, and an underline under the whole word for the active view. 11:11Z: keep the colour, because it tells you it is a two-stage thing: v activates the others.

## Principle
One cue per meaning (qh9ep1 check 9). Replaces a82j3n's solid filled v (v0.7.81) with a steady underline on v, and today's bold-only active view with a whole-word underline. Blink from the existing vCursor ticks (530 ms), no new timer. NO_COLOR still readable: bold and underline carry everything but the step.

## Relevant details
Rest, list view: v o l b ! bold and coloured by step; 'list' underlined as a whole word (its l included). After v: v bold, accent, steady underline; o l b ! bold, violet, underline blinking; the active word keeps its underline on the letters other than its key letter, which blinks with the others. After the second key or Esc: rest look, the new view underlined. Code: huginn/internal/tui/viewline.go viewLine, render.go (viewPrefixKey, viewKey, viewCurrent, viewPicked), goldens testdata/header/view-*.txt.

## Depends on

## Acceptance
- [x] At rest the active view's whole word is underlined, no other word is, and keys are bold with no underline (golden 100x30 and 160x45)
- [x] After v, v has a steady underline and no fill, o l b ! underlines blink together on vCursor ticks, the active word's other letters keep a steady underline (test with two frames 0.53 s apart)
- [x] Colour: v in the accent, o l b ! in violet, in every state; NO_COLOR shows the same bold and underlines (golden)
- [ ] ~~Jørgen looks at the line in hug tui after the release and can tell the active view and what v offers~~ (not relevant: moved to its release ticket (v0.7.82) as an ask, per 75mxf3 trial, crab, 2026-11-19)

## Notes


r80svs: failing tests first (r80svs_test.go, worktree 8889481d): TestViewLineRestUnderlinesActiveWord: '100 overview at rest: "v" at column 0 bold false, a key true'; TestViewLineAfterVBlinksTheChoices/100x30 and /160x45: 'after v, blink on: "v" at column 0 is filled: "\x1b[38;2;11;22;32;48;2;0;191;255mv"'; TestViewLineNoColorKeepsCues: 'NO_COLOR at rest: "v" at column 0 bold false, a key true'. Finding: termenv Ascii (NO_COLOR) drops every attribute, so the view line must write bold/underline itself under NO_COLOR.
check 1: 686d0955 (branch r80svs): TestViewLineRestUnderlinesActiveWord checks every cell of the view line at rest at 100 and 160 in each view (whole active word underlined, key letter too, no other word, keys bold and not underlined, no fill); golden testdata/header/view-blink-{100x30,160x45}.txt idle cues row 'B     U_______  B     B      B'. Failed before the fix: '100 overview at rest: "v" at column 0 bold false, a key true'. Local niced go test ./...: 40 ok, 0 FAIL.
check 2: 686d0955: TestViewLineAfterVBlinksTheChoices/100x30 and /160x45 take two frames one real vCursor tick apart (tick 500-900 ms, bubbles 530 ms): blink on cues 'U     U_______  U     U      U', blink off 'U     B_______  B     B      B' (v steady underline, no fill; o l b ! underlines blink together; 'verview' keeps its underline); after v l / v ! / Esc back to rest, new view underlined. Also TestViewKeysBlinkAfterV (frames differ only in o l b ! underlines), TestPickedVIsUnderlined. Failed before the fix: 'after v, blink on: "v" at column 0 is filled: "\x1b[38;2;11;22;32;48;2;0;191;255mv"'.
check 3: 686d0955: checkCues asserts v #00bfff and o l b ! #c9b3ff in every state (rest, both blink phases, after v l/!/Esc); goldens' colours row 'a.... v.........v.....v......v....' in all 5 states. TestViewLineNoColorKeepsCues: with NO_COLOR=1 no colour SGR, and bold/underline equal cell for cell to the colour line at rest and both phases; golden 'NO_COLOR idle: "\x1b[1mv\x1b[0miew: \x1b[1;4mo\x1b[0m\x1b[4mverview...' (view-cursor.txt, view-blink-*.txt). Failed before the fix: 'NO_COLOR at rest: "v" at column 0 bold false, a key true'.
r80svs: goldens rewritten: 4 files (testdata/header/view-blink-100x30.txt, view-blink-160x45.txt, view-cursor.txt, testdata/colour/depends-140.txt), each diff reviewed: only the view line's SGR changed, plus new cue/colour rows and NO_COLOR frames in view-blink-*; screen text unchanged; tmux goldens (TestTmuxGolden, 4 subtests) unchanged and passing. capture prompt s/c keys keep their underlined, not bold look (capture.go no longer borrows viewPrefixKey as-is).
r80svs: CI run 44516640510 on ci/r80svs, headSha 686d09555739b24e19ce3a87c4423935a06c74e1: conclusion success, 8/8 jobs success (secrets, bench, test macos/ubuntu/windows-latest, test-windows-cli rest/sync/shards). Branch r80svs pushed at the same SHA, not merged. Check 4 is Jørgen's, after release.
crab 11:52Z: reviewed viewline.go (bold keys, colour by step, underline for state, cue() keeps bold/underline under NO_COLOR), render.go styles, 4 goldens. Main had not moved: fast-forwarded to 686d0955 (CI 44516640510 8/8 at that SHA). Release v0.7.82: CHANGELOG 2a3690b8, merge 67652429, release.yml 44518564332. Per 75mxf3 trial, check 4 moves to the v0.7.82 release ticket as an ask.
Reflection (unanswered, 2026-11-19, crab)
Jørgen approved the view line on release ticket 5zs05h, 2026-11-20 10:19Z: 'Yes, this looks okay. Approved.'
## History
- 2026-11-19T12:11 todo -> doing/active (take)
- 2026-11-19T12:51 doing/active -> done (done); was doing_since: 2026-11-19T12:11
