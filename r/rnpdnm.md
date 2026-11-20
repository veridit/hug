---
id: rnpdnm
title: "Shared project memories: is today's filter good enough to turn sharing on?"
state: "doing"
type: feature
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzr1"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T21:36"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "Sharing is off for every repo today, so nothing is pushed. Before you turn it on for one: A keep today's check and read the dry-run first, B first teach the writer to put agent rules in global memory (about 75 calls, a day), or C approve each memory before it is pushed?"
asked_for: "jhf"
asked_since: "2026-11-20T12:54"
asked_by: "crab"
asked_check: "3"
---

## Goal
Jørgen decides whether a repo may turn on memory sharing with today's filter, or whether we first make the filter stricter.

## Purpose
A repo that shares memories shows them to everyone who clones it, so wrong ones must stay on this machine.

## Principle
Choose from what was measured; change nothing for any repo until Jørgen picks.

## Relevant details
WHAT HAPPENS TODAY, IN PRACTICE
1. After a session, hug writes memories into your local store on this machine (muninn.db). Each has a place: personal, global, a client, or this project.
2. The memories of this project are recalled in this project's sessions. That is all local; nothing leaves the machine.
3. Sharing is a per-repo switch, off by default: [memory] share = true in hug.toml. Only when it is on does hug mem sync write the project's memories into memories.jsonl on the repo's hug/memory branch and push it, so everyone who clones the repo gets them.
4. Today no repo has it on. Nothing is written to or pushed on any branch.

WHAT 'HELD' MEANS
When sharing is on, sync checks each project memory before writing it to the branch. A memory that names a home folder (/Users/...), another project or client, or you, or states a personal preference, is not written to the branch. It stays in your local store and is still recalled here; only the push skips it. That is all 'held' means.

THE PROBLEM
The check misses some. Of new project memories, about 3 in 100 that would be pushed should not be (8 of 254, by hand). Most are working rules for agents ('poll CI with a timeout') stored as this project's when they belong in global memory. If sharing were on, those would be pushed to the repo.

WHAT WE ARE CHOOSING (only matters for a repo where you turn sharing on)
- A, stop here: keep today's check. Before turning sharing on for a repo, run hug mem sync --dry-run and read what would be pushed. Costs nothing.
- B, try again: teach the writer to put agent working rules in global memory, then measure again. About 75 cheap model calls and a day of new sessions; two attempts like this failed. Side effect: those rules would also be recalled in your other projects, where they belong.
- C, approve each one: nothing is pushed until you approve it in a list in hug tui. Catches everything, but you review every memory, and it is a new feature to build.

Attempt reports: /Users/jhf/.jcode/scratch/rnpdnm-c3/REPORT.md and /Users/jhf/.jcode/scratch/rnpdnm-step2/REPORT-STEP2.md.

Previous Goal: A sharing repo pushes only memories that pass both the extractor's placement and a local, model-free sync gate.
Previous Purpose: On hug's 868 public memories, 58 (6.7%) should not have been pushed. A+ and B together leaked 0 of 699 in the measurement (gav9nw OPTIONS.md), with 13.7% held back. A pushed memory can't be taken back, so a held one is the cheap mistake.
Previous Principle: Pushed only when both say push. B also covers memories written before it. A held line is released by rescope or a local exception, never by loosening the default. Confirm on fresh captures before any repo opts in (34r1ft).
Previous Details: A sharing repo pushes only memories both A+ and B pass (gav9nw, Jørgen 19:40Z). A+: the capture extractor places a memory as personal, project:<other> or client:<name> in its existing call (no extra paid call), from the catalog hug mem tidy place builds. B: a local, model-free gate at sync, beside LooksSecret (sync.go), holds any line that names a home path, another known project or client, the person, or is a preference/correction. Its term list lives in local config, never in the repo.

