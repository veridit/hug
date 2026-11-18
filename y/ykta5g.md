---
id: ykta5g
title: "In a task, k after j scrolled past the last check jumps to the top"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
j and k in the detail are mirror images: after j steps through the checks and then scrolls the page past the last one, k first scrolls the page back up one line at a time until the marked check is in view, and only then steps to the previous check; the view never jumps

## Purpose
Jørgen 2026-11-18 19:55 local: 'J goes down but K goes to top again, that doesn't make any sense'. Reproduced on a copy of hug's board at 100x24 on zhvkmj: 25 j leave acceptance at check 6 (last) with the page scrolled to y=23; one k sets acceptance 5 and revealAcceptance re-centres to y=0

## Principle
Fix the key, not the symptom: no special case per task. The marked item and the scroll position move together in both directions. j/k on a list keep their behaviour while the marked item is on screen

## Relevant details
huginn/internal/tui/tui.go:1608 (case j/down/k/up in the detail): moving the selection calls revealAcceptance (detail.go:941), which sets YOffset to i-Height/2 whenever the marker is off screen. After j scrolled the page past the last check, the marker is above the view, so the first k both moves the selection and re-centres. Repro test: /Users/jhf/.jcode/scratch/kprobe/zz_probe2_test.go (ZZ_BOARD=board copy).

## Depends on

## Acceptance
- [ ] A key test: in a task whose checks end above the bottom of a 24-row screen, j until the page scrolls N lines past the last check, then k N times scrolls back up one line each with the last check still marked, and the next k marks the previous check with no jump (fails first on main)
- [ ] The same holds for dependencies, links and files lists (table test over the four panes)
- [ ] Jørgen confirms j/k in a task feel symmetric after release

## Notes
