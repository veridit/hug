---
id: 4vxqp4
title: "Board lint: a task the message only mentions is not the session's task"
state: "todo"
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: ""
stage: "ready"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A missing-section finding goes to a session only for a task it owns or acted on, not one its message merely mentions (an example, a dependency, a task it forbids)

## Purpose
After tpnd3n, 5 of the 12 false board lines on the 6 replay sets are such findings at session start: s6ff5k to hog and dromedary told not to start it, vk9td0 to kitten and poodle, n8zmjw to evergreen told not to take it (muninn/docs/BOARD4.md, tpnd3n)

## Principle
Measured by replay with the blind judge on the same sets and fresh data; a message mention may still count for other findings; no change ships without a CI-clear gain

## Relevant details
In tpnd3n's arm 6 lines on message-only tasks were judged right (2 return, 4 start): a rule that drops all message mentions loses them too. Data: scratch/board4/x325pp (tpnd3 arm, judgments3.json).

## Depends on

## Acceptance
- [ ] Replay (board4, blind judge): false lines on the 6 sets fall below tpnd3n's 12 and right lines stay at or above 101
- [ ] Fresh sessions after the change keep their right lint lines

## Notes

crab 14:53: test the start-of-session rule (a task only mentioned in the brief gets no lint line) on its own first: it is 5 of the 9; the other 4 (own/noted task at a wrong moment) after.


hatchling 15:00, start rule alone (crab): at session start only the message's first board task (lead) counts among tasks the message only mentions. Test TestBoardStartLintOnlyOnLeadTask fails without it ('start line on a task the brief only mentions: s6ff5k...'). Replay vs tpnd3n on the same live boards (fresh2 pinned to 14:00Z): six sets false 12 -> 10, right 101 -> 100, lint 55 -> 52. All 5 targeted start lines are gone (poodle, dromedary, hog, kitten vk9td0/s6ff5k, evergreen n8zmjw; evergreen now gets p39w4a, judged right). But the rotation hands the freed findings to other sessions: 3 new false return lines (crab vk9td0, poodle vk9td0, palmtree n8zmjw) and palmtree loses a right zjedy0 line. Fresh hug (38.8 h): lint 3 right -> 2 right + 1 false. guppy's start brief listed a queue ('ez50rb, then s6ff5k'): lead-only dropped s6ff5k (judged right) and gave guppy jrj1vq (judged wrong). Fresh sets together: no gain (fresh2 +1/-1, fresh hug -1 right +1 false). Not shipped; kept as wip commit on ci/4vxqp4 (worktree guppy-lint).
hatchling 15:01, pass bar for owner-only (crab ruling 15:00), written before any output is read. Disclosure: the replay run was launched at 15:00:50, seconds before the ruling arrived; none of its output has been looked at. Arm 'mine': a missing section is handed only to the task's own session (taken, assignee, or worktree/branch = mine()), never by mention or action. Baseline tpnd3b (tpnd3n, same live boards, fresh2 to 14:00Z). PASS needs all of: (1) six sets: false lines < 12 and right lines >= 101; (2) fresh seid + fresh2 + fresh hug together: right lint lines >= tpnd3n's (5+7+3 = 15) and false lint lines <= tpnd3n's (0+1+0 = 1). Otherwise park: 4vxqp4 back to todo with the evidence, then s6ff5k replay. No third variant.
hatchling 15:03, owner-only measured once (crab ruling): FAILS the pre-stated bar, parked. Six sets: false 12 -> 9, right 101 -> 95, lint 55 -> 47 (right lint 45 -> 39). Fresh seid+fresh2+fresh hug lint: tpnd3n 15 right / 1 false / 1 unclear; owner-only 10 right / 2 false / 1 unclear (fresh2 lint 9 -> 7, right 15 -> 12; fresh hug lint 3 right -> 1 right). Bar (1) right >= 101: fail; bar (2) right lint >= 15 and false <= 1: fail. Both variants tried (lead-only 3f7a071, owner-only) remove the 5 start lines but cost right lines: the judge counts many section lines on a task the session named (seedling n8zmjw, maple p39w4a, guppy s6ff5k) as right. Idea for later, untested: decide by what the brief says about the task (assigned vs forbidden/dependency), not by position or ownership. Arms in scratch/board4/x325pp (lead, mine, tpnd3b), judgments3.json.
hatchling 15:08: detached from tpnd3n so tpnd3n can close on landing (b8f9000); this is follow-up work from tpnd3n's replay, parked as todo.
## History
- 2026-11-17T15:53 todo -> doing/active (take)
- 2026-11-17T16:03 doing/active -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-17T15:53
