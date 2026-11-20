---
id: 41hhgn
title: "Questions to a person get their own doing stage, 'asked'"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzmh"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T11:29"
done_release: "v0.7.70"
---

## Goal
A question to a named person is its own doing stage, asked, separate from waiting on things.

## Purpose
Jørgen 23:48-23:49 local: 'drop the ? in favor of question instead of waiting'; 'interaction with the user is a thing in itself, a specific thing, not just general waiting'. Today a question to him is a waiting with waiting_for plus a '?' glyph: two ways of saying one thing, and the '?' collided with the scroll arrows.

## Principle
One fact in one place: the stage says it in words, no glyph. No backward compatibility: hug upgrade migrates (preview, then --apply) and waiting_for goes away. Answering needs no editor (hug task answer ID, as now). Nothing writes silently.

## Relevant details
doing gets a stage 'asked': the owner asked a named person something and can't go on until they answer. Interaction with a person is its own thing, not a kind of waiting. Fields: asked_what (the question), asked_for (who), asked_check (the check it is about, optional), asked_since, asked_by, asked_show. 'waiting' keeps only waits on things with no person (CI, a release). The TUI drops the '?' glyph: the row reads 'asked: you · check 9 · <question>', and questions for the viewer come first. hug upgrade moves every waiting with waiting_for into asked, and a leading 'check N:' into asked_check.

Today: states.go doing/waiting carries waiting_on, waiting_for, waiting_since, waiting_asked_by, waiting_show; 'hug task wait ID --on NAME' sets it; render.go marks it '?' and 'waiting on you'. After: 'hug task ask ID --to NAME [--check N] [--show FILE] "question"' sets doing/asked; 'hug task wait' loses --on and refuses it with 'fix: hug task ask ID --to NAME "..."'. MCP task_update gains action ask. The at3qep lint questions skip stage asked (wdkqzx's rule, said by stage). 0gztsy's waiting_check becomes asked_check. todo/design (a question before starting) stays for now; crab flags the overlap for a later look. Owner: kestrel (model, CLI, MCP, upgrade); nautilus renders it in 0gztsy. Checks below that say 'question' mean stage asked (Jørgen 23:49).

## Depends on

## Acceptance
- [ ] ~~x~~ (not relevant: placeholder from creation, crab, 2026-11-17)
- [x] hug task states lists doing/question with its fields; 'hug task ask ID --to jhf --check 9 "look at ..."' sets stage question with them, and a bad N is refused with 'task has 9 checks' and a fix: line (test)
- [x] hug task wait ID --on NAME refuses with 'fix: hug task ask ID --to NAME "..."'; plain wait still works (test)
- [x] hug upgrade previews, and --apply moves each doing/waiting with waiting_for into doing/question, a leading 'check N:' (real N) into question_check; on hug's own board the preview lists the tasks and the apply leaves 0 waiting_for fields (test, plus the board numbers in a note)
- [x] The TUI shows no '?' glyph; a question row's stage cell reads 'question' and questions for the viewer sort first (golden)
- [x] MCP task_update action ask behaves like the CLI (test)
- [x] Jørgen confirms on his board after release
- [x] A multi-line answer survives every path: TUI (Alt-Enter adds a line, a paste keeps its lines, Enter sends), 'hug task answer ID -' from stdin, and MCP; History holds it as an indented block, and show, the detail and the asker's notice print every line verbatim (test with a 3-line answer holding a quote and a colon)
- [ ] ~~The overview has an ASKED section at the top ('ASKED · for you [N]', oldest first) holding the tasks asked of the viewer; WAITING holds only waits with no person and questions asked of others; the header's '? N for you' equals the ASKED count (golden on hug's board shape: 16 asked, 2 waiting)~~ (not relevant: Jørgen 00:13: rows show the title, the focused row adds the question (next check), crab, 2026-11-18)
- [x] The overview has an ASKED section at the top ('ASKED · for you [N]', oldest first) holding the tasks asked of the viewer; each row shows its title, and only the focused row adds a second line '└ asked: check N · <question>'; WAITING holds only waits with no person and questions asked of others; the header's '? N for you' equals the ASKED count (golden on hug's board shape)
- [x] After a person answers, the asker's next write on that task shows under the answer as its outcome, and the coordinator gets one cue line naming the task and what was done (harness test)
- [x] show --json and MCP task_show give the current stage's fields as an object named after the stage next to 'stage' (asked: what, for, check, since, by), and the fields map no longer repeats them (test on a waiting and an asked task)
- [x] hug upgrade moves a wait on a person to asked only when it reads as a question (has a '?', or names a check N); any other wait (a park such as s6ff5k's 'parked per crab ruling, no fourth arm') stays waiting, and the preview lists the waits it leaves, each with its id and text (test, plus the preview on a board clone)

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 7):
  Questions to a person now have their own stage, asked. On your board, does ASKED show what is asked of you, apart from what merely waits?
  answered 2026-11-20T11:29 by jhf:
    answered by ticking check 7

