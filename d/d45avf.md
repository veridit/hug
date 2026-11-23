---
id: d45avf
title: "Editing keys in the answer and request box: ctrl+left/right and a key table"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-22
updated: 2026-11-22
design_for: "jhf"
design_since: "2026-11-22T23:10"
design_asked_by: "crab"
design_question: "In hug's answer and request box these already work in v0.7.108 (tested in tmux): ctrl+w, ctrl+a, ctrl+e, ctrl+u, ctrl+k, alt+b, alt+f, alt+backspace, Home, End. Only ctrl+left and ctrl+right do nothing. Which keys failed for you, and in which board? If it was in Seid on p39w4a, every key there takes about a second (1tjq11), so the edit may have looked missing until the page caught up."
stage: "design"
---

## Goal
Every common line-editing key works in the answer and request box, as in a shell, and F1 lists them.

## Purpose
Jørgen 2026-11-22 21:37Z: regular ctrl+w and others should work to navigate while answering or making a request.

## Principle
Shell and readline keys, nothing invented; Enter still sends. Test each key in tmux, not only in the model.

## Details
crab 21:43Z tested v0.7.108 in tmux (iTerm2 sends the same keys) on hug's board, box opened with r on hd74ct, text 'alpha beta gamma', '|' typed after the key to show the cursor:
WORKS: ctrl+w -> 'alpha beta |'; ctrl+a / Home -> start; ctrl+e / End -> end; ctrl+u, ctrl+k, ctrl+h; alt+b / alt+left -> 'alpha beta |gamma'; alt+f / alt+right; alt+d; alt+backspace; ctrl+b; ctrl+f; Left.
DOES NOTHING: ctrl+left, ctrl+right (cursor stays at the end).
These are bubbles textarea v1.0.0's KeyMap (textarea.go:76-95); ctrl+left/right are not in it.
On a big task every key takes 0.8-1.4 s (1tjq11), so a ctrl+w there looks like it does nothing until the page catches up; that is 1tjq11, not this task.
Which key failed for Jørgen is asked in question 1.

## Depends on

## Acceptance
- [ ] Jørgen names the keys that did not work for him (question 1)
- [ ] ctrl+left and ctrl+right move a word in the box, and every key Jørgen names works (tmux test per key)
- [ ] F1 in the box lists the editing keys
- [ ] Jørgen edits an answer with his keys after release

## Questions
- 2026-11-22T23:10 question 1 by crab to jhf (check 1):
  In hug's answer and request box these already work in v0.7.108 (tested in tmux): ctrl+w, ctrl+a, ctrl+e, ctrl+u, ctrl+k, alt+b, alt+f, alt+backspace, Home, End. Only ctrl+left and ctrl+right do nothing. Which keys failed for you, and in which board? If it was in Seid on p39w4a, every key there takes about a second (1tjq11), so the edit may have looked missing until the page caught up.

## Notes


## History
- 2026-11-22T23:10 todo -> todo/design (ask)
