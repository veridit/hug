---
id: 8bsx4y
title: "TUI: inline code in a quiet tint, not red"
status: "done"
type: bug
priority: "medium"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
Inline code and code blocks in the task detail read as code without shouting: light blue-grey text on a faint background, no padding, no red; red stays for blocked and lint problems

## Purpose
Jørgen 2026-11-16 on fpp2ns: the red-on-grey inline code everywhere in the description 'isn't it'. Red is reserved for diagnostic meaning. A real syntax colouring tool comes later, not now.

## Principle
Override glamour's Code and CodeBlock styles in huginn/internal/tui/detail.go (next to the heading colours). Dark: text #c9d6e3, background #1a2630, no prefix/suffix padding. Light: #2b3a48 on #e8eef3. Code blocks: same background, no syntax colours. NO_COLOR: plain text.

## Relevant details
Agreed palette from crab's proposal, approved by Jørgen 'Yes'.

## Depends on

## Acceptance
- [x] the detail of a task with inline code shows no red; inline code uses #c9d6e3 on #1a2630 on a dark terminal (golden)
- [x] light terminal uses #2b3a48 on #e8eef3; NO_COLOR shows plain text (test)

## Notes

3678b10 on origin/main (ff from 02066bd), CI run 43914364156 green: test (ubuntu-latest) success, test (macos-latest) success, test (windows-latest) success. Golden 140x40-color/detail.txt: '-Deliver compiler-lang support with  huginn  and readable evidence.' -> '+Deliver compiler-lang support with huginn and readable evidence.' (padding gone; tmux capture holds no SGR). TestCodeQuietTint (code_tint_test.go:27) renders through markdown() and asserts fg #c9d6e3 bg #1a2630 on dark and no red SGR (redSGR, line 23): '--- PASS: TestCodeQuietTint (0.04s)'

TestCodeQuietTint, line 33 case {false, "#2b3a48", "#e8eef3"} asserts the light pair; lines 53-61 assert NO_COLOR emits no colour SGR and unchanged code text: 'ok github.com/veridit/hug/huginn/internal/tui 0.063s'

version corner, from Jørgen: commit 82273c1. The key bar ends with 'hug '+Version (huginn/run.Version, the -X string hug version prints; default changed from stale v0.1.6 to dev), dim, right-aligned; dropped when fewer than 2 spare columns. Golden 140x40-no-color/detail.txt last line: 'j/k check · Space tick · E files · n/p task · Tab section · Esc back · : cmd · ? help · q quit ... hug v0.0.0'; every 80x24 golden drops it (e.g. list.txt key bar 76 cols, no version). TestVersionCorner (version_test.go) checks list/help/palette/detail/sort at 140x40, 100x30, 80x24: bar never wider than the screen, hints are a prefix (none pushed off), version shown iff room >= len+2; width 40 drops it. CI run 43931868360 on ci/zgr01t (60d3f4b = 82273c1 version corner + 8368c53 z6kz20 + 60d3f4b zgr01t, ff onto bb665ea v0.7.31, now origin/main): test (ubuntu-latest) success, test (macos-latest) success, test (windows-latest) success.
