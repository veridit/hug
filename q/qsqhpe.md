---
id: qsqhpe
title: "Ticking the check a question is about answers the question"
state: "doing"
type: bug
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzr"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T15:11"
stage: "active"
model: "claude-opus-5-5"
branch: "qsqhpe"
worktree: "/Users/jhf/.jcode/scratch/qsqhpe"
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
- [ ] jhf ticking the asked check moves the task from asked to active with a History line 'answered by ticking check N', and the asker sees it; a test fails first on main.
- [ ] Someone other than the asked person ticking that check leaves the question open (test).
- [ ] gmzd87's delivery sends the tick-answer to the coordinator like a typed answer (test or note once gmzd87 lands).
- [ ] hug task ask --check N on a check that is already ticked is refused (exit 2, 'check N is already ticked: ask about an open check, or add one with hug task set ID --accept ...' and a fix: line); the TUI never shows a question on a ticked check (test, failing first)

## Notes


Design (mizaru): Store.Check, when the Store's Actor is asked_for (case-insensitive), the stage is asked and N is asked_check, writes the answer through the same answerOn that hug task answer uses, text 'answered by ticking check N'. One History block 'answered asked (check N), by jhf, for ASKER (doing/asked -> doing/active)', asked fields cleared, Answer seen by the asker. Result.Answered makes cli.Execute (CLI and MCP) and the TUI's Space call the same Deliver(kind answer) as a typed answer. Base 19a5fc77 (gmzd87), branch qsqhpe, worktree /Users/jhf/.jcode/scratch/qsqhpe.
## History
- 2026-11-18T15:11 todo -> doing/active (take)
