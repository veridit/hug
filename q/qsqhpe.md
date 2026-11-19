---
id: qsqhpe
title: "Ticking the check a question is about answers the question"
state: "done"
type: bug
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzxe"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "qsqhpe"
worktree: "/Users/jhf/.jcode/scratch/qsqhpe"
done_at: "2026-11-18T20:54"
---

## Goal
When the person a question is asked of ticks the check the question is about (hug task check, or the TUI), the question counts as answered: the task leaves asked, the asker is told, and the person is not asked to also type an answer.

## Purpose
Jørgen 2026-11-18 on e3sevr: he ticked check 4 (the check the question was about) and the task stayed in stage asked, still in his ASKED list, so he answered as well. Either one should have been enough.

## Principle
One action, one meaning: a tick by the person asked is their answer ('it holds'). No double bookkeeping. A tick by someone else does not answer a person's question.

## Relevant details
e3sevr at 14:40 local after dd07edee (check 4 ticked by jhf): stage still asked, asked_check 4, asked_for jhf. Only d27d31f3 (hug task answer) moved it to active. Delivery to the coordinator (gmzd87) must fire for this tick-answer too.

## Depends on

## Acceptance
- [x] jhf ticking the asked check moves the task from asked to active with a History line 'answered by ticking check N', and the asker sees it; a test fails first on main.
- [x] Someone other than the asked person ticking that check leaves the question open (test).
- [x] gmzd87's delivery sends the tick-answer to the coordinator like a typed answer (test or note once gmzd87 lands).
- [x] hug task ask --check N on a check that is already ticked is refused (exit 2, 'check N is already ticked: ask about an open check, or add one with hug task set ID --accept ...' and a fix: line); the TUI never shows a question on a ticked check (test, failing first)

## Notes


Design (mizaru): Store.Check, when the Store's Actor is asked_for (case-insensitive), the stage is asked and N is asked_check, writes the answer through the same answerOn that hug task answer uses, text 'answered by ticking check N'. One History block 'answered asked (check N), by jhf, for ASKER (doing/asked -> doing/active)', asked fields cleared, Answer seen by the asker. Result.Answered makes cli.Execute (CLI and MCP) and the TUI's Space call the same Deliver(kind answer) as a typed answer. Base 19a5fc77 (gmzd87), branch qsqhpe, worktree /Users/jhf/.jcode/scratch/qsqhpe.
crab 15:25 local, from Jørgen on vk9td0 (screenshot 14:22Z): crab asked on vk9td0 check 1, which was already ticked; the question sat on a done check under his resolved 12:22 request and read as answered, yet stayed in his ASKED [12]. Same rule as this task from the other side: a tick is the answer, so a ticked check cannot be asked. Added as the last check here.
Evidence at fbb3a174 (branch qsqhpe on gmzd87 19a5fc77). Failing-first, saved in $JCODE_SCRATCH_DIR/qsqhpe-ff/: check 1/2: tickanswer_test.go:19: r.Answered undefined (task-ff.txt); check 3: deliver_test.go:306: receipt: <nil>, tui deliver_test.go:96: delivered [] (message "checked aaaaaa"), delivery_instructions_test.go:204: DeliveryFlow no longer says "or ticks the check ..."; check 4: ask_test.go:168 exit 0 on --check 1 of a ticked check, asked_test.go:250 '? ☑' drawn, asked_test.go:257 row still 'check 2 · look at ...'. Mutants: dropping the asked_for==Actor guard gives tickanswer_test.go:65 'the asker ticking check 2 answered: true'; dropping the check-number guard gives ':65 the person asked, another check ticking check 1 answered: true'. Full suite GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./... -count=1: 34 ok, 0 FAIL (suite2.txt). Pushed qsqhpe and ci/qsqhpe.
crab review 15:41 local: accepted. fbb3a174 (3a9a9b02 + fbb3a174 on gmzd87 19a5fc77): Store.Check by asked_for on asked_check answers through answerOn (same as hug task answer) and Deliver(kind answer); ask --check on a ticked check refused (ticked-check, exit 2, fix names the first open check); TUI draws no question on a ticked check. 7 failing-first lines, 2 mutants. Who ticks = HUG_USER / session / git email via [people]: right. I rebase onto main (gmzd87 landed as 4e70099c) and land after exact-SHA CI.
crab 14:44Z: accepted (ticker from HUG_USER, session or git email is right); crab rebases onto main and lands after exact-SHA CI. CI 44325825385 at fbb3a174 so far: secrets, bench, ubuntu, windows success; test-windows-cli in progress, macos queued.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed e5f4057 on main 2026-11-18: Ticking the check a question is about answers the question (qsqhpe)
shipped in v0.7.66 (e5f4057)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T15:11 todo -> doing/active (take)
- 2026-11-18T15:44 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T15:11; waiting_asked_by: mizaru; waiting_on: crab rebases onto main and lands after exact-SHA CI; waiting_since: 2026-11-18T15:44
