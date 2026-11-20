---
id: 0gztsy
title: "Detail: one place for what a task asks of you; open with focus on it"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzkh"
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-18T01:39"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "A task asked of you opens with focus on its question, shown once. Open 6ephqk (or any task in your ASKED): does it read clearly? Your 08:20Z note on 911j70 (the picture line gets no focus and no jump) is filed on 8a0ybb check 7."
asked_for: "jhf"
asked_since: "2026-11-20T09:26"
asked_by: "crab"
asked_check: "4"
---

## Goal
A task's detail shows what it asks of you in one place and opens with focus on it.

## Purpose
Jørgen 2026-11-17 23:27 local, on 6ephqk's detail: the waiting line, a '⚠ line 1: shipped, but doing ... fix: hug task done 6ephqk' lint line, and a '? waiting: check 9: look at the state/stage rows ... asked by nautilus · 3 h ago · a answer' block all say the same thing at the top, and overlap with '□ 9. Jørgen confirms the look on his board after release' further down. 'This is crowded ... would be better served with that in focus when I open, or this can be structured better and receive focus to jump down ... check 9 should look the same both places.'

## Principle
One fact, one place; the cursor starts where the reader's answer goes. No new key: Space and a already act on a focused check. Nothing writes on open.

## Relevant details
A task's detail shows what it waits on you for once, not three times. The header keeps one stage line ('└─ waiting on you: ... since ...'). The separate '? waiting:' question block goes when the wait names a check: that check in the acceptance list carries the question marker instead, looks exactly as it does in the list (same box, number, text), and the detail opens with focus on it, so Space ticks it and 'a' answers. A wait that names no check keeps one question block, styled like an acceptance row, and gets focus on open. The lint line for a person-wait is gone via wdkqzx.

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
- [x] a task whose wait names check N opens with focus on check N, which shows the ? marker and the same text as the list; no separate question block (test + golden)
- [x] a wait that names no check shows one question block styled like an acceptance row, focused on open; Space/a act on it (test)
- [ ] ~~the detail of a task waiting on you shows the wait once in the header and no lint line for it (golden at 80 and 140)~~ (not relevant: replaced by check 5: header-spec says the question is not in the header (crab 01:06 ruling), nautilus, 2026-11-18)
- [ ] Jørgen opens 6ephqk and says it reads clearly
- [x] a task asked of you shows the question once: the context has only a one-word stage badge ([asked], [design], [review]) and no stage line; with a check, the question sits under that check; without, one block at the top of the details, whole and wrapped, never cut; asked of someone else or waiting on a thing, the context line stays (crab 01:06; golden at 80 and 140)

## Questions
- 2026-11-20T09:26 question 1 by crab to jhf (check 4):
  A task asked of you opens with focus on its question, shown once. Open 6ephqk (or any task in your ASKED): does it read clearly? Your 08:20Z note on 911j70 (the picture line gets no focus and no jump) is filed on 8a0ybb check 7.

## Requests
- 2026-11-20T11:28 request 1 by jhf:
  This ticket seems to not have the goal and have a long description, which I talked about before. It doesn't make sense. The only question is that if I scroll down, should the goal or purpose jump into the subject so that I see what this is all about? I don't think so. I think the current view is okay. And I did jump into the question, so this is better.

## Notes

crab 2026-11-17 23:28 ruling on how a wait names its check: field waiting_check: N set by hug task wait --check N (refuses a bad N with a fix line; MCP task_update gets check). No read-time fallback (no backward compatibility): hug upgrade's task part moves an exact leading 'check N:' with a real N into waiting_check (preview, then --apply, like waiting_for). The TUI reads only the field. hug task wait prints a hint when the reason starts with 'check N' and no --check. After release: run hug upgrade on hug's own board, preview first (14 of 23 waits start 'check N').