OPTIONS: /Users/jhf/.jcode/scratch/gav9nw/OPTIONS.md (scripts, labels, logs alongside). The A+ row was estimated by a labeler, not measured on the extractor itself. Term list: local config (hug's global config), seeded from the place catalog plus the person's names from [people]. hug mem sync reports how many it held and why, one line per reason.

## Depends on

## Acceptance
- [x] B: sync holds a line naming /Users/, ~/, a cataloged other project/client, or the person, and a preference/correction; the rest of the project scope pushes (table test); the term list is read from local config, not the repo (test)
- [x] A+: the extractor's place field accepts personal, project:<other>, client:<name>; a captured personal memory is never in memories.jsonl (test with a recorded extraction)
- [ ] On fresh captures (not the 868 used to choose), A+ and B leak at most 1 in 200 pushed, judged blind; withheld rate reported with its interval
- [x] hug mem sync prints how many were held, by reason, and how to release one (fix: line)

## Questions
- 2026-11-19T15:23 question 1 by crab to jhf:
  Memory placement attempt 2 failed its bar: of 295 memories that would be pushed to the shared repo, 36 should not be (24 are team working rules written as if they only apply to this repo). Shall I pre-register a third approach on new data, or drop this check and keep what already ships (personal and other-project memories held back by the sync gate)?
  answered 2026-11-20T11:26 by crab:
    Withdrawn by crab 2026-11-20: unclear (Jørgen request 1). Re-asked plainly as question 2 with options A, B and C in Details.
- 2026-11-20T11:26 question 2 by crab to jhf (check 3):
  Shared project memories: A stop here with today's filter (about 3 in 100 wrong ones get through; costs nothing), B try a third attempt aimed at agent working rules (about 75 cheap calls, a day's wait), or C a person approves each memory before it is pushed (new feature)? Details explain each.
  answered 2026-11-20T12:54 by crab:
    Withdrawn by crab 2026-11-20: still unclear (Jørgen request 2). Re-asked as question 3 with the practical consequences.
- 2026-11-20T12:54 question 3 by crab to jhf (check 3):
  Sharing is off for every repo today, so nothing is pushed. Before you turn it on for one: A keep today's check and read the dry-run first, B first teach the writer to put agent rules in global memory (about 75 calls, a day), or C approve each memory before it is pushed?

## Requests
- 2026-11-20T11:21 request 1 by jhf:
  I don't understand the question of this ticket. It is confusing what is the third approach, what you mean pre-register, what you mean holding back. Try to be grounded and speak plainly. What are we choosing between? So simplify the things. Also, the description, the purpose is not clear. And what does it mean to be held back? Clarity and groundedness, please. I will try again once you've made this ticket practical.
  resolved 2026-11-20T11:26 by crab:
    Rewritten plainly. Title, Goal, Purpose and Principle now say what we choose: keep today's filter or make it stricter. Details start with WHAT WE ARE CHOOSING BETWEEN: what a shared memory is (today no repo shares, nothing is pushed), what 'held back' means (kept on this machine, not pushed), what was measured (3 in 100 wrong get through today, the bar was 1 in 200), and options A stop here (my pick), B a third attempt, C a person approves before push, each with its cost. The old question 1 is withdrawn; question 2 asks A, B or C. The old text sits under it in Details.
- 2026-11-20T12:54 request 2 by jhf:
  I still don't understand what hold means, being held back. So in practical terms, we import memories, they're stored in a separate branch. Are we discussing if that separate branch should be pushed or not? Or are we discussing if some memories are placed in global storage or client storage and not in the project itself? So you don't lay out the practical consequences here.
  resolved 2026-11-20T12:54 by crab:
    Laid out in practice at the top of Details. Memories are written locally; a repo's memories go to its hug/memory branch and get pushed only if that repo turns sharing on, and today none does, so nothing is pushed. 'Held' means: when sharing is on, a memory that names a home folder, another project or client, or you, is not written to the branch; it stays local and is still recalled here. The choice is only about whether today's check is good enough to turn sharing on (A), or whether to first move agent rules to global memory (B) or approve each push (C). The old question is withdrawn; question 3 asks it plainly.
- 2026-11-20T12:58 request 3 by jhf:
  As you see in the screenshot for this, it does scroll down. I don't notice that I can scroll up again because I don't have the arrows that I discussed in the previous task where there is a line and an arrow if I can go up and an arrow if I can go down. But I'm guessing that is coming in a later task. So hopefully I can view that. Now let me look at the task again. So this, I just used this task to give you a screenshot to give you that feedback
  resolved 2026-11-20T12:58 by crab:
    Thanks, noted where it belongs: the missing up/down arrows are 4ddcwa (↑ on the rule under the title when you can scroll up, a bottom rule with ↓ and the page status). Raised to high so it comes soon. Nothing changed on rnpdnm itself; its question 3 (A, B or C) still stands.
