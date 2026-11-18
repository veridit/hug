---
id: 6fqhg9
title: "Hook: a hug command quoted inside another's argument is not a call"
state: "doing"
type: feature
priority: ""
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzu"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T13:32"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
harness.HugCalls skips a hug task command that sits inside a quoted argument (a --goal, a note, a quoted lint line), so the board nudge does not count a task as taken, checked or done because an agent quoted the command

## Purpose
4vxqp4 replay: crab's hug task new --goal quoted 'next: hug task take vk9td0', so the nudge counted vk9td0 as crab's and handed crab and poodle its missing Principle (judged false). On 16 sessions the quote-aware parser drops 20+ quoted take/done strings, which also moves take/work/done lines

## Principle
Measure on board4 replays with the blind judge before shipping; a command a shell runs (sh -c, $(...)) still counts; unbalanced quotes count nothing as quoted; no paid calls in the hook

## Relevant details
Patch: /Users/jhf/.jcode/scratch/board4/4vxqp4/quote.patch (muninn/internal/harness/toolcalls.go HugCalls + shellQuoted, 7 test cases in toolcalls_test.go). Arms on the 6 board4 sets, right/false: base 101/12, read2 (4vxqp4 as shipped) 104/8, quote (read2 + this patch) 105/7. Fresh hug (367 sessions, 2026-11-16 20:00 to 11-17 14:00): read2 lint 3 (3 right), false 3; quote lint 2, false 4 (guppy loses the s6ff5k start line, judged right, and gets a take line on jrj1vq, judged wrong). Arms and scripts: scratch/board4/4vxqp4 (run.sh, score.py, diff.py, judgments4.json).

## Depends on

## Acceptance
- [ ] replay on the 6 board4 sets plus fresh hug: every take/work/done line the patch adds or removes is blind-judged, and false lines do not rise on any set
- [ ] fresh hug keeps its right lint lines (3 under 4vxqp4)

## Notes

duckling (4vxqp4) 13:10: the blind judge is unstable on start-lint lines. The 15 lines that differ between base and quote were rejudged twice from empty caches (judge_board.py, DeepSeek Flash): base-only lines 2 correct / 5 wrong / 1 unclear, then 8 / 0 / 0; rule-only lines 3 / 3 / 1, then 5 / 1 / 1. hog s6ff5k (named only in a quoted prompt in a brief that forbids other tasks) was judged wrong in rejudge 1 ('not this worker's task') and correct in rejudge 2 ('board lint is independent'). The judge prompt says a lint reminder 'does not need to relate to the current work', so verdicts on missing-section lines flip on that reading. Files: scratch/board4/4vxqp4/rejudge1.json, rejudge2.json, rejudge-summary.txt.


## History
- 2026-11-18T13:32 todo -> doing/active (take)
