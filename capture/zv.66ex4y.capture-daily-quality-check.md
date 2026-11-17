---
id: 66ex4y
title: "Capture: daily quality check"
state: "doing"
type: chore
priority: "medium"
assignee: ["hatchling"]
labels: ["capture"]
created: 2026-11-10
updated: 2026-11-10
doing_since: "2026-11-17T21:58"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Captured memories are reviewed daily for wrong place, triviality and duplicates while capture runs live

## Purpose
Capture runs live on every agent session; a bad memory is recalled into later prompts until someone retires it, so a daily look catches what the eval sets miss.

## Principle
Judge the real store, fix only what is clearly wrong with hug mem commands (forget, reinforce, rescope) and list every change; no capture code changes from this check: a finding that needs code becomes a ticket for crab.

## Depends on

## Acceptance
- [ ] a short daily note of what was wrong and what was fixed

## Notes


## History
- 2026-11-17T21:58 todo -> doing/active (take)
