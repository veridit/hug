---
id: 9gp0tz
title: "hug task commands accept named flags beside positional arguments"
state: todo
type: feature
priority: "low"
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzu"
created: 2026-11-25
updated: 2026-11-25
---

## Goal
Each hug task command that takes values accepts them as named flags, so a call reads plainly and a misplaced value is refused.

## Purpose
Jørgen 2026-11-25 on dswkgj: 'When you have positional arguments, I think it's quite brittle.'

## Principle
Named over positional for anything beyond the task id; old positional forms keep working, and help shows the flag form first.

## Details
Examples today: hug task check ID N "evidence", hug task answer ID N "text", hug task strike ID N --reason. A swapped or missing N reads ambiguously. Proposal: --check N, --question N, --text; ID stays first. Inventory every command's positional slots first; the MCP tools already take named fields.

## Depends on

## Acceptance
- [ ] An inventory of every positional argument across hug task commands, attached, with the flag each would take
- [ ] The flag forms work and help shows them first; positional forms still work (tests)

## Notes
