---
id: t64v7d
title: "TUI: ? typed in an answer or any text prompt is text, not help"
state: "doing"
type: bug
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T02:03"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
In every text input (answer, note, new title, filter text, / find, : palette argument) the ? key types a ? like any other character. Help in a text input is on F1, which the prompt's footer names.

## Purpose
Jørgen 2026-11-18 01:52 local, answering hibiscus on Statbus xz3jej ('Hibiscus, can you get it done by morning'): pressing ? at the end of his question opened 'IN THIS PROMPT' help instead of typing the ?. tui.go:1163 lets ? win over text input on purpose ('Help always wins over text input'). That rule is wrong for a text field: a person writing a question needs ?.

## Principle
Inside text, every printable key is text. Only keys that cannot be text (Enter, Esc, Tab, F-keys, ctrl-chords) act. Outside text input ? stays help. No key that works only in some prompts.

## Relevant details
Code: huginn/internal/tui/tui.go:1162-1170 (the ? branch runs before the prompt gets the key). The help's 'IN THIS PROMPT' section and TestHelpKeysAreBound must list F1 for help there. 0gztsy rebuilds the answer box as a multi-line textarea: the rule must hold there too.

## Depends on

## Acceptance
- [ ] In each text input, typing 'done by morning?' leaves exactly that text in the field and help stays closed (test per input, fails first on main)
- [ ] F1 in a text input opens help with the typed text intact underneath; Esc or F1 returns to it (test)
- [ ] Outside text input ? still opens help (existing tests pass)

## Notes


## History
- 2026-11-18T02:03 todo -> doing/active (take)
