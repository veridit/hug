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
parent: "d634kr"
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

2026-11-16 feedback from another agent on the recall header (via Jørgen): it did not act on 'Fix: memory_forget <id>, memory_remember' because (1) it read the header as a caution, not a task; (2) 'not the user' read as 'not mine to edit'; (3) it assumed the tools were missing and did not check; (4) it was in wait-for-owner mode; (5) the .0 .1 .2 table hides the one instruction. Suggested: plain imperative ('fix it now without asking'), 'part of your job', name the tool server, one note per line. Coordinator's reading: consistent with item 1 (0 of 454 sessions called memory_forget). The 2026-11-10 5-agent test (b56828e: F1 38/40 vs old 28/40, old 0/10 on fixing a wrong note) measured whether agents know HOW to fix a note once it is plainly wrong; it did not measure whether they act unprompted mid-task. Risk in the suggestion: 'fix it now' without separating wrong from merely off-topic would delete good memories; off-topic must never delete (item 2). Proposal for item 3: rerun the 5-agent test with two added scenarios, (a) a stale note appears while the agent works on something else (measures unprompted fixing), (b) an off-topic but true note appears (must NOT be forgotten); compare F1 against the suggested wording and a variant that only names the server. Ship only a wording that wins (a) without losing (b) or the original scenarios.
