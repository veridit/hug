---
id: bpvevz
title: "Mid-turn recall: after a day on, decide whether it is the default"
state: "doing"
type: chore
priority: "medium"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzw"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-20T11:23"
doing_with: "p87ngv"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_asked_by: "crab"
waiting_until: "2026-11-21"
waiting_on: "a day of mid-turn recall on (switched 2026-11-20 10:22Z)"
waiting_since: "2026-11-20T11:23"
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
- [ ] A note counts, for the day after 11-20 10:22Z and the day before: mid-turn messages, those given a recall block, parked blocks gone stale, with the commands used
- [ ] Jørgen decides: default on (old path removed), default on (old path kept), or stays opt-in

## Notes


## History
- 2026-11-20T11:23 todo -> doing/active (take with p87ngv)
- 2026-11-20T11:23 doing/active -> doing/waiting (wait until 2026-11-21)
