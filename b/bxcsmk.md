---
id: bxcsmk
title: "Blinking \u003e for the currently focused row"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "00d"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
Blinking > for the currently focused row

## Depends on

## Acceptance
- [ ] Verify Blinking > for the currently focused row

## Requests
- 2026-11-20T13:10 request 1 by jhf:
  I think I've written this before. Maybe you have another task for it, but it's still not done. And with the wraparound, the other ticket I wrote, so you can get no column focused. I think that whenever there is something I can type, there's a blinking underscore or glowing underscore. And whenever there's something that's just selected that where I'm currently at, there is a blinking greater than
  resolved 2026-11-20T13:10 by crab:
    You did write it before: it is q3sh69 (high priority, first in the TUI queue): the > on the focused row blinks like a cursor, with no underline, in lists and in a task's page. I added your general rule there: wherever you can type, a blinking underscore; wherever you are just positioned, a blinking >; never both at once. Your screenshot is attached to q3sh69. This ticket is dropped as a duplicate. The no-column step when wrapping is hktdgs, handled next.

## Notes

file:screen-1309.ans (13756 bytes, sha256:ea041843) screen capture from hug tui


## History
- 2026-11-20T13:10 request 1 by jhf
- 2026-11-20T13:10 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T13:10 request 1 acknowledged by crab
- 2026-11-20T13:10 resolved request 1 by crab
