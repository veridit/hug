---
id: rzfrgp
title: "Board take line: never name a task the session only read"
state: "doing"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:50"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
The board take line (changing code, no task doing: 'this looks like ID') names a task the session's message, commit or check/note points at, never one it only opened with hug task show

## Purpose
6fqhg9 replay: with quoted hug commands no longer counted as takes, guppy 2026-11-16 20:16 got 'this looks like jrj1vq ... hug task take jrj1vq' (judged wrong by both judgings) because guppy ran 'hug task show jrj1vq' at 18:38 to read a dependency; show feeds addNamed in muninn/internal/recall/boardcue.go. This blocks 6fqhg9 (quote-aware HugCalls).

## Principle
Measure on board4 replays with the fixed blind judge (e66eeb65, two judgings) before shipping; write the bar before any run; no paid calls in the hook

## Relevant details
boardcue.go detect: 'if h.Verb == "show" || h.Verb == "check" || h.Verb == "note" { addNamed(h.ID) }'. Arms and judgings: scratch/board4/6fqhg9 (judgeA.json, judgeB.json, score.py, run.sh). Parser patch for 6fqhg9: scratch/board4/6fqhg9/quote-parked.patch.

## Depends on

## Acceptance
- [ ] a test: a session that only ran hug task show X and edits code gets no take line naming X
- [ ] replay (6 sets + fresh hug, two judgings of the new prompt): take/work/done/owner false lines do not rise on any set and right lines do not fall

## Notes


## History
- 2026-11-18T13:50 todo -> doing/active (take)
