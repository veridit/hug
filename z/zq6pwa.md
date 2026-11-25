---
id: zq6pwa
title: "In a task, focus follows the screen: h/l pick what is in view"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzw3"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
In a task's page, h and l focus what is in view, and scrolling shows what can take focus without stealing it.

## Purpose
Jørgen 2026-11-20 10:12Z (p87ngv request 4): after scrolling down with j or J, pressing l jumps back up to where focus last was; scrolling into the question section moves one question at a time; nothing shows what can take focus as he scrolls.

## Principle
Scrolling moves the view, h/l move focus, and focus never pulls the view away from where the reader is; what can take focus is marked subtly, not loudly.

## Relevant details
Keys today in a task's page: j/k scroll (or step inside a focused list), J/K page, h/l and Tab step between focusable things (checks, questions, files, links, ids). Problems seen on p87ngv (v0.7.88): (1) after scrolling, l returns to the old focus off screen; (2) j stepping into ## Questions walks them one by one instead of scrolling; (3) nothing marks focusable items while scrolling. Wanted: l/h take the next/previous focusable item on screen; if none is on screen, the next one beyond it; j/k scroll unless a list is explicitly entered; focusable items carry a subtle marker (dim gutter sign) as they come into view. Keys provisional until bfxq93's review.

## Depends on

## Acceptance
- [ ] After scrolling with j or J so the old focus is off screen, l focuses the first focusable item in view, h the last; with none in view, the next or previous beyond it (tmux test)
- [ ] j and k scroll the page through ## Questions and Acceptance; they step item by item only inside a list you entered with l or Enter (tmux test)
- [ ] Focusable items in view carry one subtle gutter marker; the focused one carries the cursor (golden 100x30)
- [ ] Jørgen reads p87ngv top to bottom with j/J and reaches any question with h/l without the view jumping away

## Notes
