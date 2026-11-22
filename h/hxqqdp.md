---
id: hxqqdp
title: "An old hug writing set --details adds a second Relevant details section"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-22
updated: 2026-11-22
---

## Goal
A file with both '## Details' and '## Relevant details' shows both texts, lints, and hug upgrade merges them.

## Purpose
ram 2026-11-22 (6q1d1t): v0.7.101 still running (hug tui/mcp from 11-17..11-19) writing set --details to a '## Details' file adds '## Relevant details'; v0.7.104 show then prints only the Details text, hiding what the old hug wrote.

## Principle
Never hide text: if a file holds both names, show and lint it; upgrade merges them.

## Details
Old binaries cannot be fixed; the new side can: Parse sees both headings, lint reports 'two Details sections' with fix: hug upgrade, and hug upgrade merges them (old text after new). Restarting the old processes ends the risk.

## Depends on

## Acceptance
- [ ] A file with both '## Details' and '## Relevant details' shows both texts and lints with a fix: line (test)
- [ ] hug upgrade merges them into one '## Details', no text lost (test)

## Notes
