# g9dg32 replay (check 4), wyvern 2026-11-17

Binary: ci/g9dg32 build. Board: a scratch clone of a mirror of ~/Projects/hug
($JCODE_SCRATCH_DIR/replay/hug, remote = the mirror, so nothing reached the real
board). Main line: the mirror's refs/remotes/origin/main (82 commits since v0.7.30).

Command: `hugbin task deduce --since v0.7.30` -> exit 0, 64 "reached main" lines
(replay2.txt). Run again: "nothing new on the main line or in release tags".

Hand comparison, git log subject vs ticket Notes "landed" lines:
- t709he: 5 commits (4921209 b9b1ae2 387cfe3 a08bb34 d74af5d) = 5 landed lines
- 44w1ev: 14 commits d4ae7e7 .. 62a340c = 14 landed lines, same order
- m1c28r: 38ecd7d, 2c27e42 = 2 lines
- 0sbvnq: 04216f3 = 1 line
- 6ephqk: 62c98a3 954e165 893cd50 65a7129 25f5f5f d198e01 = 6 lines

Subjects with a 6-char word in a trailing group but no landed line (4), all correct:
- c6e0bff "(and drop a stray conflict marker from 3187cf5)": no ticket id
- 6190bbd "(the rebase onto v0.7.38 put them under its header)": no ticket id
- 9d1f382 "(the rebase onto v0.7.30 placed them there)": no ticket id
- 46be742 "(Windo..." : names no existing ticket

Not measured: release-tag lines in this replay. The mirror carried tags
v0.7.31.. but main here had no new tag after the first pass; the first replay
(stale local main) wrote 36 "shipped in" lines, not compared by hand.

## v0.7.41 tag (crab 14:23 ask 1)

Fresh mirror clone r3 (its remote is a mirror with no upstream), main = ~/Projects/hug origin/main 38ff464.
`deduce --since v0.7.40`: 26 landed and 13 "shipped in v0.7.41" lines (replay3.txt), in one commit.
A plain `deduce` after it: "nothing new on the main line or in release tags"; `sync` after that: no new deduce commit.
Lines for older releases: 0. Duplicate "shipped in v0.7.41" lines across the board: 0.

Hand check against `git log v0.7.40..v0.7.41`:
- 0sbvnq (doing): commit 04216f3 -> 1x "shipped in v0.7.41 (04216f3)", no done_release
- m1c28r (done): commit 2c27e42 -> 1x "shipped in v0.7.41 (2c27e42)", done_release "v0.7.41"
- t709he (done): commit d74af5d -> 1x "shipped in v0.7.41 (d74af5d)", done_release "v0.7.41"

Bug found and fixed on the way. The first v0.7.41 replay on clone r41 ran on the older binary. There, `--since` skipped the releases at or before since without marking them read, so the next background pass deduced them anyway: it wrote "shipped in v0.7.40" lines and a second deduce commit on old tickets. The fix marks a skipped release as read. The new test TestDeduceReplayFirstLeavesOldReleases fails before the fix ("pass after the replay wrote [observed: release v0.1.0 ...]") and passes after it.

## The note as it lands (crab 14:23 ask 2), t709he

Ticket Notes:
    landed d74af5d on main 2026-11-17: hug task tests: the commit-msg gate's cost is counted, not timed: 2 git and 1 awk per commit (t709he)
    shipped in v0.7.41 (d74af5d)
Report (command reply and the pass log):
    observed: commit d74af5d naming t709he reached main; marked: landed d74af5d; if wrong: hug task note t709he "not landed: d74af5d (why)"
    observed: release v0.7.41 contains commit d74af5d naming t709he; marked: shipped in v0.7.41; if wrong: hug task note t709he "not in v0.7.41 (why)"
