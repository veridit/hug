---
id: ersks7
title: "Release v0.7.111: a dated wait leads with the date"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzh"
created: 2026-11-24
updated: 2026-11-24
design_question: "v0.7.111 is out (brew upgrade veridit/tap/hug). A wait for a date now leads with it: 'waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08' (before: 'waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'). In hug tui's list the stage cell says 'til 11-24', because 'until 11-24' does not fit the column. Does that read right to you, or would you rather widen the column (every title loses 2 characters)?"
design_for: "jhf"
design_since: "2026-11-24T00:57"
design_asked_by: "crab"
stage: "design"
---

## Goal
v0.7.111 is out and a wait for a date reads with the date first, once, everywhere.

## Purpose
One ticket per release; its shipped tasks hang under it.

## Principle
Ask one question, with a real line from the board.

## Details
Main ed882297 (merge of c5925996 on 0c09d4ee), CI 45290106766 8/8, release.yml 45291771370, brew hug v0.7.111. Ships atex3d (from jhf's z2vmte request 11-20) and tfe9nr (tests only: JSON parsed from stdout).

## Depends on

## Acceptance
- [ ] Jørgen confirms the dated-wait line and its 'til 11-24' list cell read right

## Questions
- 2026-11-24T00:57 question 1 by crab to jhf (check 1):
  v0.7.111 is out (brew upgrade veridit/tap/hug). A wait for a date now leads with it: 'waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08' (before: 'waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'). In hug tui's list the stage cell says 'til 11-24', because 'until 11-24' does not fit the column. Does that read right to you, or would you rather widen the column (every title loses 2 characters)?

## Notes

Shipped atex3d (worker elephant, Opus 5.5 high): 15 of 15 mutants caught, 1 new golden (rows/dated-wait-120.txt), no existing golden changed, no schema bump. Old tasks keep their stored fields and show the new line. Also in this release: tfe9nr (tests only). Before: 'stage: waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'. After: 'stage: waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08'. Two choices were crab's (23:07Z): the list STAGE cell stays 10 wide and says 'til 11-24' (no room for 'until'), and 'since' keeps each screen's own format.


## History
- 2026-11-24T00:57 todo -> todo/design (ask)
