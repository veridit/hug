---
id: at3qep
title: "Observations hug can't settle become questions to the owner"
status: "doing"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
parent: "47sxn2"
doing_since: "2026-11-17T17:39"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Lint observes states that need a person or agent to settle them and turns each into a question on the ticket, addressed to its owner, with the observation and a fix command: (a) landed on main but checks unticked; (b) doing with no note, check or commit from its taker for 2 hours; (c) shipped in a release but not done; (d) a commit on main naming a ticket that is todo; (e) a doing ticket whose recorded branch is gone. Questions reach the agent at its next break (x325pp) and in show/next, and are answered in place (3jtfw9). Answering or fixing clears them.

## Purpose
Epic 47sxn2, level 2. Today Jørgen noticed these by hand: tickets shipped with 0 checks ticked, 5 tickets in doing for one agent, tickets nobody was on.

## Principle
Ask, never guess. Each question names what was observed, from what, and one fix command. One question per ticket at a time, the most important first, so nobody is flooded. Never in the listing beyond the ! marker (11:01 ruling).

## Relevant details
Uses level 1 facts (landed, shipped lines) from the git deduction ticket. Activity = notes, checks, commits naming the ticket, by its taker. Delivery: x325pp nudges, board cue. Answer: 3jtfw9.

## Depends on

- g9dg32
## Acceptance
- [ ] each of (a)-(e) on a fixture board produces one question with observation and fix line; fixing it clears it (test per case)
- [ ] the owner's agent gets the question at its next break and in hug task show (test)
- [ ] on hug's own board today, the questions raised match the drift Jørgen saw (0hkaa5, m1c28r, t709he would each have been asked), shown before and after

## Notes

hatchling 12:41, design before code (not taken: depends on g9dg32, wyvern doing). Existing pieces: (e) is today's branch-gone finding (task/branch.go BranchFindings); (b) sits next to stale-owner (task/owner.go, v3k6cp: no live session 24 h), but is per ticket and from the taker's own activity: last note/check by assignee (ticket history) or commit naming the id, older than 2 h. (a) and (c) need g9dg32's 'landed <sha>' / done_release lines; (d) needs the same commit scan on a todo ticket. Plan: task/questions.go Questions(ts, facts, now) []Question{ID, Owner, Code, Observed, From, Fix, Rank}; rank a>c>e>d>b, keep only the top one per ticket (principle: one at a time). Each is also a Finding (codes landed-unticked, shipped-not-done, branch-gone, todo-has-commit, doing-idle) so lint and show carry it via fkpyg3's per-task block, rows keep only the ! marker. Delivery through the recall hook's board line at a break (x325pp's stop/between rules) and the 3jtfw9 board cue for answers. Fix commands: (a) hug task check ID N / hug task done ID; (b) hug task note ID "..." or hug task set ID --status todo; (c) hug task done ID; (d) hug task take ID or hug task note ID; (e) hug task done ID or hug task set ID --status todo. Clears on fix because each is recomputed from state, nothing stored.


hatchling 16:47, design (taken; s6ff5k waits on its H3 replay). (a) landed-unchecked, (c) shipped-not-done and (d) todo-has-commit are facts in the ticket's own file (g9dg32's 'landed SHA on main' and 'shipped in TAG' notes, status, checks), so Parse emits them like done-unchecked: cached, and in lint, show, take and the board cue with no extra git call. Only the top of a/c/d per ticket (a>c>d, principle: one question at a time). (a) = doing, a landed note, and no check settled (0 of N ticked or struck): 'shipped with 0 checks' is what Jørgen saw; a ticket with some ticks is in progress, not drift. (c) = a shipped note and status todo/doing. (d) = todo with a landed note. (e) is today's branch-gone (git), (b) doing-idle (git): stage active, doing_since older than 2 h, and no commit touching its file on hug/tasks and none on main naming it in 2 h. Two git calls, lint only (the recall hook gets a/c/d; e and b stay out of the per-prompt path). A git-derived question is dropped when the ticket already has a file question (rank a>c>e>d>b). Author cannot tell the taker (every agent commits as Jørgen), so (b) counts anyone's commit on the ticket as activity; said in the finding. Delivery: question codes join ownerOnly in the board cue (missing-section's rule, tpnd3n): only a session that is the ticket's assignee or touched it is told. Clears on fix: recomputed from state, nothing stored. Cache schema 17 (parse findings change).
file:questions-1234.txt (895 bytes, sha256:ab1dbbf3) check 3 replay: questions-1234.txt
file:questions-now.txt (208 bytes, sha256:0837dd44) check 3 replay: questions-now.txt
## History
- 2026-11-17T17:39 todo -> doing/active (take)
