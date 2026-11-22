---
id: tfe9nr
title: "TestTmuxAutoRefresh parses stderr as JSON from hug task new"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzz9"
created: 2026-11-21
updated: 2026-11-21
---

## Goal
TestTmuxAutoRefresh reads only stdout from 'hug task new --json', so a stderr line can never break its parse.

## Purpose
ladybug 2026-11-21 (4ddcwa): tmux_refresh_test.go:65: invalid character 'h' looking for beginning of value, once, passed on rerun. The run helper uses CombinedOutput (tmux_refresh_test.go ~49), so any stderr line (a git hint:, a hug notice) lands before the JSON.

## Principle
Fix the cause, not the retry: take stdout for parsing, keep stderr for the failure message. Find which line starting with 'h' was printed and say whether it should print at all.

## Relevant details
Same pattern may exist in other tests that json.Unmarshal CombinedOutput: grep for it.

## Depends on

## Acceptance
- [ ] A test that makes hug task new print a stderr line still parses (fails first with CombinedOutput)
- [ ] The stderr line starting with 'h' is named, and either fixed or shown to be correct
- [ ] Other tests that unmarshal CombinedOutput are listed and fixed

## Notes
