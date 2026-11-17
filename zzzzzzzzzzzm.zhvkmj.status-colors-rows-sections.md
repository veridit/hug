---
id: zhvkmj
title: "Status colors: rows and sections share one subtle scheme; done shows a count"
status: "doing"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_since: "2026-11-17T14:30"
waiting_asked_by: "nautilus"
waiting_on: "jhf: look at the colours after v0.7.41"
stage: "waiting"
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

## Notes

Jørgen 2026-11-17 08:42, on NEXT UP: the pie and fraction are useful on todo tasks that were started and paused (h176dp 4/5, c3hszg 3/4, n7jbmx 1/4, 0sw0dd 1/5): they show resumable work. Keep the fraction on todo, doing and blocked whenever some checks are ticked. Only done drops it, to ● N. Todo with nothing ticked: dim ○ N.

vole: on ci/zhvkmj (run 44058841019). check 1: progress.go progressText; TestProgressCellByStatus: done 5/5 -> '● 5' done colour, todo 0/3 -> '○ 3' dim, todo 4/5 '◕ 4/5' yellow, doing 1/4 '◔ 1/4', dropped blank (per Jørgen's 08:42 note: fraction stays on todo/doing/blocked whenever some are ticked). Goldens counted: before 58x '● 2/2' + 2x '● 12/12', after 60x '● N', 0x '● N/M', 0x '○ 0/'. check 2: sectionStyle: IN PROGRESS doing, NEXT UP todo, BLOCKED red, RECENTLY DONE green, LINT warning; TestSectionTitlesShareStatusColours (truecolor SGR prefix of each title = its status style; done row status word in done colour). The tmux color goldens store plain capture text, so colour is asserted by this test, not by the golden. check 3: TestNoColorReadsByText (no SGR, '● 12 ' and '○ 12 ' and status words present, no '12/12'); no-color goldens regenerated. check 4 (Jørgen confirms after release) open.

landed on main c6e0bff..c818f34 (ff). CI run 44066450337 (ff3c2a3, the same code; only crab's v0.7.40 CHANGELOG commit came in after): ubuntu, macos, windows and bench success. Checks 1-3 ticked with the evidence above. Check 4 (Jørgen confirms the look on his board after release) stays open, so the task stays doing until then.


## History
- 2026-11-17T14:30 doing -> doing/waiting (wait)
