---
id: 9ahvgt
title: "stale-owner lint misses owners whose sessions are gone"
state: "done"
type: bug
priority: ""
assignee: ["iwazaru"]
labels: []
group: ""
rank: "zzzzzzzzzzoj"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T20:54"
---

## Goal
hug task lint reports stale-owner for every doing task whose assignee has had no live harness session for over a day, including workers spawned in other jcode homes and names with no session file at all.

## Purpose
retriever 2026-11-18 12:34Z: lint --code stale-owner said 'no findings' while 26+ doing tasks belonged to gone sessions: hatchling 6, nautilus 11 (journal last 07:46), kestrel 8 (no session file), wyvern 1. Requests and answers then reach nobody.

## Principle
A finding must reflect what hug can actually see; when it cannot see sessions (unreadable home) it says so once, rather than staying silent.

## Relevant details
Code: huginn/internal/task/owner.go:95 builds the finding; it reads ~/.jcode/sessions only when readable. Check: does it match assignee names to session names (session_<name>_<ts>), and how does it treat a name with no session file (kestrel)? Scratch: /Users/jhf/.jcode/scratch/board-orphans/REPORT.md.

## Depends on

## Acceptance
- [x] On hug's board, lint reports stale-owner for nautilus, kestrel and hatchling tasks; a test with a gone owner and with an owner that has no session file fails first on main.
- [x] A live worker's task is not reported (test).
- [x] hug task take on a task whose owner's session is gone (the same test stale-owner uses) ends with a runnable fix line that takes it over (e.g. hug task set ID --assignee ME, then take), not only 'fix: hug task next' (test; mizaru on txjkrm 2026-11-18 hit it)

## Notes


