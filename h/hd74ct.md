---
id: hd74ct
title: "Overview: rules only around the focused section"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-22
updated: 2026-11-22
---

## Goal
In the overview and lists only the focused section sits between rules; the others show a plain title line, so fewer lines cross the screen.

## Purpose
Jørgen 2026-11-22 21:30Z (request 1, screen-2230.ans of v0.7.108): with a titled rule and an end rule on every section the overview has too many horizontal lines; he asks to show them only for the active section.

## Principle
One frame on screen: the rules mark where you are, so only the section you move in has them. Same rule style, arrows and focus cue as v68n82 and 4ddcwa; nothing new is added.

## Details
Today (v0.7.108, v68n82 variant A): every section has a titled top rule '─── WAITING [19] · 1–3 shown ───' and an end rule with ↓, so 6 sections draw 11 rules (screen-2230.ans).
Proposed (variant C):
     ASKED · for you [34] · 1–20 shown
        bxqtkt   Contributions: ...
     IN PROGRESS · assigned to [0] · None
  ─── WAITING [19] · 1–3 shown ────────────────
  >     s6ff5k   Measure amortized cleanup ...
        c3hszg   Recall: user-weighted ...
  ↓───────────────────────────────────────────
     NEXT UP · rank order [31] · 1–3 shown
        a1dx7s   seid-lang: ...
- The focused section keeps its titled top rule and end rule, bright, with ↑/↓ on them when rows are hidden.
- Every other section shows its title on a plain dim line (no dashes) and no end rule; its count ('1–3 shown' of [19]) says rows are hidden.
- Moving focus (h/l, n/p) moves the rules with it. Lists (one section) and the board are unchanged unless the frames show they should follow.
- Rows: each unfocused section gives back its end rule, about 5 task rows more at 100x30 than variant A's 12.
This answers v68n82 check 2 (A or B) with a third option; Jørgen's pick here settles it.
Build in huginn/internal/tui/sectionframe.go (sectionFrame, sectionRule); B is sectionFrame="shared" today.

## Depends on

## Acceptance
- [ ] Verify Too many horisontal lines?

## Requests
- 2026-11-22T22:31 request 1 by jhf:
  How about just showing the for the active section?

## Notes

file:screen-2230.ans (18179 bytes, sha256:caeb9471) screen capture from hug tui


## History
- 2026-11-22T22:31 request 1 by jhf
- 2026-11-22T22:31 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-22T22:31 request 1 acknowledged by crab
- 2026-11-22T22:31 request 1 in progress by crab: hug task set hd74ct
