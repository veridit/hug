---
id: q2adr8
title: "Ask several questions at once, each answered on its own"
state: "done"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzw1"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T11:34"
done_release: "v0.7.70"
---

## Goal
Several questions on a task are asked and answered each on its own, and the task stays asked until all are answered.

## Purpose
Jørgen 2026-11-18 19:27Z on rxr4z8: 'If you ask three questions, I should have three questions to answer in the ticket. This makes no sense. Let's structure this better.' nautilus put three questions into one ask, and Jørgen had to answer all three in one text.

## Principle
One question, one answer. Agents are taught to ask one thing per question. A numbered list inside one ask is either split or refused with a fix: line. Answers stay verbatim in History, each beside its question. Delivery to the coordinator stays one line per answer.

## Relevant details
An agent with several open questions on a task asks each separately, and the person answers each in its own place (CLI, MCP, hug tui) without one answer covering all of them. The task stays asked until every question is answered.

Today: hug task ask sets one asked/asked_check/asked_show on the stage (41hhgn); hug task answer answers 'the question the stage asks'. Look at: repeated hug task ask adding questions (asked 1..N), hug task answer ID N, the TUI ASKED row showing 'N questions', the help text for ask/answer and the agent instructions (DeliveryFlow) saying one question per ask. Consider a lint/ask warning when the question text holds '(1)' '(2)' or '1.' '2.' lines.

## Depends on

## Acceptance
- [x] hug task ask twice on one task keeps both questions; show lists them numbered; the stage stays asked until both are answered (test)
- [x] hug task answer ID N answers question N only, and the asker sees which (test); answering by ticking --check N still works
- [x] hug tui shows each open question with its own answer action (golden at 100x24)
- [x] An ask whose text holds a numbered list of questions is refused with 'fix: ask each question on its own: hug task ask ID --to NAME "..."' (test)
- [x] The ask/answer help and the agent instructions say: one question per ask
- [x] Jørgen confirms on his board after release

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 6):
  Several questions on one task are now asked and answered each on its own. On your board, does that work for you?
  answered 2026-11-20T11:34 by jhf:
    answered by ticking check 6

## Notes

crab 20:30 local: queued; next free TUI/CLI worker after the windows-cli shard lands. Until then coordinators and workers ask one question per hug task ask (memory saved).


check 4 (iwazaru): branch q2adr8 on ci/34r1ft-stack d67ecba3. Failing first: askmany_test.go:28 'ask "(1) Is the header right? (2) ... (3) Ship today?": exit 0, want 2 with the fix'. Rule: >=2 numbered items each a question (ends ? or holds its own ' or '). Replay over hug board history: 34 distinct questions, 1 refused = rxr4z8's (Jørgen's complaint). Format A (## Questions) chosen by crab 19:49.
check 1: branch q2adr8 76c19719 on main 8dc479b2. Failing first (f72db4e9): askquestions_test.go:28 'file lacks "## Questions\n"'. TestTwoAsksKeepBothQuestions passes at 9b952f51: two asks keep '- STAMP question 1 by nautilus to jhf (check 1):' and 'question 2 ... (check 3):' in ## Questions, show lists 'question 1 to jhf (check 1): ...', stage stays asked (asked_* on the oldest open) until both answered; a third ask is question 3. Also TestAskAdoptsLegacyAndWithdrawsOnResume, TestMergeQuestionsRenumbersOursAfterBoards. CI 44387882079.
check 2: Failing first (f72db4e9): askquestions_test.go:70 'expected 2 positional arguments, got 3'. TestAnswerNAnswersQuestionN passes at 9b952f51: answer ID 2 writes the answer under question 2 only, History 'answered asked (question 2, check 3), by jhf, for nautilus', show --json answer.question_n = 2; answering 2 again and question 7 refused (exit 2, 'open: 1'); bare answer with 2 open refused with 'fix: hug task answer ID 3 "plain"'; ticking check 2 as jhf answers question 4 (about check 2) alone.
check 3: Failing first (on fba767c2): askmany_test.go:49 'lacks "└ question 1 · nautilus"'. TestSeveralQuestionsEachAnswered passes at 76c19719: 3 questions to jhf, 3 lines ending '· answer' (two blocks at the top, each its own Tab stop, one under check 2); Tab to block 2, a, the box opens under question 2 only; Enter answers question 2 alone (2 open left: 1 and 3), > moves to question 1, header 'jhf answered question 2'. Golden huginn/internal/tui/testdata/detail/several-questions-100.txt at 100x24, reviewed.
check 4 (iwazaru): rebased: failing test b7cc3b84, fix f75763be on main 8dc479b2. Not ticked by me before: ticking now.
check 4: Failing first b7cc3b84: askmany_test.go:28 'ask "(1) Is the header right? (2) ... (3) Ship today?": exit 0, want 2 with the fix'. f75763be refuses >=2 numbered items each a question with 'fix: ask each question on its own: hug task ask ID --to NAME "<first>"'. Replay over hug's board history: 34 distinct asks, 1 refused (rxr4z8's).
check 5: Failing first a1f09ba7: askone_test.go:17 'help ask does not say one question per ask', :21 'help answer does not name answer ID N', :24 'AGENTS.md section does not name hug task answer ID N'. Passes at 78d05f78: help ask, help answer (answer ID [N]) and the AGENTS.md section ('Questions: one question per ask ... hug task answer ID N') say it; MCP ask/answer descriptions too.
iwazaru: branch q2adr8 = ci/q2adr8 at 335a39a5 on main 8dc479b2. CI 44391011491 completed success (8 jobs). Full niced go test ./...: exit 0 (cli 368.074s, tui 133.184s, task 11.300s, mcp 15.510s). Report: scratch/q2adr8-REPORT.md. Check 6 is Jørgen's after release.
landed 854286b on main 2026-11-18: cli: failing test, an ask holding several numbered questions is refused (q2adr8…
landed fffc54e on main 2026-11-18: task: refuse an ask that is several numbered questions, fix asks each on its own…
landed f3e08e8 on main 2026-11-18: cli: failing tests, several questions on one task, each answered on its own…
landed 4223588 on main 2026-11-18: cli: failing test, help and instructions say one question per ask (q2adr8 check…
landed 2849cde on main 2026-11-18: task, cli, mcp: several questions on one task, each answered on its own (q2adr8…
landed d4682bb on main 2026-11-18: cli: help and AGENTS.md section say one question per ask, answer ID N (q2adr8…
landed fbf0663 on main 2026-11-18: tui: each open question with its own answer action; cache schema 27; CHANGELOG…
shipped in v0.7.70 (854286b)
## History
- 2026-11-18T20:43 todo -> doing/active (take)
- 2026-11-18T21:39 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: iwazaru's session ended (closed 2026-11-19 02:27)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: iwazaru; waiting_on: v0.7.70 release, then Jørgen confirms on his board (check 6); accepted by crab at 335a39a5, stacked as ci/q2adr8-rebased d5c66a1f; waiting_since: 2026-11-18T21:39
- 2026-11-20T11:34 answered asked (check 6), by jhf, for crab (doing/asked -> doing/active):
  > Several questions on one task are now asked and answered each on its own. On your board, does that work for you?
  answered by ticking check 6
- 2026-11-20T11:34 doing/active -> done (done after jhf ticked check 6 (release v0.7.70)); was doing_since: 2026-11-18T20:43
- 2026-11-20T11:34 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:34 answer 1 acknowledged by crab
