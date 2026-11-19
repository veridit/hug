---
id: h8ys9s
title: "Release v0.7.63: lint mentions, prompt history, framed file viewer"
state: "done"
type: chore
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzxb"
created: 2026-11-18
updated: 2026-11-18
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T14:44"
---

## Goal
v0.7.63 ships 4vxqp4 (8d26d050), xfv9pw (c87da5a8) and z82qsy once it lands, each from exact all-green CI, published before targeted install.

## Purpose
Jørgen tests releases live; prompt history and the framed viewer are his requests from this morning.

## Principle
Exact-SHA all-job CI green; release.yml success before brew upgrade; no --machine migration; released CHANGELOG sections never edited.

## Depends on

## Acceptance
- [x] CHANGELOG v0.7.63 heading covers exactly the landed entries, no conflict markers, earlier releases byte-identical
- [x] release.yml succeeds on the tag before targeted brew upgrade; hug version prints v0.7.63

## Notes


Tag v0.7.63 on 8a869ef7 (CHANGELOG commit: heading over 4vxqp4, xfv9pw, z82qsy; 4vxqp4 line now also gives the re-judged 106/9; 0 removed lines from released sections vs v0.7.62; no conflict markers). Release merge 832167b8 on main. Content CI, exact SHAs: 8d26d050 44304709971, c87da5a8 44307241896, 161381f3 44309663696 (WindowsCLI first attempt hit the 6ep7y2 push stall, rerun green). Install only after release.yml success.
release.yml run 44315534923 success on tag v0.7.63 (8a869ef7) before install; brew upgrade veridit/tap/hug then hug version: 'hug v0.7.63'. No --machine migration.
## History
- 2026-11-18T14:08 todo -> doing/active (take)
- 2026-11-18T14:37 doing/active -> doing/waiting (wait)
- 2026-11-18T14:44 doing/waiting -> done (done); was doing_since: 2026-11-18T14:08; waiting_asked_by: crab; waiting_on: release.yml for v0.7.63; waiting_since: 2026-11-18T14:37
