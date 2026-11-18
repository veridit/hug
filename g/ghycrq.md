---
id: ghycrq
title: "tmux test harness: server start and cleanup deadlines trip under load"
state: todo
type: bug
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
Every tmux-backed TUI test starts and stops its tmux server reliably on a loaded machine: startTmuxServer waits on the server's own readiness and its exit, not on fixed 5 s and 3 s deadlines.

## Purpose
hfkt1n's 50-run campaign under 16 CPU hogs (load ~260) passed 48 of 50 after its fix; both failures were in the harness, not the reload path: 'reload_tmux_test.go:86: foreground tmux server did not become ready' (5 s) and 'tmux_test.go:82: tmux server did not exit after explicit cleanup' (3 s). Every tmux test shares this harness, so each such flake costs a rerun.

## Principle
Find the cause under forced load first (as ad62aa, mq5ger, hfkt1n). A deadline is fine only if what it waits for is real; no raised timeout or retry without a named cause.

## Relevant details
Evidence: /Users/jhf/.jcode/scratch/hfkt1n-ev/REPORT.md (campaign logs). Harness: huginn/internal/tui/tmux_test.go:82, reload_tmux_test.go:86, startTmuxServer. Run niced and bounded (load is often 200+).

## Depends on

## Acceptance
- [ ] both failures reproduced under a CPU hog with their verbatim lines and the cause named with file:line
- [ ] fixed at the cause: TestTmuxReload and TestTmuxGolden pass -count=50 niced under the same hog with 0 harness failures

## Notes
