---
id: 5mmqjk
title: "hug completion: one script for task, mem and the top level"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-18
updated: 2026-11-18
parent: "gem59b"
---

## Goal
hug completion bash|zsh|fish|powershell prints one completion script for the hug command: top-level commands (task, mem, tui, mcp, apps, setup, upgrade, version), hug task's commands and flags, and hug mem's cobra commands and flags.

## Purpose
p277bp check 2 removed hug mem completion (crab 22:21, option A): it completed the old command name muninn. hug has no completion of its own.

## Principle
One script for the one command users type (hug). Scripts already installed for muninn keep working through the hidden __complete command. No completion per part.

## Relevant details
cobra's completion scripts take the program name from c.Root().Name() (cobra v1.10.2 completions.go, lines 774/824/861/889/911) and call '<name> __complete ...'; muninn's root is Use: "muninn" (muninn/run/main.go:60). cmd/hug/main.go dispatches an unknown first word (such as __complete) to huginn.Main, so 'hug __complete mem ...' reaches huginn, not muninn. hug task (huginn) is not cobra: it has its own Commands list (huginn/internal/cli/cli.go:61) and commandHelp usage map, so a hug script must combine both.

## Depends on

## Acceptance
- [ ] hug completion zsh output, sourced in zsh, completes 'hug ta<Tab>' to task, 'hug task ta<Tab>' to take, 'hug mem se<Tab>' to search (test drives __complete)
- [ ] muninn __complete still answers for scripts installed before (test)
- [ ] README Install names the completion line for each shell

## Notes
