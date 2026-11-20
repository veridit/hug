---
id: xgwwhn
title: "Seid switches its pre-commit to hug task lint"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzc"
created: 2026-11-18
updated: 2026-11-18
aliases: ["seid-hug-task-lint"]
---

## Goal
On Seid, hug task lint reports what tools/evidence.py check reports, so its pre-commit hook can switch to hug.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 1. Jørgen (the King in Seid) wants hug lint to own these rules for every repo.

## Principle
Declarative in hug.toml, no Seid-specific names in code. Every finding has a fix: line an agent can run (hug task attach ID FILE, or mark it (lost)). Lint never deletes anything.

## Relevant details
on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch

Promoted from item 4 of task:cvbpdy.

From cvbpdy check 4 (promoted 2026-11-18, crab/Jørgen 00:38). hug's side is shipped in v0.7.43 (95079e0): forbid_cite, ' (lost)', .hug/tasks/ID/NAME citations. On a copy of Seid's board hug finds 530 forbidden-citation findings on 464 lines, the same lines as tools/evidence.py (0 on one side only). Left, Seid's own work: set forbid_cite in hug.toml, clear the 530 lines (attach or ' (lost)'), switch the pre-commit hook. Owner orangutan.

## Depends on

## Acceptance
- [ ] on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch

## Notes
