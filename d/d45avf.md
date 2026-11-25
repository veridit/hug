---
id: d45avf
title: "Editing keys in the answer and request box: ctrl+left/right and a key table"
state: todo
type: feature
priority: "high"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzza"
created: 2026-11-22
updated: 2026-11-22
stage: "ready"
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
- [x] Jørgen names the keys that did not work for him (question 1)
- [ ] ctrl+left and ctrl+right move a word in the box, and every key Jørgen names works (tmux test per key)
- [ ] F1 in the box lists the editing keys
- [ ] Jørgen edits an answer with his keys after release
- [ ] In the box, alt+left and alt+right move a word too, since macOS takes ctrl+left/right for switching Spaces; F1 names them (tmux test)

## Questions
- 2026-11-22T23:10 question 1 by crab to jhf (check 1):
  In hug's answer and request box these already work in v0.7.108 (tested in tmux): ctrl+w, ctrl+a, ctrl+e, ctrl+u, ctrl+k, alt+b, alt+f, alt+backspace, Home, End. Only ctrl+left and ctrl+right do nothing. Which keys failed for you, and in which board? If it was in Seid on p39w4a, every key there takes about a second (1tjq11), so the edit may have looked missing until the page caught up.
  answered 2026-11-25T11:16 by jhf:
    It works But notice that control left and control right don't do anything because they are tied to switching spaces

## Notes


check 1: Jørgen 2026-11-25: 'It works. But notice that control left and control right don't do anything because they are tied to switching spaces' (macOS takes them before the terminal)
## History
- 2026-11-22T23:10 todo -> todo/design (ask)
- 2026-11-25T11:16 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > In hug's answer and request box these already work in v0.7.108 (tested in tmux): ctrl+w, ctrl+a, ctrl+e, ctrl+u, ctrl+k, alt+b, alt+f, alt+backspace, Home, End. Only ctrl+left and ctrl+right do nothing. Which keys failed for you, and in which board? If it was in Seid on p39w4a, every key there takes about a second (1tjq11), so the edit may have looked missing until the page caught up.
  It works But notice that control left and control right don't do anything because they are tied to switching spaces
- 2026-11-25T11:16 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-25T13:02 crab read the answer: hug task set d45avf
