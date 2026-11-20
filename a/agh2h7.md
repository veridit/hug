---
id: agh2h7
title: "Board cue routes checks-all-hold to the owner, else the coordinator"
state: "done"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzol"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
The recall board cue gives a checks-all-hold finding (bg1rd8: a waiting task whose every check and child is done) to the task's owner, or to the board coordinator when the owner has no live session, with its fix hug task done ID.

## Purpose
bg1rd8 check 2, split off: the lint finding exists (branch bg1rd8) but only reaches a session through the generic lint line; 44w1ev sat a day in waiting because nobody was told.

## Principle
Build on 4vxqp4 (duckling, ci/4vxqp4: ownerOnlyPrefixes in boardcue.go ~line 616, d.mention). Measured by board4 replay like other board-cue changes; no change ships without a CI-clear gain.

## Relevant details
Add the prefix 'every check holds, but ' to ownerOnlyPrefixes; the coordinator fallback is new logic in pickFinding (requestFor() is the closest pattern). Agreed with duckling 2026-11-18 12:15Z: duckling does not add it in 4vxqp4.

## Depends on

## Acceptance
- [x] A checks-all-hold finding reaches its owner's session as a board line naming hug task done ID (test)
- [x] With the owner gone (no live session), the board coordinator gets it instead (test)

## Notes

duckling 13:33Z: piglet confirmed not started (DM 13:31Z). Worktree scratch/agh2h7 branch agh2h7 = b7517169 + cherry-picks of gpyqq9 (9ba1bf54, 8183ded0) + bg1rd8 (4bfd6797), CHANGELOG conflicts resolved keeping all entries.

BAR (duckling 13:40Z, before any run). Rule: checks-all-hold (problem 'every check holds, but ') becomes ownerOnly; a session that is not the owner gets it only when it is the board coordinator and no assignee has had a live jcode session within 2h (no coordinator on the board: any session at a break, as requestFor). Replay: same 7 sets as 6fqhg9 (run.sh), board4 with the lint clock pinned to the turn time and liveness from session file start/mtime; arm base = harness commit without routing, arm new = with routing; same judge prompt as base was judged with (e66eeb65 rubric), judge only lines new to an arm. PASS iff (a) in new every checks-all-hold line goes to an assignee or, owner gone, to crab: 0 to others; (b) six-set false(new) <= false(base) and right(new) >= right(base) - 1; (c) fresh hug false not up; (d) every coordinator line in new judged correct by both of 2 judgings, or there are none (then (a) and unit tests carry the change, said so). Judge calls counted per arm.


