---
id: zbdbtm
title: "A file named in a task opens from the task in the embedded viewer"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzr"
created: 2026-11-20
updated: 2026-11-20
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

## Notes

Jørgen 2026-11-20 10:12Z (p87ngv request 4): p87ngv's questions name several files (MOCK.md, before.md, after.md, history.sdd, cli.txt, tui-100x30.txt); he has to remember them, go to FILES and open each. Wanted: a file named in a question (or anywhere in the text) can be viewed in line, or jumped to, and jumped back from. Open for him: which key jumps (Enter, Space, g, or another)? Keys provisional until bfxq93.
