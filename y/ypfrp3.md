---
id: ypfrp3
title: "Release v0.7.85"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-19
updated: 2026-11-19
design_asked_by: "crab"
design_question: "x8zzzn: shall I switch mid-turn recall on in your config now ('recall': {'jcode_midturn': true}), so messages you type while I'm working get their memories, and you decide after a day whether it replaces the old path?"
design_for: "jhf"
design_since: "2026-11-19T22:00"
stage: "design"
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

## Notes

check 1: release.yml 44613144851 success; brew upgrade: hug v0.7.85, hug/muninn/huginn -> Cellar/hug/0.7.85


## History
- 2026-11-19T22:00 todo -> todo/design (ask)
