---
id: s6ff5k
title: "Measure amortized cleanup: forgets and corrections after recall"
status: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-12
updated: 2026-11-12
---

## Goal
Know whether agents act on the recall header's Fix line, and add an off-topic signal that never deletes

## Depends on

## Acceptance
- [ ] stats: share of recall blocks followed by memory_forget/remember of a shown id within the session
- [ ] memory_feedback <id> off-topic lowers recall of that memory for similar messages, no deletion
- [ ] header wording change only if the 5-agent eval does not regress

## Notes
