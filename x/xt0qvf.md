---
id: xt0qvf
title: "The status message sits on the line just above the keys"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzp"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
hug tui's status message sits on the line right above the keys, with no blank line kept for it when there is none.

## Purpose
Jørgen 2026-11-20 09:57Z (tz9z14 answer 1, screen-1055/1056.ans): 'reloaded hug v0.7.88' shows third from the bottom, then a blank line, then the keys; when there is no message the blank line stays, so a status update looks like part of the page.

## Principle
A status update reads as one: next to the keys, marked as a status (a dim sigil or tint), not mixed into the page; no line reserved for nothing.

## Relevant details
Today (v0.7.88): ...page / message / blank / keys. In the detail: 'Task 2 of 30 · ...' / blank / keys. Options for Jørgen: (A) message directly above the keys, the blank line gone when there is no message; (B) a blank line above the message, separating it from the page; either way a small sigil or tint marks it as a status. Show both as visible ANSI frames first.

## Depends on

## Acceptance
- [ ] Jørgen picks the layout from two ANSI frames (message right above the keys vs separated from the page) attached here
- [ ] With a message, it sits where Jørgen chose, marked as a status; with none, no blank line is reserved for it (golden 100x30, overview and detail)

## Notes
