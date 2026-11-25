---
id: hktdgs
title: "H/L column marker: a 'no column' step between last and first"
state: "doing"
type: "feature"
priority: "medium"
assignee: ["crab"]
labels: []
group: ""
rank: "008"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-21T23:56"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "ta6fy6"
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
- [x] In overview, list and board, L from the last shown column and H from the first land on no column (marker gone, as at open); the next press goes on to the first or last (key test, fails on main first)
- [x] Jørgen steps past both ends with H and L and lands on no column
- [ ] Column headings stay shown in the active section with no column marked; marking one only adds its marker (golden at 100x30, fails on v0.7.112 first)

## Requests
- 2026-11-20T13:07 request 1 by jhf:
  When I try to select different columns for sorting, I would like to have a zero state. Like I can, if I go left of the leftmost column, I would be no column selected rather than wrap around. So it should be an intermediate step between the last and the first. So when you wrap around, you step into no column selected territory where I started.
  resolved 2026-11-20T13:10 by crab:
    Turned into a task: H/L get a 'no column' stop between the last and the first column, exactly as the view opens, instead of wrapping straight round. Check 2 is a key test in overview, list and board that must fail on today's code first; check 3 is your try after release. Queued in the TUI lane.

## Questions
- 2026-11-22T00:23 question 1 by crab to jhf (check 3):
  Jørgen steps past both ends with H and L and lands on no column
  answered 2026-11-25T10:22 by jhf:
    This works as advertised except I would like to see the columns despite not having one being selected because sometimes because that is for clarity so in the current active section I can always see the columns and when I select the column I have the other markers that are also there but I don't think we should hide the columns at any point

## Notes

file:screen-1307.ans (26090 bytes, sha256:9f2a756e) screen capture from hug tui


file:hktdgs-board-no-column-100x30.ans (7405 bytes, sha256:2b367130) Board at 100x30 (built binary at 1d864738, tmux capture-pane -e, seeded fixture): after 6 L from opening (id title checks type priority, then past priority) the column marker is gone, rows 1-27 byte-identical with ANSI to the board as it opened, footer 'H/L pick a column, then o orders by it'. One more L marks id again.
Branch hktdgs @ 1d864738 (test e7000635 then fix). Failing first on main 8611f2b6: 'hktdgs_test.go:51: overview: L from the last column (assignee) marks id, want no column' (list: '(priority) marks id', board: '(priority) marks id'). Fix: huginn/internal/tui/columns.go moveColumn steps at+=delta; past either end sets m.column=-1. Every marker user goes through markedColumn(): footer hint, o, the s picker's c (currentColumn: dim 'mark one with H/L first' on no column, as at open), the header tint and the row > (cursorColumn), and the saved view state (reload.go saves no column when m.column<0). No other code keeps its own marker. Mutants 7/7 killed (M1 modulo wrap, M2-M4 the bound, M5 the -1, M6/M7 entry from none), columns.go sha restored. Goldens: only the help row 'H / L  mark the previous / next column, none past either end' (5 files). Old tests that pinned the wrap updated: TestTagsColumnCursor, TestMarkerSkipsSharedColumns. Local: go vet and go test ./... clean.
check 2: CI run 44960662438 on 1d864738 (branch ci/hktdgs): 8/8 jobs success (secrets, bench, test ubuntu/macos/windows-latest, test-windows-cli rest/sync/shards). TestColumnMarkerHasANoColumnStop/{overview,list,board} failed on main 8611f2b6 ('hktdgs_test.go:51: overview: L from the last column (assignee) marks id, want no column') and passes on the fix; 7/7 mutants killed; frame hktdgs-board-no-column-100x30.ans attached.
landed 1d86473 on main 2026-11-22: tui: H/L step onto no column past either end, the marker gone as when the view…
shipped in v0.7.99 (1d86473)
check 3: Jørgen 2026-11-25: 'This works as advertised' (H/L land on no column past either end)
## History
- 2026-11-20T13:07 request 1 by jhf
- 2026-11-20T13:07 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:07 request 1 acknowledged by crab
- 2026-11-20T13:10 request 1 in progress by crab: hug task set hktdgs
- 2026-11-20T13:10 resolved request 1 by crab
- 2026-11-21T23:56 todo -> doing/active (take)
- 2026-11-22T00:23 doing/active -> doing/asked (ask)
- 2026-11-22T01:53 handed to crab: lobster's session ended (closed 2026-11-22 00:22)
- 2026-11-25T10:22 answered asked (check 3), by jhf, for crab (doing/asked -> doing/active):
  > Jørgen steps past both ends with H and L and lands on no column
  This works as advertised except I would like to see the columns despite not having one being selected because sometimes because that is for clarity so in the current active section I can always see the columns and when I select the column I have the other markers that are also there but I don't think we should hide the columns at any point
- 2026-11-25T10:22 answer 1 delivered to crab (session_crab_1795558631109_253787e0f5d2bbbb)
- 2026-11-25T13:02 crab read the answer: hug task check hktdgs 3 "Jørgen 2026-11-25: 'This works as advertised' (H/L land on no column past either end)"
