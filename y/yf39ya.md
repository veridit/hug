---
id: yf39ya
title: "Shipped work leaves the board clean, not a pile of lint"
state: "doing"
type: feature
priority: ""
assignee: ["chipmunk"]
labels: []
group: ""
rank: "zzzzzzzzzzw8"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:56"
stage: "waiting"
model: "claude-opus-5-5"
branch: "jrj1vq-c3"
worktree: "/Users/jhf/.jcode/scratch/jrj1vq"
waiting_asked_by: "bonehound"
waiting_on: "landing: branch yf39ya cfaa511d (3 commits on v0.7.69), CI 44391027727 green; crab merges/releases, then check 5 counts lint after the release and check 6 is Jørgen's"
waiting_since: "2026-11-18T21:34"
---

## Goal
Work that ships leaves no lint behind: the flow closes, hands over or asks the moment each case happens.

## Purpose
Jørgen 2026-11-18 (screenshot screen-2048.ans: 17 broken, 3 advice): 'Things that have shipped have a validation failure, so they show up in lint. So somehow we are leading ourselves into a place of lots of lint errors.' Lint that is mostly expected bookkeeping hides the real findings.

## Principle
Fix the flow, not the lint: no finding is silenced or downgraded to hide it. Each state change has an owner who acts at that moment: the release closes what it ships when every check holds; a check that waits on Jørgen after release becomes a question asked of him, not a lint error; a worker's tasks go back to the coordinator when its session ends. Nothing is closed without its checks holding.

## Relevant details
Work that ships does not leave lint behind. Today after each release the board shows tasks that are done in fact but still open: checks-all-hold (11 tasks waiting on 'crab review/landing' long after they shipped), shipped-not-done (5 shipped tasks still open on a 'Jørgen confirms after release' check), and stale-owner (17 tasks whose worker session ended). The flow closes, hands over or asks at the moment each happens, so lint shows only real problems.

Measured 19:53Z on hug's board (hug task lint): 40 findings, 37 broken, on 31 tasks: stale-owner 17, checks-all-hold 11, shipped-not-done 5, missing-file 2, landed-unchecked 1, missing-task-ref 1, missing-section 2, long-title 1. Cleaned by hand at 19:54Z: 11 checks-all-hold tasks were all in v0.7.64-v0.7.66 and closed with hug task done; 17 stale owners reassigned to live workers or crab. The release step lives in the coordinator's release procedure (CHANGELOG commit, Release merge, tag); hug knows which tasks a release contains (shipped-not-done already reads it).

## Depends on

## Acceptance
- [ ] ~~Verify Figure out the better flow right now. Things that have shipped have a validation failure, so they show up in lint. So somehow we are leading ourselves into a place of lots of lint errors~~ (not relevant: replaced by checks 2-6: the request is now a stated goal with observable checks, crab, 2026-11-18)
- [x] A release names the tasks it contains; each whose checks all hold is closed by the release step (hug command the coordinator runs, or hug upgrade/release hook), with the version in History (test)
- [x] A shipped task whose only open check is the person's after-release confirmation shows as asked of that person (ASKED for them), not as a broken lint finding (test + golden)
- [x] When a worker's session ends, its doing tasks go back to the board coordinator with a note, so stale-owner fires only when there is no coordinator either (test)
- [ ] On hug's board after the next release, hug task lint shows no checks-all-hold, shipped-not-done or stale-owner that the above should have prevented (count in a note)
- [ ] Jørgen confirms the lint view stays clean after a release
- [x] A task with an open question to a person never shows shipped-not-done (txjkrm 2026-11-20: todo, question 1 to jhf open since 11-18, still broken) (test)
- [ ] The release step asks or hands over every shipped task whose open check is not a person's: the coordinator's own checks (x8zzzn c3, ynh84t c4 on 11-20) become its active work, never a broken finding (test)
- [ ] A task in depends whose dependency is done or shipped is not shipped-not-done; its person check is asked once the dependency holds (0gztsy on kg07bt, 2026-11-20) (test)

## Requests
- 2026-11-18T20:48 request 1 by jhf:
  Clean up this ticket. Take a look at the screenshot so you can see all the lint errors and when I take a screenshot, it would be nice if I could also attach it to the current ticket or attach it to a selected ticket or create a ticket
  resolved 2026-11-18T20:56 by crab:
    Cleaned up. The title is now 'Shipped work leaves the board clean, not a pile of lint'. The ticket has a Goal, Purpose and Principle and 5 observable checks: the release closes what it ships, an after-release confirmation shows as a question for you rather than a broken finding, and a worker's tasks return to the coordinator when its session ends. From your screenshot I also cleaned the board by hand: 11 shipped tasks whose checks all held are closed, and 17 tasks of ended workers are handed to live workers or crab. Attaching a screenshot to another task is its own task, hjbfzd.
- 2026-11-18T20:49 request 2 by jhf:
  And I didn't mean attach it to the current selected ticket, but I could say, okay, attach to ticket so-and-so or S for select or C for create. And if I press select, I can go pick one. And if I pray, if I put create, I can create a new one
  resolved 2026-11-18T20:56 by crab:
    Done as you describe, in hjbfzd (high): after S, the prompt takes a task ID, or s to pick one from the list, or c to create a new task with the capture attached.
- 2026-11-18T20:49 request 3 by jhf:
  Also, help fix this ticket with linting and other requirements so you can get it going
  resolved 2026-11-18T20:56 by crab:
    yf39ya now lints clean ('no findings'), with a Goal, Purpose, Principle and 5 checks, so it can be taken. The old placeholder check is struck.

