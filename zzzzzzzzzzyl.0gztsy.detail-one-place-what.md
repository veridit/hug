---
id: 0gztsy
title: "Detail: one place for what a task asks of you; open with focus on it"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-18T01:39"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "t64v7d lands first (crab 00:53); 0gztsy WIP on nautilus/0gztsy at 0109717e"
waiting_since: "2026-11-18T02:03"
waiting_asked_by: "nautilus"
---

## Goal
A task's detail shows what it waits on you for once, not three times. The header keeps one stage line ('└─ waiting on you: ... since ...'). The separate '? waiting:' question block goes when the wait names a check: that check in the acceptance list carries the question marker instead, looks exactly as it does in the list (same box, number, text), and the detail opens with focus on it, so Space ticks it and 'a' answers. A wait that names no check keeps one question block, styled like an acceptance row, and gets focus on open. The lint line for a person-wait is gone via wdkqzx.

## Purpose
Jørgen 2026-11-17 23:27 local, on 6ephqk's detail: the waiting line, a '⚠ line 1: shipped, but doing ... fix: hug task done 6ephqk' lint line, and a '? waiting: check 9: look at the state/stage rows ... asked by nautilus · 3 h ago · a answer' block all say the same thing at the top, and overlap with '□ 9. Jørgen confirms the look on his board after release' further down. 'This is crowded ... would be better served with that in focus when I open, or this can be structured better and receive focus to jump down ... check 9 should look the same both places.'

## Principle
One fact, one place; the cursor starts where the reader's answer goes. No new key: Space and a already act on a focused check. Nothing writes on open.

## Relevant details
Mock, 6ephqk opened:
  Task state and stage: a hard-coded tree replaces status
  [doing] [feature] 6ephqk  group:root  assigned:nautilus · claude-opus-5-5/high · main
  └─ waiting on you: check 9 · since 19:50 · asked by nautilus
  ...
  ACCEPTANCE 8/9
    ■ 1. ...
  > ? □ 9. Jørgen confirms the look on his board after release      ← focus on open; Space ticks, a answers
 How the wait names a check: waiting_on text starting 'check N' (as agents write it today) or a field set by hug task wait --check N; pick one and say which. Code: huginn/internal/tui detail (stage line, question block from 3jtfw9, focus on open from ny39q0-era logic).

## Depends on

## Acceptance
- [ ] a task whose wait names check N opens with focus on check N, which shows the ? marker and the same text as the list; no separate question block (test + golden)
- [ ] a wait that names no check shows one question block styled like an acceptance row, focused on open; Space/a act on it (test)
- [ ] ~~the detail of a task waiting on you shows the wait once in the header and no lint line for it (golden at 80 and 140)~~ (not relevant: replaced by check 5: header-spec says the question is not in the header (crab 01:06 ruling), nautilus, 2026-11-18)
- [ ] Jørgen opens 6ephqk and says it reads clearly

## Notes

crab 2026-11-17 23:28 ruling on how a wait names its check: field waiting_check: N set by hug task wait --check N (refuses a bad N with a fix line; MCP task_update gets check). No read-time fallback (no backward compatibility): hug upgrade's task part moves an exact leading 'check N:' with a real N into waiting_check (preview, then --apply, like waiting_for). The TUI reads only the field. hug task wait prints a hint when the reason starts with 'check N' and no --check. After release: run hug upgrade on hug's own board, preview first (14 of 23 waits start 'check N').

Correction (supersedes my earlier note recording crab's waiting_check ruling): kestrel's 41hhgn adds an 'asked' doing stage with field asked_check, and the ? glyph goes away. Jørgen 00:12/00:13 via crab: the overview gets 'ASKED · for you [N]' at the very top, above IN PROGRESS; WAITING keeps only no-person waits and questions asked of others. ASKED rows show the title like every row; only the focused row adds a second line '└ asked: check 9 · <question> · nautilus · 3 h' (header-spec.md 'Overview sections' has the picture). 0gztsy's detail rendering waits for 41hhgn to land. (nautilus)


## History
- 2026-11-18T01:39 todo -> doing/active (take)
- 2026-11-18T02:03 doing/active -> doing/waiting (wait)
