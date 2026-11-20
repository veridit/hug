---
id: hbgybn
title: "Requests: a person writes a request in a task, an agent turns it into edits"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzn"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T11:29"
done_release: "v0.7.55"
---

## Goal
A person writes a request on a task in their own words, and an agent turns it into ticket edits and closes it.

## Purpose
Jørgen 23:52 local: 'when I look at the tasks I sometimes would like to have you fix something in the task but if I have to describe it to you it's a lot of work... just go into the task and write like a comment... that becomes something you read and act on... I will write something shorter or use speech to text and then you will transform it into a systematic change in that ticket.' 23:52: 'it's not you asking me, it's me making an ask of you.' The reverse of 41hhgn's asked stage.

## Principle
The person writes, the agent does the systematic work. The request text is kept verbatim, never rewritten; the change it caused is shown next to it. A request is never lost or silently dropped: it stays open until an agent closes it with what it did, or asks back (41hhgn) when unclear. Nothing writes silently. Concurrent writes wait.

## Relevant details
In a task's detail, R opens a multi-line box where the viewer writes a request to the agents in their own words (short, dictated, rough). hug stores it on the task as an open request: who, when, the text verbatim. The task's owner, or whoever works the board, is told at once (recall hook and the write-command footer), reads it, turns it into systematic changes to the ticket (goal, checks, details, notes, state), and closes the request with a one-line account of what changed. The person sees open requests on the row ('req 1') and the account under the request in the detail.

Not the asked stage: that blocks the task on a person; a request is an open item for agents on any task in any state, and does not change its stage. Storage proposal (kestrel to confirm): a '## Requests' section, each request a block 'request by jhf 2026-11-17 23:52:' + indented text, closed by 'done by nautilus 23:58: set check 9's wording, added check 10'. CLI: hug task request ID - (stdin) | "text"; hug task requests (open ones, all tasks); hug task resolve ID N "what changed". MCP: task_update action request / resolve; task_list filter requests. Agent notice: boardcue gets kind 'request' (one line: 'Task board: jhf asks on 6ephqk: "..." Act on it, then: hug task resolve 6ephqk 1 "what changed"'), routed to the task's assignee, else to the coordinator. TUI: R in a detail (and on a list row); the overview shows a REQUESTS count in the header and the row marks 'req N'. Multi-line input shared with 41hhgn's answer box. Owner: kestrel (model, CLI, MCP, cue) then nautilus (TUI).

## Depends on

## Acceptance
- [x] hug task request ID - stores a multi-line request verbatim with who and when, and hug task requests lists every open one across the board (test)
- [x] hug task resolve ID N "what changed" closes request N with the account and who; resolving an unknown N refuses with a fix: line (test)
- [x] The assignee's session gets one recall line naming the task, the request text and the resolve command, on its next turn; a task with no assignee routes to the coordinator (harness test)
- [x] In the TUI, R in a detail opens a multi-line box (Enter sends, Alt-Enter adds a line); the row shows 'req 1' while open, and the detail shows each request with its account once resolved (golden)
- [x] MCP task_update request and resolve behave like the CLI (test)
- [x] Jørgen writes a real request on hug's board after release and an agent resolves it into ticket changes he agrees with

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 6):
  Write a real request on a hug task after this release: does the agent turn it into ticket changes you agree with?
  answered 2026-11-20T11:29 by jhf:
    answered by ticking check 6

## Notes

Jørgen 23:53 local: 'it's like a managerial ask, and the consequence for the ticket will be figured out as you process it.' So a request names no field or check and has no form: it is intent. The agent decides what it means for the ticket (which fields, checks, state, maybe a new task or a split) and the resolve account says what it decided. The agent applies hug's task rules (goal, purpose, principle, details, acceptance; titles under 80; fix lines) when it turns the ask into edits; the person never has to.


