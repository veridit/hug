---
id: z2vmte
title: "Wait for a date: wait --until and ask --after, woken by hug"
state: "done"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzlh"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-24T00:57"
---

## Goal
A doing task can wait until a date, then returns to its owner or asks its question that day.

## Purpose
Jørgen 2026-11-18 02:28 local, on 47sxn2 ('a week of use: has he had to remind an agent to update a ticket? look again 2026-11-24'), which hug upgrade put in his ASKED today: 'waiting for time to pass is a sensible state and when the time has passed the system brings it back... it's not waiting on me... maybe it should be a native thing.' A question that can only be answered later is not his to address now.

## Principle
Time is a thing a task waits on, like CI; a person is asked only when they can answer. hug wakes the task on its own: no agent has to remember. A date reads in the person's local time. No new stage: --until is a wait with a date, --after is an ask that starts as that wait.

## Relevant details
A doing task can wait for time to pass: hug task wait ID --until DATE "why" parks it in WAITING until DATE, then it returns to its owner, first in hug task next; hug task ask ID --to NAME --after DATE "question" parks it in WAITING until DATE, then the question appears in that person's ASKED. Nothing about it shows in ASKED before the date.

Today: hug task dep --add 'when date YYYY-MM-DD' holds a todo task out of next, but a doing task has no timed wait. Fields: waiting_until (and, for --after, the parked question with its asked_for/asked_check, moved to asked_* at the date). Waking happens on any hug read (ls, next, tui refresh) once the date has passed, written as one commit with a History line; the TUI WAITING row shows 'until 24 Nov'. 3nhp67's rule holds: the wait's fields are cleared when it wakes. MCP task_update wait/ask take until/after. Also: hug upgrade turns a wait or ask whose text says 'look again DATE' or 'on DATE' into this form, listed in the preview.

## Depends on

## Acceptance
- [x] wait --until DATE: WAITING with 'until DATE' before; on the first hug read after DATE it is doing/active and first in its owner's hug task next, with one History line (test with a fixed clock, fails first)
- [x] ask --after DATE: not in the person's ASKED before DATE; after it, in ASKED with the question verbatim and its check (test with a fixed clock)
- [x] 47sxn2 on hug's board becomes ask --to jhf --after 2026-11-24 and leaves Jørgen's ASKED today (done with hug commands)
- [x] Jørgen sees 47sxn2 come back into his ASKED on 2026-11-24 without anyone touching it
- [x] done when task:atex3d is done (A waiting task's stage line leads with the date: 'waiting until 2026-11-24 · on WHAT · since 11-18', with no second 'until' and no 'on the date' wording; the same in hug tui's header, its list row stage cell ('until 11-24') and hug task show (golden, test))

## Requests
- 2026-11-20T13:05 request 1 by jhf:
  So I love this ticket. It will come back to my ask queue on a date. My problem is why can't I see in the heading what date we're waiting for? So waiting on check 3, grab. Now crab converts. Check 4, Jrgen sees it in asked, but the wait date wait until shouldn't that show the date? And shouldn't it just be wait until and the date? Or you prefer the wait date and then wait until for clarity or wait until date. Wouldn't that be better and then show the date?
  resolved 2026-11-20T13:05 by crab:
    Agreed: a wait for a date should say 'until' and the date first. Two changes. (1) This ticket itself waited without a date, only words naming 2026-11-24; it now waits until 2026-11-24 on 'Jørgen sees 47sxn2 back in his ASKED', so its heading shows the date. (2) New check 5: every dated wait's line leads with the date, 'waiting until 2026-11-24 · on WHAT · since 11-18', without today's doubled 'on the date 2026-11-24 ... until 2026-11-24', in hug tui's header, the list's stage cell and hug task show.

## Notes


Landed 88a6ac89.
- Failing first on main: task timedwait_test.go fails with 's.WaitUntil undefined'. cli TestTimedWaitCommands fails with 'unknown flag --after; valid: --as, --model, --to, --check, --show'.
- Tests: TestWaitUntilWakesOnTheDate, TestAskAfterAsksOnTheDate, TestWaitReplacesHeldAsk, TestWakeMarksNothingRead, TestTimedWaitRefusesBadDates and TestTimedWaitCommands (cli: one wake commit for 2 tasks, nothing committed the day before or on a second read). MCP until/ask_after are covered in TestAskOverMCP.
- Mutants: 12 of 12 killed, including due-a-day-late ('due on 24 Nov: []') and read-never-wakes.
- Race ok: task 25.9s, cli 111.9s, mcp 9.5s, cache 11.1s. tui -short ok 43.0s.
- Cache schema 22. TUI WAITING row shows '· until 24 Nov' (nautilus ok'd waitrow.go).
- Check 3 holds until the release is installed. A v0.7.54 hug that writes on the task moves the unknown waiting_ask_* fields to History, and it never wakes the task.
landed 88a6ac8 on main 2026-11-18: hug task wait --until DATE and ask --after DATE: hug wakes the task on the first…
shipped in v0.7.55 (88a6ac8)
check 3 (crab, 05:08 local, v0.7.55): hug task ask 47sxn2 --to jhf --after 2026-11-24 --check 2 '...' -> 'stage: waiting on the date 2026-11-24, then ask jhf since 2026-11-18 05:08 · until 2026-11-24'. 47sxn2 left Jørgen's ASKED.
file:screen-1304.ans (30551 bytes, sha256:82681517) screen capture from hug tui
check 4: Held untouched: commit 650bdf53 at 2026-11-24 00:00 local 'hug task wake: 47sxn2, z2vmte (the date came)', made by hug's own read. HUG_USER=jhf hug task ls --asked (v0.7.110) now lists '47sxn2  2  a week of use: has he had to remind an agent to update a ticket?  crab  since 00:00'.
Reflection (open, 2026-11-24, crab): All 5 checks hold: c4 by hug's own wake at 00:00 local (650bdf53), c5 by atex3d in v0.7.111.
## History
- 2026-11-18T03:38 todo -> doing/active (take)
- 2026-11-18T04:23 doing/active -> doing/waiting (wait)
- 2026-11-20T13:05 request 1 by jhf
- 2026-11-20T13:05 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:05 request 1 acknowledged by crab
- 2026-11-20T13:05 doing/waiting -> doing/active (resume); was waiting_asked_by: kestrel; waiting_on: check 3: crab converts 47sxn2 after the release with 88a6ac89; check 4: Jørgen sees it in ASKED on 2026-11-24; waiting_since: 2026-11-18T04:23
- 2026-11-20T13:05 request 1 in progress by crab: hug task resume z2vmte
- 2026-11-20T13:05 resolved request 1 by crab
- 2026-11-20T13:05 doing/active -> doing/waiting (wait until 2026-11-24)
- 2026-11-24T00:00 doing/waiting -> doing/active (woke: the date 2026-11-24 has come); was waiting_asked_by: crab; waiting_on: 47sxn2 returns to ASKED (check 4); waiting_since: 2026-11-20T13:05; waiting_until: 2026-11-24
- 2026-11-24T00:00 crab back on it after the wait
- 2026-11-24T00:01 doing/active -> doing/waiting (wait)
- 2026-11-24T00:57 doing/waiting -> doing/active (resume); was waiting_asked_by: crab; waiting_on: atex3d done (check 5 promoted to it); waiting_since: 2026-11-24T00:01
- 2026-11-24T00:57 doing/active -> done (done); was doing_since: 2026-11-18T03:38
