# gmzd87: Requests reach the board coordinator by design (REPORT, mizaru)

**Verdict.** hug now delivers every request and answer a person writes to the board coordinator's live jcode session. An idle session starts a turn, and a busy one gets the message as a soft interrupt. Each delivery leaves a receipt in History. A request is only marked acknowledged by hug's hook inside the coordinator's own session, once the message is in that session's transcript. Both paths ran live on the real daemon, idle into crab and busy into mizaru. The instruction text is one constant, embedded in AGENTS.md and the MCP descriptions, and a test drives each of its claims. Branch `gmzd87`, not merged, no tag.

## Design

**Wake path (investigated, ~/Projects/jcode 28ac5e603).** None of jcode's hooks can start a turn in a different, idle session: all of them are per-session observers or feedback hooks. The swarm DM path needs a caller session. What can do it is the daemon socket request `{"type":"notify_session","id","session_id","message"}` (jcode-protocol `wire.rs:269`), which jcode's own scheduler uses. `handle_notify_session` (jcode-app-core `server/client_actions.rs:51`) behaves three ways:
- idle and live: `run_live_system_turn_if_idle` starts a turn.
- busy: `queue_soft_interrupt_for_session`.
- not live: `error "Session 'S' is not currently live"`.

The reply is `ack` and then `done` or `error`, which I confirmed live with a probe. jcode appends the message before `turn_start` fires (`agent/turn_execution.rs:193-210`). No desktop input and no debug socket are involved.

**Flow (one function: `cli.App.Deliver`).** Every write path calls it after its commit: `hug task request`, `hug task answer`, MCP `task_update` request/answer (these go through `Execute`), and the TUI's `r` and `a` boxes (via `tui.Options.Deliver`). It reads `hug.toml [board] coordinator`, then:
1. finds that name's newest sessions (`session_<name>_*.json`) in every discovered jcode home;
2. writes an inbox entry `<JCODE_HOME>/hug-inbox/<session>/<task>-<kind>-<n>.json` *before* notifying, because an idle session's turn starts before Notify returns;
3. sends `notify_session` to `<home>/runtime/jcode.sock`, then JCODE_SOCKET, JCODE_RUNTIME_DIR, XDG_RUNTIME_DIR and TMPDIR;
4. removes the inbox entry if no session took the message;
5. records the daemon's answer on the task and commits it.

