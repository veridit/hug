---
id: m1c28r
title: "Environment variables are HUG_*; hug upgrade migrates, no old names"
status: "doing"
type: chore
priority: "medium"
assignee: ["dragon"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Every environment variable is HUG_* (HUG_USER, HUG_HOME, HUG_NO_EMBED, ...). The old HUGINN_*/MUNINN_* names are gone from the code. hug upgrade rewrites any place hug itself wrote them (installed hooks, harness configs, its own config).

## Purpose
Users type hug; help that says HUGINN_USER or MUNINN_HOME teaches the old names (Jørgen 2026-11-16: 'HUGINN -> HUG?')

## Principle
No backwards compatibility (Jørgen 2026-11-16: 'You upgrade, you run the latest and greatest'). The code reads only HUG_*; hug upgrade is the one migration step and moves everything hug owns. Test-only variables are renamed too. One shared lookup helper, not scattered os.Getenv calls. If an old name is set in the environment, hug says once which new name to use, with a fix: line.

## Relevant details
Counted 2026-11-16 in the repo: MUNINN_HOME 40 uses, HUGINN_USER 18, MUNINN_NO_EMBED 9, HUGINN_UPDATE_GOLDEN 7, MUNINN_FULL_SPEED 5, plus 11 rarer ones. AGENTS.md and the golden instructions name HUGINN_UPDATE_GOLDEN. et85h1 (identity) should read HUG_USER first.

## Depends on

## Acceptance
- [ ] hug task take with HUG_USER=x assigns x (test); grep finds no HUGINN_/MUNINN_ in Go code
- [ ] hug upgrade rewrites old names in hooks and configs hug installed (test on a fixture home)
- [ ] with MUNINN_HOME or HUGINN_USER set, hug prints the new name to use and a fix: line (test)
- [ ] help, README, AGENTS.md and errors name only HUG_* (grep shows no HUGINN_/MUNINN_ in user-facing text)

## Notes