## Notes

Jørgen 23:49 local: the stage is named 'asked' (crab offered asked / ask / question). Read 'question' as 'asked' throughout this ticket: stage doing/asked, fields asked_what (the text), asked_for, asked_check, asked_since, asked_by, asked_show; command 'hug task ask ID --to NAME [--check N] [--show FILE] "..."'; row shows 'asked: you · check 9 · <text>'.

Jørgen 23:50 local: 'my answers can be multiple lines.' Today the TUI answer box is a one-line textinput (tui.go:428), pasted line breaks are lost, and task/answer.go:121 stores the answer strconv.Quote'd on one History line, so a multi-line answer shows as \n. Spec (crab): the answer box is a textarea that grows up to half the screen; Enter sends; Alt-Enter (or Shift-Enter where the terminal reports it) starts a new line; pasted and dictated text keeps its lines. History stores the answer as a block: 'answered asked (check 9), by jhf, for nautilus:' then the answer lines indented two spaces, read back verbatim. CLI: hug task answer ID - reads the answer from stdin; MCP text keeps its newlines. The asker's notice and the detail show every line.


kestrel design 23:56 (building it now; say if any of this is wrong).
Model (task/states.go): doing gains stage 'asked', set by agent, means 'the owner asked a named person something and cannot go on until they answer'. Fields: asked_what (required, the question), asked_for (required, who), asked_check (optional, 1..N), asked_since (hug), asked_by (hug, the asker), asked_show (optional task file). DoingOrder: asked, waiting, depends, review, active. waiting keeps waiting_on, waiting_since, waiting_asked_by (who waits; resume clears it); waiting_for and waiting_show go: a wait on a thing names no person and no file.
Transitions: doing/active -> doing/asked 'hug task ask ID --to NAME [--check N] [--show FILE] "question"'; doing/asked -> doing/active 'hug task answer ID "..."' (and resume, for an owner who stops asking).
CLI: ask as above; a bad --check N says 'ID has 9 checks' with fix 'hug task ask ID --to NAME --check 9 "..."'. wait --on NAME refuses: 'wait is for things; a question to a person is hug task ask' + fix 'hug task ask ID --to NAME "..."'. answer ID - reads stdin.
History, multi-line (check 8): the answered line becomes
  - TIME answered asked (check 9), by jhf, for nautilus (doing/asked -> doing/active):
    > question line 1
    > question line 2
      answer line 1
      answer line 2
  The question's lines quoted with '> ', the answer's indented 2 spaces, both verbatim; a one-line answer uses the same block (one shape, no quoting). Old answered lines (strconv.Quote) still read.
