---
id: txjkrm
title: "Tags: listed per board in hug.toml, shown on rows, Seid's labels cleaned up"
status: todo
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
Labels become tags: each board lists its tags in hug.toml, every task's tags show dimmed on its row before the title (compiler parser  Compile-time bounds…), lint holds every task to the list, and hug upgrade cleans an existing board (merges spelling variants, moves labels that duplicate type, stage or priority into those fields, drops one-off tags) after showing the diff.

## Purpose
Jørgen 2026-11-17 10:24-10:27: backlog's board on Seid shows [Compiler][Parser] on each row and hug's does not, so the area is lost at a glance. State and stage are hard-coded because they carry semantics; tags are the per-project freedom to organise and see where work belongs, and listing them gives the consistency people and AI need. Measured on Seid (2,516 tasks): 891 distinct labels, 420 used once, variants like perf/performance, backend-c/c-backend, compiler/Compiler, test/tests/testing, and labels that are not tags (bug 467 = type, design 244 = stage, task-632 = a reference, low-pri = priority).

## Principle
Tags carry no meaning to hug: display, filter, group only. The list in hug.toml is the truth; a tag not on it is a lint finding with the fix (add it to the list, or hug task set ID --tag). No backward compatibility: the field is tags, hug upgrade renames labels. The cleanup never writes before Jørgen approves the diff. Subtle on rows; tags are cut after the title, before the owner (102bc0 order).

## Relevant details
Data: Seid label counts in /Users/jhf/.jcode/scratch/seid-labels.txt (regenerate with the grep in the notes). Seid board: ~/Veridit/src/seid-lang/.hug/tasks. Related: 6ephqk (state and stage, which absorbs design/king-ruled), kx2ndm (g groups; by tag is a candidate view), 102bc0 (row layout). Preview first: a mock of the overview with tags, from real Seid tasks, for Jørgen's look before building.

## Depends on

## Acceptance
- [ ] a preview of the Seid overview with tags on rows, built from real tasks, is shown to Jørgen and approved before the TUI work starts
- [ ] hug.toml tags = [...] is read; lint flags a task tag not on the list with a fix: line (test)
- [ ] rows show the task's tags dimmed before the title and cut by 102bc0's order (goldens at 80 and 140 columns)
- [ ] filter tag:compiler and g grouping by tag work (test)
- [ ] hug upgrade --tags on Seid proposes the merges, field moves and drops as a diff with counts, writes nothing until approved; after approval lint is clean of tag findings

## Notes

Jørgen 2026-11-17 10:28 (row layout): the state is the section header, so rows drop the status column. Row order: pie, id, type, priority, tags, stage, text. A stage with extra data shows 'stage: reason' (waiting: CI run N · 12 min; depends: 44w1ev (doing)) instead of the title; a stage without it shows the title. Mock in tmp/lanes/txjkrm-preview.md (Revision). Open: reason replaces the title fully, or reason then dim title.
