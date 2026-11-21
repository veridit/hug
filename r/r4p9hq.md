---
id: r4p9hq
title: "Release v0.7.95: the focus \u003e pulses, text cursors blink, one thing moves"
state: todo
type: chore
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-21
updated: 2026-11-21
---

## Goal
Jørgen confirms v0.7.95: the > where you are pulses like a cursor, and wherever you type an underscore blinks instead.

## Purpose
An underlined, still > read as a key to press; he wants one moving mark that says where he is.

## Principle
One question per shipped task; on/off frames attached so it can be judged without installing.

## Relevant details
Shipped: q3sh69 checks 1, 2, 4 (88724001..dc737984, CI 44906215953 8/8). The > (and ⋗ on a marked row) turns from bold accent to dim and back every 530 ms; no underline. While typing (answer box, /, :, #, filter field) the > is still and the text cursor blinks as an underline. Idle CPU of hug task tui: about 0.2% before, 0.6-2.4% after (one tick redraws one line). Try it: brew upgrade veridit/tap/hug, hug task tui, open a task asked of you.

## Depends on

## Acceptance
- [ ] Jørgen confirms the pulse draws his eye and only one thing moves

## Notes

file:list-on.ansi (2841 bytes, sha256:fd29f725) list-on

file:list-off.ansi (2840 bytes, sha256:f9ba6605) list-off
