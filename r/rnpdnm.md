---
id: rnpdnm
title: "Memory placement: extractor places personal/other projects, sync gate rule"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzy"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:36"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_on: "fresh captures after 2026-11-19 for the (a)+(b) attempt"
waiting_since: "2026-11-18T22:24"
waiting_asked_by: "bonehound"
---

## Goal
A sharing repo pushes only memories both A+ and B pass (gav9nw, Jørgen 19:40Z). A+: the capture extractor places a memory as personal, project:<other> or client:<name> in its existing call (no extra paid call), from the catalog hug mem tidy place builds. B: a local, model-free gate at sync, beside LooksSecret (sync.go), holds any line that names a home path, another known project or client, the person, or is a preference/correction. Its term list lives in local config, never in the repo.

## Purpose
On hug's 868 public memories, 58 (6.7%) should not have been pushed. A+ and B together leaked 0 of 699 in the measurement (gav9nw OPTIONS.md), with 13.7% held back. A pushed memory can't be taken back, so a held one is the cheap mistake.

## Principle
Pushed only when both say push. B also covers memories written before it. A held line is released by rescope or a local exception, never by loosening the default. Confirm on fresh captures before any repo opts in (34r1ft).

## Relevant details
OPTIONS: /Users/jhf/.jcode/scratch/gav9nw/OPTIONS.md (scripts, labels, logs alongside). The A+ row was estimated by a labeler, not measured on the extractor itself. Term list: local config (hug's global config), seeded from the place catalog plus the person's names from [people]. hug mem sync reports how many it held and why, one line per reason.

## Depends on

## Acceptance
- [x] B: sync holds a line naming /Users/, ~/, a cataloged other project/client, or the person, and a preference/correction; the rest of the project scope pushes (table test); the term list is read from local config, not the repo (test)
- [x] A+: the extractor's place field accepts personal, project:<other>, client:<name>; a captured personal memory is never in memories.jsonl (test with a recorded extraction)
- [ ] On fresh captures (not the 868 used to choose), A+ and B leak at most 1 in 200 pushed, judged blind; withheld rate reported with its interval
- [x] hug mem sync prints how many were held, by reason, and how to release one (fix: line)

## Notes

crab 21:36 local: dependency on gav9nw removed. The decision is written (8a1b0c05, DESIGN.md 'Personal vs shared memories', CI 44386929067 green) and accepted; it lands in the next stack.


B (checks 1, 4): a3790d1d on 8bf24c02, CI 44397313515 8/8 success. muninn/internal/project/gate.go Gate.Hold beside LooksSecret in syncFile: home folder (/Users/, ~/, /home/, C:\Users\), configured term as whole word with ø/æ/å folded, preference/correction. Terms in local muninn config.json sync_gate.terms (additive), seeded once from store places + hug.toml [people]; never printed. Sync output: 'held N: REASON' per reason, history note once for lines already pushed, 'fix: hug mem sync --held' (read-only; release named as a sentence for a person). Tests: project TestGateHoldsByRule (table: Jørgen's/Jorgen/jhf:/the user/C:\Users\//home/ hold; research, jhfx, internal/home/x.go, own name push), TestSyncGateWritesOnlyWhatPasses (fails without the gate call); run TestCLISyncGateHoldsAndReports (terms read from local config, not in output or git log -p hug/memory; fails with Gate nil), TestCLISyncSaysHeldLinesStayInHistory. Measured on the 868 (shipped Go gate, real seed, 0 paid calls): leak 13/705=1.8% [1.1,3.1], withheld 118/810=14.6% [12.3,17.2].
A+ (check 2): aa708f00 (CI 44398795828 pending). extract.wide_places off by default; off prompt sha256 pinned with and without places (fecb1569..., 7a83d752...). On, where personal is writable (never a client session): places personal + other (place_name); both stored in scope personal, other tagged about:NAME (crab ruling: never into another project's scope). Model sees meanings only. Tests: app TestCaptureWidePlacesPersonalNeverSynced (recorded extraction; personal and other never in memories.jsonl after project.Sync; fails with the mapping disabled), TestCaptureWidePlacesOffAndClient, extract TestCapturePromptWidePlacesOffIsByteIdentical, TestParsePlaceName. Local ./muninn/... ./internal/... 25 ok 0 FAIL.
check 3 pre-registered: /Users/jhf/.jcode/scratch/rnpdnm-c3/PREREG.md sha256 22844c7fc29f3cff (crab approved 21:14Z with blind hand read, dedupe vs the 868, same-model caveat). Fresh set: 52 chunks from 8 hug sessions after 12:05Z. Budget: extraction <=120, judge <=15, hard cap 140 Flash calls.
check 3 hand labels sealed before unblinding: hand_labels.json sha256 cf6cee4d6f9fb8bc28d200a8e6e45073ea7818e508d60167d0295dd0239598aa (297 blind items: 254 pushed + all 43 held, since held < pushed; shuffled seed 1120; 275 repo, 13 team, 5 other, 4 personal). Extraction: 40 Flash calls, stopped at 254 pushed after dedupe (0 duplicates of the 868).
check 3 FAIL (bar pre-registered, PREREG.md): 254 pushed of 297 fresh candidates (0 duplicates of the 868), allowed floor(254/200)=1 leak. Blind hand 8/254=3.1% [1.6,6.1] (7 team rules, 1 personal; all placed here by A+, no gate term), blind judge 17/254=6.7% [4.2,10.5]. Withheld: hand 29/275=10.5% [7.4,14.7], judge 18/255=7.1% [4.5,10.9]. Extractor and judge both deepseek-flash. Spend: 50 Flash calls (40 extraction + 10 judge, 0 retries), cap 140. extract.wide_places stays off. Report: /Users/jhf/.jcode/scratch/rnpdnm-c3/REPORT.md
check 2 CI: aa708f00 CI 44398795828 8/8 success (secrets, bench, test ubuntu/macos/windows, windows-cli rest/sync, shards).
next attempt design (crab 21:23, measured only on captures made after 2026-11-19 00:00Z; the 254 are training data now; pre-register bar and budget before any call): (a) B gate gains a coordination term set held as its own reason ('names the team or this machine'): the coordinator's and live agents' names from hug.toml [board] coordinator and the session list, plus fixed phrases 'this machine', 'this environment', 'agents should', 'the agent gate'; terms stay in local config. Cost check first, free: withheld on the 868 and the 297. (b) wide prompt adds: 'A working rule for agents (how to edit, commit, poll, report, which tools or limits apply) is global even when said while working in this project.' Both behind the existing wide_places switch; the off prompt stays pinned. (c) team scope (873tqd) is a larger design for Jørgen, noted only. DESIGN.md result: rnpdnm-design 423a39c4 on gav9nw-design 8a1b0c05.
crab 21:42: PREREG2 approved as written: /Users/jhf/.jcode/scratch/rnpdnm-c3/PREREG2.md sha256 38758e19f04e2f1f10e249a5e8a00f1ccc896601462092bb60094f501114f007 (piglet, 81 lines). Start floor 60 post-00:00Z chunks stands. piglet builds the frozen branch (a)+(b) on aa708f00 and runs step 1 (free) now; step 2 (paid, fresh data) runs only after the start condition holds, by piglet or bonehound tomorrow, under this prereg unchanged.
piglet 23:18 local, PREREG2 (sha256 38758e19...) frozen code and step 1. Code: branch rnpdnm2 c8c9335e on aa708f00 (tests 3cdcbd9e failing first; build 3ed93192; c8c9335e canonical paths in agents.NamesIn after CI 44406772764 failed macOS/windows 'no team reason line'), CI 44408727005 8/8 success. (a) Gate.Team, reason 'names the team or this machine' on its own sync line; terms read at each sync (hug.toml [board] coordinator, agents whose jcode sessions worked in the repo or its worktrees, 4 fixed phrases), never stored, printed or pushed; only with extract.wide_places. (b) sentence verbatim in the wide prompt; off-prompt pins fecb1569/7a83d752 unchanged. 7 mutants fail (Hold ignores Team, team as HeldName, not behind wide_places, NamesIn ignores roots, terms printed, coordinator not read, symlinked root). Step 1 (free, read-only, 0 paid calls; cost2.txt sha256 9ebb960d...; team.local 30 terms sha256 910b78af..., not printed): 868: B alone withheld 118/810=14.6% [12.3,17.2], B+(a) withheld 133/810=16.4% [14.0,19.1]; (a) holds 18 more, 16 labelled repo; leak among pushed 13/705=1.8% [1.1,3.1] -> 12/689=1.7% [1.0,3.0]. 297 hand: withheld 29/275=10.5% [7.4,14.7] -> 37/275=13.5% [9.9,18.0]. 297 judge: withheld 18/255=7.1% [4.5,10.9] -> 25/255=9.8% [6.7,14.1]. In-sample fit (not evidence): (a) holds 5 of attempt 1's 8 hand leaks (pushed leaks 8/254 -> 3/241) and 6 of 17 judge leaks. GO: highest withheld 16.4% <= 25%. Step 2 not run: waits for >= 60 chunks after 2026-11-19T00:00Z; no paid call made.
landed d142216 on main 2026-11-18: hug mem sync: a placement gate keeps a memory naming a home folder, another…
landed 6b5d7a1 on main 2026-11-18: hug mem capture: extract.wide_places (off) places a memory personal or about…
shipped in v0.7.71 (d142216)
hibiscus 08:09Z, PREREG2 (sha256 38758e19) step 2 NOT STARTED: official start count with the harness chunking (attempt 1 c3items, since 2026-11-19T00:00Z, seed 1121, built on c8c9335e, CI 44408727005 8/8) = 41 chunks in 11 hug sessions (crab 20, mizaru/piglet/sunflower 3, cactus/clover/iwazaru/macaque/mushroom 2, duckling/microbe 1; 560 post-00:00Z messages), below the floor of 60. 0 paid calls, terms not read. Also: harness/capture4c3 is the aa708f00 build without the (b) sentence; step 2 must rebuild it on c8c9335e. Report: /Users/jhf/.jcode/scratch/rnpdnm-step2/REPORT2.md
## History
- 2026-11-18T21:36 todo -> doing/active (take)
- 2026-11-18T22:24 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 handed to crab: bonehound's session ended (closed 2026-11-18 22:24)
