---
id: hjbfzd
title: "Screen capture: attach to a chosen task, pick one, or create a new task"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzwc"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:16"
stage: "asked"
model: "claude-opus-5-5"
branch: "hjbfzd"
worktree: "/Users/jhf/.jcode/scratch/hjbfzd"
asked_since: "2026-11-20T09:51"
asked_by: "crab"
asked_check: "6"
asked_what: "Is this wording right: 'Screen capture: attach to 000002 · select · create', with the task's title moved to the keys line below? Today vs proposed, in the real theme: capture-wording-100x30.ans in Files (Space shows it); all three steps are in capture-wording.md."
asked_for: "jhf"
---

## Goal
After S captures the screen in hug tui, the person picks the task it attaches to or creates a new one.

## Purpose
Jørgen 2026-11-18 (yf39ya requests 1-2): 'when I take a screenshot, it would be nice if I could also attach it to ... ticket so-and-so or S for select or C for create. And if I press select, I can go pick one. And if I put create, I can create a new one.'

## Principle
One keystroke per choice, shown in the prompt. Selecting a task reuses the list's own navigation; creating one asks only for a title, and lint's missing sections are filled later through the ticket. The capture is never lost: Esc says 'not attached' and keeps nothing on disk.

## Relevant details
After S captures the screen in hug tui, the prompt offers where it goes: type a task ID (completion), s to pick a task from the list, or c to create a new task with the capture attached; Enter attaches, Esc cancels. Today the capture only attaches to the focused task.

Capture today: S takes the frame (tui.go captured, picture.go attachCapture, render.go:1029 'Attach this screen to ID as NAME'). Attach via store.Attach. Pairs with 5z4zt4 (attach screenshots) and q2adr8 (one question per ask). The prompt follows the calm-TUI rules: underline marks the key letter, no background fills.

## Depends on

## Acceptance
- [x] After S, the prompt reads e.g. 'Attach this screen to: ID_ · s select · c create · Esc' and typing a task ID (prefix completes) then Enter attaches it there (key test)
- [x] s opens the task list in pick mode; Enter on a row attaches the capture to that task and returns (key test)
- [x] c asks for a title, creates the task with the capture attached, and opens it (key test; the new task has the file in Files)
- [x] Golden of the prompt at 80 and 140 columns
- [ ] Jørgen confirms on his board after release
- [ ] The prompt reads 'Screen capture: attach to ID · select · create' (s and c underlined), the offered task's title on the keys line; after c 'Screen capture: new task, title:'; after s 'Screen capture: select the task' (golden at 80 and 140, per capture-wording.md once Jørgen approves)

## Questions
- 2026-11-18T23:24 question 1 by crab to jhf (check 5):
  Jørgen confirms on his board after release
  withdrawn 2026-11-20T09:51 (doing/asked -> doing/active)
- 2026-11-20T09:44 question 2 by crab to jhf (check 6, show capture-wording-100x30.ans):
  Is this wording right: 'Screen capture: attach to 000002 · select · create', with the task's title moved to the keys line below? (All three steps are in capture-wording.md.)
  withdrawn 2026-11-20T09:51 (doing/asked -> doing/active)
- 2026-11-20T09:51 question 3 by crab to jhf (check 6):
  Is this wording right: 'Screen capture: attach to 000002 · select · create', with the task's title moved to the keys line below? Today vs proposed, in the real theme: capture-wording-100x30.ans in Files (Space shows it); all three steps are in capture-wording.md.
- 2026-11-20T09:51 question 4 by crab to jhf (check 5):
  The capture prompt shipped in v0.7.70: on your board, does S then attach, select or create work for you (the wording itself is question 2)?

## Requests
- 2026-11-20T09:43 request 1 by jhf:
  So I tried this and I could attach it to this ticket. You should find it, but the wording isn't good. It says attach to, and then it says screen capture, and then it has a long text, and then you can create select or create. But it should be screen capture colon attached to the pre-filled task or select or create new or create, just do not even need new. Can you make a suggestion and put it as an attachment to this task?
  resolved 2026-11-20T09:44 by crab:
    Proposal attached: capture-wording.md (the exact strings for all three steps) and capture-wording-100x30.ans (today vs proposed in the real theme). Main line: 'Screen capture: attach to 000002 · select · create'; the long title moves to the keys line; 'new' is dropped from create. Added check 6 to build it; asked you to approve the wording.
- 2026-11-20T09:52 request 2 by jhf:
  When I look at these files, some are new and some are old. It would be nice to know when they were captured because then I can see, ah, that's the old one. Ah, this is the new one. Now there's just like we have capture wording, we have screen so-and-so, screen so-and-such. But we already know the creation dates of these things or the update, the M time maybe.
  resolved 2026-11-20T09:53 by crab:
    Filed as its own task, 65kjn8 'FILES shows when each file was attached': each row in FILES gets a dim age after its size (e.g. '12.4 KiB · 10 min ago'), taken from when hug committed the file, not its mtime, which a clone or sync rewrites. It applies to every task's files, not only captures.

## Notes


