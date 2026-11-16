---
id: me51r2
title: "hug mem list --json, and forget that survives concurrent writers"
status: todo
type: bug
priority: "high"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Agents can export memories as JSON (hug mem list --json, also on search) and forget many at once (hug mem forget ID ID ... or from stdin) without failures when other agents write at the same moment

## Purpose
2026-11-16 in seid: 'hug mem list --json' failed with 'unknown flag: --json', so the agent parsed text; then 233 of 239 'hug mem forget' calls failed while other agents wrote, and it had to retry one at a time with sleeps

## Principle
Same output conventions as hug task (--json, --jsonl). Writers wait for the lock with a bounded retry instead of failing; a real failure says why with a fix: line. Batch forget is one transaction.

## Relevant details
Reproduce: run 20 hug mem forget in parallel against a store while hug mem remember runs in a loop; count failures. Find the failing error text first (orangutan's run did not print it).

## Depends on

## Acceptance
- [ ] hug mem list --json and hug mem search --json print one JSON object with a memories array (test)
- [ ] hug mem forget accepts several ids and ids on stdin, in one transaction (test)
- [ ] 20 parallel forgets with a concurrent remember loop: 0 failures (test)

## Notes
