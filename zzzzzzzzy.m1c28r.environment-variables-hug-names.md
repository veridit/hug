---
id: m1c28r
title: "Environment variables: HUG_ names, old HUGINN_/MUNINN_ still read"
status: todo
type: chore
priority: "medium"
assignee: []
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Every documented environment variable is HUG_* (HUG_USER, HUG_HOME, HUG_NO_EMBED, ...); the old HUGINN_*/MUNINN_* names keep working as fallbacks, the same way huginn.toml became hug.toml

## Purpose
Users type hug; help that says HUGINN_USER or MUNINN_HOME teaches the old names (Jørgen 2026-11-16: 'HUGINN -> HUG?')

## Principle
New name wins when both are set; old names are read silently, never documented; tests and internal-only variables (HUGINN_UPDATE_GOLDEN, MUNINN_CAPTURE_KILL_*) may move too but users never need them. One shared lookup helper, not scattered os.Getenv calls.

## Relevant details
Counted 2026-11-16 in the repo: MUNINN_HOME 40 uses, HUGINN_USER 18, MUNINN_NO_EMBED 9, HUGINN_UPDATE_GOLDEN 7, MUNINN_FULL_SPEED 5, plus 11 rarer ones. AGENTS.md and the golden instructions name HUGINN_UPDATE_GOLDEN. et85h1 (identity) should read HUG_USER first.

## Depends on

## Acceptance
- [ ] hug task take with HUG_USER=x assigns x; HUGINN_USER still works; HUG_USER wins when both are set (test)
- [ ] HUG_HOME and MUNINN_HOME behave the same way for hug mem (test)
- [ ] help, README, AGENTS.md and errors name only HUG_* (grep shows no HUGINN_/MUNINN_ in user-facing text)

## Notes
