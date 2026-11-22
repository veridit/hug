---
id: 7zg9mf
title: "Filters can be negated (!), and the list opens on 'state: not done'"
state: "doing"
type: "feature"
priority: "medium"
assignee: ["owl"]
labels: []
group: ""
rank: "zzzzzzzzzzzf"
created: 2026-11-20
updated: 2026-11-20
doing_since: "2026-11-22T03:13"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "0s652z"
asked_by: "crab"
asked_check: "5"
asked_what: "Jørgen opens the list after release and sees only open work, and can flip a filter with !"
asked_for: "jhf"
asked_since: "2026-11-22T05:05"
---

## Goal
A filter can say 'not' (state: !done), and the list view opens with state: !done so finished tasks stay out until asked for.

## Purpose
The list opens on all 317 tasks, most of them done, so the open work is buried.

## Principle
One negation mark, !, the same in the picker, the : command and the filter line; the default is visible on the filter line and Esc clears it.

## Relevant details
Today (v0.7.90): filter is one value per field (task.Filter in huginn/internal/task/store.go: State, Type, Priority, Label, Group, Mine, Epic), no negation; the list opens with 'filter: none' and shows 317/317 (screen-1334.ans). Proposal: (1) a field value may start with !, meaning every value but that one: ': state !done', and in the f picker ! before choosing (or ! on a highlighted value) flips it; the filter line reads 'filter: state !done'. (2) The list view opens with state !done; overview and board keep their own sections. The open default is a setting for later; Esc clears it as any filter and the line then says 'filter: none'. Jørgen 2026-11-20 12:34Z (request 1): 'there has to be a way to select a filter and then inverse it with an exclamation mark ... Or put the exclamation mark first, and then you get a selection of the filter. So you can select not something, and that this is the default filter when you open up the list view.'

## Depends on

## Acceptance
- [ ] ~~Verify Default filter not done for list~~ (not relevant: replaced by checks 2-5, crab, 2026-11-20)
- [x] ': state !done' and ': type !bug' keep every task but those; the filter line shows the !; CLI hug task ls --state '!done' does the same (tests)
- [x] In the f picker, ! flips the highlighted value to its negation, before or after picking (key test)
- [ ] ~~The list view opens with 'filter: state !done', and Esc clears it to 'filter: none' (tmux test)~~ (not relevant: Esc keeps filter chips since v0.7.94 (5cjy88); F clears all, ⌫ the last; replaced by check 6, crab, 2026-11-22)
- [ ] Jørgen opens the list after release and sees only open work, and can flip a filter with !
- [x] The list view opens with 'filter: state !done' shown as a chip; F (or ⌫ on it) clears it to 'filter: none', and Esc does not, as for any chip since 5cjy88 (tmux test)

## Requests
- 2026-11-20T13:34 request 1 by jhf:
  This also means there has to be a way to select a filter and then inverse it with an exclamation mark or something. Or put the exclamation mark first, and then you get a selection of the filter. So you can select not something, and that this is the default filter when you open up the list view
  resolved 2026-11-20T13:35 by crab:
    Turned into a task: filters get a 'not', written with ! (': state !done', or ! in the f picker before or after choosing a value), shown on the filter line, and the CLI takes the same. The list view then opens with 'filter: state !done', so done tasks stay out until you clear it with Esc. Checks 2-4 are tests (the default one a tmux test), check 5 your look after release.

## Questions
- 2026-11-22T05:05 question 1 by crab to jhf (check 5):
  Jørgen opens the list after release and sees only open work, and can flip a filter with !

## Notes

file:screen-1334.ans (15300 bytes, sha256:a5fc83a1) screen capture from hug tui


