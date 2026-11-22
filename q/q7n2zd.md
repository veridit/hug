---
id: q7n2zd
title: "Paste the clipboard into a task as an attachment"
state: "doing"
type: feature
priority: "medium"
assignee: ["pig"]
labels: []
group: ""
rank: "zzzzzzzzzzxt"
created: 2026-11-20
updated: 2026-11-20
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-22T08:46"
---

## Goal
One key in hug tui attaches what is on the clipboard (a screenshot or text) to a task, the same way S attaches a screen capture.

## Purpose
Jørgen copies screenshots and text elsewhere and wants them on a ticket without saving a file and running attach.

## Principle
Reuse the S flow: inside a task it attaches there; in a list it asks: attach to ID · select · create.

## Relevant details
Key: V (Jørgen 2026-11-20 11:52Z), next to S for screen; shown in the keys line and ? help. Provisional until the key review (bfxq93).

How it reads the clipboard: an image cannot come through the terminal (Cmd+V sends text only), so the key reads the system clipboard itself: macOS via osascript (PNG from «class PNGf»), Linux via wl-paste or xclip; text via pbpaste/wl-paste/xclip. Image -> paste-HHMM.png; text with ANSI colour -> .ans; other text -> .txt. Stored with the same hug task attach path (name, size, sha256 note, secret check). Over SSH or with no clipboard tool, the message says so with a fix: line (hug task attach ID FILE). Also: a Cmd+V (bracketed paste) inside a task with no box open could offer to attach the pasted text. Key: P and p already paste a yanked task (3xcesq); keys stay provisional until bfxq93. Related: 5z4zt4 (attachments shown inline), hjbfzd (S capture wording), 65kjn8 (when files were attached). Jørgen 2026-11-20 10:23Z: 'is there a way to paste a screen? ... I can go into a ticket or I can press a paste key that takes ... whatever is in the current paste buffer and attach it to something a bit like the screenshot'.

## Depends on

## Acceptance
- [x] Jørgen picks the key
- [ ] Inside a task the key attaches a clipboard image as PNG and clipboard text as .txt (.ans with colour), with the attach note, and FILES shows it at once (tmux test with a fake clipboard)
- [ ] In a list the key asks attach to ID · select · create, as S does, and Esc attaches nothing (tmux test)
- [ ] An empty clipboard, no clipboard tool or an SSH session gives a message with a fix: line, nothing attached (test)
- [ ] Jørgen copies a screenshot in macOS, presses the key in a task, and sees it in FILES

## Questions
- 2026-11-20T11:23 question 1 by crab to jhf (check 1):
  Which key pastes the clipboard into a task: V (free today, next to S for screen), A (free, 'attach'), or another you prefer?
  answered 2026-11-20T12:52 by jhf:
    Yes, let's try uppercase V

## Notes


check 1: Jørgen 2026-11-20 11:52Z (answer 1): 'Yes, let's try uppercase V'. Key: V (provisional until bfxq93).
crab 2026-11-20 12:04Z: back to todo. It sat in doing under crab with nobody building it; it is queued for a worker (machine load over 10). Found in the 47sxn2 request-1 check.
file:q7n2zd-task-after-V-100x30.ans (2600 bytes, sha256:bbc2b706) V in a task at 100x30: tmux capture-pane -e of the built binary (0f28cd8c) with HUG_TEST_CLIPBOARD as a fake clipboard holding two coloured lines; FILES shows paste-1006.ans at once and the rule says 'clipboard text (2 lines) · attached as paste-1006.ans to cccccc'
## History
- 2026-11-20T11:23 todo -> doing/active (take with p87ngv)
- 2026-11-20T11:23 doing/active -> doing/asked (ask)
- 2026-11-20T12:52 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > Which key pastes the clipboard into a task: V (free today, next to S for screen), A (free, 'attach'), or another you prefer?
  Yes, let's try uppercase V
- 2026-11-20T12:52 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T12:52 answer 1 acknowledged by crab
- 2026-11-20T12:52 crab read the answer: hug task check q7n2zd 1 "Jørgen 2026-11-20 11:52Z (answer 1): 'Yes, let's try uppercase V'. Key: V (provisional until bfxq93)."
- 2026-11-20T13:04 doing/active -> todo/ready (give back; was assignee: crab); was doing_since: 2026-11-20T11:23; doing_with: p87ngv
- 2026-11-22T08:46 todo/ready -> doing/active (take)