- 2026-11-20T12:59 request 4 by jhf:
  I don't know what the local store is, you have to be concrete. So what are the stores? Where are they physically and how would this work? How like short, succinct, grounded.

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
crab 08:10Z ruling on step 2 start: (0) My 821 at 08:00Z was wrong: I sized messages with their JSON content (tool calls and results included); the harness counts only text through ReadCaptureSession. The official count is hibiscus's: 41 chunks at 08:08:31Z, below the floor of 60. Wait, as PREREG2 says. 0 of 140 calls spent. (1) Recount: crab runs the count-only binary rnpdnm-step2/c3items2 (sha 7087ed26) every 3 h from its own session, reading only the stderr stats; at >= 60 a fresh worker starts step 2. (2) Sources: 'the measuring session' is the session that runs step 2, not the coordinator; crab's session is a hug session after 00:00Z and counts, per PREREG2's letter. Its share (20 of 41 now) is reported per source; round-robin across sessions already limits it. Changing that would be a new pre-registration. (3) Harness: rebuild capture4c3 from c8c9335e with the same scratch -wide/-ledger/-cap patch as attempt 1; record the patch's sha256 in the report. Tooling only, not a design change.
hibiscus 08:11Z, recount command (crab ruling 08:10): cwd /Users/jhf/.jcode/scratch/rnpdnm-step2, GOMAXPROCS=2 nice -n 19 ./c3items2 "$(paste -sd, sessions11.txt)" > verify11.jsonl 2> verify11.stats (sessions11.txt sha256 6a6a3201..., the 11 ids). Correction: the 08:08Z run was given 303 ids (ls session_NAME_* matched older same-name sessions); on exactly the 11 at 08:10:53Z the count is the same 41 chunk ids: {"cactus":2,"clover":2,"crab":20,"duckling":1,"iwazaru":2,"macaque":2,"microbe":1,"mizaru":3,"mushroom":2,"old":39242,"piglet":3,"sunflower":3,"used":562}. Build the list fresh at each recount; count per Session from the items (the stats key is the agent name and collides on repeated names). DM to crab failed (Unknown target), so this note carries it.
crab 11:10Z recount (stats only, no content): 57 chunks from 15 sessions, below the 60 floor. Command per REPORT2.md: GOMAXPROCS=2 nice -n 19 ./c3items2 "$(paste -sd, sessions-1110.txt)" (binary 7087ed26), list built fresh (working_dir hug, mtime after 00:00Z), no repeated names. Stats: {blossom 2, cactus 2, clover 2, crab 29, daisy 2, duckling 1, hibiscus 2, iwazaru 2, macaque 2, microbe 1, mizaru 3, mushroom 2, piglet 3, sunflower 3, tulip 1; old 40271, used 671}. Items count 57/15 sessions agrees. Next recount 14:10Z.
crab 14:10Z recount (stats only, no content): 73 chunks from 20 sessions, floor 60 MET. Command per REPORT2.md: GOMAXPROCS=2 nice -n 19 ./c3items2 "$(paste -sd, sessions-1410.txt)" (binary 7087ed26), list built fresh (20 ids, no repeated names). Stats: crab 37, sunflower 3, piglet 3, mizaru 3, 2 each hibiscus iwazaru evergreen daisy clover mushroom rose macaque maple blossom cactus, 1 each microbe palmtree duckling tulip seedling; old 41226, used 824. Items count 73/20 agrees. PREREG2 sha 38758e19 verified. Step 2 starts under PREREG2 unchanged; the step-2 session is excluded from sources (it reads the list at its own start).
herb 14:19Z, PREREG2 step 2 hand labels sealed before unblinding: hand_labels2.json sha256 70ec4c69329ce186d92e4b5235d7936fa8a4d66f9b3976b779ab987fc093eaab (394 blind items: 295 pushed + all 99 held, since held < pushed; shuffled seed 1123; 295 repo, 57 team, 23 other, 19 personal). Extraction: 61 Flash calls (ext_calls2.json), stopped after batch 3 at 295 pushed after dedupe.
herb 14:22Z, PREREG2 (sha256 38758e19) step 2: FAIL. n_pushed 295 (>=200), allowed floor(295/200)=1; blind hand leaks 36/295=12.2% [8.9,16.4] (team 24, other 7, personal 5), blind judge leaks 26/295=8.8% [6.1,12.6]. Withheld: hand 36/295=12.2% [8.9,16.4], judge 32/301=10.6% [7.6,14.6] (no 25% flag). Contamination (13 pushed: 10 name rnpdnm/gav9nw, 3 repeat attempt 1 leaks) excluded: 33 hand / 24 judge of 282, still FAIL. Spend: extraction 61 (ext_calls2.json), judge 14 (judge_calls2.json), total 75/140. Sources: 20 sessions (sessions-step2.txt sha256 ff374a4b...), 73 chunks, 60 used; crab 37/73 chunks available, 24/60 used, 116/295 pushed, 19/36 hand leaks. Harness: capture4c3 from c8c9335e + bonehound journal patch (patch body sha256 2ed9524d..., diff df1ba66d...), binary a9321bc6..., (b) sentence present (strings: 1). Team set 45 terms, team2.local d62e1cd5.... Hand labels sealed 70ec4c69... before unblinding. Per PREREG2: extract.wide_places stays off; next idea needs a new pre-registration on new data. Report: /Users/jhf/.jcode/scratch/rnpdnm-step2/REPORT-STEP2.md
crab 14:23Z ruling on step 2: FAIL accepted, by PREREG2's own bar (sha 38758e19 verified). Verified myself: ext_calls2.json {calls 61, cap 120}, judge_calls2.json {calls 14, cap 15} = 75 of 140; hand_labels2.json sha 70ec4c69 equals herb's 14:19Z sealing note, made before the 14:22Z verdict note. n_pushed 295, allowed 1; hand leaks 36 (team 24, other 7, personal 5), judge 26; still FAIL without the 13 contaminated (33/24 of 282). Consequences per PREREG2: extract.wide_places stays off; rnpdnm2 stays off main; not rerun. Learned: 24 of 36 hand leaks are team working rules written as repo-scoped; prompt sentence (b) did not move them, and none hit the team term set (a). The next idea needs a new pre-registration on new data and is not started without Jørgen. Check 3 stays open.
file:screen-1258.ans (40181 bytes, sha256:67505c40) screen capture from hug tui
## History
- 2026-11-18T21:36 todo -> doing/active (take)
- 2026-11-18T22:24 doing/active -> doing/waiting (wait)
- 2026-11-18T23:24 handed to crab: bonehound's session ended (closed 2026-11-18 22:24)
- 2026-11-19T15:23 doing/waiting -> doing/asked (ask); was waiting_asked_by: bonehound; waiting_on: fresh captures after 2026-11-19 for the (a)+(b) attempt; waiting_since: 2026-11-18T22:24
- 2026-11-20T11:21 request 1 by jhf
- 2026-11-20T11:21 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:21 request 1 acknowledged by crab
- 2026-11-20T11:26 request 1 in progress by crab: hug task set rnpdnm
- 2026-11-20T11:26 answered asked, by crab, for crab (doing/asked -> doing/active):
  > Memory placement attempt 2 failed its bar: of 295 memories that would be pushed to the shared repo, 36 should not be (24 are team working rules written as if they only apply to this repo). Shall I pre-register a third approach on new data, or drop this check and keep what already ships (personal and other-project memories held back by the sync gate)?
  Withdrawn by crab 2026-11-20: unclear (Jørgen request 1). Re-asked plainly as question 2 with options A, B and C in Details.
