---
id: qxfqxh
title: "Release v0.7.73"
state: "done"
type: chore
priority: "high"
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzzl"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
effort: "high"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T01:20"
---

## Goal
v0.7.73 published: main f17745ad, release merge 2fc72037, tag v0.7.73 on CHANGELOG commit 898f1782

## Purpose
Hourly release

## Principle
Exact-SHA green CI only; released CHANGELOG sections never edited

## Relevant details
CI 44425299288 8/8 on f17745ad. Content: rxr4z8 marks, hfkt1n reload test, nanvnt withdrawn questions. release.yml 44426761530

## Depends on

## Acceptance
- [x] release.yml run for v0.7.73 succeeds
- [x] brew upgrade veridit/tap/hug; hug version prints v0.7.73

## Notes


check 1: release.yml 44426761530 completed success
check 2: brew upgrade veridit/tap/hug was killed by my 600 s watcher timeout between unlink and link, leaving /opt/homebrew/bin/hug, muninn and huginn missing until 00:20:45Z; brew link --overwrite hug relinked 0.7.73; hug version -> hug v0.7.73; ~/.local/bin/muninn version -> hug v0.7.73
Reflection (dismissed, 2026-11-19, crab): A timeout around brew upgrade can kill it mid-link and take the user's hug off PATH: run brew upgrade on its own, never inside a timed watcher.
Reflection 1 dismissed as already applied (crab 2026-11-19): saved as memory k1pnzb ('When cutting a hug release, run brew upgrade veridit/tap/hug on its own, never inside a timed background watcher ...'); it is a release-operator practice, not an instruction text.
## History
- 2026-11-19T01:20 todo -> doing/active (take)
- 2026-11-19T01:20 doing/active -> done (done); was doing_since: 2026-11-19T01:20
