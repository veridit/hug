---
id: et85h1
title: "Agents take tasks under their own session name, not the git user"
status: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
When an agent takes a task, the assignee is the agent's session name (crab, poodle), so the board shows who is really working on it; people still take under their own name

## Purpose
Jørgen saw every doing task assigned to him (git user.name) while agents were doing the work; the board lied about who is busy

## Principle
Identity order: HUGINN_USER, then the harness session name (jcode: JCODE_SESSION_ID session_<name>_..., other harnesses where they expose one), then git config user.name. No new config needed for agents. Never guess a name from anything else.

## Relevant details
huginn/internal/cli/cli.go:180 identity() reads HUGINN_USER, else git config user.name. take.go calls it; MCP take and the TUI callback share it. --mine uses the same identity. 2026-11-16 qawmnb and ksv323 were reassigned by hand to crab and poodle.

## Depends on

## Acceptance
- [ ] hug task take inside a jcode session named poodle assigns poodle (test with JCODE_SESSION_ID set)
- [ ] with HUGINN_USER set it wins; with neither, git user.name as before
- [ ] hug task ls --mine and MCP task_take use the same identity
- [ ] take help and ls help name the identity order

## Notes