- 2026-11-20T11:26 doing/active -> doing/asked (ask)
- 2026-11-20T11:26 resolved request 1 by crab
- 2026-11-20T12:54 request 2 by jhf
- 2026-11-20T12:54 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T12:54 request 2 acknowledged by crab
- 2026-11-20T12:54 request 2 in progress by crab: hug task set rnpdnm
- 2026-11-20T12:54 answered asked (question 2, check 3), by crab, for crab (doing/asked -> doing/active):
  > Shared project memories: A stop here with today's filter (about 3 in 100 wrong ones get through; costs nothing), B try a third attempt aimed at agent working rules (about 75 cheap calls, a day's wait), or C a person approves each memory before it is pushed (new feature)? Details explain each.
  Withdrawn by crab 2026-11-20: still unclear (Jørgen request 2). Re-asked as question 3 with the practical consequences.
- 2026-11-20T12:54 doing/active -> doing/asked (ask)
- 2026-11-20T12:54 resolved request 2 by crab
- 2026-11-20T12:58 request 3 by jhf
- 2026-11-20T12:58 request 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T12:58 resolved request 3 by crab
- 2026-11-20T12:58 request 3 acknowledged by crab
- 2026-11-20T12:59 request 4 by jhf
- 2026-11-20T12:59 request 4 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
