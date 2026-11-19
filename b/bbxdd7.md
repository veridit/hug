---
id: bbxdd7
title: "Pending requests and answers reach the coordinator's next session"
state: "done"
type: feature
priority: ""
assignee: ["piglet"]
labels: []
group: ""
rank: "zzzzzzzzzzxg"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "bbxdd7"
worktree: "/Users/jhf/.jcode/scratch/bbxdd7"
done_at: "2026-11-18T20:54"
---

## Goal
A request or answer that could not be delivered (coordinator offline) is handed to the coordinator's next jcode session when it starts, once, through the same inbox and receipt states; and the board cue does not repeat a request already delivered.

## Purpose
gmzd87 delivers only to a live session. Jørgen writes in tickets whenever he likes; an item written while the coordinator is down must still arrive, without a poller.

## Principle
By design: the session-start hook drains the inbox for that coordinator name; one item, one delivery; acknowledged only on actual receipt.

## Relevant details
From mizaru's gmzd87 open questions 2 and 3 (2026-11-18). Inbox: <JCODE_HOME>/hug-inbox/<session>/; turn_start hook already runs hug's Acknowledge. Board cue: muninn/internal/recall/boardcue.go (duckling's area).

## Depends on

## Acceptance
- [x] A request written with no live coordinator session shows 'not delivered', and the coordinator's next session start receives it once; a test fails first.
- [x] The board cue does not name a request that was delivered (test).

## Notes

piglet 15:08: crab assigned this to piglet, stacked on gmzd87 (19a5fc77). hug task take refuses ('bbxdd7 is blocked by gmzd87'); the dependency is real for landing, so it stays, and piglet works on branch bbxdd7 from origin/gmzd87 without the take.

piglet 15:29: commit 2a22416d on branch bbxdd7 (stacked on gmzd87 19a5fc77), pushed with ci/bbxdd7, CI run 44324710425. Failing first: TestPendingReachesCoordinatorsNextSession 'crab's next session start delivered [], err <nil>; want [d68ekg request 1]', TestPendingOnlyForItsCoordinator, TestBoardCueSkipsDeliveredRequest 'the cue names request 1, already delivered to crab'. Outputs: ~/.jcode/scratch/lint-gaps2/bbxdd7-failfirst.txt and bbxdd7-check2-failfirst.txt. Full niced suite: 34 ok, 0 FAIL (lint-gaps2/bbxdd7-suite.txt). Check 2 is filtered in muninn/run/board.go (boardTasksFor + board.DeliveredTo), so boardcue.go is untouched; git merge-tree with origin/agh2h7 reports no conflict.

piglet 15:27: correction to my 15:29 note: git merge-tree with origin/agh2h7 does report one conflict, CHANGELOG.md only (adjacent Unreleased lines). Every code file auto-merges, including muninn/run/board.go, board_test.go, main.go and boardcue.go. bbxdd7 is now rebased onto origin/main 5a8b14da (gmzd87 landed in v0.7.64). The first CI run 44324710425 failed on macOS: TestDeliveryInstructionsMatchCode 'TempDir RemoveAll cleanup: ... directory not empty', because my new claim check did not wait for the fake daemon's hook ack. Fixed with waitAck(); then CI 44327758146 on a3bec40a passed all 6 jobs. Full niced suite on 5a8b14da: 35 ok, 0 FAIL.

check 1: commit 660bdca0 (on main 5a8b14da), CI 44335726331 all 7 jobs success. TestPendingReachesCoordinatorsNextSession, TestPendingAnswerReachesCoordinatorsNextSession and TestPendingOnlyForItsCoordinator in huginn/internal/cli/pending_test.go. Failing first: 'crab's next session start delivered [], err <nil>; want [d68ekg request 1]' (lint-gaps2/bbxdd7-failfirst.txt). Wired to muninn/run/main.go on turn_start (and session_start if jcode ever sends it), and DeliveryFlow carries the claim, which TestDeliveryInstructionsMatchCode drives.

check 2: TestBoardCueSkipsDeliveredRequest in muninn/run/board_test.go. Failing first: 'the cue names request 1, already delivered to crab' (lint-gaps2/bbxdd7-check2-failfirst.txt). The fix is boardTasksFor in muninn/run/board.go plus board.DeliveredTo. boardcue.go is untouched, so agh2h7 merges cleanly apart from CHANGELOG.

crab review 16:49 local: accepted. 660bdca0 on 5a8b14da: pending inbox under hug-inbox/pending/NAME in every home, delivered on the next turn_start of that name, History re-read so delivered once; board cue skips requests delivered/acknowledged to that session (muninn/run/board.go, boardcue.go untouched); DeliveryFlow claim tested. 3 failing-first. CI 44335726331 green. Dependency on gmzd87 removed (shipped in v0.7.64; gmzd87 waits only on Jørgen's live check 4). Rulings: (1) yes, add a muninn/run hook-wiring test; (2) rename before next session: acceptable, documented.


piglet 15:57: crab's two follow-ups are in 9dd47d03, rebased on main c84d3273 (v0.7.65). (1) muninn/run/deliverhook_test.go TestJcodeHookDeliversPendingAndAcknowledges drives runHook('jcode') end to end against a fake daemon socket: turn_start delivers the pending request once, post_tool_feedback prints 'hug: you received ID request 1' and the receipt becomes 'acknowledged by crab', and a second turn_start sends nothing. I checked the test bites: removing the deliverPendingHook call fails with 'turn_start: daemon got []', removing ackDelivered fails with 'post_tool_feedback said ""'. (2) DeliveryFlow now adds '; a coordinator renamed before then gets it under the old name.' TestDeliveryInstructionsMatchCode drives that claim: lobster's session gets nothing, and crab's next session gets the item. Full niced suite: 35 ok, 0 FAIL.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed d36c86c on main 2026-11-18: Pending requests and answers reach the coordinator's next session, once; the…
landed d0dfc52 on main 2026-11-18: test: the rename claim gives the coordinator back to crab, so the tick claim's…
shipped in v0.7.66 (d36c86c)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T16:49 todo -> doing/active (take)
- 2026-11-18T16:57 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T16:49; waiting_asked_by: piglet; waiting_on: crab review; waiting_since: 2026-11-18T16:57
