---
id: 805p6y
title: "Checks show what is being worked on now, with a subtle line in the TUI"
status: todo
type: feature
priority: ""
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
A ticket shows the work in progress, not only what is ticked: one acceptance check can be marked as current with a short now-text (for example 'waiting for CI run 44049348743'), and the TUI shows that line, dimmed, under the IN PROGRESS row and on the check in the detail view.

## Purpose
Jørgen 2026-11-17: tickets read like 'I did, I did, I did'; he wants to see what is happening now, such as waiting for CI as part of an acceptance check. The coordinator's own work this morning was invisible on the board: the landings waited on CI and on a worker fixing SQLITE_BUSY, and only the harness todo list showed it. This makes the board, not the harness, the place to see current work (see 5dxzxe, tzc9hb).

## Principle
Subtle: one dim line, only on tasks in progress, never more than one line per task. The task file stays the truth and plain markdown: the now-text is an indented line under the check, so other renderers show it as text. One current check per task. Ticking the check or writing a new now-text replaces the old one. No history kept beyond the ticket's Notes.

## Relevant details
Proposed command: hug task now ID N "text" (MCP task_update action now); hug task check ID N clears it. File form under the check: '  now: text (agent, time)'. TUI: IN PROGRESS row gets a dim second line '↳ 3. CI green on all three systems: waiting for run 44049348743 · 4 min'; the detail view marks the check ◐. Later, with 16dxsr, a now-text naming a condition (CI run, branch merged) can be checked by hug and nudge when it ends. Coordinate with 102bc0 (rows fit one line) and zhvkmj (status colors).

## Depends on

## Acceptance
- [ ] hug task now ID N "text" writes the now-line under check N; hug task check ID N removes it (test)
- [ ] the IN PROGRESS row shows one dim line with the check number, its text and the now-text, cut to the terminal width (golden)
- [ ] the detail view marks the current check and shows its now-text (golden)
- [ ] MCP task_update supports action now (test)
- [ ] Jørgen confirms the look on his board after release

## Notes

Jørgen 2026-11-17 09:00, on the IN PROGRESS section: the board should know what each in-progress task is focused on right now (e.g. 'waiting for CI'), shown on the row, and Enter opens the detail for more. Today the coordinator wrote it as Notes on 44w1ev, m1c28r, t709he, 0hkaa5 and x1t1t1 by hand; this ticket makes it a first-class now-line. Priority: right after zhvkmj.

Jørgen 2026-11-17 09:01 correction: the current-work line is NOT an acceptance check. Acceptance says what must be true for the task to be done; this line is the agent managing itself: what it is doing or waiting on right now. So it is its own field on the task, not tied to check N. Name pending Jørgen's choice (crab proposed: focus). Rework Goal and Acceptance before building.

Jørgen 2026-11-17 10:03: inside IN PROGRESS, some tasks are really waiting on something outside the agent (a CI run, a review, a reply), and the agent can do other work meanwhile. Proposal (crab): two fields on a doing task, not acceptance checks. stage = design | plan | build | verify | review (what kind of work); waiting = what it waits on, e.g. 'CI run 44057678727' (empty when the agent is working). IN PROGRESS shows waiting rows dimmed with the wait text; NEXT for an agent prefers its non-waiting tasks. Word 'waiting', not 'pending' (jcode todos use pending for not started) and not 'blocked' (hug uses blocked for unmet dependencies). With 16dxsr a waiting text that names a condition is checked by hug and ends itself. Awaiting Jørgen's choice of words.

Jørgen 2026-11-17 10:04: 'I think it's called status'. So waiting is a status value, not a separate field: todo, doing, waiting, blocked, done, dropped. waiting = the agent's own task, work done for now, waiting on something outside it, with a reason text. blocked stays for unmet dependencies. stage (design/plan/build/verify/review) is still open.