Upgrade: each doing/waiting with waiting_for -> doing/asked: asked_what = waiting_on minus a leading 'check N:' (N real -> asked_check), asked_for = waiting_for, asked_since/asked_by/asked_show from the waiting_ fields. Preview lists ID: from -> to; --apply commits once.
MCP: task_update action ask (to, check, show, text); wait loses on.
Header picture for all of tonight's rulings is attached to e3sevr as header-spec.md (crab 00:00 local). Build to it. Where it and this ticket differ, tell crab.
Jørgen 00:12 local, on the WAITING section (18 rows, most '?'): split asked from waiting. 'asked should be a separate section that I really need to address, and waiting is just a place to live for things that aren't active right now.' Ruling (crab): the overview gets an ASKED section at the top, above IN PROGRESS: 'ASKED · for you [N]', the tasks whose stage is asked and asked_for is the viewer, oldest question first. WAITING keeps doing/waiting (CI, a release) and questions asked of someone else. Each ASKED row shows the question in place of the title ('asked: check 9 · look at …'), per the 6ephqk row rule. The header's '? N for you' counts the ASKED section. 1bbe96's rule applies: in ASKED the stage column is uniform, so it is not drawn.
Jørgen 00:13 local, replacing 'question in place of the title' for ASKED rows: each row shows its title (the context); the focused row shows the question on a second line under it ('└ asked: check 9 · look at …'), as tje4sx expands only the focused row. 'Otherwise I have the question outside of context and would need to read the ticket.'
Jørgen 00:26 local: he answered 4 tickets from the TUI (6ephqk, scrxak, bejd89, 3jtfw9) and expected the outcome to come back to him through crab; nothing did. The answer reaches the asker (3jtfw9 check 3), but nobody reports what the asker did with it. Add: after an answer, the asker's next write on that task (tick, note, done) is shown under the answer in the detail as its outcome, and the coordinator's session gets a one-line cue 'nautilus acted on jhf's answer on 6ephqk: ticked 9, done'. Same mechanism as hbgybn's resolve account.
Statbus agent via Jørgen 00:43 local: 'hug task show --json doesn't return waiting_for, though the file has it' (yq5tr6, j2g6wd). Verified by crab: it is there, but nested in task.fields.waiting_for, next to top-level 'stage': 'waiting'. Agents read the top level and miss it. With asked: show --json and MCP task_show give the stage's own fields as one object named after the stage, beside 'stage': e.g. "stage": "asked", "asked": {"what": ..., "for": "jhf", "check": 9, "since": ..., "by": ...}; likewise "waiting": {"on", "since", "by"}, "design", "review", "deferred". The flat fields map stops carrying stage fields (no duplicates).
kestrel 00:48, check 4 board numbers. Preview on the live board with the branch binary (/Users/jhf/Projects/hug, read-only): exit 5, 'tasks: would: 13 questions to a person moved from waiting to asked (s6ff5k, bxqtkt, 47sxn2, zhvkmj, vzghj8, ggydj2, 5z4zt4, 89y0b7, np2kk2, vnyz8z, jr55s8, 4bwcgp, tje4sx)'. 13 = the files with a front-matter waiting_for: (grep -l '^waiting_for:'). Apply on a clone of the board at 3ef696a7 (/Users/jhf/.jcode/scratch/kestrel-41hhgn-boardcopy): 'waiting_for left: 0  waiting_show left: 0', 'asked: 13  waiting: 4  asked_check: 9  asked_show: 2', 0 old-layout findings after, second upgrade 'tasks: nothing to do'. Bug found on the clone: the first apply left all 13 waiting_for lines, because waiting_for stopped being a node field so Parse dropped it from Task.Fields. Fixed: MoveToAsked reads waiting_for/waiting_show off the file (Document.legacyField) and removes by raw key; lint reports a leftover waiting_for:/waiting_show: as old-layout with fix hug upgrade. Test TestUpgradeMovesWaitingForFromFile. Mutants: removal by Task.Fields -> 'people_test.go:188: a waiting_ field stayed:'; who from Task.Fields -> 'people_test.go:179: upgrade did: layout: nothing to do; tasks: nothing to do; ...'. The live board apply waits for the release: hug v0.7.50 does not know stage asked.
kestrel 00:48, check 12: taskJSON (cli.go stageObjects) moves each stage's fields into an object named after the stage (asked: what/for/check(int)/since/by/show; waiting: on/since/asked_by; design: question/for/since/asked_by/show; review: by/since/show; deferred: reason/until). The flat fields map keeps the rest (doing_since ...) and is dropped when empty. Task's own JSON (the cache payload) is unchanged. Tests: cli TestShowJSONStageObject (asked, waiting, review, design), mcp TestAskOverMCP (task_show's asked object). Mutant without the stageObjects call: 'ask_test.go:123: show --json asked: stage asked, asked <nil>'.
kestrel 00:48, check 6: TestAskOverMCP (mcp/ask_test.go) runs over an in-memory MCP session. ask with check 9 of 2 is refused with 'has 2 checks' and the fix 'hug task ask ID --to jhf --check 2'. ask with to/check/text (2 lines)/agent writes stage asked, asked_what verbatim, asked_for/check/by. wait with 'on' gives 'unknown argument on'. answer (2 lines) writes the History block. task_show returns the answer text. Found on the way: MCP ask refused agent ('argument agent does not apply to action ask'); now passed as --as.
landed 7defaf1 on main. Race clean before push: task/mcp/cache, cli (157.6 s), tui -short/task/mcp after the rebase. Checks 2, 3, 4, 6, 12 ticked (evidence in the notes above and in cli/ask_test.go, cli/people_test.go TestUpgradeMovesWaitingForFromFile, mcp/ask_test.go). 8 is held: the storage, stdin and MCP paths are covered by TestMultiLineAnswerKeepsItsLines and TestAskOverMCP, but the TUI textarea (Alt-Enter) is nautilus's. Also held: 5 and 10 (TUI, nautilus 0gztsy), 7 (Jørgen, after release), 11 (harness). The tmux tests did not run here: loadGuard waited at load ~700. CI runs them.
kestrel 01:15, check 11 built with hbgybn (one writer for boardcue.go, crab 00:57). Task side: the asker's first write after an answer appends 'NAME read the answer: hug task note ID "..."' (the write as the command reads, texts cut to 120, '-' said as '(from stdin)'). Task.Outcome and show --json 'outcome' {stage, by, for, at, text} read it, and older lines without ': WHAT' still read. Cue: kind 'answered', said once to the coordinator only (hug.toml [board] coordinator), never to the asker: 'Task board: nautilus acted on jhf's answer on 41hhgn (asked): hug task note 41hhgn "rows now list the open ones first". hug task show 41hhgn'. Tests: cli TestAnswerOutcomeRecorded, recall TestBoardAnswerOutcomeToCoordinator. Mutants: read line without the action -> 'requests_test.go:144: the outcome is not under the answer'; cli never sets Action -> same line; asker hears own -> 'boardcue_test.go:524'; anyone hears -> 'boardcue_test.go:527: a non-coordinator heard it'.
landed 8737541 on main 2026-11-18: hug task request/resolve/requests: a person's request on a task, turned into…
shipped in v0.7.52 (7defaf1)
crab 02:05: v0.7.52 hug upgrade --apply on hug's board moved 7 waits to asked. One was wrong: s6ff5k ('parked per crab ruling, no fourth arm') is a park, not a question, but moved to asked_for crab because it waited on 'crab'. New check added for it. Repair of s6ff5k is for its owner hatchling (hug task resume s6ff5k, then hug task wait s6ff5k "parked per crab ruling, no fourth arm") once wdkqzx frees the doing slot.
check 13 landed as 2c7ea8b on main. Rule: a waiting_for wait moves to asked only when it has a '?', names a check of the task, or starts with a request verb (approve, answer, confirm, decide, choose, pick, review, rule, look, try, please). Otherwise it stays a wait: waiting_for goes, History says so, and the preview lists it with 'hug task ask ID --to NAME "...?"'.
Failing first: 'upgrade_park_test.go:55: preview (exit 5) lacks "2 questions to a person moved from waiting to asked"', because the old rule moved all 3.
Tests: TestUpgradeLeavesAParkAsWait (cli) and TestReadsAsQuestion (task, a table of all 12 waiting_for waits on hug's board (7, from 8a18427b) and Statbus's (5)). 9 mutants, each failing a test.
Clone of hug's board at 8a18427b^ (/Users/jhf/.jcode/scratch/kestrel-41hhgn13-clone): preview exit 5, 'would: 6 questions to a person moved from waiting to asked (bxqtkt, 47sxn2, zhvkmj, vzghj8, 5z4zt4, jr55s8)' and 'would: 1 wait on a person left waiting ...: s6ff5k waiting_for crab: "H3 failed Part 2 ...; parked per crab ruling, no fourth arm"'. Apply: commit 'hug upgrade: 6 questions to a person moved from waiting to asked, 1 waits left waiting (no question)', waiting_for left 0, s6ff5k stage waiting.
Statbus (read only): 3 of its 5 would stay waits ('the hug release that stops linting...', 'owner looks at the new-record prototype...', 'owner confirms time frames...').
race: 'ok task 25.700s', 'ok cli 132.796s'.
landed 2c7ea8b on main 2026-11-18: hug upgrade: a wait on a person moves to asked only when it asks something ('?'…
landed 3b47f26 on main 2026-11-18: hug upgrade: a request verb anywhere in a wait's first clause, base, -s or -ies…
check 13 widened (crab 02:18) as 3b47f26: a request verb anywhere in the first clause, in base, -s or -ies form. -ed/-ing and the noun senses rule/test don't count.
Fail first against 2c7ea8b's rule, 6 lines, e.g. 'askrule_test.go:45: readsAsQuestion("owner confirms time frames on Norway after the release (rc.02 shipped)") = false, want true'. 7 mutants each fail the table.
hug clone at 8a18427b^: 6 moved, s6ff5k left waiting. Statbus copy: 'would: 4 questions to a person moved from waiting to asked (yq5tr6, ezp3h0, mtw92d, hmgadt)', 'would: 1 wait on a person left waiting ...: j2g6wd waiting_for jhf: "the hug release that stops linting done tasks (installed now: v0.7.50)"'.
race: 'ok task 27.784s', 'ok cli 180.194s'.
landed 4e475c8 on main 2026-11-18: tui: ASKED section first, for you, oldest first; empty sections are their title…
landed 5c99d88 on main 2026-11-18: tui: tests for ASKED on top (41hhgn check 10), the question on its check and in…
landed 1d260a6 on main 2026-11-18: CHANGELOG: ASKED, the question once, the answer textarea (0gztsy, 41hhgn)
nautilus: TUI parts landed 48b1151f..4b93356a (0gztsy), CI 44218985753 6/6 success, -race tui ok. Check 5: no ? glyph in rows (TestWaitingOnYou's marked regexp, stage_test.go); an asked row's stage cell reads 'asked' (golden testdata/overview/asked.txt); questions for the viewer come first, in ASKED. Check 8 (TUI path): TestAnswerBoxKeepsLines: typed 'he said: "keep it"', Alt-Enter, 'second: yes', a bracketed paste 'pasted one\r\npasted: two', ctrl-j, 'last' -> the box holds every line (CRLF one break each; the rune-batch split no longer splits a paste into a text field), grows (height >= 3), Enter sends; the task file's History holds each line indented two spaces; the asker's notice prints each line then 'unread by nautilus'. On the base the box held 'he said: "keep it"second: yespasted one  pasted: two'. stdin and MCP paths were kestrel's (TestMultiLineAnswerKeepsItsLines, TestAskOverMCP). Check 10: TestAskedSectionOnTop: 'ASKED · for you [3]' first, oldest first (cccccc 08:10, aaaaaa 09:50, eeeeee unstamped), only the focused row adds '└─ asked: check 2 · look at the state/stage rows on your board · nautilus', WAITING holds bbbbbb (review by Jørgen) and dddddd (asked of carol), '? 3 for you' equals ASKED; golden testdata/overview/asked.txt at 80 and 140; on main 'at 80 ASKED is not first with 3'. Board-shape note: the fixture is the stageBoard shape, not hug's 16-asked board.
shipped in v0.7.53 (2c7ea8b)
check 11 parked (iwazaru, for 34r1ft): branch 41hhgn-11 = ci/41hhgn-11 at 66034a19 on main 80c8ac28 (failing tests 680c8966). The outcome is delivered to the coordinator in one line with outcome_receipt, offline at the next session start, and the board cue skips a delivered outcome. Touched packages ok (cli 570.161s, task 18.376s, board 5.231s, mcp 35.670s, cache 2.401s, muninn/run 10.393s). huginn+muninn tui timed out under load 315 and pass alone. CI 44362462921: ubuntu, windows, bench, secrets success; macos and windows-cli were still running at 17:58Z. Report: /Users/jhf/.jcode/scratch/41hhgn-11-REPORT.md. Not ticked yet: tick after CI is green.
check 11 CI green: run 44362462921 on ci/41hhgn-11 @ 66034a19: secrets, test-windows-cli, test (macos-latest), bench, test (windows-latest), test (ubuntu-latest) all success. Ready for crab's review; parked while 34r1ft goes first.
crab 20:16 local: check 11 66034a19 accepted on review (outcome line via Deliver, offline receipt, answer cut 120, board cue skips delivered outcome; CI 44362462921 6/6). Lands after 34r1ft.
check 11: on main 8dc479b2 (CI 44385513689 9/9), from 66034a19 accepted on review
landed 08ab507 on main 2026-11-18: cli: failing tests, an answer's outcome reaches the coordinator (41hhgn check…
shipped in v0.7.70 (08ab507)
Reflection (open, 2026-11-20, crab): Nothing misled me.
## History
- 2026-11-17T23:53 todo -> doing/active (take)
- 2026-11-18T00:45 doing/active -> doing/waiting (wait)
- 2026-11-18T00:45 doing/waiting -> doing/active (resume); was waiting_asked_by: kestrel; waiting_on: parked a minute while kestrel closes d2n0h7 (crab 00:44); resumed next; waiting_since: 2026-11-18T00:45
- 2026-11-18T00:56 doing/active -> doing/waiting (wait)
- 2026-11-19T04:01 handed to crab: iwazaru's session ended (closed 2026-11-19 02:27)
- 2026-11-20T09:26 doing/waiting -> doing/asked (ask); was waiting_asked_by: kestrel; waiting_on: checks 5, 8 (TUI textarea) and 10: nautilus's 0gztsy on top of 7defaf1; 7: Jørgen after the release; 11: harness cue, owner per crab; waiting_since: 2026-11-18T00:56
- 2026-11-20T11:29 answered asked (check 7), by jhf, for crab (doing/asked -> doing/active):
  > Questions to a person now have their own stage, asked. On your board, does ASKED show what is asked of you, apart from what merely waits?
  answered by ticking check 7
- 2026-11-20T11:29 doing/active -> done (done after jhf ticked check 7 (release v0.7.70)); was doing_since: 2026-11-17T23:53
- 2026-11-20T11:29 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:29 answer 1 acknowledged by crab
- 2026-11-20T11:29 crab read the answer: hug task done 41hhgn
