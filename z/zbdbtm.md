---
id: zbdbtm
title: "A file named in a task opens from the task in the embedded viewer"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzn"
created: 2026-11-20
updated: 2026-11-20
stage: "ready"
---

## Goal
Any file a task names, attached or in the repo, opens from the task in hug's embedded viewer at the named section, in one key.

## Purpose
Jørgen 2026-11-20 08:55Z (hpqt99 request 1): hpqt99 pointed at docs/DESIGN.md section 'Imported data' on a branch; he could not open it from the ticket and had to leave for another editor. He asks (1) open referenced files from the ticket in the embedded viewer, (2) reach them when jumping between anchors, (3) per-task documents are attachments, not general docs.

## Principle
A reference is a link: the reader follows it in place and comes back with Esc. Task-specific documents are attachments; repo docs are linked, never copied.

## Relevant details
Link forms, in the p87ngv mark syntax: .target:(.File .name:"design.md") for an attachment, .target:(.File .path:"docs/DESIGN.md" .section:"Imported data") for a repo file, optionally .ref:"BRANCH". The detail's anchor stepping (Tab, the id stops of 0ga6m0) stops on each link; Enter opens it in the file frame scrolled to the section; Esc returns to the link. 8a0ybb covers a question's own target; this covers every file named anywhere in the task. Agent side (fd0j4p): instructions say a document for one task is attached to it.

## Depends on

## Acceptance
- [ ] Enter on a link to an attached file opens it in the file frame; Esc returns to the link (tmux test)
- [ ] Enter on a link to a repo file with a section (e.g. docs/DESIGN.md#Imported data) opens it scrolled to that heading, from the board's worktree or the named branch (test)
- [ ] Tab and Shift+Tab stop on every file link in the detail (test)
- [ ] AGENTS.md and the setup template say: a document for one task is attached to that task; only decisions go into repo docs (test pins the wording)
- [ ] A file named in plain text inside a question, check or note (e.g. 'MOCK.md', 'tui-100x30.txt') is recognised when the task has that file, and works as a link: one key shows it in line under the line that names it, another opens it in the file frame, Esc returns (tmux test on p87ngv's board copy)

## Questions
- 2026-11-20T11:12 question 1 by crab to jhf:
  For a file named in a task's text (e.g. MOCK.md in p87ngv's question): Space shows it in line under that line, Enter opens it in the file frame, Esc returns, matching Space and Enter in FILES today. Is that the right pair, or do you want a separate jump key (g)?
  answered 2026-11-20T12:51 by jhf:
    Let's try that and make sure you put it into this file as well so that when I review this task again when it is resolved I can use the task itself to check it

## Notes

Jørgen 2026-11-20 10:12Z (p87ngv request 4): p87ngv's questions name several files (MOCK.md, before.md, after.md, history.sdd, cli.txt, tui-100x30.txt); he has to remember them, go to FILES and open each. Wanted: a file named in a question (or anywhere in the text) can be viewed in line, or jumped to, and jumped back from. Open for him: which key jumps (Enter, Space, g, or another)? Keys provisional until bfxq93.


file:sample-notes.md (244 bytes, sha256:76521eb4) sample files: this task's own links point at them, so Jørgen can try it here
file:sample.txt (50 bytes, sha256:2df59eb0) sample files: this task's own links point at them, so Jørgen can try it here
## History
- 2026-11-20T11:12 todo -> todo/design (ask)
- 2026-11-20T12:51 answered design, by jhf, for crab (todo/design -> todo/ready):
  > For a file named in a task's text (e.g. MOCK.md in p87ngv's question): Space shows it in line under that line, Enter opens it in the file frame, Esc returns, matching Space and Enter in FILES today. Is that the right pair, or do you want a separate jump key (g)?
  Let's try that and make sure you put it into this file as well so that when I review this task again when it is resolved I can use the task itself to check it
- 2026-11-20T12:51 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T12:51 answer 1 acknowledged by crab
- 2026-11-20T12:51 crab read the answer: hug task attach zbdbtm /Users/jhf/.jcode/scratch/zb/sample-notes.md /Users/jhf/.jcode/scratch/zb/sample.txt