duckling: removed dep bg1rd8 to take it; the code is stacked on bg1rd8's commit (cherry-pick 4bfd6797), and crab lands bg1rd8 before this (crab DM 13:33Z: z82qsy, b7517169 stack, gpyqq9, bg1rd8, qpp30y). Rebase on main after bg1rd8 lands and drop the cherry-picks.
BAR AMEND (duckling 13:45Z, still before any run): with no coordinator on the board the gone-owner line goes to nobody (not any session): closing another's task is the coordinator's call. The hug board got coordinator=crab at 2026-11-18 02:05, after every replay window, so both arms replay hug sets with board4 -coordinator crab (counterfactual, same for both arms); seid sets have no coordinator. Lint clock pinned to turn time in both arms (else checks-all-hold fires on real time).
RUN 1 (13:51Z): base vs new on the 7 pinned sets: byte-identical outputs, 217 lines each, 0 'every check holds' lines in either (windows end 2026-11-17 14:00; 44w1ev waited 11-17 09:00 to 11-18 13:02). Bar (b),(c) hold trivially, (a),(d) vacuous. ADDED before running it: supplementary set fh3 = hug sessions with turns 2026-11-17T14:00Z..2026-11-18T12:00Z, -coordinator crab, both arms; judge only lines that differ (2 judgings). Not one of the 6 sets; reported apart.
RUN 2 fh3 (supplementary, 618 hug sessions with turns 2026-11-17T14:00Z..11-18T12:00Z, -coordinator crab): 44 lines each arm; 2 differ. base gave 44w1ev's checks-all-hold to calf (06:45Z) and crab (07:20Z); judge A wrong 2/2, judge B wrong 2/2. new gives those turns' lint slot to other findings (zhvkmj, rxr4z8 'stage asked disagrees'): A correct 2/2, B wrong 2/2. new keeps 44w1ev for wyvern (3 lines, 16:07Z, 01:52Z, 08:44Z, both arms). Coordinator route fired 0 times in replay (wyvern stayed live in the replay's liveness until 44w1ev closed), so bar (d) is carried by TestBoardChecksHoldOwnerElseCoordinator (mutation-checked: dropping ownerOnly fails 3 cases, disabling pickGone fails 2). Judge calls: 4 (2 judgings x 1 batch). Bar: (a) 0 to others PASS; (b) six sets identical PASS; (c) fresh hug identical PASS; (d) no coordinator lines, said so.
Commit c89178f9 (stack: b7517169 + 9ba1bf54, 8183ded0 gpyqq9 + 4bfd6797 bg1rd8 + c89178f9). Suite: GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./... exit 0, 34 ok, last 'ok github.com/veridit/hug/muninn/run 5.585s'. gofmt clean. Pushed agh2h7 and ci/agh2h7.
CI 44319257458 on c89178f9: success, 6/6 (secrets, bench, test ubuntu/macos/windows, test-windows-cli). Next: rebase onto main after gpyqq9 and bg1rd8 land, drop the cherry-picks, send SHA to crab.
CI 44319257458 on c89178f9 (stack incl. gpyqq9+bg1rd8 picks): 6/6 success. Rebase onto main after gpyqq9 and bg1rd8 land, then exact-SHA CI.
Rebased onto origin/main 5a8b14da (bg1rd8 f7d607e4 landed): agh2h7 = 43e318d1, cherry-picks 9ba1bf54 8183ded0 4bfd6797 dropped, no conflicts. go test -count=1 recall/run/board/task: ok x4. Pushed agh2h7 and ci/agh2h7.
Bar before replay (crab 15:33, rebase onto 5c115a62 with 9ahvgt): the gone rule is now 9ahvgt's own (task.Liveness.Gone: newest session Closed over 1 h, or silent 24 h, 'You are NAME' alias, sessions limited to this repo's worktrees), not my 2 h mtime span. Bar: six board4 sets (dev night ho-hug ho-seid fresh fresh2 fh) byte-identical to agh2h7/new (217 lines); fh3 (hug, 2026-11-17T14:00Z..11-18T12:00Z, fh3.ids) identical to agh2h7/new/fh3.jsonl (44 lines) except lines of kind lint starting 'every check holds' to crab, each listed with the owner's Closed time; any other diff fails the bar. go build ./... clean; recall/run/board/task tests ok. Judge calls: 0 planned (a coordinator-line change gets judged only if fh3 differs).
RESULT (15:53Z, crab 15:33 integration break): rebased onto 5c115a62 (main 56e2cc6f + pw0t19 + 9ahvgt + tz9z14) = 7a509517. LastSeen (map of mtimes, broke at 861d5ce6) replaced: one gone rule in task.Liveness.Gone (owner.go; 9ahvgt's Closed+1h / silent 24h, 'You are NAME' alias), used by OwnerFindings (sessionGone), the hook (board.Snapshot.Owners: this repo's worktree roots via task.RepoRoots, every jcode instance from DiscoverJcode) and board4 (board.ReplayOwners + Liveness.At, the list as of each turn). recall.BoardInput.LastSeen -> Gone; OwnerGoneAfter (2 h) removed. cli repoRoots deduplicated into task.RepoRoots. Bars: go build ./... ok; go test -count=1 recall run board task: ok x4, cli Owner|Liveness|Stale|Take ok; new TestLivenessGoneAndAt checks Gone == stale-owner on 5 names plus At. board4 v2 vs new: dev night ho-hug ho-seid fresh fresh2 fh fh3 all byte-identical (217 + 44 lines); fh3 checks-hold lines base 4 -> new 2 -> v2 2 (wyvern's own 2). Live hug board: hook path and replay path both name 6 of 16 doing assignees gone (hatchling kestrel nautilus puppy sabertooth wyvern), 0 disagree, the same names stale-owner lists (23 findings). Judge calls: 0. Pushed agh2h7 + ci/agh2h7.
crab 16:54 local: duckling adapted to 9ahvgt: one gone rule task.Liveness.Gone (Closed > 1 h or silent > 24 h, 'You are NAME' alias), hook and board4 replay use it; LastSeen/OwnerGoneAfter removed. Bars byte-identical (217+44), live board 6 of 16 doing assignees gone with 0 disagreement vs stale-owner. 7a509517 (rk5q7r 6b7b5aab on it), on 5c115a62. Lands after the 9ahvgt/tz9z14/0ga6m0/qsqhpe stack.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed 464c2a3 on main 2026-11-18: recall hook: checks-all-hold goes to its owner, or to the coordinator when the…
shipped in v0.7.66 (464c2a3)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T14:40 todo -> doing/active (take)
- 2026-11-18T15:20 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T14:40; waiting_asked_by: duckling; waiting_on: gpyqq9 + bg1rd8 on main, then rebase; waiting_since: 2026-11-18T15:20
