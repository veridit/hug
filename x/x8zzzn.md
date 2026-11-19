---
id: x8zzzn
title: "Prototype: recall memories delivered into the dialogue by harness notify"
state: "doing"
type: spike
priority: ""
assignee: ["palmtree"]
labels: []
group: ""
rank: "zzzzzzzzzzzg"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T15:02"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A prototype, switchable per harness, delivers recalled memories straight into the agent's dialogue by the harness's own notify path (jcode notify_session, as hug task delivery gmzd87 does), instead of parking them for the first tool call. Measured against the current path; if it works it replaces it on harnesses that support it, with the hook path as the fallback where notify is not available.

## Purpose
Jørgen 2026-11-19 09:38-09:44Z (crab session): he sees very few memories come back recently, and answers pasted into the console interact with the running turn; memories injected the way hug task answers arrive would go straight into the dialogue. Evidence found 09:44Z: on jcode, turn_start parks the recall block and only the turn's first successful tool call prints it (muninn/internal/harness/harness.go header, park.go); turn_end drops what no tool call took (z9q42h). A conversational turn with no tool call therefore never shows its memories. Injections: 699 on 11-18, 93 on 11-19 by 09:38Z (18 sessions); crab's session 572 since 11-04, deduped per (session, content_hash).

## Principle
Prototype behind a switch, measured before switching over; recall itself stays local and free. Delivery must never start an unprompted turn on an idle session for a memory alone, and never cut into the person's input: a memory rides with the message it belongs to (soft interrupt in the same turn). Per harness: use its native push if it has one, else the existing hook path. No change to what is recalled, only how it arrives.

## Relevant details
Step 1 (evidence, no code): from capture/hook logs and injections, count for the last 3 days how many turns had a recall block parked and dropped by turn_end (no tool call), against delivered; and whether a soft-interrupt message (pasted mid-turn) runs turn_start recall at all. Step 2: prototype on jcode: on turn_start recall, deliver the block by notify_session to the same session (deliver.go), flag in config, default off. Step 3: per-harness availability table: jcode (notify_session socket), Claude Code and Codex (UserPromptSubmit additionalContext already lands before the answer, so nothing to do?), others. Related: z9q42h (park), gmzd87 (deliver.go), memory 5kxgm9.

## Depends on

## Acceptance
- [x] Step 1 counts in a note: recall blocks parked vs delivered vs dropped per turn for 3 days, and whether mid-turn pasted messages get recall
- [ ] Prototype behind a config switch delivers the block into the dialogue on jcode in the same turn, idle sessions never woken for a memory alone (test)
- [ ] Per-harness table of push availability in a note, with sources
- [ ] Jørgen runs with the switch on for a day and decides whether it replaces the old path

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
## History
- 2026-11-19T15:02 todo -> doing/active (take)
