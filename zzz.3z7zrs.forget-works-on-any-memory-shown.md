---
id: 3z7zrs
title: "forget works on any memory shown to you"
status: "done"
type: bug
priority: ""
assignee: []
labels: []
created: 2026-11-11
updated: 2026-11-11
---

## Goal
memory_forget accepts every id recall showed in this session

## Depends on

## Acceptance
- [x] forget of a parent-folder memory from ~/Projects/muninn succeeds when recalled there

## Notes

2026-11-11: d8jb1b (scope folder:~/Projects) was recalled at ~/Projects/muninn, but 'muninn forget d8jb1b' there said 'no visible memory'. Worked only from ~/Projects.

Fixed in hug 819fde8 (2026-11-12) with regression tests.
