---
id: 3jtfw9
title: "Answer a ticket's question in place; the asking agent is told"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
stage: "active"
---

## Goal
A question to a person lives in the task as a field of its stage (design_question, waiting_on with a question, review_by with what to look at). In hug tui the detail shows it at the top with its context. Pressing a opens a one-line or multi-line input right under the question (not $EDITOR), and Enter writes the answer into the task beside the question: the answer field of that node, the answering person's name and the time. The stage then re-derives (design -> ready, waiting -> active, review -> done or back to active). The agent that asked sees it at its next turn: the recall hook's board cue says 'jhf answered your question on ID: <first line>', hug task next and show put it first, and MCP task_show returns it. CLI: hug task answer ID "text".

## Purpose
Jørgen 2026-11-17 11:45: AI drives most work and sometimes needs him. He wants to find the ticket, see the question, and write the answer straight where the agent needs it, without opening an editor on the whole file, and the agent should notice that the answer landed.

## Principle
The question and the answer sit together in the task, never in a chat. The answer is typed where the question is. Answering is one key, and the stage follows from the fields (6ephqk rule 2). The agent is told by what it already reads (recall cue, next, show, MCP); no new daemon. An answer is never lost: History keeps question and answer after the stage moves on.

## Relevant details
Builds on 6ephqk (stages, node fields, ? marker, '? N for you' header) and needs-you mock tmp/lanes/needs-you.md. Board cue: muninn/internal/recall/boardcue.go (hatchling's area). TUI input: reuse the note prompt. The agent that asked = the one that set the field (record asked_by). Persistent wake-up of an idle agent is 5dxzxe (our own harness spike); until then the coordinator watches the board.

## Depends on

## Acceptance
- [x] in the detail of a task whose stage waits on you, a opens an input under the question; Enter stores the answer next to the question with your name and time, and the stage re-derives (golden + test)
- [x] hug task answer ID "text" does the same from the CLI, and MCP task_update action answer (test)
- [x] the agent that asked sees 'NAME answered your question on ID: ...' in its recall board cue on its next prompt, and first in hug task next/show (test)
- [x] after the stage moves on, History holds the question, the answer, who and when (test)
- [ ] Jørgen answers a real question from a worker in the TUI and the worker picks it up without a DM

## Notes

nautilus 13:27 (crab 13:26): dependency on 6ephqk removed. 3jtfw9 needs only 6ephqk checks 3 (moves, design_asked_by/waiting_asked_by) and 8 (for you, design_for, waiting_on NAME), both on main at d74af5d..d198e01. Design ruling (crab 13:26): the answer is a History line 'answered design: <q> -> <a>, by NAME, DATE TIME, for ASKER'; show and next --mine put the newest answer first until the asker writes anything on the task after it (that write counts as seen); the board cue reads the same. No new front-matter fields, nothing new for lint. answer re-derives the stage; d stays separate.

nautilus 13:47: built on nautilus/3jtfw9, commit fd95919, CI 44096102067 (queued). Not on main. Check 1: TUI a under the question, golden huginn/internal/tui/testdata/detail/answer-100.txt, TestAnswerInDetail. Check 2: CLI TestAnswerCommand, MCP bridge case in server_test.go, task TestAnswerDesign, TestAnswerWaitingAndReview, TestNextMineAnswerFirst. Check 4: History line 'answered STAGE: "q" -> "a", by NAME, for ASKER (from -> to)', asserted in the same tests. Check 3: huginn/board.Task.Answer is exposed (board_test). The cue line is for hatchling: tmp/lanes/3jtfw9-cue-for-hatchling.md.


file:needs-you.md (3264 bytes, sha256:d2e9eb8a) design, moved here from tmp/lanes (git-ignored, so other clones never saw it)
file:waiting-rows.md (1583 bytes, sha256:5108b054) two-line waiting rows mock (Jørgen 14:55)
file:detail-stage.md (1564 bytes, sha256:a6a935a7) detail: stage first mock (Jørgen 14:47)
nautilus 15:14: rebased on main b8f9000 as 99edb5f (conflicts in CHANGELOG, cache.go, cli.go, help.go, mcp/server.go resolved; cache SchemaVersion 14, fkpyg3 took 13). Local: go test ./huginn/... all 11 packages ok. Pushed ci/3jtfw9. hatchling's cue commit (check 3) is rebased on 99edb5f in ci/3jtfw9-cue; I cherry-pick it when it's green, so both land together.
nautilus 16:06: checks 1, 2, 4 ticked. 1: tui TestAnswerInDetail. 2: cli TestAnswerCommand, task TestAnswerWaitingAndReview, mcp server_test answer case. 4: task TestAnswerDesign (History line holds question, answer, who, for whom, when). landed on main c48b30c (stack d107f9a..c48b30c: 6ef55df 3jtfw9, 842b403 cue, 998438f+c48b30c x1ra0t); local go test tui/task/cli/mcp/cache/harness ok after the rebase; earlier CI 44111992156 green on 99edb5f; cue abb9b76 CI green per hatchling. Check 3 is hatchling's cue (842b403 on main); check 5 needs Jørgen to answer a real question in the TUI.
hatchling 16:32, check 3 ticked: 842b403 on main (abb9b76 rebased by nautilus onto d107f9a). Tests: recall TestBoardAnswerForAsker; run TestBoardAnswerReachesAsker (end to end through a real git board: the answer reaches the asker's next prompt as 'jhf answered your question on ID: TEXT (stage: QUESTION). hug task show ID'). Main CI 44121775854 (c48b30c, contains 842b403): success on test ubuntu/macos/windows, secrets, bench. 44121274410 (d107f9a) attempt 2 success after a macOS runner-capacity cancel.
shipped in v0.7.43 (6ef55df)
## History
- 2026-11-17T14:56 doing -> doing/waiting (wait)
- 2026-11-18T00:15 answered waiting: "check 5: answer a real worker question in hug tui (a under the question); 1, 2, 4 on main 6ef55df; check 3 is hatchling's cue 842b403" -> "Well, here we go. I'm gonna answer this ticket. If you can read this answer, the ticket is done, and you can mark you can fly that acceptance criteria and the ticket becomes done, right Or when you mark the last one, you're reminded that you need to mark it as done. How is the interaction today", by jhf, for nautilus (doing/waiting -> doing/active)
