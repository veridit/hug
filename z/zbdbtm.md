---
id: zbdbtm
title: "A file named in a task opens from the task in the embedded viewer"
state: "doing"
type: feature
priority: ""
assignee: ["shrimp"]
labels: []
group: ""
rank: "zzzzzzzzzzxk"
created: 2026-11-20
updated: 2026-11-20
stage: "active"
doing_since: "2026-11-22T15:55"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Any file a task names, attached or in the repo, opens from the task in hug's embedded viewer at the named section, in one key.

## Purpose
Jørgen 2026-11-20 08:55Z (hpqt99 request 1): hpqt99 pointed at docs/DESIGN.md section 'Imported data' on a branch; he could not open it from the ticket and had to leave for another editor. He asks (1) open referenced files from the ticket in the embedded viewer, (2) reach them when jumping between anchors, (3) per-task documents are attachments, not general docs.

## Principle
A reference is a link: the reader follows it in place and comes back with Esc. Task-specific documents are attachments; repo docs are linked, never copied.

## Relevant details
Keys (Jørgen 2026-11-20 11:51Z): Space shows a named file in line under the line that names it; Enter opens it in the file frame; Esc returns. Same as Space and Enter in FILES today. No separate g key.

TRY IT HERE (Jørgen: 'put it into this file as well so that when I review this task again ... I can use the task itself to check it'):
- A plain-text name: open sample.txt from here.
- An attachment link: file:sample-notes.md
- A section in it: sample-notes.md#Section two
- A repo file with a section: docs/DESIGN.md#Personal vs shared memories

Link forms, in the p87ngv mark syntax: .target:(.File .name:"design.md") for an attachment, .target:(.File .path:"docs/DESIGN.md" .section:"Imported data") for a repo file, optionally .ref:"BRANCH". The detail's anchor stepping (Tab, the id stops of 0ga6m0) stops on each link; Enter opens it in the file frame scrolled to the section; Esc returns to the link. 8a0ybb covers a question's own target; this covers every file named anywhere in the task. Agent side (fd0j4p): instructions say a document for one task is attached to it.

## Depends on

## Acceptance
- [ ] Enter on a link to an attached file opens it in the file frame; Esc returns to the link (tmux test)
- [ ] Enter on a link to a repo file with a section (e.g. docs/DESIGN.md#Imported data) opens it scrolled to that heading, from the board's worktree or the named branch (test)
- [ ] Tab and Shift+Tab stop on every file link in the detail (test)
- [ ] AGENTS.md and the setup template say: a document for one task is attached to that task; only decisions go into repo docs (test pins the wording)
- [ ] A file named in plain text inside a question, check or note (e.g. 'MOCK.md', 'tui-100x30.txt') is recognised when the task has that file, and works as a link: one key shows it in line under the line that names it, another opens it in the file frame, Esc returns (tmux test on p87ngv's board copy)
- [ ] On zbdbtm itself (TRY IT HERE in Details), each of the four names opens: Space shows it in line, Enter in the file frame at the section, Esc returns (tmux test on a copy of this task)
- [ ] Jørgen opens zbdbtm after release and tries the four links in TRY IT HERE

## Questions
- 2026-11-20T11:12 question 1 by crab to jhf:
  For a file named in a task's text (e.g. MOCK.md in p87ngv's question): Space shows it in line under that line, Enter opens it in the file frame, Esc returns, matching Space and Enter in FILES today. Is that the right pair, or do you want a separate jump key (g)?
  answered 2026-11-20T12:51 by jhf:
    Let's try that and make sure you put it into this file as well so that when I review this task again when it is resolved I can use the task itself to check it

## Notes

Jørgen 2026-11-20 10:12Z (p87ngv request 4): p87ngv's questions name several files (MOCK.md, before.md, after.md, history.sdd, cli.txt, tui-100x30.txt); he has to remember them, go to FILES and open each. Wanted: a file named in a question (or anywhere in the text) can be viewed in line, or jumped to, and jumped back from. Open for him: which key jumps (Enter, Space, g, or another)? Keys provisional until bfxq93.


file:sample-notes.md (244 bytes, sha256:76521eb4) sample files: this task's own links point at them, so Jørgen can try it here
file:sample.txt (50 bytes, sha256:2df59eb0) sample files: this task's own links point at them, so Jørgen can try it here
Jørgen 2026-11-20 11:51Z (answer 1): 'Let's try that and make sure you put it into this file as well so that when I review this task again when it is resolved I can use the task itself to check it.' Done: sample-notes.md and sample.txt attached, TRY IT HERE added at the top of Details, checks 6-7 added.
file:zbdbtm-1-tab-marks-link-100x30.ans (4261 bytes, sha256:d8319d67) Tab marks file:▸sample-notes.md in TRY IT HERE, a stop like a task id (check 3); key bar: Space show · Enter open file
file:zbdbtm-2-space-in-line-100x30.ans (4167 bytes, sha256:3c4a2562) Space on ▸sample.txt: the file in line under its line, ┌ name · size, └ Enter opens it · Esc or Space closes (checks 5, 6)
file:zbdbtm-3-enter-file-frame-100x30.ans (1487 bytes, sha256:afffdb48) Enter on sample.txt: the file frame, crumb zbdbtm › z/zbdbtm/sample.txt, key bar u/Esc link (check 1)
file:zbdbtm-4-esc-back-to-link-100x30.ans (4261 bytes, sha256:743124be) Esc from the frame: back on the page with ▸sample.txt still marked (check 1)
file:zbdbtm-5-attachment-at-section-100x30.ans (1923 bytes, sha256:9517b649) Enter on sample-notes.md#Section two: the file frame opens with 'Section two' as its first line (checks 2, 6)
file:zbdbtm-6-repo-file-in-line-100x30.ans (4197 bytes, sha256:2e527281) Space on docs/DESIGN.md#Personal vs shared memories: that section of the repo file in line (check 6)
file:zbdbtm-7-repo-file-at-section-100x30.ans (5016 bytes, sha256:6683d8c6) Enter on the repo link: docs/DESIGN.md · 9.8 KiB · repo, scrolled to 'Personal vs shared memories' (check 2)
file:zbdbtm-8-p87ngv-MOCK-in-line-100x30.ans (3924 bytes, sha256:f7877693) p87ngv question 6: plain 'MOCK.md' is a link; Space shows 12 rows in line, '30 more lines' (check 5)
## History
- 2026-11-20T11:12 todo -> todo/design (ask)
- 2026-11-20T12:51 answered design, by jhf, for crab (todo/design -> todo/ready):
  > For a file named in a task's text (e.g. MOCK.md in p87ngv's question): Space shows it in line under that line, Enter opens it in the file frame, Esc returns, matching Space and Enter in FILES today. Is that the right pair, or do you want a separate jump key (g)?
  Let's try that and make sure you put it into this file as well so that when I review this task again when it is resolved I can use the task itself to check it
- 2026-11-20T12:51 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T12:51 answer 1 acknowledged by crab
- 2026-11-20T12:51 crab read the answer: hug task attach zbdbtm /Users/jhf/.jcode/scratch/zb/sample-notes.md /Users/jhf/.jcode/scratch/zb/sample.txt
- 2026-11-22T15:55 todo/ready -> doing/active (take)
