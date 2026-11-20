---
id: x8zzzn
title: "Prototype: recall memories delivered into the dialogue by harness notify"
state: "doing"
type: spike
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzd"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T15:02"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
parent: "ypfrp3"
---

## Goal
A per-harness prototype delivers recalled memories into the dialogue by the harness's notify path, measured against today's.

## Purpose
Jørgen 2026-11-19 09:38-09:44Z (crab session): he sees very few memories come back recently, and answers pasted into the console interact with the running turn; memories injected the way hug task answers arrive would go straight into the dialogue. Evidence found 09:44Z: on jcode, turn_start parks the recall block and only the turn's first successful tool call prints it (muninn/internal/harness/harness.go header, park.go); turn_end drops what no tool call took (z9q42h). A conversational turn with no tool call therefore never shows its memories. Injections: 699 on 11-18, 93 on 11-19 by 09:38Z (18 sessions); crab's session 572 since 11-04, deduped per (session, content_hash).

## Principle
Prototype behind a switch, measured before switching over; recall itself stays local and free. Delivery must never start an unprompted turn on an idle session for a memory alone, and never cut into the person's input: a memory rides with the message it belongs to (soft interrupt in the same turn). Per harness: use its native push if it has one, else the existing hook path. No change to what is recalled, only how it arrives.

## Relevant details
A prototype, switchable per harness, delivers recalled memories straight into the agent's dialogue by the harness's own notify path (jcode notify_session, as hug task delivery gmzd87 does), instead of parking them for the first tool call. Measured against the current path; if it works it replaces it on harnesses that support it, with the hook path as the fallback where notify is not available.

Step 1 (evidence, no code): from capture/hook logs and injections, count for the last 3 days how many turns had a recall block parked and dropped by turn_end (no tool call), against delivered; and whether a soft-interrupt message (pasted mid-turn) runs turn_start recall at all. Step 2: prototype on jcode: on turn_start recall, deliver the block by notify_session to the same session (deliver.go), flag in config, default off. Step 3: per-harness availability table: jcode (notify_session socket), Claude Code and Codex (UserPromptSubmit additionalContext already lands before the answer, so nothing to do?), others. Related: z9q42h (park), gmzd87 (deliver.go), memory 5kxgm9. Step 2 scope (crab 19:05Z, from STEP1.md): normal turns already get their block (404/436 on 11-18, 100/107 on 11-19); the loss is messages typed mid-turn (soft interrupts): 0 of 123 got recall, and 8 of 13 stale drops come from a paste overtaking the park. Step 2 builds, behind config recall.jcode_midturn (default off): on post_tool_feedback, if the session journal holds user messages newer than the parked turn's message and not yet recalled, recall each and deliver its block in that tool result, never dropping the parked block as stale because of a paste (deliver both). No new daemon; idle sessions never woken for a memory alone. Plus logging: injections.delivered_at (or a delivered table), a hook.log line when a park is overwritten, session id on hook timeout lines. When jcode adds a soft-interrupt hook (maintainer asked 14:30Z), switch to it.

## Depends on

## Acceptance
- [x] Step 1 counts in a note: recall blocks parked vs delivered vs dropped per turn for 3 days, and whether mid-turn pasted messages get recall
- [x] Prototype behind a config switch delivers the block into the dialogue on jcode in the same turn, idle sessions never woken for a memory alone (test)
- [ ] Per-harness table of push availability in a note, with sources
- [ ] ~~Jørgen runs with the switch on for a day and decides whether it replaces the old path~~ (not relevant: moved to its release ticket ypfrp3 (v0.7.85) as an ask, per 75mxf3 trial, crab, 2026-11-19)

## Notes


