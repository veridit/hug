---
id: tmz060
title: "File viewer: .md files show as formatted Markdown, in subtle colours"
state: "doing"
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-18T06:51"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When Space shows a .md file inline (scrxak's viewer, glance.go), it renders the Markdown with the same quiet renderer the detail uses for notes (detail.go markdown(): accent headings, blue-grey code, no syntax colours, no red), wrapped to the pane. Other files stay plain text. A key (M) flips one file between formatted and raw source.

## Purpose
Jørgen 23:29-23:30 local: wants a simple Markdown colour formatter in the inline viewer. Task FILES are mostly .md designs and evidence (6ephqk has three), and today the viewer shows them as raw source.

## Principle
Reuse the detail's renderer and palette, no second style. Escaping still holds: control characters and ESC sequences in the file are escaped before rendering, so a file can never drive the terminal. Raw source stays one key away. Nothing is written. No new dependency (glamour is already in go.mod).

## Relevant details
glanceText(b, size, width) today: escapeControls + tab expansion + wrap, line by line. For a name ending in .md or .markdown it would call the detail's markdown(), after escapeControls. The 1 MiB and binary limits stay. The in-text search of 4s3g5q should match in this view too, on the rendered text. Owner: nautilus after 4s3g5q, or whichever TUI worker is free first; touches glance.go plus goldens.

## Depends on

## Acceptance
- [ ] Space on a .md file shows headings in the accent colour, inline code and code blocks in the blue-grey tint, lists indented, no raw '#' or backticks (golden at 80 and 140 columns)
- [ ] A .md file holding \x1b[2J and a BEL shows them escaped, not acted on (test)
- [ ] M flips the shown file between formatted and raw source, and the footer names the key; a .txt or .go file is unchanged (test)
- [ ] Opening a .md file under 1 MiB renders within 200 ms on a 900 KiB fixture, or shows a spinner while it renders (counting test, not a timer)
- [ ] Jørgen confirms on his board after release

## Notes


## History
- 2026-11-18T06:51 todo -> doing/active (take)
