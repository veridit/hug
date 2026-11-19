---
id: 2nmdm0
title: "rebase-land.sh misses a CHANGELOG entry cut by a conflict"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzz1"
created: 2026-11-19
updated: 2026-11-19
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-19T05:22"
---

## Goal
Restacking with scratch/rebase-land.sh fails loudly when a CHANGELOG entry from the source branch is missing or shorter after the rebase, as it already does for code diffs.

## Purpose
v0.7.75: 5mmqjk's 5-line entry reached main as its first line only (5b6f0de7), because a conflict resolution kept one line. CODE-DIFF IDENTICAL excludes CHANGELOG, so nothing flagged it; crab found it by reading Unreleased at release time and restored it in 4591627d.

## Principle
Check, don't auto-fix: the script reports the entry and stops; a person resolves it.

## Relevant details
Compare the source branch's added CHANGELOG lines (git diff FROM..SRC -- CHANGELOG.md, + lines) with the landed commit's (BASE..NEW); every added line must appear verbatim.

## Depends on

## Acceptance
- [x] a test or dry run: a rebase that drops a line of an added CHANGELOG entry makes rebase-land.sh exit non-zero naming the missing line
- [x] the normal restack prints CHANGELOG IDENTICAL next to CODE-DIFF IDENTICAL

## Notes

Cause was the script, not a conflict: rebase-land.sh line 14 extracted entries with grep '^\+- ', so a wrapped entry's indented continuation lines were dropped. Fixed in scratch/rebase-land.sh: entries are a '- ' line plus its '  ' continuation lines (awk), and after the cherry-picks every CHANGELOG line the branch added must be in the landed CHANGELOG verbatim, else 'CHANGELOG MISSING LINES:' and exit 3 without pushing.

check 1: dry run (push replaced by echo) replaying 5mmqjk 8610a01a on 50fa8681: with the old grep extraction -> 'CHANGELOG MISSING LINES:' 'not pushed', exit 3; with the fix -> 'CHANGELOG IDENTICAL', exit 0


check 2: real landing of nmn6wf e98a12ac at 04:21Z printed 'CODE-DIFF IDENTICAL' / 'released lines removed: 0' / 'CHANGELOG IDENTICAL' / 'pushed ci/nmn6wf-rebased dceb95a0'
Reflection (open, 2026-11-19, crab): I first blamed a rebase conflict for the cut entry. The fix only became obvious once I read the script's own extraction line, which was the actual cause.
## History
- 2026-11-19T05:01 todo -> doing/active (take with p277bp)
- 2026-11-19T05:22 doing/active -> done (done); was doing_since: 2026-11-19T05:01; doing_with: p277bp
