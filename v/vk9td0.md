---
id: vk9td0
title: "Decide when dream items are handed out"
state: "doing"
type: spike
priority: "high"
assignee: ["sabertooth"]
labels: ["dream"]
group: "dream"
rank: "zz8"
created: 2026-11-10
updated: 2026-11-10
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
aliases: ["decide-when-dream-items-are-handed"]
doing_since: "2026-11-18T13:25"
---

## Goal
Dreaming runs at the start of the person's usual quiet stretch: from at least one week of activity, hug builds a per-hour-of-week histogram (168 buckets), finds the longest stretch of no activity, and starts dreaming at its beginning; the hand-out moment for dream items follows from that.

## Purpose
Jørgen 2026-11-18: an earlier sync design used an activity histogram (per hour of day and per hour of week) to find the longest quiet stretch; the week is a natural cycle and captures that days differ. Dreaming should use the same idea instead of a fixed cue on return.

## Principle
Learn the rhythm from data, don't guess a clock time. Hour-of-week, not hour-of-day. Until a week of data exists, keep today's behaviour. Local and free: no paid calls at read time.

## Relevant details
Activity source: session/turn timestamps already in the jcode and claude journals and hook.log. Today: the dream cue fires on a break turn (start, return after 20 min, wrap-up word) at most once per 6 h (recall/dream.go); hatchling's open question was return vs wrap-up. Find the earlier sync histogram design (Jørgen recalls an agent built it) and reuse it.

## Depends on

## Acceptance
- [x] decision recorded in DESIGN.md with reason
- [ ] With 7+ days of activity, hug computes a 168-bucket hour-of-week histogram and names the longest quiet stretch; shown by a command or debug line (test with a synthetic week)
- [ ] Dreaming starts at the beginning of that stretch, and with under a week of data the current behaviour stays (test)

## Requests
- 2026-11-18T12:22 request 1 by jhf:
  I think that for sync thing I had an agent make a histograph of when there was activity and then figure out the longest stretch of non-activity and then at the start of such a stretch when you have one week of data start the dreaming. And the histogram was per hour per day and per hour per week. So you can have an idea. And I think it gives you an idea of, but I think the per hour of the week is a sensible thing because then you can kind of capture that there are different days throughout the week, and a week is a natural cycle
  resolved 2026-11-18T13:03 by crab:
    Taken as the design: goal, purpose, principle and details now describe a 168-bucket hour-of-week histogram from a week of activity, dreaming at the start of the longest quiet stretch, today's cue kept until a week of data exists. Two checks added. It goes to the next free worker.

## Notes

Built a first answer (2026-11-12): cue at a break (session start, return after 20 min, wrap-up), at most once per 6 h per session; Response.Break reports which fired. User wants to discuss after trying it.


hatchling 2026-11-18: summary for Jørgen sent to crab, no code. h00cs2 (1151adad) changes when items are made: a waiter runs the pass at least 20 h after the last, once he has been away 30 min and load is below 120 (dreamauto.go QuietFor, DreamEvery, DreamLoadBelow). It does not change the hand-out: still the 11-12 cue on a break turn (start, return after 20 min, wrap-up word), at most once per session per 6 h (recall/dream.go). So the next cue will land on his first message after returning. Live store: 9 items from the one manual pass on 11-12; 5 resolved, 4 handed and all of those resolved, 4 never handed. 27 of 103 sessions cued, the last 11-16 10:57Z. Open for Jørgen: keep the cue on his return, or move it to wrap-up/turn end only.
crab 12:25Z: hatchling's session is gone; handing to retriever with Jørgen's histogram design.
sabertooth 2026-11-18: built on branch vk9td0, commit 4f0465fb (not merged). PRIOR ART: ~/Projects/syncthing internal/activityhist/activityhist.go (patch/43 81da0bd0b, patch/53 a5f6b1290): minute-of-day + hour-of-week histograms, 7 days before predicting, start at the beginning of the deepest idle run. hug reuses the idea, not the code (that is a different repo): new muninn/internal/activity, hour-of-week only, it counts the person's turns. DESIGN: activity_hours table (turns per clock hour, 8 weeks); recall noteTurn counts turns except swarm DMs and harness notices; seeded once from hook.log; histogram over the last 4 weeks; an hour is quiet if it had a turn in <=1/4 of the weeks; each day's longest quiet stretch is a window; DreamGate waits for it (ready >= 7 days); under 7 days the old 30-min rule; the waiter deadline is stretch start + 12h. REAL (copy of hook.log, scratch HUG_HOME, live store untouched): 3352 turns seeded; 'Longest quiet stretch: Sun 01:00 for 6 h'; 'Dream starts each day at: Mon 02:00 for 3 h, Tue 06:00 for 2 h, Wed 05:00 for 1 h, Thu 05:00 for 1 h, Fri 03:00 for 1 h, Sat 01:00 for 5 h, Sun 01:00 for 6 h'. SUITE niced: 35 ok, 0 FAIL. Report: /Users/jhf/.jcode/scratch/vk9td0/REPORT.md
check 1: docs/DESIGN.md '## When dream runs (vk9td0, decided 2026-11-18)': Decision, Reason (activityhist, week as the cycle), How, Not taken. Commit 4f0465fb.
## History
- 2026-11-18T06:31 todo -> doing/active (take)
- 2026-11-18T06:32 doing/active -> doing/waiting (wait)
- 2026-11-18T12:22 request 1 by jhf
- 2026-11-18T13:03 resolved request 1 by crab
- 2026-11-18T13:24 doing/waiting -> todo/ready (give back; was assignee: hatchling); was doing_since: 2026-11-18T06:31; waiting_asked_by: hatchling; waiting_on: Jørgen's answer on the dream cue moment (crab recommended wrap-up only, 05:32); waiting_since: 2026-11-18T06:32
- 2026-11-18T13:25 todo/ready -> doing/active (take)
