---
id: ddpxs8
title: "Release v0.7.86"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzg"
created: 2026-11-20
updated: 2026-11-20
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T11:27"
---

## Goal
v0.7.86 (one-sentence Goals) is published, installs via brew, and Jørgen says whether the shorter Goals read better.

## Purpose
Release-as-group trial (75mxf3): one ticket groups what ships and asks one question per user-facing change.

## Principle
Shipped tickets are children; one ask per user-facing change; test-only fixes are listed, not asked about.

## Relevant details
main 00dfa4cf (CI 44673649756, 8/8), CHANGELOG 2d1cdbd6, merge a09e9d63, release.yml 44675088441 success. Ships 0pr03h (Goal one sentence <= 140 chars; new/set refuse, lint warns; 47 open hug Goals rewritten, old text at the top of Details). Test-only: wacc33 (tmux leak), 6zhyq2 (blink/settled), 4gj7z6 (offline sync test). Docs: h9ymk8 (AGENTS.md env Known rule).

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.86 with links into the new Cellar

## Questions
- 2026-11-20T05:41 question 1 by crab to jhf:
  0pr03h: every open hug task now has a one-sentence Goal (47 rewritten, the old text moved to the top of Details). Open a few in hug tui: does the shorter Goal read better, and is anything you need missing from it?
  withdrawn 2026-11-20T06:06 (doing/asked -> done)
- 2026-11-20T06:06 question 2 by crab to jhf:
  0pr03h: every open hug task now has a one-sentence Goal (47 rewritten, the old text moved to the top of Details). Open a few in hug tui: does the shorter Goal read better, and is anything you need missing from it?
  answered 2026-11-20T11:27 by jhf:
    Yes, they do

## Requests
- 2026-11-20T11:26 request 1 by jhf:
  Well now we have purpose principle relevant details, but I think description should go below the others, right? I'm not sure if we should have description anymore It overlaps, doesn't it? Or it should be just details instead of relevant details or should description be there. I think description is when you have title and description, but now we have title or subject and then we have purpose-principle details. Maybe that's enough.
  resolved 2026-11-20T11:27 by crab:
    Agreed: no Description. What hug tui heads DESCRIPTION is the Goal under the wrong name, so it overlaps. A ticket becomes Title, then Purpose, Principle, Goal, Acceptance and Details, and 'Relevant details' is shortened to 'Details'. Added as check 13 on 8a0ybb, which already reorders the page this way (check 11), with your words in its Details. Old task files keep reading.

## Notes


check 1: release.yml 44675088441 success; brew upgrade: hug v0.7.86, hug/muninn/huginn -> ../Cellar/hug/0.7.86/bin/*
Jørgen 2026-11-20 10:27Z (answer 1): 'Yes, they do' (shorter Goals read better, nothing missing). Request 1 (no Description heading, 'Details') went to 8a0ybb check 13.
Reflection (open, 2026-11-20, crab): I asked the same question twice; check open questions before asking.
## History
- 2026-11-20T05:41 todo -> doing/active (take with p277bp)
- 2026-11-20T05:41 doing/active -> doing/asked (ask)
- 2026-11-20T06:06 doing/asked -> done (done); was asked_by: crab; asked_for: jhf; asked_since: 2026-11-20T05:41; asked_what: 0pr03h: every open hug task now has a one-sentence Goal (47 rewritten, the old text moved to the top of Details). Open a few in hug tui: does the shorter Goal read better, and is anything you need missing from it?; doing_since: 2026-11-20T05:41; doing_with: p277bp
- 2026-11-20T06:06 done -> todo/ready (reopen); was done_at: 2026-11-20T06:06
- 2026-11-20T06:06 todo/ready -> todo/design (ask)
- 2026-11-20T11:26 request 1 by jhf
- 2026-11-20T11:26 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:26 request 1 acknowledged by crab
- 2026-11-20T11:27 answered design (question 2), by jhf, for crab (todo/design -> todo/ready):
  > 0pr03h: every open hug task now has a one-sentence Goal (47 rewritten, the old text moved to the top of Details). Open a few in hug tui: does the shorter Goal read better, and is anything you need missing from it?
  Yes, they do
- 2026-11-20T11:27 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:27 resolved request 1 by crab
- 2026-11-20T11:27 crab read the answer: hug task resolve ddpxs8 1 "Agreed: no Description. What hug tui heads DESCRIPTION is the Goal under the wrong name, so it overlaps. A ticket becom…"
- 2026-11-20T11:27 answer 1 acknowledged by crab
- 2026-11-20T11:27 todo/ready -> doing/active (take with p87ngv)
- 2026-11-20T11:27 doing/active -> done (done); was doing_since: 2026-11-20T11:27; doing_with: p87ngv
