---
id: 8pwc0k
title: "Release hug v0.7.57"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzmh"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T07:11"
---

## Goal
Release newest all-green main f767f6d8 as v0.7.57 and verify installed version

## Purpose
Deliver tested search, column layout and secret-boundary fixes without mixing layout 3 migration

## Principle
Only exact all-job green SHA, reviewed changelog, release success before installation, tool recall stays off

## Relevant details
CI 44247018416. Layout w40cmk remains off main and must ship separately.

## Depends on

## Acceptance
- [x] Tag and merge pushed, release workflow succeeds, brew-installed hug reports v0.7.57

## Notes


Released v0.7.57 from f767f6d8, exact CI 44247018416 all six jobs success. Tag 0052c823, main merge e9c65c6f. Release workflow 44248615828 succeeded before brew upgrade. CHANGELOG merge has no extra diff versus tag; layout3 absent. Installed version verified below.
Installed verification: hug version printed "hug v0.7.57" after successful release workflow 44248615828 and targeted brew upgrade 0.7.56 -> 0.7.57.
## History
- 2026-11-18T07:04 todo -> doing/active (take)
- 2026-11-18T07:11 doing/active -> done (done); was doing_since: 2026-11-18T07:04
