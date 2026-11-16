---
id: me51r2
title: "Concurrent writes wait their turn; hug mem list --json; batch forget"
status: "doing"
type: bug
priority: "high"
assignee: ["dragon"]
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

2026-11-16 Jørgen: many agents acting at once is the normal case, not an error. Every hug write (mem and task) waits its turn and then succeeds; it fails only if the wait is far longer than any normal write, and then names who holds the lock and gives a fix: line. Applies to remember, forget, sync, and hug task writes alike.

2026-11-16 15:39 crab: worker hare's session ended at the 15:21 server reload, uncommitted nothing for me51r2. Its repro (tmp/lanes/me51r2-repro.sh, run in /tmp/me51w on a copy of the real store) captured the failures verbatim: 17x 'hug mem: database is locked (5) (SQLITE_BUSY)' and 5x 'hug mem: index: database is locked (5) (SQLITE_BUSY)'. Profiling scripts: tmp/lanes/me51r2-prof.sh, me51r2-reindex1.sh, me51r2-time1.sh; profile /tmp/me51w/forget.prof.

cause (dragon, measured on a copy of hare's snapshot, tmp/lanes/me51r2-cold.sh): busy_timeout(30000) is a budget for the whole queue, not per holder, and SQLite's busy handler polls (no queue). Every hug mem forget holds the write lock for full index passes (forgetCmd pre-Reindex + App.Forget Reindex after rewriting the 14.6 MB file, 1.53 s warm alone). On the copy the first pass is cold: 'units added=12723 ... 32.31s sources=14.714s communities=17.391s'; 19 of 20 parallel forgets failed at exactly 31 s with 'hug mem: index: database is locked (5) (SQLITE_BUSY)', memories still active in the file. Warm copy: 20/20 ok, 3-11 s each, so on the real store the queue length times ~1.5 s crosses 30 s (seid: 239 forgets). Also: NoWait (hook) passes reset the connection's busy_timeout to 5000 after BEGIN, not 30000 (pipeline.go:174).

check 1 (dragon, e3c32f0): TestListAndSearchJSON: 'hug mem list --json' and 'search --json' each one object {"memories":[...]} (id, kind, scope, category, type, tags, text, updated; search adds score), empty search prints '{"memories":[]}', no next: hint on stdout. 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 -run TestListAndSearchJSON|TestForgetSeveral|TestParallelForgets ./run/' -> 'ok  	github.com/veridit/hug/muninn/run	8.781s'