Jørgen 2026-11-17 10:04: statuses stay a fixed set chosen by design (no custom statuses); he adds waiting to that fixed set because we need it. Fixed set: todo, doing, waiting, blocked, done, dropped. Build: status waiting with a required reason (hug task wait ID "reason"; MCP task_update action wait), doing again on take or when a 16dxsr condition clears; lint and hug upgrade know it; zhvkmj gives it a color; next skips an agent's waiting tasks. Stage still open.

Jørgen 2026-11-17 10:07: a hard-coded tree of states with substates, and each substate tailors the view (the dim line under the row, the detail). Crab's draft: todo{draft, design, ready, blocked}, doing{active, waiting}, done, dropped. Leaf names are unique, so the file stores one word (status: waiting) and the parent is implied. draft, ready and blocked are derived by hug (lint completeness, dependencies); design, active and waiting are set by the agent; waiting and design carry a reason. Second line per substate: draft = what is missing; design = first open question; ready = none; blocked = the unmet dependency; active = the first unticked check; waiting = reason and how long; done = release or date; dropped = reason. Supersedes the separate stage field. Awaiting Jørgen's confirmation of the tree.

Jørgen 2026-11-17 10:07 (rule): the root state picks the main display (which section, the row's color and order); the substate fine-tunes it (the second line, the marker, what the detail shows first). Build to that rule. Open for Jørgen: blocked under todo or its own root; review as a substate or waiting-with-reason.

Jørgen 2026-11-17 10:07: each substate must say where its extra data is written. Crab's rule: the status field holds the leaf; data that hug derives is never written (blocked reads Depends on, draft reads lint, active reads the first unticked check); data only the agent knows goes in ONE front-matter field, status_reason (waiting: what it waits on; design: the open question; dropped: why), plus status_since (date-time set on every status change, for '· 12 min'). Changing status replaces both; the old pair goes to History. Example: status: "waiting" / status_reason: "CI run 44057678727" / status_since: 2026-11-17T09:41. Lint: waiting, design and dropped without status_reason is a finding.

Jørgen 2026-11-17 10:09 (ruling): two fields written in the task file, state (the root) and stage (the substate), because the tree is then written down and easy to read. Replaces status; hug upgrade migrates (no backward compatibility). Draft: state todo -> stage draft|design|ready|blocked; state doing -> active|waiting; state done; state dropped. stage_reason and stage_since carry the agent-only data (waiting, design, dropped). Stages hug can derive (draft, ready, blocked) are written by hug itself on every write and checked by lint, never by hand. Open: blocked under todo or its own state; review as a stage.

Jørgen 2026-11-17 10:09-10:10 (ruling): write state, and stage within the state; the tree is hard-coded because every node has semantic meaning that hug acts on. Extra fields belong to a node and are named after it: <state>_<name> or <stage>_<name> (doing_since, waiting_on, waiting_since, design_question, dropped_reason, done_at). A node's fields exist only while the task is in that node. Full draft: tmp/lanes/805p6y-states.md. Open: blocked under todo or own state; review as a stage.

Jørgen 2026-11-17 10:10 (why it is hard-coded): every node carries semantics hug acts on (which section, what next picks, what lint checks, what ends a wait). A configurable tree would still need this hard-coded tree underneath, with user nodes mapped onto it; that is two trees for one meaning. So the tree is fixed in code and changed only by design, with hug upgrade migrating the files.

Jørgen 2026-11-17 10:12 (ruling): blocked is a stage of both todo (nobody has it) and doing (someone has it). Same meaning in both; the state decides the section. Draft updated: tmp/lanes/805p6y-states.md.

Jørgen 2026-11-17 10:13 (ruling): review is a stage of both todo (work done, waits for a reviewer; nobody reviewing) and doing (someone is reviewing it now). Same pattern as blocked: the stage says what, the state says whether someone has it. review_by names the reviewer. Draft updated.

Superseded by 6ephqk (Task state and stage: a hard-coded tree replaces status), which carries Jørgen's rulings of 2026-11-17 10:03-10:14.
