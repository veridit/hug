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
doing_since: "2026-11-17T23:20"
stage: "active"
model: "claude-opus-5-5/high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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


## History
- 2026-11-17T23:20 todo -> doing/active (take)
