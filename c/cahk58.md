---
id: cahk58
title: "Hand a doing task to another agent, with its model and branch"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
A coordinator hands a doing task from one agent to another with one command that records the new assignee, model, branch and why.

## Purpose
Seid 2026-11-25 13:52 CET (orangutan, gk6sve): calf's take was refused ('taken by wyvern'), so it used set --assignee calf, which left model deepseek-v4-pro and wyvern's branch in the header; the real model sat only in a note.

## Principle
The header is the truth: one handover writes every field take writes, keeps the history, and names who handed it over and why.

## Details
Seen on seid-lang hug/tasks: 34db1397a (13:51, hug task set gk6sve) changed only assignee wyvern -> calf; 68f71a319 (13:52, hug task take) then set model claude-opus-5-5, branch gk6sve-gate-gaps and the lane worktree, so the header is right now, but only because take ran after set; between them it read assignee calf, model deepseek-v4-pro. Today (v0.7.112): take refuses a task another agent holds (huginn/internal/cli/take.go:63, fix 'hug task next'); take does take over from an owner whose session is gone (9ahvgt), but wyvern was stopped cleanly and still counted. set --assignee changes one field (help: 'replace the field with one value'). Proposal: hug task take ID --from OWNER --model M [--as NAME] --reason "why" takes it over in one write (assignee, model, branch, worktree, doing_since kept or reset), notes 'handed over from wyvern (deepseek-v4-pro) to calf (claude-opus-5-5): why'; the refusal's fix: line names it. set --assignee on a doing task warns that model and branch stay and points to take --from.

## Depends on

## Acceptance
- [ ] hug task take ID --from OWNER --reason "..." moves a doing task to the taker in one commit: assignee, model, branch and worktree are the taker's, and a note names both owners and models (test)
- [ ] The 'taken by' refusal's fix: line names take --from; set --assignee on a doing task warns that model and branch stay (test)
- [ ] Seid's coordinator hands a task over with it and the header is right without a second command

## Notes
