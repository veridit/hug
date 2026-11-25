---
id: psd8gz
title: "Release v0.7.109: a big task's page answers keys at once"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzc"
created: 2026-11-23
updated: 2026-11-23
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-25T13:01"
---

## Goal
v0.7.109 ships 1tjq11: scrolling and typing in a big task's page answer at once; Jørgen tries it on p39w4a in Seid.

## Purpose
One ticket per release so Jørgen reviews what shipped in one place.

## Principle
Shipped tasks are children; one question at a time.

## Details
Main 3125fd29 (Release v0.7.109, merge of 8c16709c, CI 8/8 on that exact SHA). release.yml run 45098166000 success; brew gives ../Cellar/hug/0.7.109/bin/hug, hug version v0.7.109. crab's own check in a real terminal (tmux 180x50, copy of seid-lang's board, p39w4a, load ~10): one j took 210-472 ms on v0.7.108 and 14-39 ms on v0.7.109 (time until the screen changed, tmux polling included). monkey's numbers: generated 700 KB task, j 0.36 s -> ~2 ms, a typed character 0.29 s -> ~2 ms; 16 invalidation tests, 16 of 16 mutants caught; goldens unchanged. Not changed: opening a big board still takes seconds (the board load), outside this task.

## Depends on

## Acceptance
- [x] Jørgen scrolls p39w4a in Seid and types an answer there, and finds it quick

## Questions
- 2026-11-23T03:15 question 1 by crab to jhf (check 1):
  v0.7.109 is out (brew upgrade veridit/tap/hug; restart your hug tui windows to get it). In Seid, open p39w4a, scroll with j and k, and type an answer (Esc to leave it unsent if you like). In my test one j went from about 0.3 s to under 40 ms. Is it quick now for you?
  answered 2026-11-25T11:17 by jhf:
    Again, I've just answered this. It is quick now. Why the second question? I don't know what to do now

## Notes


check 1: Jørgen 2026-11-25 on 1tjq11: 'Yes, quite nice and quick. Thank you' (and here: 'It is quick now')
Reflection (open, 2026-11-25, crab): nothing; confirmed by Jørgen 2026-11-25
## History
- 2026-11-23T03:15 todo -> todo/design (ask)
- 2026-11-25T11:17 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.109 is out (brew upgrade veridit/tap/hug; restart your hug tui windows to get it). In Seid, open p39w4a, scroll with j and k, and type an answer (Esc to leave it unsent if you like). In my test one j went from about 0.3 s to under 40 ms. Is it quick now for you?
  Again, I've just answered this. It is quick now. Why the second question? I don't know what to do now
- 2026-11-25T11:17 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-25T13:00 todo/ready -> doing/active (take with hktdgs)
- 2026-11-25T13:00 crab read the answer: hug task take psd8gz
- 2026-11-25T13:01 doing/active -> done (done); was doing_since: 2026-11-25T13:00; doing_with: hktdgs