Correction (supersedes my earlier note recording crab's waiting_check ruling): kestrel's 41hhgn adds an 'asked' doing stage with field asked_check, and the ? glyph goes away. Jørgen 00:12/00:13 via crab: the overview gets 'ASKED · for you [N]' at the very top, above IN PROGRESS; WAITING keeps only no-person waits and questions asked of others. ASKED rows show the title like every row; only the focused row adds a second line '└ asked: check 9 · <question> · nautilus · 3 h' (header-spec.md 'Overview sections' has the picture). 0gztsy's detail rendering waits for 41hhgn to land. (nautilus)


landed 48b1151f..4b93356a (6 commits). CI 44218985753 on 4b93356a: 6/6 success. Local -race tui ok 120.0s, task ok 29.5s; go test ./huginn/... ./cmd/... no failures. Check 1: TestAskedCheckIsFocusedOnOpen (asked_test.go): aaaaaa asked of jhf --check 2 opens with focus on check 2 (detailPane focusAcceptance, acceptance 1), '> ? ☐ 2. b' then the question under it then '└ nautilus · 3 min ago · answer'; the question shows once on screen; Space asks to tick, a opens the answer; golden testdata/detail/asked-check.txt at 80 and 140. On the base (f993e4fb): 'at 80: focus pane 1 check 1, want the checks, check 2'. Check 2: TestAskedWithoutCheckIsOneBlock: a 2-line, 160-char question with no check is one block '> ? which colour …' at the top of the details, wrapped whole (no …), focused (focusQuestion); Space says 'a question has nothing to tick: a answers it'; bar leads with 'answer'; golden asked-block.txt. On the base: 'at 80: focus pane 1, want the question'. Check 3 struck, replaced by 5 (crab 01:06). Check 5: same tests assert the header holds only '[asked]' (no '└─', no question text); asked of someone else / waiting on a thing keep the context line (TestAskedSectionOnTop, stage_test). Goldens answer-100.txt and picture.txt updated.
landed 4e475c8 on main 2026-11-18: tui: ASKED section first, for you, oldest first; empty sections are their title…
landed 3435fc7 on main 2026-11-18: tui: gofmt (0gztsy)
landed 1d260a6 on main 2026-11-18: CHANGELOG: ASKED, the question once, the answer textarea (0gztsy, 41hhgn)
landed 4b93356 on main 2026-11-18: tui: tmux overview goldens with ASKED, from ubuntu CI run 44218017150 (0gztsy)
shipped in v0.7.53 (4e475c8)
file:screen-1457.ans (24745 bytes, sha256:f97b075e) screen capture from hug tui
crab 15:00 local: Jørgen's answer on check 4 (14:58) = focus right, styling too loud (ACCEPTANCE heading competes; question bold+background+colour). Filed as ; check 4 stays open until  ships and he confirms there.
crab 15:01 local: Jørgen's answer on check 4 (14:58): focus right, styling too loud (the ACCEPTANCE heading competes; the question is bold + background + colour). Filed as kg07bt (calm heading, cursor-only emphasis); check 4 stays open until kg07bt ships and he confirms there.
crab 2026-11-20 08:26Z: removed the kg07bt dependency (kg07bt shipped in v0.7.66; its last check is now asked of jhf, so it no longer blocks); this task's own check 4 asked.
## History
- 2026-11-18T01:39 todo -> doing/active (take)
- 2026-11-18T02:03 doing/active -> doing/waiting (wait)
- 2026-11-18T02:59 doing/waiting -> doing/asked (ask)
- 2026-11-18T12:56 doing/asked -> doing/asked (settle: one stage, asked); was waiting_asked_by: nautilus; waiting_on: t64v7d lands first (crab 00:53); 0gztsy WIP on nautilus/0gztsy at 0109717e; waiting_since: 2026-11-18T02:03
- 2026-11-18T14:58 answered asked (check 4), by jhf, for nautilus (doing/asked -> doing/active):
  > after the release: open 6ephqk (or any task asked of you): does the question read clearly, once, on its check, with the cursor there?
  So while the focus was correct it says Jrgens Jrgen opens and says it reads clearly There is too much visual noise on the page specifically the acceptance heading is too drawing too much attention so that it competes with the line that is showing below So I think the acceptance heading has to be maybe bold or something but not the background color so it has to be slightly less dominant and I think the current full color of the question while sensible should rather Yeah, let's try that one because it's both bold and background and strongly colored so it's like too much so I'm wondering if it should be toned down and have something interact like a like the glowing markers like this is where you're at that's the one place you're at rather than trying to do the visual competition
- 2026-11-20T09:26 doing/active -> doing/asked (ask)
- 2026-11-20T11:28 request 1 by jhf
- 2026-11-20T11:28 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
