---
id: w4qmr1
title: "Recall skips jcode image notices: exact prefixes from jcode's source"
state: "doing"
type: feature
priority: ""
assignee: ["mizaru"]
labels: []
group: ""
rank: "zzzzzzzzzzzz"
created: 2026-11-19
updated: 2026-11-19
parent: "d634kr"
doing_since: "2026-11-19T02:24"
doing_with: "p277bp"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Recall injects nothing for jcode's own image notices (screenshot-omitted, attached-image and image-omitted markers sent in the user role), matched by an exact listed prefix taken from jcode's source; every other message, including one that only starts with '[', recalls as today

## Purpose
Discovery (ta306h A0, 2026-11-19, 240 fresh short messages from 2026-11-16..18, blind judge): 33 of the 240 'short typed' messages are jcode image notices in 6 sessions; their shipped-recall hits are 1 relevant and 24 irrelevant, 23% of all irrelevant hits on the set (103) for 1 relevant (of 163). recall.IsHarnessNotice (muninn/internal/recall/recall.go:107) already skips '**Background task**', swarm awaits, schedules, '[auto] ' and system reminders, but not these. This set found the lead, so it cannot confirm it (crab 01:22): the bar is checked on a new fresh set built after the rule is written.

## Principle
Exact, listed prefixes only, each cited to jcode's source (file:line) and counted verbatim in transcripts; no fuzzy match, no regex on '['. A skip only withholds recall for that message; it never touches memories. Recall stays local and free. Ships only under the recall rule (AGENTS.md): pre-registered, blind-judged, on a fresh set built after the rule is written; 0 paid calls beyond the judge. Separately (not a recall change): recall4/build_fresh.py's NOTICE filter learns the same prefixes so later fresh sets leave them out; ta306h's frozen set stays as it is.

## Relevant details
Prefixes (jcode at /Users/jhf/Projects/jcode 28ac5e603, 2026-11-04) with verbatim counts among user-role text messages in the six recall3 homes, all dates (scratch/ta306h/notice_counts.py): '[Older tool screenshot omitted from history to bound transcript size: ' crates/jcode-compaction-core/src/lib.rs:882, 1286; '[Attached image associated with the preceding tool result: ' crates/jcode-app-core/src/agent/tools.rs:117 (also jcode-base/src/provider/image_clamp.rs:44, session/render.rs:274), 344; '[Image omitted during request-size recovery: ' jcode-compaction-core/src/lib.rs:802, 10; '[Image omitted during emergency context recovery: ' lib.rs:655, 0; '[Image omitted: the provider rejected an image in this conversation ' lib.rs:705, 0; '[Image omitted: this provider/model does not support image input; ' jcode-base/src/provider/image_clamp.rs:110, 0. Not prefixes of these (must still recall): '[image 1]' 84 (a user's pasted image label), '[transcription] ...' (voice input), typed text starting with '['. Where: harnessNotices in muninn/internal/recall/recall.go:107 (Skipped: 'harness notice'). Discovery data: scratch/ta306h/{replay.json,judgments.json,handread-orangutan-fox.md}, scratch/recall4/real/ta306h/a0.jsonl.

## Depends on

## Acceptance
- [ ] harnessNotices lists exactly the cited prefixes; a unit test per prefix (the jcode string with a sample payload is skipped as 'harness notice') and near-miss tests that still recall: '[image 1] what is this?', '[transcription] the screenshot looks wrong', '[Older screenshot] see above'
- [ ] recall4/build_fresh.py NOTICE filter includes the same prefixes (later fresh sets exclude them); ta306h's set unchanged
- [ ] PREREG with sha in a note before any paid call; new fresh set built after the rule is written (messages after the rule's commit, deduped by text against earlier sets, >= 8 sessions); shipped vs rule: irr/msg drops (upper 95% CI < 0) and rel/msg does not drop (CI includes 0), blind judge, 0 paid calls beyond the judge

## Notes


## History
- 2026-11-19T02:24 todo -> doing/active (take with p277bp)
