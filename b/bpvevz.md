---
id: bpvevz
title: "Mid-turn recall: after a day on, decide whether it is the default"
state: "doing"
type: chore
priority: "medium"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxs"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-20T11:23"
doing_with: "p87ngv"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "One day with mid-turn recall on (counted on check 1): 13 messages you typed into running turns were recalled, 10 got memories in the tool result (half a second each, 1 missed), and stale turn blocks halved (6 to 3). Before, those 13 got nothing. Make it the default? A: default on, remove the old off path. B: default on, keep the off switch. C: keep it opt-in. My pick: B."
asked_for: "jhf"
asked_since: "2026-11-21T11:29"
asked_by: "crab"
asked_check: "2"
---

## Goal
After a day with mid-turn recall on, Jørgen decides from counted evidence whether it becomes the default and replaces the old path.

## Purpose
Jørgen switched it on at once; the counts tell whether it should stay on for everyone.

## Principle
Read the live store and logs read-only; change the default only on Jørgen's word.

## Relevant details
Switched on 2026-11-20 10:22Z: recall.jcode_midturn true in /Users/jhf/Library/Application Support/muninn/config.json (backup config.json.bak-20261120-ypfrp3 beside it; hug mem config shows it). Built in x8zzzn (shipped v0.7.85). Count from 11-20 10:22Z to 11-21 10:22Z: messages typed into a running turn, how many got a recall block in the tool result, how many parked blocks went stale, compared with the same span before the switch. Jørgen 2026-11-20 10:22Z on ypfrp3: 'Yes, clearly. I wanted to activate it right now. You want to make it staggered, but fine, make a follow-up ticket then.'

## Depends on

## Acceptance
- [x] A note counts, for the day after 11-20 10:22Z and the day before: mid-turn messages, those given a recall block, parked blocks gone stale, with the commands used
- [ ] Jørgen decides: default on (old path removed), default on (old path kept), or stays opt-in

## Questions
- 2026-11-21T11:29 question 1 by crab to jhf (check 2):
  One day with mid-turn recall on (counted on check 1): 13 messages you typed into running turns were recalled, 10 got memories in the tool result (half a second each, 1 missed), and stale turn blocks halved (6 to 3). Before, those 13 got nothing. Make it the default? A: default on, remove the old off path. B: default on, keep the off switch. C: keep it opt-in. My pick: B.

## Notes


check 1: Counted 2026-11-21 10:28Z from ~/Library/Application Support/muninn/hook.log (python over lines in each UTC window; recall.jcode_midturn true since 11-20 10:22Z, config.json mtime 11:22:36 local). AFTER (11-20 10:22Z..11-21 10:22Z): 13 messages typed into a running turn were recalled mid-turn ('midturn ... msg=' lines), 10 of them got a recall block (hits>0, 32 memories in all), median 0.5 s, max 1.1 s; 1 mid-turn message missed ('midturn ... missed=1', orangutan 13:16 local); parked turn-start blocks 185 (163 with hits), delivered 176, gone stale 3 ('dropped=stale'). BEFORE (11-19 10:22Z..11-20 10:22Z): 0 mid-turn recalls (switch off), parked 253 (229 with hits), delivered 246, gone stale 6. Spread: seid 4, brovs 4, statbus 4, global 1, plus the 1 miss; sessions orangutan, sheep, hibiscus, rabbit, sabertooth, crab. Without the switch those 13 typed messages would have had no recall at all, and each would have made the turn's parked block stale; stale halved (6 -> 3) on fewer turns (253 -> 185).
## History
- 2026-11-20T11:23 todo -> doing/active (take with p87ngv)
- 2026-11-20T11:23 doing/active -> doing/waiting (wait until 2026-11-21)
- 2026-11-21T00:54 doing/waiting -> doing/active (woke: the date 2026-11-21 has come); was waiting_asked_by: crab; waiting_on: a day of mid-turn recall on (switched 2026-11-20 10:22Z); waiting_since: 2026-11-20T11:23; waiting_until: 2026-11-21
- 2026-11-21T02:55 doing/active -> doing/waiting (wait)
- 2026-11-21T11:28 doing/waiting -> doing/active (resume); was waiting_asked_by: crab; waiting_on: until 2026-11-21 10:22Z: a full day of mid-turn recall, then the count (check 1); waiting_since: 2026-11-21T02:55
- 2026-11-21T11:29 doing/active -> doing/asked (ask)
