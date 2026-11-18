---
id: gpyqq9
title: "ask is refused by the shipped-not-done finding that asking would settle"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzq"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
hug task ask on a shipped task whose remaining decision belongs to a person succeeds and settles the shipped-not-done finding, instead of being refused because of it.

## Purpose
Crab tried to ask Jørgen whether 4vd1mz (shipped, all checks ticked, owner decision pending) is done; ask refused with 'not ready to hand to jhf: shipped-not-done ... fix: hug task done 4vd1mz', and wait refused too, pointing back to ask. No command records 'the owner decides'.

## Principle
A finding whose fix is a person's decision must be settleable by asking that person. The readiness gate for ask excludes questions ask itself answers (stageSaysWhy already silences shipped-not-done once asked).

## Relevant details
questions.go noteQuestion: case shipped != nil && stageSaysWhy(t) returns nil, so asked would clear it; the ask command's readiness check runs lint before the stage changes. Repro: 4vd1mz at 2026-11-18 11:56Z.

## Depends on

## Acceptance
- [ ] hug task ask on a shipped, all-ticked task succeeds, sets stage asked, and lint no longer reports shipped-not-done for it; a test fails first on main.
- [ ] Other broken findings (e.g. missing-file) still block ask (test).

## Notes
