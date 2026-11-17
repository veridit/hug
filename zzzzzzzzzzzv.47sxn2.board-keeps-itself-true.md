---
id: 47sxn2
title: "The board keeps itself true: deduce what happened, ask when it can't"
status: todo
type: feature
priority: "high"
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
Tickets match what really happened without a person reminding anyone. Two levels: (1) when hug can deduce an event with certainty from git or the board (a commit naming the ticket reached main, a release tag contains it, a branch recorded on take is merged), it writes that onto the ticket itself; (2) when it observes something it cannot settle (landed but checks unticked, a doing ticket with no activity, a commit on main naming no ticket, a merged branch whose ticket is still doing), it raises a clarification request on the ticket, addressed to its owner, who answers in place.

## Purpose
Jørgen 2026-11-17 11:55: 'the fact that I am reminding you to update tickets means that our system is deficient.' Today: 0hkaa5 shipped in v0.7.38 and stayed doing; m1c28r and t709he shipped in v0.7.40 with 0 checks ticked; hatchling held 5 tickets in doing while working on one. Every update depends on someone remembering. 11:58: 'we can't always know what happened in git; if we can deduce it, we do it; if not, we trigger a request to clarify, because we made an observation. Two levels.'

## Principle
Never guess: deduce only what is certain, and say what it was deduced from. Everything else becomes a question to the owner, never a silent change. Observations come from what hug already reads (fetch, take, lint); no daemon. A question is a field on the ticket, answered in place (3jtfw9), delivered at the agent's next break (x325pp).

## Relevant details
Children: deduce from git (wyvern), observations become questions (hatchling), one ticket in doing per agent (kestrel). Evidence for the ids: 58 of 179 main commits in 2 days end with (id). Related: 0hkaa5 background sync (fetch is where new main commits are seen), x1t1t1 (branch recorded on take), x325pp (nudges at a break), 3jtfw9 (answer in place), 6ephqk (stages, waiting_on, asked_by).

## Depends on

## Acceptance
- [ ] all three children are done
- [ ] a week after they ship, Jørgen has not had to remind an agent to update a ticket (noted with the date)

## Notes

Jørgen 2026-11-17 11:58 (refines the two levels): every observation is told to the owner, not only the unclear ones. Level 1, deduced: hug acts and tells: 'observed: commit a08bb34 naming m1c28r reached main; marked landed. If wrong: hug task note m1c28r "..." / <fix command>'. Level 2, unclear: hug does not act and asks: 'observed: ci/m1c28r merged but 2 of 4 checks unticked: tick them, or say what is left'. Both name the observation, what hug did or did not do, and the fix command.