kestrel 00:58, storage proposal (please confirm before I build; check 11 of 41hhgn is built with it, same file boardcue.go, one writer):
1. File: a '## Requests' section placed before ## Notes (task.Tidy's order gets Requests after Acceptance and others). Each request is one block:
   - 2026-11-18T00:58 request 1 by jhf:
     <text verbatim, every line indented 2 spaces, blank lines kept as '  '>
   closed by a line under it:
     resolved 2026-11-18T01:10 by nautilus: <account, verbatim, may span lines, indented 4>
   N is the request's number on the task (1, 2, ...), never reused, so 'resolve ID N' stays stable after others close.
2. Parse: Task.Requests []Request{N, By, At, Text, Resolved *{By, At, Text}}; Task.OpenRequests int (cached; cache schema 20).
3. CLI: hug task request ID "text" | - (stdin) [--as NAME]; hug task requests [--all] lists open ones across the board (id, N, by, age, first line), fix line per row; hug task resolve ID N "what changed" | - [--as NAME]. A resolve of an unknown N gives 'ID has requests 1, 2 (open: 2)' with fix 'hug task requests'. Resolving an already resolved N is refused, with show as the fix. Commits: 'hug task request ID' / 'hug task resolve ID N'.
4. MCP: task_update action request (text, agent) and resolve (item = N, text, agent); task_list filter requests: true.
5. Cue (boardcue.go): kind 'request', said once per request (key request:ID#N, like answer: not rate-held), to the assignee's session. Line: 'Task board: jhf asks on 6ephqk: "..." (request 1). Act on it, then: hug task resolve 6ephqk 1 "what changed"'.
6. Unassigned routing and 41hhgn check 11 both need 'the coordinator', which hug does not know today. Proposal: hug.toml [board] coordinator = "crab" (hug task people-style command 'hug task coordinator NAME'); with none set, an unassigned request goes to every session on the board once (first session whose turn comes, key per request, so said once in total, not once per session). crab to rule.
7. 41hhgn check 11: after an answer, the asker's next write on that task is recorded under the answer (History '  outcome by X: <the write's history line>'), and kind 'answered' tells the coordinator once: 'Task board: nautilus acted on jhf's answer on 41hhgn: <outcome line>'. Same routing as 6.
crab 00:58 ruling. (a) The storage format of the 00:58 note is approved as written. (b) Routing: hug.toml [board] coordinator = NAME, written by 'hug task coordinator NAME' (no editor). When it is unset, every session on the board hears an unowned request once each, because 'first session hears it' could leave it with an agent that drops it, and a request must never be lost quietly. hug setup lists 'no coordinator set' as a step with that fix line. An open request is also the first thing 'hug task next' prints for whoever it routes to (the assignee, else the coordinator, else everyone), until it is resolved, so the cue is not its only path. crab sets crab as coordinator on hug's board after the landing.
kestrel 01:15, built on kestrel/hbgybn (4d2c8a5, rebased on eeb8f93). Evidence per check:
1. TestRequestAndResolve (task) and TestRequestCommands (cli): a 2-line request from stdin with a quote and a colon is stored as '## Requests' / '- TIME request 1 by jhf:' / '  make it per board: "tags"' / '  and split the TUI part', before Notes. hug task requests lists it with '  fix: hug task resolve ID 1 "what changed"'.
2. Resolve 2 before 1 keeps the numbers; a third request is 3. 'resolve ID 9' gives 'ID has no request 9; its requests are 1, 2, 3 (open: 3)' with fix 'hug task requests ID'; an already resolved N gives 'request 1 on ID is already resolved, by nautilus' with fix 'hug task show ID'. The commit is 'hug task resolve ID 1'.
3. TestBoardRequestRouting (recall). Exact line: 'Task board: jhf asks on 6ephqk: "make it per board and split the TUI part" (request 1). Act on it, then: hug task resolve 6ephqk 1 "what changed"'. Said once per request. It routes to the assignee of a doing task; on a task nobody holds, to the coordinator; with none, to every session once each. TestRequestRoutesThroughNext (cli): hug task next prints it first for the same people until it is resolved. hug task setup step 'coordinator' says 'no coordinator set' with 'hug task coordinator NAME'.
5. TestRequestOverMCP: task_update request and resolve (item = N), task_requests, task_show requests[].resolved.
Mutants, 13 in total, each failing a test. 5 in task/cli, e.g. Next ignoring requests -> 'requests_test.go:79: next for wyvern: ...hug task take'; numbers reused -> 'requests_test.go:29'. 4 in recall, e.g. a held task going to the coordinator -> 'boardcue_test.go:479: coordinator heard a held task's request'. 4 for 41hhgn check 11.
Open: 4 (TUI R box, nautilus), 6 (Jørgen after release).
landed 8737541 on main
nautilus 01:23: check 4 (TUI) is queued for nautilus after e3sevr and 0gztsy's answer textarea (41hhgn check 8). The R box reuses that textarea (Enter sends, Alt-Enter adds a line, paste keeps lines). Model from kestrel 01:23: task.Task.Requests []task.Request{N, By, At, Text, Resolved *Resolution}, task.OpenRequests(t), Store.Request/Resolve, '## Requests' before Notes, show --json 'requests'. Note: R today is reload (the corner's 'R reloads'); in a detail R must not shadow reload, so where both apply, which wins is a ruling for crab. (nautilus)
crab 01:30 ruling on the key: (c). r = request (r marked in 'request'); the old r refresh in a detail goes (the board updates live); R stays reload. So check 4 reads: r in a detail and on a list row opens the request box. (nautilus)
check 4 landed 88a6ac89..d75c1691 (nautilus, crab 01:30 ruling r = request, r marked in 'request'; the old r refresh removed, the board updates live). r in a task's detail, or on its row (opens the detail then the box), opens the answer's textarea under a REQUESTS heading: Enter sends, Alt-Enter / shift+enter / ctrl-j add a line, a paste keeps its lines; an empty request is refused. The row draws 'req N' before the title while N requests are open; the detail's REQUESTS lists 'N. who · X ago · open|resolved', the text verbatim, then '└ by: account'. Test huginn/internal/tui/request_test.go TestRequestBox (80 and 140; types two lines, sends, checks store.Requests[0].Text/By, the REQUESTS block, 'req 1' on the row, then store.Resolve and the account) with golden testdata/detail/request.txt; fails first on main 28791dea: 'request_test.go:32: at 80 r on a row: prompt "" detail false'. Key bar: S screen and E edit files moved with the writers (E only at >=100, so 80 keeps 'u up'). CI run 44228773327 green on every job (ubuntu, macos, windows, windows-cli, bench); tui + task suites green locally after the final rebase; -race on tui: only the tmux drift that CI goldens then fixed, no DATA RACE.
landed d75c169 on main 2026-11-18: tui: r opens the request box in a task and on its row; REQUESTS in the task, req…
shipped in v0.7.55 (d75c169)
Approved follow-on ec6z6j: single and multi-task feedback should converge on one board-coordinator inbox, with pending/acknowledged/in-progress/answered receipt tracking. A selected-set request is one linked record, not independent copies per task. Approval 2026-11-18 11:08Z, implementation still pending.
## History
- 2026-11-18T00:57 todo -> doing/active (take)
- 2026-11-18T01:17 doing/active -> doing/waiting (wait)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: kestrel; waiting_on: check 4: nautilus's TUI R box on top of 8737541; 6: Jørgen after the release; waiting_since: 2026-11-18T01:17
- 2026-11-20T11:29 answered asked (check 6), by jhf, for crab (doing/asked -> doing/active):
  > Write a real request on a hug task after this release: does the agent turn it into ticket changes you agree with?
  answered by ticking check 6
- 2026-11-20T11:29 doing/active -> done (done after jhf ticked check 6 (release v0.7.55)); was doing_since: 2026-11-18T00:57
- 2026-11-20T11:29 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:29 answer 1 acknowledged by crab
