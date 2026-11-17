---
id: 41hhgn
title: "Questions to a person get their own doing stage, 'asked'"
state: "doing"
type: feature
priority: ""
assignee: ["kestrel"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T23:53"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
doing gets a stage 'asked': the owner asked a named person something and can't go on until they answer. Interaction with a person is its own thing, not a kind of waiting. Fields: asked_what (the question), asked_for (who), asked_check (the check it is about, optional), asked_since, asked_by, asked_show. 'waiting' keeps only waits on things with no person (CI, a release). The TUI drops the '?' glyph: the row reads 'asked: you · check 9 · <question>', and questions for the viewer come first. hug upgrade moves every waiting with waiting_for into asked, and a leading 'check N:' into asked_check.

## Purpose
Jørgen 23:48-23:49 local: 'drop the ? in favor of question instead of waiting'; 'interaction with the user is a thing in itself, a specific thing, not just general waiting'. Today a question to him is a waiting with waiting_for plus a '?' glyph: two ways of saying one thing, and the '?' collided with the scroll arrows.

## Principle
One fact in one place: the stage says it in words, no glyph. No backward compatibility: hug upgrade migrates (preview, then --apply) and waiting_for goes away. Answering needs no editor (hug task answer ID, as now). Nothing writes silently.

## Relevant details
Today: states.go doing/waiting carries waiting_on, waiting_for, waiting_since, waiting_asked_by, waiting_show; 'hug task wait ID --on NAME' sets it; render.go marks it '?' and 'waiting on you'. After: 'hug task ask ID --to NAME [--check N] [--show FILE] "question"' sets doing/asked; 'hug task wait' loses --on and refuses it with 'fix: hug task ask ID --to NAME "..."'. MCP task_update gains action ask. The at3qep lint questions skip stage asked (wdkqzx's rule, said by stage). 0gztsy's waiting_check becomes asked_check. todo/design (a question before starting) stays for now; crab flags the overlap for a later look. Owner: kestrel (model, CLI, MCP, upgrade); nautilus renders it in 0gztsy. Checks below that say 'question' mean stage asked (Jørgen 23:49).

## Depends on

## Acceptance
- [ ] ~~x~~ (not relevant: placeholder from creation, crab, 2026-11-17)
- [ ] hug task states lists doing/question with its fields; 'hug task ask ID --to jhf --check 9 "look at ..."' sets stage question with them, and a bad N is refused with 'task has 9 checks' and a fix: line (test)
- [ ] hug task wait ID --on NAME refuses with 'fix: hug task ask ID --to NAME "..."'; plain wait still works (test)
- [ ] hug upgrade previews, and --apply moves each doing/waiting with waiting_for into doing/question, a leading 'check N:' (real N) into question_check; on hug's own board the preview lists the tasks and the apply leaves 0 waiting_for fields (test, plus the board numbers in a note)
- [ ] The TUI shows no '?' glyph; a question row's stage cell reads 'question' and questions for the viewer sort first (golden)
- [ ] MCP task_update action ask behaves like the CLI (test)
- [ ] Jørgen confirms on his board after release
- [ ] A multi-line answer survives every path: TUI (Alt-Enter adds a line, a paste keeps its lines, Enter sends), 'hug task answer ID -' from stdin, and MCP; History holds it as an indented block, and show, the detail and the asker's notice print every line verbatim (test with a 3-line answer holding a quote and a colon)

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
## History
- 2026-11-17T23:53 todo -> doing/active (take)
