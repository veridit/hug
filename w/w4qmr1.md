---
id: w4qmr1
title: "Recall skips jcode image notices: exact prefixes from jcode's source"
state: "doing"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzu8"
created: 2026-11-19
updated: 2026-11-19
parent: "d634kr"
doing_since: "2026-11-19T02:24"
doing_with: "p277bp"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_asked_by: "crab"
waiting_until: "2026-11-26"
waiting_on: "fresh notice data: daily count at 01:45Z needs \u003e= 40 notices from \u003e= 8 sessions (1 in 1 on 11-20); on 11-26 crab asks jhf about the floor"
waiting_since: "2026-11-20T09:24"
---

## Goal
Recall injects nothing for jcode's own image notices, matched by exact prefixes taken from jcode's source.

## Purpose
Discovery (ta306h A0, 2026-11-19, 240 fresh short messages from 2026-11-16..18, blind judge): 33 of the 240 'short typed' messages are jcode image notices in 6 sessions; their shipped-recall hits are 1 relevant and 24 irrelevant, 23% of all irrelevant hits on the set (103) for 1 relevant (of 163). recall.IsHarnessNotice (muninn/internal/recall/recall.go:107) already skips '**Background task**', swarm awaits, schedules, '[auto] ' and system reminders, but not these. This set found the lead, so it cannot confirm it (crab 01:22): the bar is checked on a new fresh set built after the rule is written.

