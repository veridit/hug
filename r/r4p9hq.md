---
id: r4p9hq
title: "Release v0.7.95: the focus \u003e pulses, text cursors blink, one thing moves"
state: "doing"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxq"
created: 2026-11-21
updated: 2026-11-21
stage: "active"
doing_since: "2026-11-25T11:22"
doing_with: "hktdgs"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Jørgen confirms v0.7.95: the > where you are pulses like a cursor, and wherever you type an underscore blinks instead.

## Purpose
An underlined, still > read as a key to press; he wants one moving mark that says where he is.

## Principle
One question per shipped task; on/off frames attached so it can be judged without installing.

## Relevant details
Shipped: q3sh69 checks 1, 2, 4 (88724001..dc737984, CI 44906215953 8/8). The > (and ⋗ on a marked row) turns from bold accent to dim and back every 530 ms; no underline. While typing (answer box, /, :, #, filter field) the > is still and the text cursor blinks as an underline. Idle CPU of hug task tui: about 0.2% before, 0.6-2.4% after (one tick redraws one line). Try it: brew upgrade veridit/tap/hug, hug task tui, open a task asked of you.

## Depends on

## Acceptance
- [x] Jørgen confirms the pulse draws his eye and only one thing moves

## Questions
- 2026-11-21T14:05 question 1 by crab to jhf (check 1):
  v0.7.95: the > where you are now pulses (bold accent, then dim, every 530 ms, no underline), and wherever you type an underscore blinks while the > holds still. Frames attached (on/off for a list, a task page, the answer box). Does the pulse draw your eye without being restless?
  answered 2026-11-25T09:52 by jhf:
    Finally, it's so much better. Subtle one thing brings your attention, the one thing moving where you can enter something

## Notes

file:list-on.ansi (2841 bytes, sha256:fd29f725) list-on

file:list-off.ansi (2840 bytes, sha256:f9ba6605) list-off

file:page-on.ansi (3481 bytes, sha256:198832ad) page-on

file:page-off.ansi (3480 bytes, sha256:f5354fd1) page-off

file:answer-box-on.ansi (2982 bytes, sha256:00aa32c7) answer-box-on

file:answer-box-off.ansi (2955 bytes, sha256:1e53ab94) answer-box-off


shipped: tag v0.7.95 (c4389611); release.yml 44907060648 attempt 1 failed on TestSettleWaitsOnProgressNotAClock/progressing ('settleWait returned "" after 1m32.6s, want about 1m30s', unrelated to q3sh69, same failure on ci/qcq3tm 44720742380; filed 3qa9bt), attempt 2 success; brew -> ../Cellar/hug/0.7.95/bin/hug, hug v0.7.95
check 1: Jørgen 2026-11-25: 'Finally, it's so much better. Subtle one thing brings your attention, the one thing moving where you can enter something'
## History
- 2026-11-21T14:05 todo -> todo/design (ask)
- 2026-11-25T09:52 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.95: the > where you are now pulses (bold accent, then dim, every 530 ms, no underline), and wherever you type an underscore blinks while the > holds still. Frames attached (on/off for a list, a task page, the answer box). Does the pulse draw your eye without being restless?
  Finally, it's so much better. Subtle one thing brings your attention, the one thing moving where you can enter something
- 2026-11-25T09:52 answer 1 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
- 2026-11-25T11:21 crab read the answer: hug task check r4p9hq 1 "Jørgen 2026-11-25: 'Finally, it's so much better. Subtle one thing brings your attention, the one thing moving where yo…"
- 2026-11-25T11:22 todo/ready -> doing/active (take with hktdgs)
