---
id: yf39ya
title: "Shipped work leaves the board clean, not a pile of lint"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Work that ships does not leave lint behind. Today after each release the board shows tasks that are done in fact but still open: checks-all-hold (11 tasks waiting on 'crab review/landing' long after they shipped), shipped-not-done (5 shipped tasks still open on a 'Jørgen confirms after release' check), and stale-owner (17 tasks whose worker session ended). The flow closes, hands over or asks at the moment each happens, so lint shows only real problems.

## Purpose
Jørgen 2026-11-18 (screenshot screen-2048.ans: 17 broken, 3 advice): 'Things that have shipped have a validation failure, so they show up in lint. So somehow we are leading ourselves into a place of lots of lint errors.' Lint that is mostly expected bookkeeping hides the real findings.

## Principle
Fix the flow, not the lint: no finding is silenced or downgraded to hide it. Each state change has an owner who acts at that moment: the release closes what it ships when every check holds; a check that waits on Jørgen after release becomes a question asked of him, not a lint error; a worker's tasks go back to the coordinator when its session ends. Nothing is closed without its checks holding.

## Relevant details
Measured 19:53Z on hug's board (hug task lint): 40 findings, 37 broken, on 31 tasks: stale-owner 17, checks-all-hold 11, shipped-not-done 5, missing-file 2, landed-unchecked 1, missing-task-ref 1, missing-section 2, long-title 1. Cleaned by hand at 19:54Z: 11 checks-all-hold tasks were all in v0.7.64-v0.7.66 and closed with hug task done; 17 stale owners reassigned to live workers or crab. The release step lives in the coordinator's release procedure (CHANGELOG commit, Release merge, tag); hug knows which tasks a release contains (shipped-not-done already reads it).

## Depends on

## Acceptance
- [ ] ~~Verify Figure out the better flow right now. Things that have shipped have a validation failure, so they show up in lint. So somehow we are leading ourselves into a place of lots of lint errors~~ (not relevant: replaced by checks 2-6: the request is now a stated goal with observable checks, crab, 2026-11-18)
- [ ] A release names the tasks it contains; each whose checks all hold is closed by the release step (hug command the coordinator runs, or hug upgrade/release hook), with the version in History (test)
- [ ] A shipped task whose only open check is the person's after-release confirmation shows as asked of that person (ASKED for them), not as a broken lint finding (test + golden)
- [ ] When a worker's session ends, its doing tasks go back to the board coordinator with a note, so stale-owner fires only when there is no coordinator either (test)
- [ ] On hug's board after the next release, hug task lint shows no checks-all-hold, shipped-not-done or stale-owner that the above should have prevented (count in a note)
- [ ] Jørgen confirms the lint view stays clean after a release

## Requests
- 2026-11-18T20:48 request 1 by jhf:
  Clean up this ticket. Take a look at the screenshot so you can see all the lint errors and when I take a screenshot, it would be nice if I could also attach it to the current ticket or attach it to a selected ticket or create a ticket
  resolved 2026-11-18T20:56 by crab:
    Cleaned up. The title is now 'Shipped work leaves the board clean, not a pile of lint'. The ticket has a Goal, Purpose and Principle and 5 observable checks: the release closes what it ships, an after-release confirmation shows as a question for you rather than a broken finding, and a worker's tasks return to the coordinator when its session ends. From your screenshot I also cleaned the board by hand: 11 shipped tasks whose checks all held are closed, and 17 tasks of ended workers are handed to live workers or crab. Attaching a screenshot to another task is its own task, hjbfzd.
- 2026-11-18T20:49 request 2 by jhf:
  And I didn't mean attach it to the current selected ticket, but I could say, okay, attach to ticket so-and-so or S for select or C for create. And if I press select, I can go pick one. And if I pray, if I put create, I can create a new one
- 2026-11-18T20:49 request 3 by jhf:
  Also, help fix this ticket with linting and other requirements so you can get it going

## Notes

file:screen-2048.ans (22856 bytes, sha256:8a9e2d1c) screen capture from hug tui


## History
- 2026-11-18T20:48 request 1 by jhf
- 2026-11-18T20:48 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:48 request 1 acknowledged by crab
- 2026-11-18T20:49 request 2 by jhf
- 2026-11-18T20:49 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:49 request 3 by jhf
- 2026-11-18T20:49 request 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:49 request 2 acknowledged by crab
- 2026-11-18T20:49 request 3 acknowledged by crab
- 2026-11-18T20:55 request 1 in progress by crab: hug task set yf39ya
- 2026-11-18T20:55 request 2 in progress by crab: hug task set yf39ya
- 2026-11-18T20:55 request 3 in progress by crab: hug task set yf39ya
- 2026-11-18T20:56 resolved request 1 by crab