crab 02:14Z before staffing: check 4 said Esc clears the default; since v0.7.94 Esc keeps chips (5cjy88 ruling, Jørgen liked lasting chips), F clears all, ⌫ the last. Struck 4, added 6 with that. The details' 'Esc clears it' reads as F/⌫ now.
file:1-list-opens.ansi (3113 bytes, sha256:b1665065) the list as it opens (0201dca6, hug binary in tmux 100x30, capture-pane -e): 'filter: [state:!done ×] ⌫ last · F clear', 4/5 tasks, cccccc (done) left out, the dropped bbbbbb kept, footer 'state=!done · F clears'
file:2-picker-not-done.ansi (1704 bytes, sha256:efbe6180) the f picker with a negated value (0201dca6, tmux 100x30): f s done !, header 'FILTER · state · not: every task but the value picked', row '> !done 4', footer 'j/k ↑↓ pick · Enter add chip · ! not · ...'
Failing first on origin/main d72f77ca (commit 9d35298a, tests only): TestNegatedFilters '[ls --state !done]: invalid-value state: !done; valid: todo, doing, done, dropped; closest: done / fix: hug task ls --state done'; TestListTakesNegatedFilters 'list map[state:!done]: error true, ran [], want [ls --state !done]' (and search, next); TestFilterNegation does not build: 'f.Keeps undefined (type Filter has no field or method Keeps)'; TestNegatedFilterCommand ': state !done: filter "", 60 of 60 shown, none done false'; TestFilterPickerBangFlips/before_the_field '[F f ! s done enter]: prompt "", state filter "done", want "!done"', on_the_value 'prompt "filter", state filter ""', '! went into the text: "done!"'; TestListOpensOnNotDone 'the list opened with filter "": filter: none · sort: rank'. All pass on 0201dca6; local full suite 42/42 ok (GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test ./...).
Design (0201dca6), for crab: (1) ! is free in the f picker: it is bound only as the key after v (v ! = lint, viewline.go:14), never as text there; in the picker ! is a toggle, never typed, so 'done!' cannot happen. (2) MCP shares the CLI code: task_list/task_search/task_next build 'hug task ls --state !done' and the same Execute runs it; the schema enums gained the ! forms (state, type), so a typo like '!don' is still refused before it runs. new/set still refuse ! (only ls, search, next, board negate). (3) board: --state stays ignored by the board's state columns, as before; ! on the other fields applies. (4) The default and ht3cdx: the default is the list's own, put up by setView when the list is shown and taken down in every other view, so a filter still keeps the view and the overview/board never see 'state !done'. A chip the reader sets in the list (': type bug') is kept across views as any chip; a state chip of theirs replaces the default for the session. F, or ⌫ on a state chip in the list, retires the default for the session; F in the overview leaves it. (5) Empty sections: in the list, a filter that empties it reads 'TASKS [0] · None with state:!done type:X', the default named like any chip; the overview and board never name it (not filtered by it). (6) Reload (R) carries it: the default is not written as a chip, a cleared one travels as no_default.
CI run 44982838522 on ad61f852f34314d79cfce88abf3254de508d4d5e (branches 7zg9mf and ci/7zg9mf): all 8 jobs success: secrets, test (ubuntu-latest), test (macos-latest), test (windows-latest), test-windows-cli (rest), test-windows-cli (sync), test-windows-cli-shards, bench. Local: full suite 42/42 ok; -race on task, cli, mcp, tui ok. Mutants (one edit each, restored with git checkout, sha256 checked): 22 run, 22 caught in the end. M6 (board's State cut) survived because board takes no --state at all, so the cut was dead code: removed, and M6b (board's Keeps call off) is caught. M11 (picker's ! not reset on f) and M16 (⌫ in the list not retiring the default) survived and were closed by new asserts in ad61f852. Goldens: 51 changed, 370 lines each way, every one reviewed: list views show [state:!done ×], 45/60 or 4/5 counts, TASKS [45] and the done rows gone, footer 'state=!done · F clears'; task pages opened from the list name the chip in the crumb ('HUG · list › state:!done › ID') and 'Task N of M' counts the shown rows; picker goldens gain '! not'. No other change.
check 2: ad61f852, CI 44982838522 all 8 green: TestNegatedFilterCommand (': state !done' and ': type !bug' keep every task but those, chip '[state:!done ×] [type:!bug ×]', '· None with state:!todo'), TestNegatedSearchTerm ('/ state:!done'), TestFilterNegation (task.Filter.Keeps, Next), TestNegatedFilters (hug task ls --state '!done', --type '!bug', both, search, next, board; bad value ends 'fix: hug task ls --state '\''!done'\'''), TestListTakesNegatedFilters (MCP task_list/task_search/task_next; '!don' refused). Failed first on d72f77ca. Mutants M1-M9, M6b, M21 caught.
check 3: ad61f852, CI 44982838522 all 8 green: TestFilterPickerBangFlips: ! before the field, before the value and on the highlighted value each give !done; ! twice gives done; ! is never text ('done', not 'done!'); the row reads '> !done' with its count; footer '! not'; a new f starts without it. tmux: TestTmuxListOpensOnNotDone, frame 2-picker-not-done.ansi. Failed first on d72f77ca. Mutants M10, M11, M12 caught.
check 6: ad61f852, CI 44982838522 all 8 green: TestTmuxListOpensOnNotDone (hug binary in tmux 100x30): v l opens on 'filter: [state:!done ×]', 4/5, done cccccc out; Esc from a task keeps it; Esc up to the overview shows 'filter: none' 5/5; F clears it to 'filter: none' with cccccc back and it stays cleared after v o v l. TestListOpensOnNotDone: the same, the board unfiltered, ⌫ clears it for the session, a reader's own state chip replaces it. TestReloadKeepsListDefault: R keeps it up, cleared or replaced. Frame 1-list-opens.ansi. Failed first on d72f77ca. Mutants M13-M20, M16 caught.
landed 9d35298 on main 2026-11-22: test: negated filters (! in : and the f picker, CLI and MCP) and the list's…
landed 0201dca on main 2026-11-22: filters can be negated with !, and the list opens on 'state !done' (7zg9mf)
landed ad61f85 on main 2026-11-22: test: close three surviving mutants: the picker's ! resets on f, ⌫ retires the…
shipped in v0.7.101 (9d35298)
## History
- 2026-11-20T13:34 request 1 by jhf
- 2026-11-20T13:34 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:34 request 1 acknowledged by crab
- 2026-11-20T13:35 request 1 in progress by crab: hug task set 7zg9mf
- 2026-11-20T13:35 resolved request 1 by crab
- 2026-11-22T03:13 todo -> doing/active (take)
- 2026-11-22T05:05 doing/active -> doing/asked (ask)