Diagnosis (iwazaru, from ~/.jcode/sessions and logs). (1) owner.go judges by mtime only, with a 24 h threshold. nautilus' newest file is 07:46 today and hatchling's 06:47, so neither is past 24 h. Yet both sessions are gone: the last journal line of each has meta status Closed, and jcode-2026-11-16.log:1492534 reads 'Reaped idle spawned swarm worker session_nautilus_1794905151412_33a1e4b1294b58f5 (idle > 1800s)'. hug never reads the status. Of the 75 journals, 53 end Closed and 22 end Active, and the 22 Active ones are the 11 live swarm workers plus others. (2) LivenessFrom reads only session_*.json, not the .journal.jsonl that is written later (nautilus json 07:07, journal 07:46). (3) kestrel has no session file in any home (global, brovs, GenderedGames, truth_tribe_and_treasure, seid, statbus). Session calf's title is 'You are kestrel, a worker on hug', so calf took bxqtkt --as kestrel, and OwnerFindings skips any name the session list does not know. 'engineer' (xfv9pw: my own take --as engineer) and 'crab-engineer' (4vxqp4, duckling) are the same shape.
Failing-first on origin/main 77d25855: TestOwnerFindingsGoneSessions 'owner_gone_test.go:74: zhvkmj not reported (its owner is gone); findings: map[]' (also tb0dra, bxqtkt); TestLintStaleOwnerSessions (cli) 'staleowner_test.go:64: nautilus's task gz0epr not reported; findings map[]' + kestrel, wyvern, and 'staleowner_test.go:89: unreadable sessions: warnings []'. Real board, read-only, branch build: hug task lint --code stale-owner gives 22 findings (hatchling 5: h00cs2 n7jbmx s6ff5k tb0dra z9q42h; nautilus 9: 0gztsy 1bbe96 4s3g5q 5z4zt4 ahejza qh9ep1 rxr4z8 tmz060 zhvkmj; kestrel 6 '(kestrel was session calf)': 41hhgn bxqtkt ggydj2 hbgybn mbdfzs z2vmte; wyvern 1 jr55s8; puppy 1 911j70). The installed hug v0.7.62 on the same board: 'no findings with code stale-owner'. Owners flagged: exactly hatchling, kestrel, nautilus, puppy, wyvern; none of the 11 live workers (macaque, tigress, duckling, kikazaru, iwazaru, bonehound, piglet, sabertooth, mizaru, crab).
Check 2: a live worker is never reported. TestOwnerFindingsGoneSessions: macaque (Active, 1 min) not reported while an older Closed macaque exists; puppy (Closed 20 min, under GoneOwnerAfter=1h) not reported; e3vw30 kestrel+macaque not reported (one live owner is enough); jhf (Model human) and claude-opus-5-5 (the model as name) never judged. TestLintStaleOwnerSessions (cli): macaque (Active, in a subfolder of the repo) and jhf (--model human) not reported. A Statbus wyvern (Active, other working_dir) does not hide hug's closed wyvern. On the real board none of the 11 live workers is flagged. Commit ffe5c0fa on 9ahvgt and ci/9ahvgt; full suite 34/34 ok; CI 44322853178 running.
Check 3 failing-first (origin/main + earlier 9ahvgt commit): TestTakeFromGoneOwner 'staleowner_test.go:134: exit 1, want 1 ending "fix: hug task set 6zhpt3 --assignee mizaru && hug task take 6zhpt3 --as mizaru --model claude-opus-5-5/high\n": taken by nautilus (6zhpt3) / fix: hug task next'. With the fix (take.go): a take refused as 'taken' runs the stale-owner test (task.OwnerFindings with the same liveness and [people]); for a gone owner it says 'taken by nautilus (ID), but has had no live session since ...' and ends 'fix: hug task set ID --assignee ME && hug task take ID --as ME --model MODEL'. The test runs both commands and the task becomes mizaru's. A live owner's task (macaque) still ends 'fix: hug task next'. The refused take writes nothing (the refusal comes before any write in transitionFull).
Ready to land: branch 9ahvgt = ci/9ahvgt at acd46d65 (43b03284 lint + acd46d65 take fix), on origin/main 62c20712. CI 44331139852: completed success (secrets, macos, ubuntu, windows, test-windows-cli, bench). Local full suite 34/34 ok on acd46d65. Earlier CI 44322853178 failed macOS (/var vs /private/var) and windows-cli (unescaped working_dir); fixed with canonical()+jsonString(). Report: ~/.jcode/scratch/9ahvgt/REPORT.md. Not merged; done left to crab.
crab review 16:24 local: accepted. 43b03284 + acd46d65 on 62c20712; owner.go gone rule (Closed + 1 h quiet, any 24 h quiet, unknown name doing > 24 h), 'You are NAME' alias, instances via the existing harness jcode_instances discovery limited to sessions in this repo, people never judged, take fix 'hug task set ID --assignee ME && hug task take ID --as ME --model MODEL'. Real board 22 findings, 0 live workers flagged. CI 44331139852 6/6. Rulings: 1 h for Closed stays; && over --force is right; role names as unknown names are fine.
Landed on main via 80c8ac28 (stack of 10 on v0.7.65; CI 44344774710 exact SHA 6/6). Ships in the next release.
landed ae04196 on main 2026-11-18: hug task lint: stale-owner sees owners whose sessions are gone (9ahvgt)
landed 83f7e05 on main 2026-11-18: hug task take: a task whose owner is gone ends with a fix that takes it over…
shipped in v0.7.66 (ae04196)
Reflection (unanswered, 2026-11-18, crab)
## History
- 2026-11-18T13:44 todo -> doing/active (take)
- 2026-11-18T16:24 doing/active -> doing/waiting (wait)
- 2026-11-18T20:54 doing/waiting -> done (done); was doing_since: 2026-11-18T13:44; waiting_asked_by: iwazaru; waiting_on: crab lands acd46d65 in the next stack (accepted 15:24Z, CI 44331139852 6/6); waiting_since: 2026-11-18T16:24
