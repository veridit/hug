---
id: 7zg9mf
title: "Filters can be negated (!), and the list opens on 'state: not done'"
state: todo
type: "feature"
priority: "medium"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzf"
created: 2026-11-20
updated: 2026-11-20
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
- [ ] ': state !done' and ': type !bug' keep every task but those; the filter line shows the !; CLI hug task ls --state '!done' does the same (tests)
- [ ] In the f picker, ! flips the highlighted value to its negation, before or after picking (key test)
- [ ] The list view opens with 'filter: state !done', and Esc clears it to 'filter: none' (tmux test)
- [ ] Jørgen opens the list after release and sees only open work, and can flip a filter with !

## Requests
- 2026-11-20T13:34 request 1 by jhf:
  This also means there has to be a way to select a filter and then inverse it with an exclamation mark or something. Or put the exclamation mark first, and then you get a selection of the filter. So you can select not something, and that this is the default filter when you open up the list view
  resolved 2026-11-20T13:35 by crab:
    Turned into a task: filters get a 'not', written with ! (': state !done', or ! in the f picker before or after choosing a value), shown on the filter line, and the CLI takes the same. The list view then opens with 'filter: state !done', so done tasks stay out until you clear it with Esc. Checks 2-4 are tests (the default one a tmux test), check 5 your look after release.

## Notes

file:screen-1334.ans (15300 bytes, sha256:a5fc83a1) screen capture from hug tui


## History
- 2026-11-20T13:34 request 1 by jhf
- 2026-11-20T13:34 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:34 request 1 acknowledged by crab
- 2026-11-20T13:35 request 1 in progress by crab: hug task set 7zg9mf
- 2026-11-20T13:35 resolved request 1 by crab
