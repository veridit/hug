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
stage: "ready"
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
- [x] Jørgen confirms the dated-wait line and its 'til 11-24' list cell read right

## Questions
- 2026-11-24T00:57 question 1 by crab to jhf (check 1):
  v0.7.111 is out (brew upgrade veridit/tap/hug). A wait for a date now leads with it: 'waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08' (before: 'waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'). In hug tui's list the stage cell says 'til 11-24', because 'until 11-24' does not fit the column. Does that read right to you, or would you rather widen the column (every title loses 2 characters)?
  answered 2026-11-25T13:25 by jhf:
    I think the age-old month or day, day or month pops up in my mind now, but I think until three letter month abbreviation space day of the month would make sense to me as well or relative like wait three days and then absolute if further away than say a week or two weeks or something sensible for humans

## Notes

Shipped atex3d (worker elephant, Opus 5.5 high): 15 of 15 mutants caught, 1 new golden (rows/dated-wait-120.txt), no existing golden changed, no schema bump. Old tasks keep their stored fields and show the new line. Also in this release: tfe9nr (tests only). Before: 'stage: waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'. After: 'stage: waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08'. Two choices were crab's (23:07Z): the list STAGE cell stays 10 wide and says 'til 11-24' (no room for 'until'), and 'since' keeps each screen's own format.


check 1: Jørgen 2026-11-25 13:25: the date-first wait is fine; he wants the date written as people say it, filed as its own task
## History
- 2026-11-24T00:57 todo -> todo/design (ask)
- 2026-11-25T13:25 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.111 is out (brew upgrade veridit/tap/hug). A wait for a date now leads with it: 'waiting until 2026-11-24 · then ask jhf · since 2026-11-18 05:08' (before: 'waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'). In hug tui's list the stage cell says 'til 11-24', because 'until 11-24' does not fit the column. Does that read right to you, or would you rather widen the column (every title loses 2 characters)?
  I think the age-old month or day, day or month pops up in my mind now, but I think until three letter month abbreviation space day of the month would make sense to me as well or relative like wait three days and then absolute if further away than say a week or two weeks or something sensible for humans
- 2026-11-25T13:25 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-25T14:09 crab read the answer: hug task check ersks7 1 "Jørgen 2026-11-25 13:25: the date-first wait is fine; he wants the date written as people say it, filed as its own task"
