---
id: r80svs
title: "View line: bold marks keys, underline marks state, colour marks the key step"
state: "doing"
type: feature
priority: ""
assignee: ["rose"]
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T12:11"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] At rest the active view's whole word is underlined, no other word is, and keys are bold with no underline (golden 100x30 and 160x45)
- [ ] After v, v has a steady underline and no fill, o l b ! underlines blink together on vCursor ticks, the active word's other letters keep a steady underline (test with two frames 0.53 s apart)
- [ ] Colour: v in the accent, o l b ! in violet, in every state; NO_COLOR shows the same bold and underlines (golden)
- [ ] Jørgen looks at the line in hug tui after the release and can tell the active view and what v offers

## Notes


r80svs: failing tests first (r80svs_test.go, worktree 8889481d): TestViewLineRestUnderlinesActiveWord: '100 overview at rest: "v" at column 0 bold false, a key true'; TestViewLineAfterVBlinksTheChoices/100x30 and /160x45: 'after v, blink on: "v" at column 0 is filled: "\x1b[38;2;11;22;32;48;2;0;191;255mv"'; TestViewLineNoColorKeepsCues: 'NO_COLOR at rest: "v" at column 0 bold false, a key true'. Finding: termenv Ascii (NO_COLOR) drops every attribute, so the view line must write bold/underline itself under NO_COLOR.
## History
- 2026-11-19T12:11 todo -> doing/active (take)
