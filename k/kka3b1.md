---
id: kka3b1
title: "Release v0.7.91: r sends one request for the marked tasks"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Jørgen confirms v0.7.91: r on marked rows opens one request for exactly that set, box under the list.

## Purpose
He marked four LINT rows and had no way to ask about them together.

## Principle
One question per shipped task; frames attached so it can be judged without installing.

## Relevant details
Shipped: ec6z6j checks 4 and 5 (d3b79155, CI 44780705119 8/8). Try it: brew upgrade veridit/tap/hug, hug task tui, Space on 2-3 rows in the lint view (!), then r, write, Enter. Frames attached: before and after at 80x24.

## Depends on

## Acceptance
- [ ] Jørgen confirms r on marked rows works as he wanted

## Notes

file:before-lint-3marked-marked.ans (5739 bytes, sha256:231e255d) before (v0.7.90): 3 rows marked, no request in the footer

file:after-lint-3marked-marked.ans (5796 bytes, sha256:41c28c0b) after: footer offers request
