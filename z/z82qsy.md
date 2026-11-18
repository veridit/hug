---
id: z82qsy
title: "File viewer: a rule under the header; no stale action message"
state: "doing"
type: bug
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T12:53"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In a task's file viewer, a horizontal rule separates the header (path, size, n of m, title) from the file's contents, and the message line never shows a leftover result from an earlier action on another task.

## Purpose
Jørgen 2026-11-18 viewing 5z4zt4/detail-stage.md: the contents start with no visual boundary, and the bottom line read 'answer vzghj8', the result of answering a different task earlier, which looks like part of this view.

## Principle
The message line reports the action just taken and clears when the person moves on (opens another task or file, changes view). One rule style shared with the detail header's existing rules.

## Relevant details
Message comes from actions.go finish(): m.message = action + ' ' + ID, never cleared on navigation. Message also reads as a command ('answer vzghj8') rather than a result; prefer 'answered vzghj8'. Header lines from files.go (size via fmt KiB). Screenshot in conversation 11:37Z.

## Depends on

## Acceptance
- [ ] The file viewer draws a horizontal rule between its header lines and the contents, in overview, list and board (golden).
- [ ] After an action on one task, opening another task or file clears the message line; a test fails first on main.
- [ ] Action results read as past tense with the id ('answered vzghj8'), not as a command (test).
- [ ] Viewing a file, everything else goes away: the top shows only the crumb (task id > file path, n of m), the bottom only the keys (next/previous file, back), and a full-width rule at the top and bottom frames the file so a screenshot's own lines read as inside it (golden; Jørgen's 5z4zt4 request 2)

## Notes


## History
- 2026-11-18T12:53 todo -> doing/active (take)
