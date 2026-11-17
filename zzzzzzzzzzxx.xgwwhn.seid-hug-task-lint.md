---
id: xgwwhn
title: "on Seid, hug task lint reports what tools/evidence.py check reports (counts in…"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-18
updated: 2026-11-18
---

## Goal
on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch

Promoted from item 4 of task:cvbpdy.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 1. Jørgen (the King in Seid) wants hug lint to own these rules for every repo.

## Principle
Declarative in hug.toml, no Seid-specific names in code. Every finding has a fix: line an agent can run (hug task attach ID FILE, or mark it (lost)). Lint never deletes anything.

## Relevant details
Seid reference: tools/evidence.py lines 91-112: SCRATCH_REF, EVID_REF, LOST=' (lost)', KEEP_EXT, MAX=256 KiB. hug code: task/files.go LintFiles (line 267; forbid check line 300 uses strings.Contains on the whole doc, so no line number and no (lost) escape), task/config.go ForbidCite. Text-extension list: hug checks content (NUL, utf8), which is stricter than an extension list; keep that.

## Depends on

## Acceptance
- [ ] on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch

## Notes
