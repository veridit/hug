---
id: ec6z6j
title: "Send one request for a selection of tasks"
state: todo
type: feature
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzx8"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
Let Jorgen mark multiple tasks on the board and give one instruction applying to that exact selection.

## Purpose
He wants to select tasks with the same structural issue and ask an agent to fix them without leaving the board or repeating a request for each task.

## Principle
Selection is distinct from keyboard focus and acceptance checks. Preserve exact target IDs and instruction. Do not report a request handled until its selected scope is accounted for.

## Relevant details
User feedback 2026-11-18: marking task rxr4z8 is unfinished; single-task requests hbgybn work. Need a batch request over an arbitrary marked set, not the group field. Storage, agent delivery and partial-resolution presentation need design before implementation.

## Depends on
- rxr4z8

## Acceptance
- [ ] A board mock shows marking several tasks, selected count, and one request action for that exact set, reviewed by Jorgen.
- [x] Submitting one request preserves its exact text and selected IDs, visible to the agent without asking the user to repeat it per task.
- [x] User can see pending versus handled targets without a partial fix silently closing the whole request.

## Notes

APPROVED by Jorgen 2026-11-18 11:08Z: Separate request, delivery, and work tasks. One request preserves the instruction and exact selected task IDs; each selected task links to it rather than owning an independent duplicate. Route to the board coordinator inbox, not an arbitrary worker. Visible lifecycle: pending -> acknowledged -> in progress -> answered. Saved is not received, acknowledgement requires actual coordinator receipt. Sending feedback does not automatically create another work task. Coordinator may act on existing tasks or propose a new coordinating task when warranted. Bulk partial completion must remain visible, not close unhandled scope. This is approved design, not implemented delivery.

Jørgen 2026-11-18 11:21Z, requirement: request delivery to the coordinator is BY DESIGN, not by an agent's own improvisation. Crab had started a background poller (hug task requests every 60s) to notice requests; Jørgen rejected that: the system must deliver it, and the instructions hug gives agents (AGENTS.md block written by hug setup, MCP/tool descriptions, hook messages) must say how requests reach the coordinator, consistent with what the code does. As other tickets resolve, those instructions must keep matching. Implications: (1) hug itself pushes a new request to the board coordinator ([board] coordinator in hug.toml) via the harness hooks, including when that session is idle, not only piggybacking on whichever agent next runs a hug command; (2) receipt is recorded (acknowledged) only when the coordinator actually got it; (3) the setup-written instructions describe this flow, with a test that the instruction text and the behaviour agree. Poller cancelled.

Jørgen 11:59Z, working style that delivery must serve: he answers questions inside tickets, in any order, with quiet focus there, and expects the coordinator to receive each answer/request and act as it comes in; chat stays for conversation. So answers (hug task answer) must reach the coordinator exactly like requests do, not only requests.

mizaru takes the non-visual part (crab 15:48): hug task request with several IDs, one request stored on the first task with links on the others, delivered once, MCP. TUI marking waits on rxr4z8, so the dependency stays. Branch ec6z6j on c84d3273, worktree /Users/jhf/.jcode/scratch/ec6z6j.

Non-visual part at 7df7c120 (branch ec6z6j on c84d3273), design A with crab's rulings 15:48/15:50. hug task request ID1,ID2 "text" / --task ID (repeatable) / MCP task_update request ids: text written once on the home ('- STAMP request N by jhf (for a, b, c):'), a link on each other ('- STAMP request M by jhf (see a request N):', no copy), one commit 'hug task request a,b,c', delivered ONCE via gmzd87 Deliver with one resolve line per target. Targets resolve in any order, the home too; home History gets 'request N target b resolved by X' and, after the last, 'request N closed: all targets resolved'; receipt answered only then. show/requests print 'targets: a resolved, b pending, c pending'; the home stays listed until closed; a link shows the home's text. Unknown id: exit 2 'no task zzzzzz: nothing was written', fix = request on the known ids, no commit. Failing first: task manyrequest_test.go:33 s.RequestFor undefined; cli manyrequest_test.go:29 unknown flag --task, :94 not-found; mcp request_test.go:217 unknown argument ids (/Users/jhf/.jcode/scratch/ec6z6j-ff/*-ff.txt). Mutants (mutants.txt), all fail, files restored identical: close on first resolve -> manyrequest_test.go:144/:188; unknown written -> :95; link resolve not recorded -> :162; home drops off requests -> cli manyrequest_test.go:68. Suite niced -count=1: build-ok, vet-ok, 37 ok, 0 FAIL. CI 44344411767. Check 1 (board mock) and the TUI marking wait on rxr4z8.

crab review 17:29 local: non-visual part accepted. 7df7c120 on c84d3273: request a,b / --task / MCP ids; home holds text + targets, links point to it; one commit, one delivery with a resolve line per target; any-order resolves, 'request N closed: all targets resolved' after the last; unknown id writes nothing. 4 mutants. CI 44344411767 7/7. TUI marking waits on rxr4z8.

crab 18:35 local: stacked as c7ccbca7. mizaru's rebase fix: deliverPendingItem now carries targets (bbxdd7 pending path), TestPendingRequestForSeveralTasks + mutant. CI pending.