Design (duckling, before code). S copies as today, then opens the where-prompt in the input line (no confirm, no background fill): 'screen copied (N lines) · Attach this screen to: ID▁ → title · [s]elect · [c]reate · Esc'. The field starts with the task in view (Enter attaches there, as today) or empty with none in view (today S then says 'no task in view'). Typing replaces it; Tab completes a prefix; Enter resolves with task.Resolve (an ambiguous or unknown prefix keeps the prompt and says why). s and c act while the field is untouched or empty; once typing they are text, so an id starting with s or c is picked with s (or typed after a first other char is not possible): stated in help. s: leaves the task for its rows, footer 'pick the task for the screen: Enter attaches · Esc not attached'; navigation keys work, writers (t d x e m y P p r N S a w - Space) only say so; Enter attaches to the row and returns to where S was pressed. c: 'title:' prompt; Enter creates the task as N does (goal = title, accept 'Verify TITLE'), attaches, one commit 'hug task new ID', opens it; Esc goes back to the where-prompt with the capture kept. Esc in the where-prompt or the pick: 'not attached', nothing on disk. The file name is fixed at capture time (screen-HHMM.ans, -2 on a clash in the chosen task). Code in tui.go/render.go/keys.go/helpkeys.go and picture.go's attach path only; not detail.go.
f68a9a5d on main 8a45b0eb (v0.7.70), branch hjbfzd + ci/hjbfzd. Fail-first: all 6 new tests failed on the old code (prompt lacks 'Attach this screen to: 000002'; S with no task: prompt "" message 'no task in view to attach it to'; s/c opened nothing; golden: old bar had '48;2;255;179;84' background). After: TUI suite ok 108.563s (pre-rebase); post-rebase one failure, TestTmuxReload ('the reload line stayed after leaving the task', reload_tmux_test.go:216, no S involved) at load ~120, then ok 19.503s and 16.038s alone. Rebase conflicts with v0.7.70 (runKey confirm, palette footer, pendingArgs) resolved, keeping both. Help row now 'copy the screen, attach: id, s select, c create' (one line at 80/140; goldens are 1:1 swaps).
check 1: f68a9a5d: TestCaptureAttachesToATypedTask (typed 000012 attaches there, commit 'hug task attach 000012'; Tab completes 000010; ambiguous/unknown keep the prompt, no file) and TestCaptureWithNoTaskInViewAsks; prompt reads 'Attach this screen to: 000002 → Bug task 02: ... · select · create' (golden); fail-first on 8a45b0eb: 'the prompt lacks "Attach this screen to: 000002"'
check 2: TestCaptureSelectPicksARow: s opens the rows (picking, not detail), 9 writers only say so, j j Enter attaches to that row with one commit and returns to 000002; Esc attaches nothing; # 000020 Enter Enter attaches there. Fail-first: 's did not open the rows to pick'
check 3: TestCaptureCreatesATask: c asks 'title:', Esc goes back to where with the capture kept, empty title creates nothing, 'Broken bar' Enter creates it with screen-1452.ans in Files (on disk and in the detail), one commit 'hug task new ID', opens it. Fail-first: 'c opened ""'
check 4: testdata/detail/capture-prompt.txt: offered, typed, title and picking at 80 and 140, text and codes; the test fails on any 48; background and any SGR with both 1 and 4. Fail-first: old bar '48;2;255;179;84'
CI 44400498760 on f68a9a5d: success, all 8 jobs. ubuntu -race huginn/internal/tui: 290.273s. Checks 1-4 ticked; check 5 (Jørgen confirms on his board after release) open.
landed 69b5ceb on main 2026-11-18: tui: S asks where the screen goes: a task id, s select, c create (hjbfzd)
shipped in v0.7.71 (69b5ceb)
file:screen-0939.ans (12730 bytes, sha256:f8b1deb2) screen capture from hug tui
file:capture-wording.md (1806 bytes, sha256:89289ad8)
file:capture-wording-100x30.ans (2801 bytes, sha256:54dc7b78)
file:screen-0952.ans (7458 bytes, sha256:7cd25fb2) screen capture from hug tui
## History
- 2026-11-18T21:16 todo -> doing/active (take)
- 2026-11-18T22:34 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 doing/waiting -> doing/asked (ask); was waiting_asked_by: duckling; waiting_on: crab lands ci/hjbfzd-rebased 69b5cebc with the stack, then a release; check 5 is Jørgen's confirmation after release; waiting_since: 2026-11-18T22:34
- 2026-11-19T04:01 handed to crab: duckling's session ended (closed 2026-11-19 02:31)
- 2026-11-20T09:43 request 1 by jhf
- 2026-11-20T09:43 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T09:43 request 1 acknowledged by crab
- 2026-11-20T09:44 request 1 in progress by crab: hug task attach hjbfzd /Users/jhf/.jcode/scratch/hjbfzd-wording/capture-wording.md
- 2026-11-20T09:44 resolved request 1 by crab
- 2026-11-20T09:44 asked question 2 of jhf, 2 open (doing/asked -> doing/asked)
- 2026-11-20T09:51 doing/asked -> doing/active (resume); was asked_by: crab; asked_check: 5; asked_for: jhf; asked_since: 2026-11-18T23:24; asked_what: Jørgen confirms on his board after release
- 2026-11-20T09:51 doing/active -> doing/asked (ask)
- 2026-11-20T09:51 asked question 4 of jhf, 2 open (doing/asked -> doing/asked)
- 2026-11-20T09:52 request 2 by jhf
- 2026-11-20T09:52 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T09:52 request 2 acknowledged by crab
- 2026-11-20T09:53 resolved request 2 by crab
