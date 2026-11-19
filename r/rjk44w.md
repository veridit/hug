---
id: rjk44w
title: "Memory fixes leave a trail: who forgot what, after which recall, and why"
state: "done"
type: feature
priority: ""
assignee: ["bonehound"]
labels: []
group: ""
rank: "zzzzzzzzzzts"
created: 2026-11-18
updated: 2026-11-18
parent: "d634kr"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
Every hug mem forget and memory_remember that corrects a note records, locally, the session, the harness, whether that id was shown in a recall block in the same session and how many turns earlier, and an optional reason (--reason / MCP reason). hug mem stats --fixes prints, per day and per instance, recall blocks shown, fixes of a shown id, fixes the owner asked for versus unprompted, and forgets later undone or contradicted.

## Purpose
s6ff5k item 1 had to be measured by regex over four jcode instances' transcripts, and the answer is still soft: the injections table holds 84 jcode sessions since 2026-11-16 12:00 but only 35 of their transcripts contain the block (the rest are absent or rewritten), and nothing says why a memory was retired. Of the 5 working sessions that forgot a shown id since then, in 3 (orangutan, spider, rabbit) the owner had told the agent to ('Okay, if you see a recalled note that is obsolete, you have to remove it'; 'Doesn't that instruction tell you to?'; 'Remove obsolete memory'). Without a trail we cannot tell header-driven fixes from owner-driven ones, or see a wrong forget at all (H2 made 28 forgets of notes that still held, found only by a blind judge in a replay).

## Principle
Local and free: a row in muninn.db per fix, no model call, no change to what recall shows or ranks. Recording never blocks the write and never asks a question. The reason is optional; absent is recorded as absent. The stats read the trail, not transcripts.

## Relevant details
Today: units has active/updated_ns only; holds_ended has reason but only for holds; injections (session, unit_id, at_ns) already ties ids to sessions, so 'shown before' is a join. Scripts and counts: /Users/jhf/.jcode/scratch/s6ff5k-feedback/forget_stats3.py, trigger.py, REPORT.md cluster C1. 173 units went inactive since 2026-11-16 12:00 with no record of which command, session or reason.

## Depends on

## Acceptance
- [x] hug mem forget ID --reason TEXT and MCP memory_forget reason are stored with session, harness, time and shown-in-session (test with a recall block injected first)
- [x] memory_remember naming an old id (supersedes or reinforce) records the same row (test)
- [x] hug mem stats --fixes prints recall blocks shown, fixes of a shown id, and fixes without a shown id per day and instance; a golden over a fixture store
- [x] A forget whose id is remembered again or reinforced within 7 days is counted as undone in the stats (test)
- [x] No recall output changes: the recall4 frozen replay is byte-identical before and after

## Notes


