---
id: qpsata
title: "New tasks are placed by priority, and hug says where one landed"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
When an agent files a task, hug says where it landed in the queue and how to move it, so new work goes ahead of older backlog.

## Purpose
Jørgen 2026-11-25: new actionable tasks crab filed landed behind 40 older ones; 'you should have been handed an instruction by our own system' to place them.

## Principle
The system prompts at the moment it matters, in the command's own output; a memory is no substitute for the instruction.

## Details
Today (v0.7.112): hug task new prints the task and 'next: hug task take ID' only (probe 1z5sw3, 2026-11-25 12:30Z, dropped). Placement exists (--top, --after REF, --group; hug task move ID --top|--after|--before) but nothing asks for it, and AGENTS.md's New work line does not mention it. Proposal: (1) new prints 'placed: N of M in todo, after <id title>' and a second next line 'next: hug task move ID --top (or --after ID) if it comes before older work'; (2) with --priority high or a bug, new places it above the first task of lower priority instead of at the end; (3) AGENTS.md and hug setup's template add one sentence to New work: place it by priority (--top or --after). Same for MCP task_new's result.

## Depends on

## Acceptance
- [ ] hug task new prints where the task landed and a next line to move it; MCP task_new returns the same (test)
- [ ] A new task with --priority high lands above the first lower-priority todo task, not at the end (test)
- [ ] AGENTS.md and hug setup's New work line say to place a new task by priority, pinned by a test

## Notes
