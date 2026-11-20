---
id: ykta5g
title: "In a task, k after j scrolled past the last check jumps to the top"
state: "doing"
type: bug
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzvv"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:00"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "Jørgen confirms j/k in a task feel symmetric after release"
asked_for: "jhf"
asked_since: "2026-11-18T23:24"
asked_by: "crab"
asked_check: "3"
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
- [x] A key test: in a task whose checks end above the bottom of a 24-row screen, j until the page scrolls N lines past the last check, then k N times scrolls back up one line each with the last check still marked, and the next k marks the previous check with no jump (fails first on main)
- [x] The same holds for dependencies, links and files lists (table test over the four panes)
- [ ] Jørgen confirms j/k in a task feel symmetric after release
- [x] Jørgen 19:37Z (gav9nw request 1): when a check gets focus (on open, or by j/k), the whole check shows, question and all, not only its first line; a check taller than the screen shows from its first line. Key test on a task asked with a long multi-line question at 100x24 (e.g. gav9nw)

## Questions
- 2026-11-18T23:24 question 1 by crab to jhf (check 3):
  Jørgen confirms j/k in a task feel symmetric after release

## Notes


check 1: 944b165d (pushed ykta5g, ci/ykta5g, CI 44381652998). TestDetailJKMirrorInEveryList/checks failed first on main 6a6b6573: 'ykta5g_test.go:121: k 1 (item 2, line 6, offset 7): the page jumped to 0'; passes after: 'ok github.com/veridit/hug/huginn/internal/tui 1.133s'. Crab's probe on kprobe-board-1856, zhvkmj at 100x24: k now y=23->22->21->20 with acc=5 still marked.
check 2: Same table test over panes checks, dependencies, links, files: all 4 failed first on main (deps 'offset 12 ... jumped to 4', links 'offset 17 ... jumped to 9', files 'offset 22 ... jumped to 14'); all 4 pass after.
check 4: 68029838 on main 939636ff (pushed ykta5g + ci/ykta5g, CI 44386677307). TestFocusedCheckShowsWhole at 100x24, asked of jhf on check 2 with a 3-paragraph question: on e9bc27ce 'ykta5g_test.go:196: on open: the focused check is not whole on screen, missing "Which option do we take?"' (page ended at '> ? ☐ 2. two: decide which option'); after: 'ok ... 0.986s'. Covers open, j onto it, k back onto it, and a 40-line question: shown from its first line, j reads it a line at a time then steps to check 3. markedBlock (detail.go) replaces the first-line search in openTop and revealAcceptance. TestTmuxReload failed once at load 151 (paused 190 s by its load guard), passed 2/2 after at load 73: 'PASS (14.26s)', 'PASS (13.65s)'.
14481822 (ykta5g, ci/ykta5g, run 44393599578) stacks on kikazaru's 57dd9e90 (zhvkmj; not yet on origin/main 8bf24c02). markedBlock searches from focusedHeading. Rebasing exposed a real conflict: openTop's showMarked scrolled zhvkmj's page past its asked capture (page line 0) to check 4 (line 53 at 140x40), and TestTmuxZhvkmjFilesOnItself failed 'never showed "capture, as it was taken"'. It passed on 57dd9e90 alone at load 78. Fix: openTop leaves the page at the top when askedPicture is set. Local: the whole huginn/internal/tui package gives 'ok ... 119.946s'.
crab 21:50 local: 14481822 accepted on review (CI 44393599578 8/8; openTop keeps the page at the top when an asked picture shows, so zhvkmj's capture stays in view). Stacked as ci/ykta5g-rebased 2c609710 on gav9nw, z77pem, mbdfzs3 over main d5c66a1f.
landed c0ac0d5 on main 2026-11-18: TUI: j and k mirror each other in a task's lists; a focused check shows whole…
shipped in v0.7.71 (c0ac0d5)
## History
- 2026-11-18T20:00 todo -> doing/active (take)
- 2026-11-18T21:02 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> doing/asked (ask); was waiting_asked_by: piglet; waiting_on: CI run 44386677307; waiting_since: 2026-11-18T21:02
- 2026-11-20T02:45 handed to crab: piglet's session went silent (last seen 2026-11-19 02:41)
