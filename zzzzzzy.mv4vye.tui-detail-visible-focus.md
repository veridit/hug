---
id: mv4vye
title: "TUI detail: visible focus, clear prompts, honest key hints"
status: "doing"
type: feature
priority: "high"
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Inside a task the reader always sees where focus is, scrolling behaves predictably, and a question from the tool cannot be missed

## Purpose
Jørgen 2026-11-16 review of the detail view: the heading 'ACCEPTANCE · [ / ] SELECT' made him press / (which opens search); Tab moved focus between acceptance and dependencies with nothing showing where focus went, and the tab bar still showed [overview] as if focus were there; the 'Confirm edit ID title?' line is bland, so he did not notice he was being asked something

## Principle
Show only keys that work, written so they cannot be misread. Focus is always visible in exactly one place. A prompt that waits for an answer is the most prominent thing on screen. Compact: no extra lines. Esc always backs out

## Relevant details
Code: huginn/internal/tui/detail.go (section headings, detailPane 0 = acceptance, 1 = dependencies), tui.go (detail key handling, j/k step through the focused list then scroll), render.go (confirm prompt near 'if m.confirm', tab bar near the views loop, keyBar in keys.go). Wanted behaviour: three focus states in detail: page (j/k scroll the page), acceptance (j/k step through checks), dependencies (j/k step through them); Tab cycles and skips an empty section; the focused section heading is highlighted and the others are not; with page focus no check carries the > pointer. Write the keys as '[ and ]' or similar, never '[ / ]'. The tab bar must not mark a view as current while a task is open; show the task as current instead (for example 'overview › s6ff5k'). The confirm line uses a strong style (warning colour or reverse) and starts with the question. Already shipped in v0.7.15: { and } step to the previous/next task, Space ticks only in detail, tick prompt reads 'Tick: ☐ 1. text'. Goldens: HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./huginn/internal/tui/ then review the diff

## Depends on

## Acceptance
- [ ] no heading or hint writes keys as '[ / ]'; pressing every key shown in a hint does what the hint says (test)
- [ ] Tab cycles page, acceptance, dependencies and skips empty sections; the focused heading is highlighted and only it
- [ ] with page focus j/k scroll the page and no check shows the pointer; with acceptance focus j/k step through checks
- [ ] while a task is open the tab bar does not mark overview/list/board as focused
- [ ] every confirm prompt is visually prominent and reads as a question; test asserts the style differs from normal text
- [ ] full tests pass niced; goldens regenerated and reviewed

## Notes

2026-11-16 Jørgen, second screenshot: the bland-prompt problem is general. The tick prompt ('Tick item 2 of s6ff5k: ...?') is as easy to miss as the edit one. Every confirm line needs the prominent style, not just edit. Since v0.7.15 the tick prompt reads 'Tick: ☐ 2. text' (render.go, the m.confirm block); keep that wording and make the whole line stand out.

2026-11-16 Jørgen: { and } are hard on a Norwegian layout. Task stepping in detail moves to ',' (previous) and '.' (next), both unbound before; { and } are removed, Ctrl-P/Ctrl-N stay. Sent to the worker as an addition to this task.

2026-11-16 Jørgen, supersedes the ',' and '.' note: in detail n = next task, p = previous task; new task moves to N in every view; { and } removed; Ctrl-N/Ctrl-P stay; in lists n/p move the selection like j/k. Sent to the worker.

2026-11-16 Jørgen on the v0.7.15 board: compact rows are 'a lot better'. Addition: the legend must connect to the column labels, e.g. 'T type: F feature ... · P priority: C critical ...', with T and P styled like the header; narrow fallback 'T: ... · P: ...'. Sent to the worker.
