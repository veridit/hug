---
id: h8ymea
title: "import files sees evidence already attached under backlog-evidence/"
state: todo
type: bug
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
hug task import files and setup count evidence already copied into task folders as in place, and only rewrite the old citations.

## Purpose
On seid-lang they report 0 of 4,738 in place while 4,741 copies exist, so following them would copy everything twice.

## Principle
Never copy a file that is already on the board; say what is there and what is left.

## Relevant details
Verified 2026-11-20 15:57Z on seid-lang (read-only): .backlog/evidence holds 4,741 files; .hug/tasks/*/ID/backlog-evidence/ holds 4,741 files, added in 1,814 commits on 2026-11-16 22:14 ('hug task attach x1eq3g', an earlier import path). hug task setup says 'import task files: 4738 files to copy, 0 already in place, 2766 citations to rewrite'. 373 task files still cite .backlog/evidence paths. The seid agent (orangutan) spotted it and asked Jørgen on mmrd22 before running it. Fix: recognise ID/backlog-evidence/REL as the placed copy of .backlog/evidence/REL (same bytes), count it in place, and let import files --cite rewrite only the citations to file: links pointing at those copies.

## Depends on

## Acceptance
- [ ] On a copy of seid-lang, hug task import files .backlog/evidence --cite .backlog/evidence --dry-run reports about 4,738 already in place and 0 to copy (test with a fixture of the 11-16 layout)
- [ ] Run for real on that copy, it copies no file and rewrites the citations; lint missing-file does not grow
- [ ] hug task setup shows the import task files step done once only citations were left

## Notes
