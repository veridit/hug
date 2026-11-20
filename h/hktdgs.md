---
id: hktdgs
title: "H/L column marker: a 'no column' step between last and first"
state: todo
type: "feature"
priority: "medium"
assignee: []
labels: []
group: ""
rank: "008"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Moving the column marker with H/L past either end lands on 'no column', as at the start, before wrapping on.

## Purpose
You can get back to no column marked without leaving the view, and you always see where the cycle starts.

## Principle
No column is a real stop in the ring, exactly as the view opens.

## Relevant details
Today (v0.7.90, huginn/internal/tui/columns.go moveColumn): the first H or L marks a column, then H/L wrap from the last column straight to the first; no column comes back only by leaving. Change: the ring is (no column), col1 ... colN, so L from colN and H from col1 land on no column, and the next press goes on. Same in overview, list and board, and in the sort picker's field order if it uses the marker. Jørgen 2026-11-20 12:07Z (request 1): 'if I go left of the leftmost column, I would be no column selected rather than wrap around ... an intermediate step between the last and the first ... no column selected territory where I started.'

## Depends on

## Acceptance
- [ ] ~~Verify Ability to select no column~~ (not relevant: replaced by checks 2-3, crab, 2026-11-20)
- [ ] In overview, list and board, L from the last shown column and H from the first land on no column (marker gone, as at open); the next press goes on to the first or last (key test, fails on main first)
- [ ] Jørgen steps past both ends with H and L and lands on no column

## Requests
- 2026-11-20T13:07 request 1 by jhf:
  When I try to select different columns for sorting, I would like to have a zero state. Like I can, if I go left of the leftmost column, I would be no column selected rather than wrap around. So it should be an intermediate step between the last and the first. So when you wrap around, you step into no column selected territory where I started.

## Notes

file:screen-1307.ans (26090 bytes, sha256:9f2a756e) screen capture from hug tui


## History
- 2026-11-20T13:07 request 1 by jhf
- 2026-11-20T13:07 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:07 request 1 acknowledged by crab
- 2026-11-20T13:10 request 1 in progress by crab: hug task set hktdgs
