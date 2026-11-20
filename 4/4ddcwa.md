---
id: 4ddcwa
title: "Detail page frame: ↑ on the top rule, a bottom rule with ↓ and the page status"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzh"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
A task's page sits between two rules: ↑ on the top one, ↓ and the page status on the bottom one.

## Purpose
You see where the ticket ends and whether there is more, and the page's own status is kept apart from the global keys.

## Principle
The page owns its frame and its status; the last line stays the global keys, and no blank line is wasted.

## Relevant details
Layout (Jørgen 2026-11-20 10:32Z, request 1 on qpp30y, screenshot screen-1130.ans there), bottom up:
- last line: the global keys, as today.
- above it, one bottom rule: '↓──' (the ↓ only when more is below), then the status of the thing you are in (today's 'Task 3 of 14 · 0% · next · previous · Esc back'), then the rule runs on to the edge. It replaces today's status line plus the blank line under it.
- top: the rule under the title gets a small ↑ at its left end when the page is scrolled down.
Sketch:
  ↑─────────────────────────────────────────────
  ...page...
  ↓── Task 3 of 14 · 0% · next · previous · Esc back ─────────────
  j/k check · Space tick · ... (global keys)
Replaces 8a0ybb check 9 (↑/↓ in the corners). Related: xt0qvf (where the status message sits in the lists). His words: 'there should be a small arrow pointing up on the long line on the left part of the below the subject ... there should be a line on the bottom of the screen where the ticket ends ... an arrow as well pointing down on that line ... that line should separate from what is below ... the one line that starts with an arrow down, a piece of line, and then any inline status for this thing I'm in now. And then the next line is the global line. Because right now you can see that there is this extra space.'

## Depends on

## Acceptance
- [ ] Jørgen approves ANSI frames at 100x30 of a long task: at the top, scrolled mid-page, and at the end
- [ ] The rule under the title shows ↑ at its left end exactly when the page is scrolled down (golden)
- [ ] A bottom rule sits right above the global keys, starts with ↓ when more is below, and carries the page status; no blank line between page, rule and keys (golden 100x30 and 160x45)
- [ ] Jørgen opens a long task after release and sees where it ends and that there is more

## Notes
