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
- [x] stats: share of recall blocks followed by memory_forget/remember of a shown id within the session
- [ ] memory_feedback <id> off-topic lowers recall of that memory for similar messages, no deletion
- [ ] header wording change only if the 5-agent eval does not regress

## Notes

Item 1 measured 2026-11-16 (local data only): of 454 jcode sessions that received recall injections (injections table, keys jcode:<session>), 0 called memory_forget as a tool (counted from tool-use records '"name":"mcp__muninn__memory_forget"', not text mentions; the recall header itself contains 'memory_forget'), and 13 memory_remember calls occurred across them. Agents essentially never act on the header's Fix line. Implication for item 2: cleanup must not depend on agents volunteering it; an off-topic feedback signal needs to be cheap and prompted, or inferred.
