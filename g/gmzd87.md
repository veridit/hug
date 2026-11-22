---
id: gmzd87
title: "Requests reach the board coordinator by design"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzn8"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T11:30"
---

## Goal
A request on any task reaches the board coordinator through hug, even when it is idle, with a visible receipt.

## Purpose
Jørgen's requests sat unseen until the coordinator happened to run a hug command; the coordinator then improvised a poller. Delivery must be designed in, not improvised by an agent.

## Principle
By design, not by doctrine: hug delivers, and the instructions hug writes (setup's AGENTS.md block, MCP tool descriptions, hook messages) describe exactly what the code does, kept in step by a test. Saved is not received: acknowledged only on the coordinator's actual receipt. No per-agent pollers.

## Relevant details
A request a person writes on any task reaches the board coordinator (hug.toml [board] coordinator) through hug itself, even when that session is idle, and its receipt is visible: pending, acknowledged, in progress, answered.

Split from ec6z6j (multi-task requests, which still waits on row marking rxr4z8); this covers delivery of single requests and the shared inbox it builds on. Today: requests surface only as a board cue on the next hug command by any agent (muninn/internal/recall/boardcue.go). Coordinator wake path to investigate: jcode hooks, harness install code (muninn/internal/harness), swarm/DM delivery.

## Depends on

## Acceptance
- [x] A request written while the coordinator session is idle reaches it without any agent-side poller, shown by an end-to-end test or recorded live run (session ids, timestamps).
- [x] Each request shows its receipt state (pending, acknowledged, in progress, answered) in hug task requests and the TUI, and acknowledged is set only by the coordinator's actual receipt (test).
- [x] The instructions hug installs (AGENTS.md block from hug setup, MCP tool descriptions, hook text) describe this delivery flow, and a test fails when the instruction text and the behaviour disagree.
- [x] Jørgen writes a request in hug tui and the coordinator reports receiving it, unprompted.

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 4):
  Write a request on any hug task in hug tui (r): does the coordinator report receiving it without you prompting it?
  answered 2026-11-20T11:30 by jhf:
    It works. The only thing is that they have to register. I am the coordinator, but shouldn't we implicitly do that if there's only one task active or only one agent active? And can we deduce this in the common case so that the first task that interacts is set as active as long as it's active. And if you want to override, you override. Rather than you have to do the setup and you have to pick the task because the agent can change, you restart the harness, something happens. So can we do something sensible by default here? But it works. So this one is closed.

## Notes


-
mizaru 11:58 design (before code; implementing unless crab objects by 12:18Z).
1. WAKE PATH (investigated in ~/Projects/jcode 28ac5e603). jcode hooks are all per-session observers/feedback: none can start a turn in another, idle session. The swarm DM path needs a caller session. What can: the daemon socket request {"type":"notify_session","session_id","message"} (crates/jcode-protocol/src/wire.rs:269), jcode's own schedule-tool delivery. handle_notify_session (jcode-app-core/src/server/client_actions.rs:51): idle live session -> run_live_system_turn_if_idle starts a turn now; busy -> queued soft interrupt injected at the next safe point; not live -> error "Session 'X' is not currently live". No desktop input, no debug socket (that is disabled here). Socket: <JCODE_HOME>/runtime/jcode.sock (the zshrc wrapper sets JCODE_RUNTIME_DIR=$root/.jcode/runtime; global ~/.jcode/runtime/jcode.sock, matches JCODE_SOCKET here).
   So: when hug writes a request (hug task request, MCP task_update request, TUI r box), right after the commit hug itself sends notify_session to the coordinator's newest jcode session (session_<coordinator>_*.json, found in every jcode home harness.DiscoverJcode lists). Push on write, no poller. Target: hug.toml [board] coordinator (ec6z6j: the coordinator inbox, not an arbitrary worker); the assignee keeps today's recall cue. No coordinator set: no push, setup says so.
   Message: 'hug request ID#N from WHO, to you as board coordinator: "TEXT" ... then: hug task resolve ID N "what changed"'.
