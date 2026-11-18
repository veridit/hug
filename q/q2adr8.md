---
id: q2adr8
title: "Ask several questions at once, each answered on its own"
state: "doing"
type: feature
priority: "high"
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T20:43"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
An agent with several open questions on a task asks each separately, and the person answers each in its own place (CLI, MCP, hug tui) without one answer covering all of them. The task stays asked until every question is answered.

## Purpose
Jørgen 2026-11-18 19:27Z on rxr4z8: 'If you ask three questions, I should have three questions to answer in the ticket. This makes no sense. Let's structure this better.' nautilus put three questions into one ask, and Jørgen had to answer all three in one text.

## Principle
One question, one answer. Agents are taught to ask one thing per question. A numbered list inside one ask is either split or refused with a fix: line. Answers stay verbatim in History, each beside its question. Delivery to the coordinator stays one line per answer.

## Relevant details
Today: hug task ask sets one asked/asked_check/asked_show on the stage (41hhgn); hug task answer answers 'the question the stage asks'. Look at: repeated hug task ask adding questions (asked 1..N), hug task answer ID N, the TUI ASKED row showing 'N questions', the help text for ask/answer and the agent instructions (DeliveryFlow) saying one question per ask. Consider a lint/ask warning when the question text holds '(1)' '(2)' or '1.' '2.' lines.

## Depends on

## Acceptance
- [ ] hug task ask twice on one task keeps both questions; show lists them numbered; the stage stays asked until both are answered (test)
- [ ] hug task answer ID N answers question N only, and the asker sees which (test); answering by ticking --check N still works
- [ ] hug tui shows each open question with its own answer action (golden at 100x24)
- [ ] An ask whose text holds a numbered list of questions is refused with 'fix: ask each question on its own: hug task ask ID --to NAME "..."' (test)
- [ ] The ask/answer help and the agent instructions say: one question per ask
- [ ] Jørgen confirms on his board after release

## Notes

crab 20:30 local: queued; next free TUI/CLI worker after the windows-cli shard lands. Until then coordinators and workers ask one question per hug task ask (memory saved).


check 4 (iwazaru): branch q2adr8 on ci/34r1ft-stack d67ecba3. Failing first: askmany_test.go:28 'ask "(1) Is the header right? (2) ... (3) Ship today?": exit 0, want 2 with the fix'. Rule: >=2 numbered items each a question (ends ? or holds its own ' or '). Replay over hug board history: 34 distinct questions, 1 refused = rxr4z8's (Jørgen's complaint). Format A (## Questions) chosen by crab 19:49.
## History
- 2026-11-18T20:43 todo -> doing/active (take)
