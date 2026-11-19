---
id: x8zzzn
title: "Prototype: recall memories delivered into the dialogue by harness notify"
state: todo
type: spike
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzt"
created: 2026-11-19
updated: 2026-11-19
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
- [ ] Step 1 counts in a note: recall blocks parked vs delivered vs dropped per turn for 3 days, and whether mid-turn pasted messages get recall
- [ ] Prototype behind a config switch delivers the block into the dialogue on jcode in the same turn, idle sessions never woken for a memory alone (test)
- [ ] Per-harness table of push availability in a note, with sources
- [ ] Jørgen runs with the switch on for a day and decides whether it replaces the old path

## Notes
