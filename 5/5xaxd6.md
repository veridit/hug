---
id: 5xaxd6
title: "Flaky on Linux: data race in bubbles textinput sanitizer across TUI tests"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzb"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
huginn/internal/cli TUI tests pass under -race on Linux without the bubbles textinput sanitizer data race.

## Purpose
CI run 44610132775 (ci/x8zzzn, SHA 9073b532, a muninn-only change) failed on test (ubuntu-latest) with 2 WARNING: DATA RACE reports in huginn/internal/cli; a red ubuntu job blocks merges of unrelated work

## Principle
Fix the sharing (one sanitizer per textinput model, or no parallel TUI programs sharing one), not the test; never drop -race

## Relevant details
huginn/internal/cli TUI tests pass under -race on ubuntu: TestTUIHistorySurvivesReopen, TestTUIPaletteNoteCommits, TestEveryWriteEndsWithTheTasksLint and TestEveryRefCommandAcceptsTaskColon no longer race in charmbracelet/bubbles runeutil.(*sanitizer).Sanitize (runeutil.go:65 read vs :93 write) when two TUI programs run at once

Stack: huginn/internal/tui/tui.go:2011 updateInput -> bubbles textinput.go:286 insertRunesFromUserInput -> runeutil.go:65/93; callers tuihistory_test.go:29 and tuipalette_test.go:26 via cli.(*App).runTUI hooks.go:184. Seen 2026-11-19T20:38Z. Log: /Users/jhf/.jcode/scratch/x8zzzn/ci-failed.log

## Depends on

## Acceptance
- [ ] go test -race ./huginn/internal/cli/ passes 20 times in a row on ubuntu CI with no DATA RACE line

## Notes
