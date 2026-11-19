---
id: rk5q7r
title: "When an open reflection is older than 3 days the coordinator's board cue names…"
state: "done"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzu"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
When an open reflection is older than 3 days, the coordinator's board cue (muninn/internal/recall/boardcue.go) names how many there are and the command hug task reflections --by-instruction, at a break in the coordinator's work, so reflections reach a reviewer instead of piling up.

## Purpose
Split from 2k68eg check 4 (crab 2026-11-18 13:35): boardcue.go is being changed by duckling and piglet now. 0 of 56 reflections on three boards were resolved by 2026-11-18; 2k68eg makes them readable (quote parsed, grouped, unanswered counted) but nothing yet tells anyone to read them.

## Principle
Nudges go to the coordinator at a break (x325pp), never to an agent mid-task. Counts from hug task reflections, no model. One cue line, with a runnable command.

## Relevant details
Parser: huginn/internal/task/reflect.go var quoted, the third alternative '([^']{6,})' matches from a possessive apostrophe. Garbled ids: fkpyg3 scrxak n2hq8p x1ra0t cvbpdy t3fqaw ce3dmq wdkqzx ygay9c f5vx4x. Counts and quotes: /Users/jhf/.jcode/scratch/s6ff5k-feedback/REPORT.md clusters B and C2. Feeds tb0dra check 5.

## Depends on

## Acceptance
- [x] When an open reflection is older than 3 days the coordinator's board cue names the count and hug task reflections --by-instruction (test)

## Notes


BAR (duckling 14:27Z, before any run): (1) six sets + fh3 replayed with agh2h7 (c89178f9) as base and rk5q7r as new, same flags (-coordinator crab on hug sets): outputs byte-identical except added lines of the new kind; (2) every new line is in a session named crab (the coordinator) at a break turn; 0 elsewhere; (3) judge calls <=10, spent only on new lines (2 judgings, if any). Count source: main's ParseReflections (open state), dated > 3 days before the turn. Report the count on today's boards under main's parser and under 2k68eg's (origin/2k68eg).
RESULT (duckling 14:41Z). Base board4 = agh2h7 c89178f9; new = rk5q7r. (1) Six sets + fh3, -coordinator crab on hug sets: 217+44 lines, all 8 outputs byte-identical (no reflection in any window was >3 days old: first reflection on the hug board 2026-11-17 12:35). (2) Synthetic: board4 -reflect-age 96h (all reflection dates moved back 4 days), window 11-17 22:00..11-18 12:00: base 35 lines, new 36, the one added line is crab's (coordinator) at 2026-11-17T22:02:07Z: 'Task board (start): 22 open reflections are older than 3 days; read them grouped by the instruction they name: hug task reflections --by-instruction'; 0 lines removed or changed. Full fh3 window aged: 0 new lines, because crab's first break (14:01Z) read 0 reflections and the daily rate held until after the window. (3) Judge calls: 0 (the only new line is synthetic; judging it says nothing about the real rule). Tests: TestBoardOldReflectionsToCoordinator (mutations: no coordinator gate fails 2 cases, no age gate fails 1), TestOpenReflections (cache and files paths). Count today via board.OpenReflections: hug 50 open (26 from 11-17, 24 from 11-18), seid 5 (11-18); old>3d now 0; on 11-21 26; on 11-22 50+. Same under 2k68eg's parser (origin/2k68eg build: hug 50, seid 5, unanswered 0), so the count does not change when 2k68eg lands; its new 'unanswered' state is not open, so not counted.
Commit 346800f5 on rk5q7r + ci/rk5q7r (stacked on agh2h7 c89178f9). Suite: 33 ok + huginn/gitbranch FAIL TestUpgradeFailuresKeepOriginalLayout/lost-ack 'TempDir RemoveAll cleanup: ... remote.git/objects: directory not empty' (untouched package), rerun alone: ok github.com/veridit/hug/huginn/gitbranch 40.921s. gofmt clean.
CI 44327807341 on 346800f5: success 6/6. Waiting for bg1rd8 on main to rebase (crab 14:55Z).
CI 44327807341 on 346800f5 (stacked on agh2h7 c89178f9): 6/6 success. Lands after bg1rd8, rebased onto main.
Rebased onto agh2h7 43e318d1: rk5q7r = 299f1d37, no conflicts. go test -count=1 recall/run/board/task: ok x4. Pushed rk5q7r and ci/rk5q7r.
RESULT (15:53Z): rebased onto agh2h7 7a509517 = 6b7b5aab (conflicts: BoardInput field, LastSeen -> Gone, kept Reflections; CHANGELOG kept both). go build ./... ok; tests recall run board task ok x4. board4 v2: six sets + fh3 byte-identical to rk5q7r/new; synthetic -reflect-age 96h 11-17 22:00..11-18 12:00 base 35 -> probe 36 lines, identical files to the 14:41 run, the one added line crab 22:02:07Z '22 open reflections are older than 3 days'. Judge calls: 0. Pushed rk5q7r + ci/rk5q7r.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed 4d80135 on main 2026-11-18: recall hook: the coordinator hears of open reflections older than 3 days…
shipped in v0.7.66 (4d80135)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T15:25 todo -> doing/active (take)
- 2026-11-18T15:55 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T15:25; waiting_asked_by: duckling; waiting_on: bg1rd8 on main, then rebase onto main; waiting_since: 2026-11-18T15:55
