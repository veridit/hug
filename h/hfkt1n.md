---
id: hfkt1n
title: "TestTmuxReload: 'reload line stayed' under load"
state: "doing"
type: feature
priority: ""
assignee: ["tigress"]
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:38"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
TestTmuxReload passes under machine load as it does alone: name why 'the reload line stayed after leaving the task' fires at load ~120 and close it at the cause

## Purpose
A red TUI run at load ~120 (duckling's full TUI run, ~/.jcode/scratch/hjbfzd-tui3.log) costs a rerun and trust in local suites; it passes alone and in CI

## Principle
Cause before fix; no sleep and no longer timeout unless the cause is a real wait; keep to the test and the reload path; DM macaque before touching tui.go/render.go (tz9z14, rxr4z8)

## Relevant details
Verbatim: 'reload_tmux_test.go:216: the reload line stayed after leaving the task:' with line 1 'HUG · repo · list   15/60 tasks · ! 2 broken · 5 advice · ⠹ refreshing' (the list) but lower lines still the detail's (checks, NOTES, HISTORY, 'reloaded hug v0.0.2', key bar 'j/k scroll · edit · done · wait · x drop ...'). The wait at reload_tmux_test.go:207-211 stops as soon as line 1 has no ›.

## Depends on

## Acceptance
- [ ] The cause is named in a note with the failing capture or a reproduction
- [ ] A reproduction under a CPU hog fails first, then passes -count=50 niced under the same hog after the fix
- [ ] No sleep added and no timeout raised unless the cause is a real wait, said in the note

## Notes


## History
- 2026-11-18T22:38 todo -> doing/active (take)
