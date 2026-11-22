---
id: ypfrp3
title: "Release v0.7.85"
state: "doing"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzx7"
created: 2026-11-19
updated: 2026-11-19
stage: "waiting"
doing_since: "2026-11-20T11:23"
doing_with: "p87ngv"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "x8zzzn check 3 (per-harness push table), its last open child"
waiting_since: "2026-11-20T11:23"
waiting_asked_by: "crab"
---

## Goal
v0.7.85 (x8zzzn mid-turn recall, off by default) is published, installs via brew, and Jørgen decides on the switch.

## Purpose
Hourly release; trial of release-as-group (75mxf3).

## Principle
Release procedure; shipped tickets are children; one ask per shipped change.

## Relevant details
main 9073b532 (CI 44610132775 attempt 2, 8/8), CHANGELOG ccf8641f, merge 97d2de73, release.yml 44613144851 success.

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.85 with links into the new Cellar

## Questions
- 2026-11-19T22:00 question 1 by crab to jhf:
  x8zzzn: shall I switch mid-turn recall on in your config now ('recall': {'jcode_midturn': true}), so messages you type while I'm working get their memories, and you decide after a day whether it replaces the old path?
  answered 2026-11-20T11:21 by jhf:
    Yes, clearly. I wanted to activate it right now. You want to make it staggered, but fine, make a follow-up ticket then.

## Notes

check 1: release.yml 44613144851 success; brew upgrade: hug v0.7.85, hug/muninn/huginn -> Cellar/hug/0.7.85


Jørgen 2026-11-20 10:22Z (answer 1): 'Yes, clearly. I wanted to activate it right now...make a follow-up ticket then.' Done at 10:22Z: recall.jcode_midturn true in his muninn config.json (backup config.json.bak-20261120-ypfrp3; hug mem config shows it). Follow-up bpvevz counts a day and asks for the default.
## History
- 2026-11-19T22:00 todo -> todo/design (ask)
- 2026-11-20T11:21 answered design, by jhf, for crab (todo/design -> todo/ready):
  > x8zzzn: shall I switch mid-turn recall on in your config now ('recall': {'jcode_midturn': true}), so messages you type while I'm working get their memories, and you decide after a day whether it replaces the old path?
  Yes, clearly. I wanted to activate it right now. You want to make it staggered, but fine, make a follow-up ticket then.
- 2026-11-20T11:21 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:22 answer 1 acknowledged by crab
- 2026-11-20T11:22 crab read the answer: hug task note ypfrp3 "Jørgen 2026-11-20 10:22Z (answer 1): 'Yes, clearly. I wanted to activate it right now...make a follow-up ticket then.' …"
- 2026-11-20T11:23 todo/ready -> doing/active (take with p87ngv)
- 2026-11-20T11:23 doing/active -> doing/waiting (wait)
