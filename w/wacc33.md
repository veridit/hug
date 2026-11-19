---
id: wacc33
title: "TUI tests leave tmux servers running after they finish"
state: "doing"
type: bug
priority: ""
assignee: ["bat"]
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T21:05"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every tmux server a test starts is gone when the test ends, pass or fail.

## Purpose
crab 12:01Z: found 3 orphaned tmux servers (parent launchd): huginn-reload-90820 (since 19 Nov 02:14, running a v0.0.2 'hug tui' in a scratch repo), huginn-golden-99210 (since 17 Nov 22:45), huginn-golden-62963 (since 19 Nov 01:11); killed. They hold memory and a TUI process on a shared machine.

## Principle
t.Cleanup kills the server even on t.Fatal or a timeout; a test fails if a server it started is still alive at cleanup.

## Relevant details
Grep huginn/internal/tui for tmux -D -L huginn-golden / huginn-reload; check whether killed tests (go test -timeout, CI cancel) skip cleanup, and whether -D detached servers survive their parent.

## Depends on

## Acceptance
- [ ] A test that fails or times out mid-run leaves no tmux server (test that runs a failing child and checks pgrep)
- [ ] After a full niced go test ./... no huginn-* tmux server remains (note with pgrep output)

## Notes


## History
- 2026-11-19T21:05 todo -> doing/active (take)
