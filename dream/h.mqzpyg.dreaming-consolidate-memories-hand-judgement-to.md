---
id: mqzpyg
title: "Dreaming: consolidate memories, hand judgement to the driving agent"
status: todo
type: feature
priority: "high"
assignee: []
labels: ["dream"]
created: 2026-11-10
updated: 2026-11-10
---

## Goal
muninn dream runs while idle at low priority, applies safe mechanical consolidation automatically with undo, and queues judgement items that the driving agent (jcode, Claude Code, Codex) resolves through MCP memory_dream batches and a one-line hook cue

## Depends on

## Acceptance
- [x] muninn dream exists; exact duplicates retired automatically with an undo receipt
- [x] judgement items queued; memory_dream MCP tool returns at most 5 with instruction and allowed tools
- [x] items handed only to sessions whose scope can see them (WebKit item only in WebKit)
- [ ] Claude Code and Codex hooks carry the same one-line cue
- [ ] morning summary line: merged, handed out, open
- [ ] dated memory text is never rewritten (decision rxw6w7)

## Notes

2026-11-12: shipped in muninn v0.6.0 (hug 0075ad1). First live pass: 15 duplicates retired (undo tidy/apply-20261112T143206Z-1006002259), 9 items queued; client item (brovs) only visible in Brovs; item 4 resolved end to end via memory_dream (kept veeeg8, forgot ywc8z5). Hand-out: start / return after 20 min / wrap-up, once per 6 h per session; tune with the user.