bonehound 15:00 plan (worktree /Users/jhf/.jcode/scratch/rjk44w from origin/main 832167b8, v0.7.63). Additive schema: new table memory_fixes (CREATE TABLE IF NOT EXISTS, as holds_ended was), no column changes. One row per forget (every surface: Binding.ForgetMany, so CLI, MCP, web, TUI), per remember naming an old id (--reinforce, new optional --supersedes / MCP supersedes: recorded only, it forgets nothing), with session, harness, jcode instance, surface, shown-in-session from the injections table and the time since it was shown, content hash, optional reason (--reason / MCP reason). Recording errors are ignored: never blocks the write. hug mem stats --fixes [--since DATE] prints per UTC day: recall blocks (distinct session+second in injections), fixes, of-shown, not-shown, undone; and per instance. Deviation from the goal text: 'how many turns earlier' is recorded as time since the recall showed it (shown_ago); injections keep no turn index. 'Owner asked vs unprompted' cannot be told from local data without transcripts: not built. Not touched: muninn/internal/recall (boardcue.go), muninn/internal/harness.
check 1 evidence (commit 58a2904f on origin/rjk44w): TestForgetLeavesTrail (muninn/internal/app/fixes_test.go) injects a recall row for session mcp:abc, forgets with a reason, and asserts session, harness, instance, surface, reason, shown=true and shown_ago; an unshown forget gives shown=false; a failed forget writes no row. TestCLIFixTrailAndStats: hug mem forget --reason with JCODE_SESSION_ID set records session jcode:<id>, surface cli. MCP memory_forget takes reason (mcpserver/mcp.go forgetIn). Niced go test ./... exit=0, 34 packages ok.
check 2 evidence (58a2904f): TestRememberNamingOldIDLeavesTrail asserts a reinforce row (shown, reason, session) and a supersede row with new_sid; supersedes retires nothing; plain remember records nothing; an unknown supersedes errors with a fix: line. TestSupersedesAnswersDuplicateQuestion (cosine 0.92 band): naming the near memory answers the duplicate question, naming another still asks. It was red without the binding.go change ('NOT STORED ... Same fact?').
check 3 evidence (58a2904f): hug mem stats --fixes [--since DATE] [--json] prints per UTC day: blocks fixes of-shown not-shown forget reinforce supersede undone, then 'by instance:' and the last 20 fixes. Golden muninn/run/testdata/fixes.golden (TestFixReportGolden); TestCLIFixTrailAndStats over a real store (json, text, --since, plain stats unchanged).
check 4 evidence (58a2904f): TestFixSummaryCountsUndone: of 3 forgets on 2026-11-17, one reinforced 2 days later counts undone=1, one superseded after 8 days does not (UndoWindow 7 days).
check 5 evidence: A/B frozen recall4 replay, heldout split (77 messages, 115 hits), with recall4 built at base 832167b8 and at rjk44w, each on its own copy of recall3/baseline-heldout.db, variant baseline. With timings removed (jq -cS 'del(.latency_ms, .response.elapsed)') both are sha256 33e6cd8f62ebaab39e9dc3497142046634d0f69a49dbb5d46ad65186094fd9d4, and cmp says IDENTICAL. The raw files differ only in .response.elapsed (wall clock) on 61 of 77 lines. git diff 832167b8 over muninn/internal/{recall,search,harness,index} and muninn/cmd/recall4 is empty. Files: $JCODE_SCRATCH_DIR/rjk44w-replay/{base,new}.jsonl.
bonehound: ready to land. origin/rjk44w = origin/ci/rjk44w = 8a38836d (a1eaface feature + 8a38836d CRLF-safe golden compare), rebased on origin/main 4e70099c. CI 44326377046 all 6 jobs success (test ubuntu/macos/windows, test-windows-cli, bench, secrets). The first CI run, 44323264838, failed only on windows TestFixReportGolden (CRLF checkout), fixed in 8a38836d. Schema: additive memory_fixes table only. Report: /Users/jhf/.jcode/scratch/rjk44w/REPORT.md
crab review 15:55 local: accepted. 8a38836d (a1eaface + CRLF test fix) on 4e70099c; additive CREATE TABLE IF NOT EXISTS memory_fixes + 2 indexes; writes never fail the forget/remember; recall replay heldout identical sha256 after removing elapsed; recall/search/harness untouched. The 4 deviations accepted (shown_ago for turns; owner-asked not built; contradicted = undone within 7 d; blocks = (session, second)). CI 44326377046 6/6. Queued for landing.
Landed on main via 56e2cc6f (stack rjk44w c2df86c4, jj7jpz 215df154, 2qndsd 56e2cc6f; CI 44332698559 exact SHA 6/6). Ships in the next release.
landed 30c8c83 on main 2026-11-18: mem: memory fixes leave a local trail; hug mem stats --fixes (rjk44w)
landed c2df86c on main 2026-11-18: mem: fixes golden compares across a CRLF checkout (rjk44w, windows CI)
shipped in v0.7.65 (30c8c83)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T14:44 todo -> doing/active (take)
- 2026-11-18T15:55 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T14:44; waiting_asked_by: bonehound; waiting_on: landing: accepted by crab 14:55, queued to land (8a38836d, CI 44326377046 6/6); waiting_since: 2026-11-18T15:55
