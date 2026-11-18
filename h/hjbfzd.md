---
id: hjbfzd
title: "Screen capture: attach to a chosen task, pick one, or create a new task"
state: "doing"
type: feature
priority: "high"
assignee: ["duckling"]
labels: []
group: ""
rank: "zzzzzzzzzzzq"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:16"
stage: "active"
model: "claude-opus-5-5"
branch: "hjbfzd"
worktree: "/Users/jhf/.jcode/scratch/hjbfzd"
---

## Goal
After S captures the screen in hug tui, the prompt offers where it goes: type a task ID (completion), s to pick a task from the list, or c to create a new task with the capture attached; Enter attaches, Esc cancels. Today the capture only attaches to the focused task.

## Purpose
Jørgen 2026-11-18 (yf39ya requests 1-2): 'when I take a screenshot, it would be nice if I could also attach it to ... ticket so-and-so or S for select or C for create. And if I press select, I can go pick one. And if I put create, I can create a new one.'

## Principle
One keystroke per choice, shown in the prompt. Selecting a task reuses the list's own navigation; creating one asks only for a title, and lint's missing sections are filled later through the ticket. The capture is never lost: Esc says 'not attached' and keeps nothing on disk.

## Relevant details
Capture today: S takes the frame (tui.go captured, picture.go attachCapture, render.go:1029 'Attach this screen to ID as NAME'). Attach via store.Attach. Pairs with 5z4zt4 (attach screenshots) and q2adr8 (one question per ask). The prompt follows the calm-TUI rules: underline marks the key letter, no background fills.

## Depends on

## Acceptance
- [ ] After S, the prompt reads e.g. 'Attach this screen to: ID_ · s select · c create · Esc' and typing a task ID (prefix completes) then Enter attaches it there (key test)
- [ ] s opens the task list in pick mode; Enter on a row attaches the capture to that task and returns (key test)
- [ ] c asks for a title, creates the task with the capture attached, and opens it (key test; the new task has the file in Files)
- [ ] Golden of the prompt at 80 and 140 columns
- [ ] Jørgen confirms on his board after release

## Notes


Design (duckling, before code). S copies as today, then opens the where-prompt in the input line (no confirm, no background fill): 'screen copied (N lines) · Attach this screen to: ID▁ → title · [s]elect · [c]reate · Esc'. The field starts with the task in view (Enter attaches there, as today) or empty with none in view (today S then says 'no task in view'). Typing replaces it; Tab completes a prefix; Enter resolves with task.Resolve (an ambiguous or unknown prefix keeps the prompt and says why). s and c act while the field is untouched or empty; once typing they are text, so an id starting with s or c is picked with s (or typed after a first other char is not possible): stated in help. s: leaves the task for its rows, footer 'pick the task for the screen: Enter attaches · Esc not attached'; navigation keys work, writers (t d x e m y P p r N S a w - Space) only say so; Enter attaches to the row and returns to where S was pressed. c: 'title:' prompt; Enter creates the task as N does (goal = title, accept 'Verify TITLE'), attaches, one commit 'hug task new ID', opens it; Esc goes back to the where-prompt with the capture kept. Esc in the where-prompt or the pick: 'not attached', nothing on disk. The file name is fixed at capture time (screen-HHMM.ans, -2 on a clash in the chosen task). Code in tui.go/render.go/keys.go/helpkeys.go and picture.go's attach path only; not detail.go.
f68a9a5d on main 8a45b0eb (v0.7.70), branch hjbfzd + ci/hjbfzd. Fail-first: all 6 new tests failed on the old code (prompt lacks 'Attach this screen to: 000002'; S with no task: prompt "" message 'no task in view to attach it to'; s/c opened nothing; golden: old bar had '48;2;255;179;84' background). After: TUI suite ok 108.563s (pre-rebase); post-rebase one failure, TestTmuxReload ('the reload line stayed after leaving the task', reload_tmux_test.go:216, no S involved) at load ~120, then ok 19.503s and 16.038s alone. Rebase conflicts with v0.7.70 (runKey confirm, palette footer, pendingArgs) resolved, keeping both. Help row now 'copy the screen, attach: id, s select, c create' (one line at 80/140; goldens are 1:1 swaps).
## History
- 2026-11-18T21:16 todo -> doing/active (take)
