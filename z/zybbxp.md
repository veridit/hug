---
id: zybbxp
title: "Page text wraps at its width, without one-word lines"
state: todo
type: bug
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzo"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
Wrapped text on a task's page fills each line to the page's width, with no line holding one stray word.

## Purpose
On kkj0mn at 100 columns, PURPOSE shows 'Run', 'the' and at 160 'minutes:', 'last' alone on lines (v0.7.112 goldens), and Details counts 8 lines where it reads as 4, so it folds.

## Principle
Wrap once, at the width the page is drawn at; the Details limit counts the lines a person sees.

## Details
Seen in huginn/internal/tui/testdata/hya1st/kkj0mn-100x30.txt and -160x45.txt on v0.7.112 (c1056b46): the text is wrapped twice, first by the markdown renderer at a narrower width, then by wrapPage at m.width (detail.go detailText return), so a line just over the first width breaks before its last word. rhino 2026-11-25 suggested this as a small separate fix. Check other goldens for the same pattern before and after.

## Depends on

## Acceptance
- [ ] kkj0mn's page at 100x30 and 160x45 has no line with one word followed by a line that would have fit it (golden), and a test pins it
- [ ] kkj0mn's Details opens at 100 columns once it fits in 6 visible lines

## Notes
