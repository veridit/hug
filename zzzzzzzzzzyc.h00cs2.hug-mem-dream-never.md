---
id: h00cs2
title: "hug mem: dream never runs on its own unless hug mem serve is running"
state: "doing"
type: feature
priority: ""
assignee: ["hatchling"]
labels: []
created: 2026-11-17
updated: 2026-11-17
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
doing_since: "2026-11-18T04:27"
---

## Goal
The daily dream pass (merge and retire duplicates) runs on every machine that captures, whether or not a hug mem serve process is up, and a user can see when it last ran.

## Purpose
On Jørgen's machine dream has never run automatically: meta dream_last_ns is unset, no hug mem serve process and no LaunchAgent run, and hug mem dream --dry-run says 'Would retire 2574 duplicate memories' (2026-11-17, found during 5sc5rw). Duplicates pile up and recall carries them.

## Principle
Find where the pass should be triggered (hook, setup-installed agent, or a cheap check in capture), measure the cost before shipping; no change to capture extraction; retiring memories stays previewable.

## Relevant details
muninn/internal/daemon/dream.go:14 QuietFor 30m, :17 DreamEvery 20h, :21 dreamWhenQuiet is only called from daemon/watch.go's 5-minute maintenance ticker inside hug mem serve. meta dream_last_ns read at dream.go:29. 2574 of the would-retire count overlap the 2651 twins 5sc5rw folds; rerun the dry run after the fold for the real residue. Filed by hatchling for crab.

## Depends on

## Acceptance
- [ ] a test: with no serve process, the trigger chosen runs dream once DreamEvery has passed and the machine is quiet (fails before)
- [ ] hug mem status (or doctor) names when dream last ran, and says never when dream_last_ns is unset
- [ ] on this machine, dream_last_ns is set within a day of upgrading, with no hug mem serve running

## Notes


Started design 22:21Z, paused for wdkqzx (crab). Facts so far: dream is called only at muninn/internal/daemon/dream.go:35 from watch.go:79 (serve's 5-min ticker). One dry-run pass on a copy of the live index: 11.9 s wall, 9.5 s user, 302 MB max RSS (niced, GOMAXPROCS=2). Its 'Would retire 2574' is mostly the 2571 active 5sc5rw twins (that copy still held the live source, 2571 active imp_ repo: units); rerun the dry run on the live store after the v0.7.49 fold for the real residue. Existing detached-pass pattern to reuse: project.Nudge / gitbranch.Background (muninn/internal/project/background.go). Design proposal to crab before building.
Measured 2026-11-18 03:28Z on the live store with brew hug v0.7.54 (niced). meta dream_last_ns is unset. 0 hug mem serve processes. No LaunchAgent. hug mem dream --dry-run: 'Would retire 3 duplicate memories. Queued 111 new items for agents; 4 waiting.' 10.14 s real, 8.97 s user, 377 MB max RSS. So the v0.7.49 fold took the 2574 down to 3. The 111 are judgement items, uncapped in dry run. A real pass caps them: VetPerPass 10, AgentPerPass 10, EndedPerPass 20 (app/dream.go:336, agents.go:25, holds.go:25). Quiet windows, from hook.log user turns over the last 7 days (3035 turns): 54 gaps of 30 min or more, at least 3 every day, the longest each day 1:29 to 7:31. So 'quiet for 30 min' happens daily without serve. Proposal to crab follows.
file:h00cs2-proposal.txt (3072 bytes, sha256:44e2d89c)
## History
- 2026-11-17T23:20 todo -> doing/active (take)
- 2026-11-17T23:22 doing/active -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-17T23:20
- 2026-11-18T04:27 todo/ready -> doing/active (take)