check 1: Step 1 in /Users/jhf/.jcode/scratch/x8zzzn/STEP1.md: since park went live 2026-11-18T03:53Z, 543 parked, 504 delivered, 18 dropped at turn_end (no tool), 13 dropped stale, 8 unresolved; mid-turn soft interrupts never run turn_start recall (jcode 446302f1b fires it only at turn_execution.rs:66/215/240; park era 103 of 104 mid-turn soft interrupts and 19 of 19 point-B ones have no recall line)
Step 1 table (palmtree, hook.log + journals + jcode server logs, UTC days; full report /Users/jhf/.jcode/scratch/x8zzzn/STEP1.md):
day | turn_start runs | parked | delivered | dropped turn_end (no tool) | dropped stale | unresolved | old transcript sent/failed/turn-ended | no block (no hits/harness notice/own injection)
11-16 | 931 | 0 | 0 | 0 | 0 | 0 | 225/95/55 | 556 (338/147/71)
11-17 | 678 | 0 | 0 | 0 | 0 | 0 | 181/56/18 | 423 (289/99/35)
11-18 | 1038 | 436 | 404 | 17 | 9 | 6 | 33/9/0 before 03:53Z | 560 (420/140/0)
11-19 to 14:15Z | 229 | 107 | 100 | 1 | 4 | 2 | 0 | 122 (63/59/0)
Parked includes 65 cue-only blocks (hits=0). Journal check of 498 parks: 444 delivered with the marker in a tool result, 15 delivered lines without it (11 tool=mcp), 0 marker without a delivered line. 8 of 13 stale drops: a soft interrupt became the latest typed message before the first tool result (harness.go Current()). Unresolved: 6 parked before the 03:53Z config rewrite, 2 still on disk (pawprint, turtle: system turns, 0 tools, no turn_end line).
Per harness: all 1876 injections since 11-16 are jcode; 0 claude/codex hook.log lines (last claude injection 2026-11-14). Per instance (11-18/19): global 314 parked/290 delivered/9 no-tool/7 stale/8 unresolved; brovs 116/110/2/4/0; seid 61/59/1/1/0; statbus 52/45/6/1/0.
Mid-turn: soft interrupts are added by inject_soft_interrupts (interrupts.rs:347, points B/C/D) and never call fire_turn_start_hook. Park era: mid-turn soft interrupts 1 recall line / 103 none; point-B soft interrupts 0/19; typed turn inputs 880/1.
Gaps: injections is written at recall time, so it cannot tell delivered from dropped (would need delivered_at); Park overwrites are not logged; soft interrupts are joined by timestamp, because jcode logs no message id at INJECT_COMMIT; hook timeouts carry no session id.
crab 14:31Z, review of step 1 (palmtree, STEP1.md): spot-checked hook.log for 11-19 (local +01:00 stamps): 107 'delivered' lines, matching the table. Reading: on the park path (since 11-18 03:53Z) most blocks are delivered (404/436, 100/107); dropped for lack of a tool call is small (17, 1). The real loss is mid-turn: a message pasted into a running turn gets no recall at all (103/104 after a tool call, 19/19 after an answer; jcode fires turn_start only at turn start, interrupts.rs:347 inject_soft_interrupts never calls it), and 8 of 13 stale drops come from a paste overtaking the parked turn. So step 2 targets pasted messages: recall each soft-interrupted message and deliver its block into the same turn. That needs jcode to tell muninn about injected messages (a hook on soft interrupt, or turn_start-like event with the message id) — a jcode change, filed to the maintainer; until then hug can only recall on the next tool call by reading the journal for messages newer than the parked one. Logging gaps 1-5 (delivered_at in injections, park overwrite line, session id on timeouts) are cheap and go into step 2.
check 1: STEP1.md (palmtree): parked/delivered/dropped per day; mid-turn pastes get no recall (103/104, 19/19); crab spot-check 107 delivered on 11-19 matches
check 2: fix 9073b532 (tests first 26ce3295) on branch x8zzzn, ci/x8zzzn; CI run 44610132775 attempt 2 success on 9073b532, 8/8 jobs (attempt 1 ubuntu: unrelated bubbles data race in huginn/internal/cli, filed 5xaxd6). recall.jcode_midturn (default off): post_tool_feedback recalls messages typed into the running turn (jcode model_usage_turn_id), parked block kept and printed first; nothing printed outside tool results. TestJcodeMidturnRecallIsByteIdentical: 'byte-identical: 902 bytes through both paths'. injections.delivered_ns + migration, hook.log replaced/midturn/missed lines, session+event on hook errors. Report /Users/jhf/.jcode/scratch/x8zzzn/REPORT2.md
crab 20:53Z: reviewed (midturn.go reads the journal from the park's offset by jcode's model_usage_turn_id; parked block delivered first, no stale drop on a paste; gated by Cfg.JcodeMidturn(), default off; injections.delivered_ns via addIntColumn migration; byte-identical test 902 bytes). CI 44610132775 attempt 2 8/8 at 9073b532 (attempt 1: bubbles runeutil data race in huginn/internal/cli, filed 5xaxd6). Main unmoved: fast-forwarded to 9073b532. Release v0.7.85: CHANGELOG ccf8641f, merge 97d2de73, release.yml 44613144851. Check 4 (Jørgen runs with the switch a day) goes to the v0.7.85 release ticket.
landed 9073b53 on main 2026-11-19: muninn: jcode mid-turn recall behind recall.jcode_midturn, delivered_ns and hook…
shipped in v0.7.85 (9073b53)
## History
- 2026-11-19T15:02 todo -> doing/active (take)