## Principle
Exact, listed prefixes only, each cited to jcode's source (file:line) and counted verbatim in transcripts; no fuzzy match, no regex on '['. A skip only withholds recall for that message; it never touches memories. Recall stays local and free. Ships only under the recall rule (AGENTS.md): pre-registered, blind-judged, on a fresh set built after the rule is written; 0 paid calls beyond the judge. Separately (not a recall change): recall4/build_fresh.py's NOTICE filter learns the same prefixes so later fresh sets leave them out; ta306h's frozen set stays as it is.

## Relevant details
Recall injects nothing for jcode's own image notices (screenshot-omitted, attached-image and image-omitted markers sent in the user role), matched by an exact listed prefix taken from jcode's source; every other message, including one that only starts with '[', recalls as today

Prefixes (jcode at /Users/jhf/Projects/jcode 28ac5e603, 2026-11-04) with verbatim counts among user-role text messages in the six recall3 homes, all dates (scratch/ta306h/notice_counts.py): '[Older tool screenshot omitted from history to bound transcript size: ' crates/jcode-compaction-core/src/lib.rs:882, 1286; '[Attached image associated with the preceding tool result: ' crates/jcode-app-core/src/agent/tools.rs:117 (also jcode-base/src/provider/image_clamp.rs:44, session/render.rs:274), 344; '[Image omitted during request-size recovery: ' jcode-compaction-core/src/lib.rs:802, 10; '[Image omitted during emergency context recovery: ' lib.rs:655, 0; '[Image omitted: the provider rejected an image in this conversation ' lib.rs:705, 0; '[Image omitted: this provider/model does not support image input; ' jcode-base/src/provider/image_clamp.rs:110, 0. Not prefixes of these (must still recall): '[image 1]' 84 (a user's pasted image label), '[transcription] ...' (voice input), typed text starting with '['. Where: harnessNotices in muninn/internal/recall/recall.go:107 (Skipped: 'harness notice'). Discovery data: scratch/ta306h/{replay.json,judgments.json,handread-orangutan-fox.md}, scratch/recall4/real/ta306h/a0.jsonl.

## Depends on

## Acceptance
- [x] harnessNotices lists exactly the cited prefixes; a unit test per prefix (the jcode string with a sample payload is skipped as 'harness notice') and near-miss tests that still recall: '[image 1] what is this?', '[transcription] the screenshot looks wrong', '[Older screenshot] see above'
- [x] recall4/build_fresh.py NOTICE filter includes the same prefixes (later fresh sets exclude them); ta306h's set unchanged
- [ ] PREREG with sha in a note before any paid call; new fresh set built after the rule is written (messages after the rule's commit, deduped by text against earlier sets, >= 8 sessions); shipped vs rule: irr/msg drops (upper 95% CI < 0) and rel/msg does not drop (CI includes 0), blind judge, 0 paid calls beyond the judge

## Notes


Code half: commit 163321cb on ci/w4qmr1 (run 44436965382, pending). 6 prefixes in harnessNotices (recall.go:110-117) with jcode 28ac5e603 file:line and transcript counts. muninn/scripts/build_fresh.py NOTICE has the same 6, held by TestBuildFreshFiltersEveryHarnessNotice (fails without the script change: 6 'NOTICE lacks' errors). TestRecallSkipsJcodeImageNotices failed first (6/6 recalled) and pins 4 near misses that must still recall, including a notice quoted mid-message. Mutants 7/7 killed (each prefix dropped: m0-m5; HasPrefix to Contains: m6): /Users/jhf/.jcode/scratch/w4qmr1/mut/results.txt. Measurement PREREG draft: /Users/jhf/.jcode/scratch/w4qmr1/PREREG-DRAFT.md (no set built; needs messages after 163321cb).
CI 44436965382 on 163321cb: completed success, 8/8 jobs (secrets, bench, test ubuntu/macos/windows, test-windows-cli rest/sync/shards). Check 1: recall.go:110-117 lists the 6 cited prefixes; TestRecallSkipsJcodeImageNotices asserts Skipped='harness notice' and no block for each, and that '[image 1] ...', '[transcription] ...', '[Older screenshot] ...' and a mid-message quote still recall. Check 2: repo muninn/scripts/build_fresh.py NOTICE has the 6 (scratch/recall4/build_fresh.py synced, cmp identical); ta306h replay.json untouched. PREREG draft sha c739... is superseded by the bar edit aligned to check 3; final sha goes in a note after crab's amendments.
Crab 01:38: stays OFF main until check 3 passes; ci/w4qmr1 frozen at 163321cb. Fresh window: >= 2026-11-19T00:00Z (ta306h set ends 2026-11-18T21:02Z), deduped against recall3, recall4/real/fresh, recall4/real/tool, ta306h. Daily count, scratch/w4qmr1/count_notices.py: 2026-11-19T01:39Z: 0 notices in 0 sessions (need >= 40 from >= 8). Script check with the cut moved to 11-16: 129 notices in 6 top-level sessions, 35 dedup hits, so it reads them. PREREG draft amended: 235489860758.
Daily count runbook (crab runs it from 2026-11-20; mizaru's wake cancelled). Command: nice -n 19 python3 /Users/jhf/.jcode/scratch/w4qmr1/count_notices.py. It is read-only and takes ~25 s. It prints one line: '<UTC now> since 2026-11-19T00:00Z: top-level N notices in S sessions; child C in K; deduped away D; ready=<bool>'. Note it with: hug task note w4qmr1 "daily count: <line>". ready=True means N >= 40 and S >= 8. Then freeze /Users/jhf/.jcode/scratch/w4qmr1/PREREG-DRAFT.md as PREREG.md (shasum -a 256), note the sha, and run it: A0 = 163321cb^, A1 = 163321cb on ci/w4qmr1, store copy niced, concurrency 2, judge cap 40. Accepted draft sha: 235489860758b07e4e5ccf9201c3669fdf942adb467a71a3f3db6adf52301dee.
Daily count 2026-11-20T01:45Z (window since 2026-11-19T00:00Z, deduped per PREREG-DRAFT.md): top-level 1 notice in 1 session; child 0 in 0; deduped away 0; ready=False (needs >= 40 notices from >= 8 sessions). Next count 11-21 01:45Z; floor question to Jørgen if sessions < 8 by 11-26.
daily count 2026-11-21 01:45Z (count_notices.py, window >= 2026-11-19T00:00Z, deduped per PREREG-DRAFT.md): 6 notices in 4 sessions (child 0, deduped away 0); bar is 40 from 8 sessions: not ready. 11-20 01:45Z was 1 in 1.
crab 2026-11-22 01:46Z daily count: 42 notices in 5 top-level sessions (child 0, deduped 0); ready=False (needs >= 8 sessions). Per session: statbus rabbit 19, statbus hibiscus 16, brovs kikazaru 4, crab 2, brovs sheep 1. The count passed 40 but 35 of 42 (83%) come from two statbus sessions, so the session floor is what holds it back. History: 11-20 1/1, 11-21 6/4, 11-22 42/5. If still under 8 sessions on 11-26, ask Jørgen about the floor.
crab 2026-11-23 02:15Z daily count (since 2026-11-19T00:00Z): top-level 77 notices in 6 sessions; child 0; deduped away 2; ready=False (needs 8 sessions). Per session: statbus hibiscus 42, statbus rabbit 19, brovs fox 7, brovs kikazaru 6, crab 2, brovs sheep 1. Two statbus sessions give 61 of 77. Next count 11-24 01:45Z; if still under 8 sessions on 11-26, ask Jørgen about the floor.
crab 14:12Z 11-24 (the 01:45Z run never fired): 228 notices in 8 sessions since 2026-11-19T00:00Z, 2 deduped away, ready=True. Per session: statbus hibiscus 124, brovs sheep 34, statbus rabbit 24, brovs bear 20, brovs fox 13, brovs kikazaru 10, crab 2, blowfish 1. hibiscus is 54% of the set. Threshold met (>= 40 from >= 8 sessions). The measurement worker waits for load <= 10 (212 at 14:12Z), after vrx1ja.
## History
- 2026-11-19T02:24 todo -> doing/active (take with p277bp)
- 2026-11-19T04:22 handed to crab: mizaru's session ended (closed 2026-11-19 03:11)
- 2026-11-20T09:24 doing/active -> doing/waiting (wait until 2026-11-26)
