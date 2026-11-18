# 44w1ev check 3: an unchanged board skips the per-file walk (design for review)

Status: design only, not implemented, not on main. For Jørgen's review (wyvern/humpback, 2026-11-17).

## What the walk costs today (measured)

Board: `board10k` (10,000 tasks from gen-scale-board.py), tree 6190bbd, nice -n 19, GOMAXPROCS=2, macOS.

| piece | cost |
|---|---|
| `cache.Sync` on an unchanged board (WalkDir + 10,000 stats, 0 hashed), in-process, 7 runs | 42-53 ms |
| `cache.Open` (blob decode), 5 runs | 23-36 ms |
| `hug task show` end to end, median of 5 | 0.208 s |
| `hug task next` end to end, median of 5 | 0.237 s |
| profile of `show`: `cache.(*Cache).sync` | 160 of 620 ms sampled (26%) |
| `git rev-parse HEAD` | 10-20 ms |
| `git status --porcelain=v2 -z --untracked-files=all` | 40-60 ms |
| `git diff-files --name-only -z` | 20-30 ms |
| `git ls-files -o --exclude-standard -z` | 20 ms |

So skipping the walk saves at most about 45 ms of a 200 ms command at 10,000 tasks. At 50,000
it would be about 225 ms if it scales linearly (an extrapolation, not measured), which is where it starts to matter. The larger fixed costs are
still the blob decode (Open) and the git spawns.

## The correctness bar

The walk exists to see changes made outside hug: an editor, `git pull`/`rebase`/`checkout`
inside `.hug/tasks`, a sync pass, or a copy. A skip is correct only if every such change
trips the signal. Same-size edits in the same mtime tick are the hard case (racy-clean, step 3).

## Candidate signals

1. **HEAD unchanged + `git status` clean**. Tempting, but `git status` itself stats every
   tracked file (that is how it knows), so it costs the same 40-60 ms as our walk plus a
   process spawn. No gain. Rejected.
2. **Directory mtimes.** A file created, deleted or renamed bumps its directory's mtime, but
   an in-place edit does not. Misses the commonest edit. Rejected alone.
3. **git's fsmonitor (`core.fsmonitor=true`, the built-in daemon on macOS and Windows).**
   `git status` then asks the daemon what changed since a token instead of statting. Our
   cache could store the daemon token and ask `git fsmonitor--daemon` (or read the
   `.git/fsmonitor--daemon` IPC) for paths changed since. That is a true O(changes) answer,
   but: not on Linux (no built-in daemon), it starts a long-lived daemon on the user's
   machine, and on a network share or after daemon restart it must fall back to the full
   walk. Possible as an opt-in accelerator, not as the default.
4. **Our own watcher.** Same as 3 but ours; a daemon per board. Too much machinery for 45 ms.
5. **Change journal written by every hug writer + the walk only when an outsider may have
   written.** hug's own writes already know their paths (board_delta). Outsiders are the
   problem, and we cannot detect them without looking. Rejected as unsound.

## Proposal

Keep the walk as the default: it is the only signal that is both cheap enough and complete.
Make it cheaper instead of skipping it:

- (a) **Walk with one `ReadDir` per group folder and `Lstat` only entries whose name is a
  task file**, instead of `WalkDir` + `d.Info()` (already the TUI's approach, step 7). Small
  win (the profile shows `os.Lstat` at 80 ms of 620 ms sampled across 5 runs).
- (b) **Parallel stat** over 2-4 goroutines (bounded, the machine is shared). Stats are
  syscall-bound, so they may overlap (not measured yet).
- (c) **Opt-in fsmonitor fast path** (`HUG_FSMONITOR=1` or when `core.fsmonitor` is already
  on for the repository): ask git for changed paths since the stored token, hash just
  those, fall back to the walk when the token is missing, invalid, or older than the last
  hug sync. Only where it is already enabled; we never turn it on ourselves.

Estimate, unmeasured: (a)+(b) might take the unchanged sync from ~45 ms to ~15-25 ms at
10,000 tasks, with no new correctness risk (same files statted, same rules). (c) would cost
one IPC round trip where fsmonitor is on. Each gets measured with bench44.py before it ships.

## Questions for Jørgen

1. Is 45 ms at 10,000 tasks (225 ms at 50,000) worth any new correctness surface, or should
   check 3 be re-worded to "the unchanged-board walk is measured and bounded" and ticked with
   (a)+(b)?
2. fsmonitor (c): acceptable as opt-in, or out of scope?

## How check 3 gates step 8 (show/note read one task)

Measured on board10k, 6190bbd, in-process: show spends Locate 36-50 ms, branch Open 17-29 ms,
cache.Open (blob decode, 14 MB heap) 28-37 ms, List 77-115 ms (of which the walk is 42-53 ms),
then 10-20 ms on the task itself. A single-row read removes the blob decode and snapshot
(~35-45 ms, ~18 MB) but the command still has to walk to be correct, and still needs every
task's id, aliases and parent for Resolve, dependency titles and children (a light-rows query
takes 12-13 ms). So step 8 alone is ~20% of show. Combined with whatever check 3 decides, it is
the bigger half. Proposed order: decide check 3, then step 8 on a markers-plus-light-rows open.

## 2026-11-17 14:00, after Jørgen's ruling: measured (wyvern)

Ruling: (a)+(b) by default; (c) fsmonitor where git already has it on, with a fallback to the
walk whenever the answer is not certain; every sync records its path.

(a)+(b) built on ci/44w1ev-walk: `walkMarkers`, one ReadDir per folder, task files folders
found from the listing in hand (no Glob per folder), DirEntry.Info on 4 goroutines.
In-process on board10k, interleaved with the old WalkDir, 21 runs each (load average 36):
old 40-46 ms median, new 25-34 ms. Same files and markers as WalkDir
(TestWalkMarkersMatchesWalkDir; dropping the files-folder skip or the symlink rule fails it).
bench44.py board10k n=5, main 5b80e87 vs walk, interleaved x2 (loaded machine; the first pair
ran at load ~36 and is noisy): show 0.328/0.266 -> 0.210/0.157 s, next 0.247/0.186 ->
0.219/0.160, board 0.244/0.184 -> 0.252/0.155, ls 0.655/0.454 -> 0.988/0.419 (the 0.988 is
the noisy first pair).

(c) fsmonitor, measured on a copy of board10k (macOS, built-in daemon, core.fsmonitor=true,
core.untrackedCache=true, daemon warm), median of 11, 3 rounds:
- `git version` alone (process start): 15-20 ms
- `git status --porcelain=v2 -z --untracked-files=all`: 40-45 ms, the same with the daemon off
- `git diff-files --name-only -z`: 22-27 ms (needs the index, misses untracked new files)
- walkMarkers: 32-42 ms in the same rounds

So through the git CLI, fsmonitor cannot beat the walk at 10,000 tasks. The status that would
answer "what changed" costs as much as walking, because git still reads and checks the
10,000-entry index and untracked cache, and every git process starts at 15-20 ms. A real win
needs to talk to the daemon's IPC socket directly (.git/fsmonitor--daemon.ipc, the
fsmonitor protocol v2 with a token). That is an unstable, undocumented wire format that git
can change between releases. On Linux there is no built-in daemon at all.
Proposal: do not build (c) now. Keep it as a recorded path name (Stats.Path "fsmonitor"
reserved) and revisit at 50,000 tasks, where the walk is ~5x and an IPC answer stays O(changes).
"Never skip on a guess" holds: no skip without a certain answer, and the CLI cannot give a
certain answer faster than the walk.
