---
id: 41hhgn
title: "Questions to a person get their own doing stage, 'question'"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
doing gets a stage 'question': the owner asked a named person something and can't go on until they answer. Interaction with a person is its own thing, not a kind of waiting. It carries question (the text), question_for (who), question_check (the check it is about, optional), question_since, question_asked_by and question_show. 'waiting' keeps only waits on things with no person (CI, a release). The TUI drops the '?' glyph: the stage cell says 'question', and questions for the viewer come first. hug upgrade moves every waiting with waiting_for into question, and a leading 'check N:' into question_check.

## Purpose
Jørgen 23:48-23:49 local: 'drop the ? in favor of question instead of waiting'; 'interaction with the user is a thing in itself, a specific thing, not just general waiting'. Today a question to him is a waiting with waiting_for plus a '?' glyph: two ways of saying one thing, and the '?' collided with the scroll arrows.

## Principle
One fact in one place: the stage says it in words, no glyph. No backward compatibility: hug upgrade migrates (preview, then --apply) and waiting_for goes away. Answering needs no editor (hug task answer ID, as now). Nothing writes silently.

## Relevant details
Today: states.go doing/waiting carries waiting_on, waiting_for, waiting_since, waiting_asked_by, waiting_show; 'hug task wait ID --on NAME' sets it; render.go marks it '?' and 'waiting on you'. After: 'hug task ask ID --to NAME [--check N] [--show FILE] "question"' sets doing/question; 'hug task wait' loses --on and refuses it with 'fix: hug task ask ID --to NAME "..."'. MCP task_update gains action ask. The at3qep lint questions skip stage question (wdkqzx's rule, said by stage). 0gztsy's waiting_check becomes question_check. todo/design (a question before starting) stays for now; crab flags the overlap for a later look. Owner: kestrel (model, CLI, MCP, upgrade); nautilus renders it in 0gztsy.

## Depends on

## Acceptance
- [ ] x
- [ ] hug task states lists doing/question with its fields; 'hug task ask ID --to jhf --check 9 "look at ..."' sets stage question with them, and a bad N is refused with 'task has 9 checks' and a fix: line (test)
- [ ] hug task wait ID --on NAME refuses with 'fix: hug task ask ID --to NAME "..."'; plain wait still works (test)
- [ ] hug upgrade previews, and --apply moves each doing/waiting with waiting_for into doing/question, a leading 'check N:' (real N) into question_check; on hug's own board the preview lists the tasks and the apply leaves 0 waiting_for fields (test, plus the board numbers in a note)
- [ ] The TUI shows no '?' glyph; a question row's stage cell reads 'question' and questions for the viewer sort first (golden)

## Notes
