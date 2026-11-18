---
id: a0gwvg
title: "Repo tool: regenerate TUI goldens from a failed CI run's log"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
A scripts/ci-goldens.sh RUNID [JOB] in the repo rewrites the TUI goldens a CI run reported as 'golden drift', from that run's GOT blocks, so goldens can be refreshed when a local HUG_UPDATE_GOLDEN run cannot finish (machine load) or the drift is platform-specific

## Purpose
2026-11-18 stack 6448e7d5 failed CI 44403710185 on ubuntu only (tmux palette goldens, txjkrm TAGS x 05r3pg palette); crab's local regen could not finish at load ~225. A scratch script fixed it as 4d7eec2d in minutes

## Principle
Never writes unless every WANT block, decoded the same way, is byte-equal to the committed golden (proves the decoding on that run's own log). Refuses when two jobs disagree on a path. Runs on macOS bash 3.2. Errors end with a fix: line. It is a convenience next to HUG_UPDATE_GOLDEN, not a replacement: the diff is still reviewed

## Relevant details
Prototype attached (ci-goldens.sh, from $JCODE_SCRATCH_DIR). Format source: huginn/internal/tui/tmux_test.go t.Errorf("golden drift %s\nGOT:\n%s\nWANT:\n%s"); testing.decorate indents every line after the first and drops the final newline. First use: CI 44403710185 -> 4 palette.txt files, commit 4d7eec2d on ci/stack71. Other golden helpers (golden(), goldenFile() in alias_search_test.go, e3sevr_test.go) may print differently: check before claiming them

## Depends on

## Acceptance
- [ ] scripts/ci-goldens.sh exists and AGENTS.md's TUI goldens line names it
- [ ] a test feeds a recorded --log-failed excerpt (with a blank line and a last-line golden) and gets byte-identical files
- [ ] a tampered WANT block makes it write nothing and exit non-zero with a fix: line

## Notes
