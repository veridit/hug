---
id: fzd0jj
title: "Release v0.7.89"
state: "done"
type: chore
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzl"
created: 2026-11-20
updated: 2026-11-20
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T12:57"
---

## Goal
v0.7.89 (qcq3tm: an asked task opens on its question; 7j706s: lint checks only live file links) is out and Jørgen tries it.

## Purpose
Each release gets one look from Jørgen, with pictures to answer from.

## Principle
One ticket per release; the shipped tasks are its children; frames are attached here.

## Relevant details
Main 20dbb6ca (Release v0.7.89, merge of a5fc5208 + CHANGELOG). qcq3tm CI 44721913557 8/8 on a5fc5208; 7j706s CI 44717024683 8/8 on 4b1a5781. Frames from cricket (100x30, before = 4b1a5781, after = a5fc5208): before/after-tb0dra, before/after-hjbfzd, after-hjbfzd-check6-link, before/after-p87ngv-a.

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.89 with links into the new Cellar

## Questions
- 2026-11-20T12:02 question 1 by crab to jhf:
  qcq3tm: restart hug tui and open p87ngv (or any task in ASKED): do you land on the question, with its file as a '→ NAME · Enter shows' line, and does a open the box in view? Before/after frames are in this ticket's Files.
  answered 2026-11-20T12:57 by crab:
    Jørgen on qcq3tm, request 2 (2026-11-20 11:57Z): 'Yes, it does ... it worked on this task after I reloaded.' The underlined > is q3sh69.

## Notes

file:before-tb0dra-100x30.ans (2975 bytes, sha256:d2e372aa) 100x30 frames from qcq3tm: before = v0.7.88 code (4b1a5781), after = v0.7.89

file:after-tb0dra-100x30.ans (3927 bytes, sha256:4c747c53) 100x30 frames from qcq3tm: before = v0.7.88 code (4b1a5781), after = v0.7.89

file:before-p87ngv-a-100x30.ans (2483 bytes, sha256:ffee7bf7) 100x30 frames from qcq3tm: before = v0.7.88 code (4b1a5781), after = v0.7.89

file:after-p87ngv-a-100x30.ans (3668 bytes, sha256:cd9adebd) 100x30 frames from qcq3tm: before = v0.7.88 code (4b1a5781), after = v0.7.89

check 1: release.yml 44723208345 completed success; brew upgrade: 'hug v0.7.89', which hug -> ../Cellar/hug/0.7.89/bin/hug


Reflection (open, 2026-11-20, crab): Nothing misled me.
## History
- 2026-11-20T12:02 todo -> todo/design (ask)
- 2026-11-20T12:57 answered design, by crab, for crab (todo/design -> todo/ready):
  > qcq3tm: restart hug tui and open p87ngv (or any task in ASKED): do you land on the question, with its file as a '→ NAME · Enter shows' line, and does a open the box in view? Before/after frames are in this ticket's Files.
  Jørgen on qcq3tm, request 2 (2026-11-20 11:57Z): 'Yes, it does ... it worked on this task after I reloaded.' The underlined > is q3sh69.
- 2026-11-20T12:57 todo/ready -> doing/active (take with 911j70)
- 2026-11-20T12:57 doing/active -> done (done); was doing_since: 2026-11-20T12:57; doing_with: 911j70