**Message (crab's condition 3).** It carries who wrote it, the task id and title, the board name and path, the text verbatim, and for an answer the check, the question and who asked. It ends with the exact closing command and a token:
```
hug: jhf wrote request 1 on vmt3h1 (gmzd87 idle probe), to you as the board coordinator:
  gmzd87 idle probe: not a real request, reply received
Turn it into changes to the ticket, then: hug task resolve vmt3h1 1 "what changed"
If it is unclear: hug task ask vmt3h1 --to jhf "question"
[hug vmt3h1 request 1 63df5805]
```
That is the live text crab received at 13:00:49Z. Since then (813a26d9), on crab's note, it also says `on board <repo> (<path>)` and `then (in <path>): hug task resolve ...`.

**Receipt state (in History; the Requests block format is unchanged, so older hugs still parse it).** Each line is written by the side that saw the step:

| state | History line | written by |
|---|---|---|
| pending (not sent / delivered / not delivered) | `request N delivered to crab (SESSION)` / `request N not delivered: REASON` / `... not delivered to crab: REASON` | the hug that wrote the item, from the daemon's reply (`Store.Delivered`) |
| acknowledged | `request N acknowledged by crab` | hug's jcode hook in crab's session (turn_start and post_tool_feedback, `muninn/run` → `huginn.Acknowledge` → `cli.Acknowledge`). Written only when a **user text message** in crab's journal (or in a snapshot rewritten after delivery) holds the token |
| in progress | `request N in progress by crab: <the write>` | `Store.finish` → `markProgress`, on the acknowledger's first write on the task |
| answered | resolve (`resolved request N by X`); for an answer, the asker's read line (41hhgn) | existing writes |

"answer N" is the Nth answer block in History. `Task.Requests[].Receipt`, `Task.Answers[]` and `Task.AnswerCount` are parsed from History (cache schema 23). A coordinator writing its own request gets nothing delivered.

**Not delivered (crab's condition 1).** No coordinator, no session, no socket, a dead session, or a non-jcode harness all record `pending, not delivered: REASON`. That shows in `hug task requests`, show/MCP `request.receipt` and the TUI REQUESTS label. The request still comes first in `hug task next`. Nothing retries or polls.

**Scope kept.** I did not touch boardcue.go, toolcalls.go, sort.go or gitbranch. The TUI change is one label: REQUESTS `open` became the receipt label, with no highlighting.

## Commits (branch gmzd87, on origin/main 4e776695)
- `2d614461` Requests and answers reach the board coordinator by design (gmzd87)
- `813a26d9` gmzd87: receipt only from a message to the session (not a tool call or result quoting the token); the delivered message names its board

Both carry `Model: claude-opus-5-5/high`. CI-only branch `ci/gmzd87` is the same commit.

## Failing-first (new tests copied onto origin/main 4e776695, verbatim excerpt)
```
huginn/internal/task/receipt_test.go:18:18: undefined: Receipt
huginn/internal/task/receipt_test.go:21:25: tk.Requests[0].Receipt undefined (type Request has no field or method Receipt)
huginn/internal/task/receipt_test.go:27:15: s.Delivered undefined (type *Store has no field or method Delivered)
huginn/internal/task/receipt_test.go:48:15: s.Acknowledge undefined (type *Store has no field or method Acknowledge)
FAIL	github.com/veridit/hug/huginn/internal/task [build failed]
huginn/internal/tui/deliver_test.go:22:9: m.opts.Deliver undefined (type Options has no field or method Deliver)
huginn/internal/mcp/request_test.go:138:45: undefined: cli.DeliveryFlow
huginn/internal/cli/deliver_test.go:146:15: undefined: Acknowledge
huginn/internal/cli/deliver_test.go:153:4: a.JcodeHomes undefined (type *App has no field or method JcodeHomes)
```
Behaviour mutants, each failing a test (verbatim):
- answers not delivered → `deliver_test.go:246: receipt: <nil>` and `delivery_instructions_test.go:115: answer receipt <nil>`
- claim with no text → `delivery_instructions_test.go:183: DeliveryFlow no longer says "stays \"pending, not delivered: REASON\" xx": change the claim with the text`
- text with no claim (added "hug retries every minute.") → `delivery_instructions_test.go:194: DeliveryFlow says "hug" with no check for it here; ...`
- ack the whole inbox without checking the transcript → `delivery_instructions_test.go:139: acknowledged before the message reached the transcript: [f8rcfp request 99]`
- 2d614461's byte matcher (a tool call or result quoting the token) → `delivery_instructions_test.go:147: a tool call or result quoting the token counted as receipt: [e2wn96 request 2] <nil>`
- no in-progress mark → `deliver_test.go:202: requests after crab's write:`
- AGENTS.md without the flow → `delivery_instructions_test.go:25: the AGENTS.md section hug setup writes does not carry DeliveryFlow verbatim:`

## Suite (local, `GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./...` at 813a26d9)
34 packages ok, 0 FAIL. Lines for the touched packages:
```
ok  	github.com/veridit/hug/huginn/internal/cli	159.541s
ok  	github.com/veridit/hug/huginn/internal/harness	(cached)
ok  	github.com/veridit/hug/huginn/internal/mcp	7.227s
ok  	github.com/veridit/hug/huginn/internal/task	(cached)
ok  	github.com/veridit/hug/huginn/internal/tui	(cached)
ok  	github.com/veridit/hug/muninn/run	4.628s
```
The first full run, before 813a26d9, failed `todos_test.go:58: agentSection names harness word "jcode"`. The AGENTS.md text must stay harness-neutral, so DeliveryFlow now says "live session" and "a harness hug cannot reach yet such as Claude Code or Codex".

## CI
Run **44310114524** (https://github.com/veridit/hug/actions/runs/44310114524) on 813a26d9: CI_RESULT_PENDING

## End-to-end delivery evidence (real jcode daemon `~/.jcode/runtime/jcode.sock`, branch-built hug)
**Idle path, into crab** (board `/Users/jhf/.jcode/scratch/gmzd87-idle`, task vmt3h1, crab's approval 13:00:17Z):
- `swarm list` at 13:00:41Z: `crab [coordinator] ... Status: ready · idle 20s`
- sent 13:00:49.590720Z with `HUG_USER=jhf hug task request vmt3h1 "gmzd87 idle probe: ..."`; returned 13:00:49.875302Z with `receipt: pending, delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)`
- crab's journal line 69 has `role user display_role system ts 2026-11-18T13:00:49.787968Z id message_1795006849787_14095820721636381869` and the message text verbatim. It arrived on the **idle path as its own turn**; crab's DM: "The probe arrived as its own turn at 2026-11-18T13:00:49.787Z (I was idle: no tool running, last turn ended ~13:00:17Z)".
- 13:01:10.319Z: the new hook (`hug mem hook jcode`, `JCODE_HOOK_EVENT=post_tool_feedback`, crab's session id) printed `hug: you received vmt3h1 request 1; it now shows as acknowledged (hug task requests)`. History then read:
```
- 2026-11-18T14:00 request 1 by jhf
- 2026-11-18T14:00 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T14:01 request 1 acknowledged by crab
```
  The acknowledgement ran by hand because crab's installed hook is v0.7.61. After release, that hook runs on crab's own turn.

**Busy path, into mizaru (soft interrupt)** (board `gmzd87-live`, task 1bgbdx, coordinator mizaru): sent 12:58:47.54Z. It arrived inside my running turn as a system message at 12:58:47.96Z, and the hook acked it at 12:59:02Z. History:
```
- 2026-11-18T13:58 request 1 delivered to mizaru (session_mizaru_1795001661330_4df3a1a1fec30343)
- 2026-11-18T13:59 request 1 acknowledged by mizaru
- 2026-11-18T13:59 request 1 in progress by mizaru: hug task note 1bgbdx "mizaru live: received"
- 2026-11-18T13:59 resolved request 1 by mizaru
```
`hug task requests` showed `receipt: pending, delivered to mizaru`, then `acknowledged by mizaru`, then `in progress by mizaru`, then `answered`.

## Instruction texts changed
**AGENTS.md section (`agentSection`, hug setup).** It previously ended its Requests paragraph with "...then hug task resolve ID N "what changed"; if it is unclear, hug task ask instead." That paragraph is unchanged, and this line follows it (`cli.DeliveryFlow`):
> Requests and answers reach the board coordinator (hug task coordinator NAME) by hug itself: when a person writes hug task request or hug task answer (or r and a in hug tui), hug hands it at once to the coordinator's live session (an idle session starts a turn, a busy one gets it at its next step) and History says delivered or not delivered with the reason. The coordinator's session acknowledges it on receipt (hug's hook), its first write on the task marks it in progress, and hug task resolve ID N answers a request (an answer: the asker's next write). hug task requests shows each state: pending, acknowledged, in progress, answered. Not delivered (no coordinator set, no live session, a harness hug cannot reach yet such as Claude Code or Codex) stays "pending, not delivered: REASON" and the request still comes first in hug task next. No agent polls.

**MCP `task_requests`.** Before: "... the text verbatim and the fix that resolves it. ... then task_update action resolve." After: "... the text verbatim, its receipt (request.receipt.state) and the fix that resolves it. ... then task_update action resolve. " followed by DeliveryFlow.

**MCP `task_update`.** Appended after "using only the selected action's fields.": "A request or answer reply says its receipt (receipt). " followed by DeliveryFlow.

**Setup step `coordinator`.** Before: "hug.toml [board] coordinator is NAME: a person's request on a task nobody holds goes to them". With none set, it was "no coordinator set: a person's request on a task nobody holds goes to every session on the board; ...". After: "hug.toml [board] coordinator is NAME: hug hands every request and answer a person writes to NAME's live jcode session (gmzd87)". With none set: "no coordinator set: a person's requests and answers are not delivered to any session (they stay pending, not delivered: no coordinator set); name the session that coordinates the agents".

**Hook text (new).** In the coordinator's tool feedback on acknowledgement: `hug: you received ID request N; it now shows as acknowledged (hug task requests)`.

## Open questions
1. **Release dependency.** Acknowledgement runs in the coordinator's installed hook, so live acks need a release, and crab and Jørgen need `brew upgrade`. Delivery only needs the writer's hug to be new; Jørgen's TUI hug must be the new one.
2. **Recall cue overlap.** boardcue.go (duckling's) still sends the old "Task board: jhf asks on ..." recall line to the assignee or coordinator. With delivery in place, the coordinator may hear a request twice: once delivered and once as a cue. I left it alone because it is out of scope. Should the cue skip items that are already acknowledged? That needs a DM to duckling or crab.
3. **Offline coordinator.** By design nothing redelivers, since that would be a poller. A request written while crab has no live session stays "pending, not delivered" until someone acts on `hug task next` or the cue. Should crab's next `session_start` (a hook, not a poll) pick up its undelivered items? I propose it as a follow-up.
4. **Answer "answered" rule.** An answer counts as answered on the asker's next write (41hhgn's read line), even when the coordinator relays it. If the coordinator should be able to close an answer itself, that needs a ruling.
5. **Sessions on other machines** are not reachable; that case reads "no jcode session named X on this machine".