## Notes

file:screen-2048.ans (22856 bytes, sha256:8a9e2d1c) screen capture from hug tui


check 2 evidence: task/deduce.go closeShipped after the shipped note (doing, active/waiting, every check settled, no open child, done refusal leaves it). History '(done, by release vX: every check holds)', done_release set, fix 'hug task set ID --state todo'. Tests cli/release_test.go TestReleaseClosesShippedTasks, TestReleaseCloseTwoClones (two clones deduce one tag: one done), both shown failing first. branch yf39ya cfaa511d (c2 4fce361f, c3 b476ead8, c4 cfaa511d); CI run 44391027727 all 8 jobs success; local niced go test ./huginn/... ./cmd/... 12 ok 0 FAIL
check 3 evidence: task/afterrelease.go asks the first person check (AddressesPerson, the wait-refusal reading) via public Store.Ask, asker = coordinator; AfterAnswer is the one helper called from CLI check (MCP rides the same Execute) and TUI Space: tick asks the next person check or closes with History '(done after jhf ticked check N (release vX))'; a text answer closes nothing. No ask code edited. Tests cli TestShippedPersonCheckIsAsked, tui TestShippedAskedClosesOnTick + golden testdata/overview/shipped-asked.txt, mcp TestShippedAnswerClosesOverMCP; each fails with its call site removed. branch yf39ya cfaa511d (c2 4fce361f, c3 b476ead8, c4 cfaa511d); CI run 44391027727 all 8 jobs success; local niced go test ./huginn/... ./cmd/... 12 ok 0 FAIL
check 4 evidence: task/handover.go PlanHandover + Store.HandOver; runs only in the background pass (cli/sync.go) and hug task sync (reply field handovers). Gives a doing task whose every assignee is a known local session gone by the stale-owner rule to hug.toml [board] coordinator when that is a person or live session; keeps stage; History 'handed to crab: orca's session ended (closed ...)'. Never hands a person, model name or unknown name. Lint never writes: stale-owner fix says 'hug task sync' only when the plan would hand it over, else keeps --assignee fix. Tests cli/handover_test.go TestGoneWorkerHandsToCoordinator (fails without the code), TestBackgroundPassHandsOver. branch yf39ya cfaa511d (c2 4fce361f, c3 b476ead8, c4 cfaa511d); CI run 44391027727 all 8 jobs success; local niced go test ./huginn/... ./cmd/... 12 ok 0 FAIL
check 5 pre-release count (matcher unchanged per crab): on hug's board now, c3's person reading will NOT ask 3 open checks, which stay shipped-not-done if shipped: 5z4zt4 c5 'Jørgen attaches a screenshot...' (attaches is not a confirm verb), ec6z6j c1 (task is todo, never closed or asked; its 'reviewed by Jorgen' wording is not taken as a person's), zhvkmj (doing/active, no open checks: closes by c2 at its next shipped note, not by c3). Will be asked: qh9ep1 c6 'Jørgen confirms on his board after release' (already asked), 0gztsy c4 'Jørgen opens 6ephqk and says...' if shipped, 5z4zt4 c4. Check 5 itself needs the next release: count lint after it.
landed 2dd99bb on main 2026-11-18: hug task: a release closes the shipped tasks whose checks all hold (yf39ya check…
landed 8fafd8f on main 2026-11-18: hug task: a shipped task's after-release person check is asked, and their tick…
landed 0155445 on main 2026-11-18: hug task: sync and the background pass hand a gone worker's tasks to the…
shipped in v0.7.70 (2dd99bb)
crab 2026-11-20 08:26Z, Jørgen 08:16Z screenshot (v0.7.87): '! 5 broken', shipped-not-done on 0gztsy, p277bp, txjkrm, x8zzzn, ynh84t, doing-idle on w4qmr1, missing-file on p87ngv. Check 5 fails. Causes by task: txjkrm todo with an open question to jhf still flagged (askAfterRelease only looks at doing active/waiting); 0gztsy in depends on kg07bt (shipped, waiting on 'crab review'); x8zzzn and ynh84t had a crab-owned check open after release, which nothing asked (crab released anyway); p277bp hid a pick for jhf in a label 'waiting-on-user'. Fixed by hand 08:22-08:25Z: ynh84t check 4 run and closed, x8zzzn waiting, w4qmr1 waiting until 11-26, p277bp asked. Checks 7-9 added for the flow gaps.
check 7: branch 72n8m4 2a2b383d (tip 1e4a3386): document.go parses ## Questions before noteQuestion, and noteQuestion is quiet while HasOpenQuestion (todo or doing); cache SchemaVersion 29. Test cli/shippedasked_test.go TestShippedWithOpenQuestionIsNotBroken (todo, shipped, asked of jhf: no finding; answered: shipped-not-done is back): failing first on 6fa18387 'shippedasked_test.go:48: m1drs8 has a question open to jhf, but lint says shipped-not-done:'; now ok. Board copy at hug/tasks 1c825160 (Jørgen's 08:16Z board): hug task lint at 6fa18387 shipped-not-done 5 (0gztsy p277bp txjkrm x8zzzn ynh84t), at 1e4a3386 3 (p277bp x8zzzn ynh84t). CI run 44710823480 on 1e4a3386: success, 8/8 jobs.
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
- 2026-11-18T20:56 resolved request 2 by crab
- 2026-11-18T20:56 resolved request 3 by crab
- 2026-11-18T20:56 todo -> doing/active (take)
- 2026-11-18T21:34 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 handed to crab: bonehound's session ended (closed 2026-11-18 22:24)
