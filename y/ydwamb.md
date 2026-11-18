---
id: ydwamb
title: "Lint reads a glob citation (compare-*.png) as a missing file"
state: "doing"
type: bug
priority: "medium"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T22:34"
stage: "active"
model: "claude-opus-5-5"
branch: "ydwamb"
worktree: "/Users/jhf/.jcode/scratch/ydwamb"
---

## Goal
hug task lint treats a citation with a shell glob, like .hug/tasks/9/911j70/compare-*.png, as the pattern it is: fine when it matches at least one attached file, a finding naming the pattern when it matches none.

## Purpose
911j70 note (crab, 2026-11-18) tells Jørgen to run: open -a Preview .hug/tasks/9/911j70/compare-*.png. Lint cut the citation at * and reports '.hug/tasks/9/911j70/compare-: file not found', fix: hug task attach 911j70 FILE --name compare-. That is a false broken finding with a fix that would attach a junk file. Ten compare-*.png files are attached.

## Principle
Lint never writes. A real missing file is still found. No special case for 911j70: any citation with * ? or [..] in the last path part is a glob, matched against the task's attached files.

## Relevant details
Finding: hug task lint 911j70 -> 9/911j70.md:0: broken [missing-file] .hug/tasks/9/911j70/compare-: file not found. The citation parser is in huginn/internal/task lint (missing-file). Run tests niced: GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./huginn/...

## Depends on

## Acceptance
- [ ] a test: a note citing .hug/tasks/X/ID/compare-*.png with two matching files gives no finding (fails before)
- [ ] a test: a glob citation matching no attached file gives one missing-file finding naming the whole pattern, and its fix: line does not suggest --name with a cut-off name
- [ ] hug task lint on this board shows no missing-file finding for 911j70

## Notes


## History
- 2026-11-18T22:34 todo -> doing/active (take)
