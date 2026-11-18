---
id: 89y0b7
title: "Waiting rows: two lines, the title on top, what it waits on below"
state: "done"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:39"
---

## Goal
A row whose stage waits on something (waiting, review, design, depends) takes two lines in every list, overview section and board: line 1 is the task as usual (pie, id, type, priority, title, owner); line 2 hangs under the title with └─ and says what it waits on, the reason directly after the colon, and since when: '└─ waiting on you: approve the design ▸ bxqtkt.md   since 14:42'. The person is named, never initials: 'on you' for the TUI user, 'on wyvern', 'on CI'. An attachment named in the wait shows as ▸ name. Active and ready rows stay one line.

## Purpose
Jørgen 2026-11-17 14:55-14:58: the one-line waiting row hides the title, so he cannot tell what he is asked about; the colon floats away from its text; 'jhf' is just text, so he cannot tell it is for him; the file it names is a path he cannot open. He approved the two-line design at 14:58 ('let's try that'). Mock attached to 3jtfw9: waiting-rows.md.

## Principle
The title always shows; the reason is what a waiting row adds, not what replaces it. Text that reads as a sentence: 'waiting on you: ...'. Two lines only where there is something to say. The 102bc0 rule still holds per line: each line fits the width, cut in a fixed order.

## Relevant details
Replaces the 10:56 'stage: reason replaces the title' rule for waiting rows. 'you' needs hug to know the TUI user: hug.toml people = [...], hug setup sets HUG_USER once, lint flags a --on naming no listed person or agent (part of this ticket). Row code: huginn/internal/tui render.go, stage.go forYou(). Section heights count two-line rows. Moving with j/k moves by task, not by line.

## Depends on

## Acceptance
- [x] a waiting, review or design row renders as two lines with └─ and the reason after the colon; active and ready rows stay one line (goldens at 80 and 140)
- [x] 'on you' when the wait names the TUI user, else the name; hug setup sets HUG_USER from hug.toml people, and lint flags a --on naming no listed person or agent (test)
- [x] j/k move by task over two-line rows, and section heights and scrolling count lines correctly (test)
- [ ] ~~Jørgen sees the hug board with the new rows and says it reads clearly~~ (not relevant: superseded: Jørgen's later rulings replaced two-line rows (tje4sx: one line, only the focused row expands; 41hhgn: ASKED section, question under the focused row); he confirms those on 41hhgn and 1bbe96, crab, 2026-11-18)

## Notes

file:waiting-rows.md (1583 bytes, sha256:5108b054) approved mock (Jørgen 14:58)


crab 15:28 ruling (nautilus asked): (a) --on writes a node field waiting_for: NAME (like design_for, review_by); waiting_on holds only the what. This reverses crab's 12:38 'no new field': a person must be checkable. Lint flags waiting_for/design_for/review_by naming no hug.toml person and no agent seen on the board. hug upgrade moves an old 'NAME: x' into waiting_for only when NAME is a listed person. (b) 'you' = HUG_USER, else the only person in people, else the header says '? who are you: set HUG_USER'. The setup step's fix is a command (hug setup you NAME adds to people and prints the export line); hug never edits the shell profile.
Jørgen 15:30 (supersedes crab's 15:28 (b)): identify the person by git email. hug.toml [people] jhf = "jorgen@veridit.no". 'you' = HUG_USER if set; else, with no agent session, the person whose email is git config user.email; in an agent session, the agent's own name (agents commit with Jørgen's email in Seid, so email never identifies an agent). hug setup offers [people] entries from commit authors as a command. Check 2's 'hug setup sets HUG_USER' reads as: 'you' resolves as above (test).
Jørgen 15:31: replace HUG_USER with git email for people; keep a mapping to a short name. So: hug.toml [people] short = "email" (jhf = "jorgen@veridit.no"); a person is found by git config user.email and shown by the short name; HUG_USER is no longer read for people (agents keep their session name). No [people] entry: show the email's local part and lint/setup offer the mapping as a command.
Jørgen 15:31b: the mapping also carries a long name, for people whose usernames and emails are obfuscated (e.g. 12345+x@users.noreply.github.com). Shape: [people.jhf] name = "Jørgen H. Fjeld", email = ["jorgen@veridit.no", "...@users.noreply.github.com"]. A person matches by any listed email; rows show the short name (jhf), the detail and help show the long name. Several emails per person, since one person commits from several.
nautilus 15:58: built on nautilus/89y0b7 (on top of the 3jtfw9+cue+x1ra0t landing stack 4a5c655), not pushed yet. Rulings: crab 15:28 (waiting_for field), Jørgen 15:31 and 15:33 ([people.SHORT] name + email list; person by git email, agent by its name; no hug setup you). What is in it: (1) rows two lines, tui/waitrow.go note()/noteRow(), window() counts lines, overviewRows in lines; (2) Store.Wait writes waiting_for (--on), Answer and resume clear it; (3) task/people.go People/PersonByEmail/EmailShort; (4) hug task people [add SHORT EMAIL --name], offers grouped by author name from git log; setup step people; (5) lint unknown-person (task/peoplelint.go), fix hug task people add NAME EMAIL; (6) hug upgrade moves old waiting_on 'NAME: what' to waiting_for when NAME is listed (asr4t2 part); (7) detail line with the long name. Tests: TestRowsShowStage golden, TestWaitingOnYou, TestYouByGitEmail, TestTwoLineRowsScrollByTask, TestDetailNamesTheLongName, cli TestPeopleAndWaitingFor, TestPeopleOffersFromCommitLog, TestUpgradeMovesWaitingFor. Full suite running.
nautilus 16:11: landed on main 2447266 (c48b30c..2447266), full local go test ./... ok (tui 83.1s), pushed straight per the 15:36 rule. Check 1: TestRowsShowStage plus the tmux goldens list/overview/board/sort/grab at 80x24 and 140x40 ('└─ depends on 000001  ▱ 0/1' under a depends row; active rows one line). Check 2: TestWaitingOnYou, TestYouByGitEmail, TestDetailNamesTheLongName; cli TestPeopleAndWaitingFor (wait --on writes waiting_for; lint unknown-person names no listed person or agent, fix hug task people add NAME EMAIL). The 'hug setup sets HUG_USER' half was superseded by Jørgen's 15:31 ruling: a person is found by git config user.email in hug.toml [people.SHORT]; the setup step 'people' offers entries from the commit log. Check 3: TestTwoLineRowsScrollByTask (j/k by task; window and section heights count lines). Check 4 is Jørgen's.
file:rows-on-main.txt (4510 bytes, sha256:33a25283)
landed 9b27bf7 on main 2026-11-17: hug upgrade: moving waits to waiting_for keeps the cache open until both board…
shipped in v0.7.43 (2447266)
## History
- 2026-11-17T16:11 todo -> doing/active (take)
- 2026-11-17T17:15 doing/active -> doing/waiting (wait)
- 2026-11-18T00:39 doing/waiting -> done (done); was doing_since: 2026-11-17T16:11; waiting_asked_by: nautilus; waiting_for: jhf; waiting_on: check 4: open hug tui on this board and say whether the two-line rows read clearly (capture attached: ▸ rows-on-main.txt; landed 2447266 + 9b27bf7); waiting_since: 2026-11-17T17:15