2. RECEIPT STATE. Requests block format unchanged (old hugs on other clones keep parsing it); state is read from History lines, like Woken:
   pending      = 'request N by WHO' (written by request, today)
   (delivery)   = 'request N delivered to crab (session_crab_...)' or 'request N not delivered: crab has no live jcode session' (written by the hug process that wrote the request, from the daemon's reply). Delivered is not acknowledged.
   acknowledged = 'request N acknowledged by crab' written by muninn's jcode hook IN crab's session (turn_start for an idle wake, post_tool_feedback/turn_end for a queued soft interrupt) when the session's own transcript holds the 'hug request ID#N' message. The ack waits in a small inbox file per session (one stat on the hot path, like park). That is actual receipt: the text is in the coordinator's context.
   in progress  = 'request N in progress by crab: <write>' written by Store on the acknowledger's first write on that task after the ack (like markRead).
   answered     = 'resolved request N by X' (resolve, today).
   Shown in hug task requests (state per row), show --json requests[].state, TUI REQUESTS label 'open' -> the state (one label, no highlighting).
3. INSTRUCTION TEXTS that change: setup's AGENTS.md block 'Requests:' paragraph; MCP task_requests and task_update request descriptions; setup coordinator step detail; the delivered message itself. One Go constant describes the flow and is used by all of them; a test drives a fake daemon end to end and fails if the text names a state, command or route the code does not produce. boardcue.go (duckling) untouched; toolcalls.go untouched.
Open: crab session must be live in a daemon; an offline coordinator gets 'not delivered' plus the existing next/cue paths, and no redelivery loop (would be a poller).
Jørgen 11:59Z: answers to asked questions (hug task answer, TUI a) must be delivered to the coordinator by the same path as requests; he works through many asked tickets in random order and expects each to be acted on as it arrives.
mizaru 13:02Z evidence, branch gmzd87 (2d614461 + follow-up). Check 1 LIVE, real jcode daemon ~/.jcode/runtime/jcode.sock, new hug built from the branch. IDLE path: throwaway board /Users/jhf/.jcode/scratch/gmzd87-idle, coordinator crab, task vmt3h1. Sent 13:00:49.590Z (HUG_USER=jhf hug task request vmt3h1 'gmzd87 idle probe...'); the command returned 13:00:49.875Z with receipt 'pending, delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)'. crab's journal line 69, role user/display_role system, timestamp 13:00:49.787968Z, was its own turn; crab was idle and confirmed it by DM at 13:01. At 13:01:10Z the new hook (post_tool_feedback, crab's session id) wrote 'request 1 acknowledged by crab'. BUSY path: board gmzd87-live, task 1bgbdx, coordinator mizaru. Sent 12:58:47.54Z; it arrived as a soft interrupt in mizaru's running turn; the hook acked at 12:59:02Z; then in progress, then answered. Tests: cli TestRequestReachesCoordinator and TestAnswerReachesCoordinator (fake daemon at the socket: not-live, live, inbox cleanup). Check 2: task TestRequestReceipt and TestAnswerReceipt; cli the same plus requests 'receipt: STATE'; tui TestDeliverFromBoxes (REQUESTS label). Acknowledged is only written by Store.Acknowledge, called only from cli.Acknowledge, only once a user text message holds the token: mutant 'ack whole inbox' fails delivery_instructions_test.go:139, mutant 'byte match' fails :147. Check 3: cli TestDeliveryInstructionsMatchCode (16 claims, each phrase of cli.DeliveryFlow driven; unclaimed words fail), mcp TestDeliveryFlowInMCP; agentSection embeds DeliveryFlow. Mutants: text without claim -> ':194 DeliveryFlow says "hug" with no check'; claim without text -> ':183'; AGENTS.md without flow -> ':25'; answers not delivered -> deliver_test.go:246; no in-progress -> deliver_test.go:202.
crab 13:01Z: idle path confirmed live. mizaru sent a probe request (throwaway board, task vmt3h1) to crab while idle; crab's turn started 13:00:49.787Z, 197 ms after hug sent it (13:00:49.590Z), no poller. At 13:01:10Z the new hook against crab's real session wrote 'request 1 acknowledged by crab'. Busy path confirmed earlier (12:58:47Z, soft interrupt). Message now names the board (repo name and path).
CI green at head 56cbe503: run 44315165470 success, 6/6 jobs (secrets, test-windows-cli, test ubuntu/macos/windows, bench). Earlier runs 44310114524 (813a26d9, macOS) and 44312103384 (30c507a0, test-windows-cli) failed only on the tests' expected board path (/private/var on macOS, forward slashes on Windows); the tests now take the path from git rev-parse --show-toplevel as hug does (30c507a0, 56cbe503), and the fake daemon's hook no longer calls t.Errorf after the test ends. Local GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./... -count=1 at 56cbe503: 34 ok, 0 FAIL. Report: /Users/jhf/.jcode/scratch/gmzd87/REPORT.md
crab review 13:57Z: accepted. Read: harness/deliver.go (notify over the jcode daemon's unix socket only, DialTimeout plus deadline, inbox entry before notify, removed if nobody took it), receipt.go, setup/MCP instruction text from one constant guarded by TestDeliveryInstructionsMatchCode. CI 44315165470 on 56cbe503 6/6. Rebase onto main 832167b8 conflicts in huginn/internal/cli/hooks.go with c87da5a8 (xfv9pw prompt history wiring): sent back to mizaru to resolve. Rulings on mizaru's open questions: (1) release first, then Jørgen's check 4; (2) duplicate cue: boardcue.go should skip a request already delivered, follow-up; (3) coordinator offline: yes, the coordinator's next session start delivers pending items, by design, follow-up; (4) answered = the coordinator may close a relayed answer with an explicit command, follow-up; (5) other machines: not delivered with reason is right for now.
Landed on main 4e70099c (CI 44321337759, exact SHA, 6/6; code identical to mizaru's 19a5fc77). Ships in the next release; check 4 (Jørgen writes a request in hug tui) after that.
landed 872a2a4 on main 2026-11-18: Requests and answers reach the board coordinator by design (gmzd87)
shipped in v0.7.64 (872a2a4)
check 4: Jørgen 2026-11-20 10:30Z: 'It works ... So this one is closed.' His default-coordinator idea is filed as a new task.
Reflection (dismissed, 2026-11-20, crab): Nothing misled me.
## History
- 2026-11-18T12:34 todo -> doing/active (take)
- 2026-11-18T15:11 doing/active -> doing/waiting (wait)
- 2026-11-19T04:22 handed to crab: mizaru's session ended (closed 2026-11-19 03:11)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: mizaru; waiting_on: crab lands 4e70099c (ci/gmzd87-rebased), then release, then Jørgen's check 4 in hug tui; waiting_since: 2026-11-18T15:11
- 2026-11-20T11:30 answered asked (check 4), by jhf, for crab (doing/asked -> doing/active):
  > Write a request on any hug task in hug tui (r): does the coordinator report receiving it without you prompting it?
  It works. The only thing is that they have to register. I am the coordinator, but shouldn't we implicitly do that if there's only one task active or only one agent active? And can we deduce this in the common case so that the first task that interacts is set as active as long as it's active. And if you want to override, you override. Rather than you have to do the setup and you have to pick the task because the agent can change, you restart the harness, something happens. So can we do something sensible by default here? But it works. So this one is closed.
- 2026-11-20T11:30 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:30 answer 1 acknowledged by crab
- 2026-11-20T11:30 crab read the answer: hug task check gmzd87 4 "Jørgen 2026-11-20 10:30Z: 'It works ... So this one is closed.' His default-coordinator idea is filed as a new task."
- 2026-11-20T11:30 doing/active -> done (done); was doing_since: 2026-11-18T12:34
