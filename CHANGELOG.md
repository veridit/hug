# Changelog

Releases are tagged `vX.Y.Z` and ship one `hug` binary. Earlier part releases
are in [muninn/CHANGELOG.md](muninn/CHANGELOG.md) and the `muninn-v*` and
`huginn-v*` tags.

## Unreleased

- hug tui opens a task that asks you something on the question itself
  (qcq3tm), scrolled into view and focused, under its check or in its
  own block. A file the question names (--show) is no longer drawn above
  the page: one dim line under the question names it, `→ NAME · Enter
  shows`; Enter shows it in the file frame and Esc goes back to the
  question. `a` always opens the answer box in view, under the question
  it answers, even when that question is far down the page (p87ngv).

- `hug task lint` checks only live file links (7j706s): `file:NAME` and
  `task:REF/NAME` outside the record. A path quoted in prose
  (`.hug/tasks/ID/NAME`), in ## History, in an answered or withdrawn
  question or in a resolved request is text, not a link, and raises no
  missing-file. A dangling `file:` link in a live section is still broken.

## v0.7.88 - 2026-11-20

- `hug task done` refuses a task with a question still open to someone
  (72n8m4): exit 3, `fix: hug task resume ID` withdraws it on purpose.
  The board cue no longer offers done for such a task.
- After a release nothing is left as shipped-not-done that the flow can
  settle (yf39ya): a todo task with an open question is quiet; a shipped
  task whose open check is not a person's is handed to its owner as
  active work (a "handed on after vX" note), and the owner's tick asks
  the person check or closes it; a shipped task waiting on a dependency
  is quiet, and its person check is asked when the dependency is done.

## v0.7.87 - 2026-11-20

- hug tui lists broken tasks before advice (z6cb03): the overview's LINT
  section and the lint view put every task with a broken finding first,
  rank order within each, as `hug task lint` does. On a board with many
  long Goals the broken ones no longer scroll out of sight.

## v0.7.86 - 2026-11-20

- The Goal is one sentence of at most 140 characters; detail goes in
  Details (0pr03h). `hug task new` and `hug task set` (and MCP
  `task_new`, `task_update` set) refuse a longer Goal with `fix: hug task
  set ID --goal "one sentence" --details "..."`. `hug task lint` warns
  on one already on disk (`goal-too-long`, advice) and never rewrites it.
  A sentence ends at `.`, `!` or `?` followed by a capital, or at a
  blank line; `e.g.`, `v0.7.85`, file names, initials and ids never
  split one. Promote puts "Promoted from item N" in Relevant details.
- The view line's blink tests no longer wait on the wall clock (6zhyq2,
  CI 44520969132: a blink measured 497 ms against 530 ms). The two tests
  that ran bubbles' real 530 ms timer now run it in a testing/synctest
  bubble, whose clock moves only when the test waits, and check the
  exact 530 ms; the tick helpers take their message with no timer.
- Sync tests wait for a background pass while it makes progress (6zhyq2,
  Windows CI 44522707343 attempt 1). settled failed any pass still
  running at 30 s; on a slow runner one was still starting git then. It
  now fails a pass with no new log line or git event for 30 s, or after
  3 minutes in all, and says which.
- `TestBackgroundSyncOfflineCatchesUp` checks that a note never waits on
  a hanging remote by order, not by time (4gj7z6, Windows CI
  44638912154: a correct note took 2.749 s against 418 ms without
  background sync). The remote now hangs until the test releases it,
  which it does only after the note has returned. A note that calls the
  remote itself fails (the hang marks a call from outside the background
  pass), and one that waits for the pass never returns and fails with
  the pass log before go test's -timeout.
- Tests leave no tmux server running (wacc33, crab 2026-11-19 12:01Z:
  three huginn-golden/reload servers, parent launchd, alive for days). A
  test's tmux server now dies with the test however it ends: pass,
  t.Fatal, a panic in a subtest, go test -timeout, or SIGKILL of the test
  process. A watchdog shell holds the server and SIGTERMs it when a pipe
  only the test process writes closes; cleanup is also a t.Cleanup, and
  it fails the test if the server still runs. Every test server is named
  huginn-NAME-PID (the muninn terminal goldens too), and each tui test
  run first reaps any huginn-* server whose test process is gone or
  whose parent is launchd. No other tmux server is touched.

## v0.7.85 - 2026-11-19

- Mid-turn recall on jcode, off by default (x8zzzn). A message typed into a
  running jcode turn (a soft interrupt) ran no recall: since 2026-11-18
  03:53Z, 122 of 123 such messages had no recall line. When such a message
  came before the turn's first tool result, it also made the turn's parked
  block "stale", and the block was dropped (8 of 13 stale drops). With
  `"recall": {"jcode_midturn": true}` in config.json, the next tool result
  recalls each such message and carries its block, after the turn's parked
  block, which is no longer dropped. The turn is jcode's own turn id
  (`model_usage_turn_id` in the session journal), so a new turn still drops
  the old turn's block. What is recalled does not change: a test shows the
  same message gives byte-identical output through both paths. Nothing is
  printed outside a tool result, so an idle session is never woken for a
  memory, and recall stays local, with no model calls.
- Delivery logging for jcode recall (x8zzzn). `injections.delivered_ns`
  records when each recalled memory reached the agent: 0 until then, and 0
  for a block dropped with no tool call. Older stores get the column on
  open. hook.log now says `replaced parked=X by=Y` when a park is
  overwritten, `midturn turn=X msg=Y` for each mid-turn recall, and
  `midturn turn=X missed=N` at turn_end for messages typed after the last
  tool call. Hook errors, timeouts among them, now name the session and
  event (`jcode session_… turn_start: context deadline exceeded`).

## v0.7.84 - 2026-11-19

- The file view in `hug tui` fills to the keys (yp3c09, Jørgen 2026-11-19
  11:30Z on p87ngv's before.md). Two empty rows sat between the file's
  bottom rule and the key bar while the file went on below: the message
  line and the input line, drawn blank under every file. Now a file, its
  raw source (M) and a picture get those rows while nothing is in them;
  a message ("first file: 1 of 3", an error) still gets its line above
  the keys and the file gives that one row back while it shows.
  Goldens at 100x30 and 160x45, in a real terminal too.

## v0.7.83 - 2026-11-19

- The sort picker in `hug tui` (s) moves like the filter picker (17w9r6,
  Jørgen 2026-11-19 11:41Z). A highlight starts on the first row; j/k,
  ↑↓ and ctrl-n/p move it. Space on a field adds it to the order, and
  Space again flips it, as the field's letter does. After j/k, Enter does
  the same, and Enter with nothing new picked applies the order. On r or
  c, Space or Enter does what r and c do. Letters still pick at once and
  move the highlight to their row, so a letter and then Enter applies as
  before. A field in the new order shows its place and direction ("1 ↓")
  instead of a `>`, so `>` marks only the highlight, as it does in filter.
  The footer and ? help list j/k ↑↓ for both pickers.

## v0.7.82 - 2026-11-19

- The view line in `hug tui` gives each cue one meaning (r80svs, Jørgen
  2026-11-19 11:09Z, 11:11Z). Bold marks a key you can press: v, o, l, b
  and !. Colour marks the step of the key sequence, v in the accent and
  o l b ! in violet, so you can see that v opens the others. Underline
  marks state: the view you are in has its whole word underlined, its key
  letter too (until now it was bold, and hard to tell apart); after `v`,
  `v` keeps a steady underline (instead of v0.7.81's filled cell) and the
  underlines of o l b ! blink together on the same 530 ms ticks, while
  the active word's other letters keep theirs. The second key or Esc
  brings the line to rest, the new view underlined. With `NO_COLOR` the
  bold and underlines are still drawn, without colour. Nothing on the line
  is filled any more.

## v0.7.81 - 2026-11-19

- After `v` in `hug tui`, the keys you can press next pulse instead of a
  second cursor (a82j3n, Jørgen 2026-11-19 10:26Z, 10:27Z). While `v`
  waits, the underlines of o, l, b and ! blink together at the cursor's
  530 ms, on the same ticks, the letters keeping their violet; there is
  no cursor cell after `view:` any more; and `v` itself is drawn solid, a
  filled accent cell, for as long as it waits. The second key or Esc
  ends the blink and puts `v` back to its underlined accent. The answer
  box cursor is unchanged. Goldens at 100x30 and 160x45 for both phases.

## v0.7.80 - 2026-11-19

- A question left open with no stage to answer it by can be answered again
  (m5pbb0). On 2026-11-19 Jørgen's `hug tui` was a v0.7.69 left running
  since 2026-11-17, from before several questions per task (q2adr8,
  v0.7.70). It answered question 1 of p87ngv into History only and
  cleared `design_*`, so p87ngv went to ready with questions 1, 2 and 3
  still open in `## Questions`. A current hug drew each with its own
  `a answer`, but every answer was refused with "p87ngv asks nobody: its
  stage is not design, asked or review". Now an open question can be
  answered whatever the stage fields say: `hug task answer ID N`, the TUI's `a` and
  MCP answer it, and the stage comes back as design (todo) or asked
  (doing), on the oldest question still open. Tests replay p87ngv's file
  as it was at 10:05:26 and at 10:05:47 (five questions), goldens at
  100x30 and 160x45.

## v0.7.79 - 2026-11-19

- hug mem: search matches query words against concepts in one read instead
  of one full-table `LIKE` scan per word, and looks up a repeated word once
  (b5jks7). Capture planning spent most of a hook's time there: replaying
  capture run 8476 (14 chunks, 65 candidates) on a copy of the live store,
  planning took 115.8 s before and 66.3 s after (chunk search 63.8 -> 20.3
  s, per candidate 800 -> 707 ms). The Go matcher is SQLite's LIKE ('_' is
  a wildcard, ASCII-only case folding); search and recall output are
  byte-identical on those 79 searches and on the 277 recall4 messages.

## v0.7.78 - 2026-11-19

- hug mem rescope records each move in the fix trail (j13xs0): one
  memory_fixes row per memory, with its old and new scope, the time, and
  how (single, or batch for `--from`). A dry run records nothing.
  `hug mem stats --fixes` counts rescopes per day and lists them in recent.
  Before, two batch moves (58 and 9 memories, 2026-11-18/19) left no row and
  had to be traced through session journals. What rescope moves and its
  backups are unchanged.

## v0.7.77 - 2026-11-19

- hug mem capture: a hook starts an extraction call only when the time left
  covers it (a6xm8n). The margin is what this run has seen: the slowest
  search, the slowest call (at least 5 s; p99 of 5482 hook calls since
  2026-11-12 was 4.7 s) and the slowest save, checked before a chunk and
  again before its call. A run that runs out of time ends with skipped
  `remaining messages deferred to next hook (budget)` and no error. Before, a
  fixed 3 s check before the search let runs 8467 and 8468 start their last
  call with about 4.5 s and 0.6 s left: it was cut, and runs with 83 and 88
  applied actions were recorded as failed. A deadline that cuts a reply body
  mid-read is now a timeout, not `invalid extraction response` (parse).

## v0.7.76 - 2026-11-19

- hug mem capture: a hook call whose reply is cut at the hook's 1024-token
  cap is sent once more at 2048, and a truncated reply is never applied
  (nmn6wf). Before, it ended the whole hook, and the next hook re-sent the
  same chunk, which at temperature 0 was cut again while its input stayed
  the same (crab_1793783281662: runs 8453, 8459, 8460). A chunk still cut is
  recorded in capture_runs (`truncated`, `deferred`), the run goes on past
  it, and a later hook of any session sends it again at the full cap, so the
  last chunk of a session_end is not lost; it also stays in the backlog for
  `hug mem capture --backfill`. Of 1905 audited hook calls, 21 (1.1%) hit
  1024.

## v0.7.75 - 2026-11-19

- `hug completion bash|zsh|fish` prints one completion script for hug: its
  commands, hug task's commands, flags, types, states and open task ids (with
  titles), and hug mem's commands and flags. The script asks `hug __complete`
  on every TAB; it reads the task files and a read-only view of the cache and
  never writes, syncs or locks (5mmqjk).

- `hug task new` dates `created` and `updated` by the board clock, as `done_at`, reflections and strikes are (z25pb2): one clock for every date hug writes. A test pins it at 23:59:59.9 on 2026-01-31 and finds that date in both fields (it found the wall clock's before). The TUI's find golden showed the wall-clock date masked as `YYYY-MM-DD`; its board now pins the clock and the golden shows `created 2026-01-31 · updated 2026-01-31`.

- The tmux tests no longer fail a working tmux server on a busy machine (ghycrq). The harness timed tmux client processes: "foreground tmux server did not become ready" when `tmux show-options` took over 1 s to start, and "tmux server did not exit after explicit cleanup" when `tmux kill-server` took over 5 s, which then never signalled the server. Under 16 CPU hogs, starting /bin/echo took up to 11.6 s. Readiness is now a connection to the server's socket from the test itself, and cleanup sends the server SIGTERM, as `tmux kill-server` does, and waits for it to exit. TestTmuxServerSlowStarts runs the harness against a tmux that takes 6 s to start.

- A task two clones close by the same release has one close in its History (j0zvee). When two machines deduced the same release tag before either synced, each closed the task by it at its own minute, and the merge kept both History lines, which differed in their minute only (CI 44412479085: "2 release lines"). It also noted a done_at conflict that was none. The merge now counts a release close once: the same transition by the same release keeps the board's line and the board's done_at, whatever minute the other clone stamped. Any other History line keeps its time, so two resumes a minute apart on two clones stay two lines.

- `muninn/docs/RECALL4.md` says where the recall4 harness keeps its databases and that a `HUG_HOME` copy of the store still points at the live `memories.json` until its source is dropped, with the commands for a clean copy (nrp268, from reflection 5sc5rw).

- Hook and MCP entries are named hug (ynh84t). `hug mem install`, `hug task install` and `hug setup` write `<hug> mem hook X` and one `hug` MCP server instead of the muninn and huginn links and servers. `hug upgrade --apply --machine` moves the entries an older hug wrote, plus Claude permissions for their tools (`mcp__muninn__*` becomes `mcp__hug__*`), with one `FILE.bak-hug-upgrade` per file; a second run has nothing to do. A `hug` entry hug did not write is left alone and reported as `names: skipped`. The old links keep working until the server is reloaded (jcode: `mcp action=reload server=hug`; Claude, Codex: restart the app).
- `hug upgrade` takes JCODE_HOME from the environment it was given, not the process's own (ynh84t). Before, a caller with its own JCODE_HOME got the process's jcode home previewed and, with `--apply --machine`, rewritten.
- Tests can no longer reach this machine's harness configs (ynh84t incident, 2026-11-19 00:32: an upgrade test rewrote 7 live jcode and Claude configs, restored from their backups). Every package that runs install or upgrade points HOME, JCODE_HOME, XDG_CONFIG_HOME and CODEX_HOME at a temp folder first, and fails the run naming any real config that changed.

- Capture no longer stops for good on one long message (8jcj3k). A message longer than one extraction call (8,192 bytes at a hook) was refused with `capture message exceeds 8192 bytes, refusing silent loss` at every later hook, so nothing more of that session was captured: one coordinator session was refused 841 times since 2026-11-16. Every capture path now cuts such a message into parts that fit, in order, sends every part, and records the cut as `split` in the run's audit record (`capture_runs`) beside each part's `original_id`. No part is dropped and no model call is added. Sessions stuck this way resume at their next hook after the upgrade.

## v0.7.74 - 2026-11-19

- On macOS, closing a file watch no longer closes a file it does not own (mq5ger). fsnotify v1.10.1's kqueue backend closed each watched file's descriptor twice when the file was deleted while the watch closed: once on the delete event, once on close. The second close hit whatever had taken that descriptor number in between, so another file hug held open lost its descriptor. On CI this showed as a test's temp-folder clean-up failing with `unlinkat ...: bad file descriptor` (run 44384875316). hug now carries fsnotify v1.10.1 in `internal/fsnotify`, with the upstream fix (fsnotify PR 789, not yet released) applied: the event reader releases the watches on its way out, and close waits for it. `TestCloseWhileFilesGoClosesNothingElse` fails first on v1.10.1 (2 of 3 runs of 300 rounds lost a held file's descriptor) and passes 10 of 10 with the fix. The copy's whole diff from v1.10.1 (four import lines, PR 789, and one call PR 789's lock needs on v1.10.1, `TestRemoveWatchedDirReturns`) is in `internal/fsnotify/upstream.patch`, held by `TestUpstreamDiff`. Linux and Windows are unchanged.

- The TUI's reload no longer waits on git (adqdfe). Every load asked git for the close times of RECENTLY DONE before it returned, so on a machine too loaded to start a process the board waited too: a change made elsewhere could take seconds to show, and TestChangeBeforeWatchIsUpIsSeen failed 4 in 1000 starved runs ('no reload after the watch came up'). The board now lands with the close times already known, and git is asked after it; RECENTLY DONE takes the new order when git answers. The TUI's own writes still read them at once.

- `hug task set` says what it wrote before any lint advice, one line per changed field (`added check 2: VALUE`, `title: OLD -> NEW`, `tags: + parser`), and `unchanged: ...` when nothing changed; MCP `task_update` set returns the same lines as `wrote` (zdgw60, from reflection asr4t2: a set whose reply ended in a lint `fix:` read as failed, and the check was added twice).

- Tests: a delivered request's acknowledgement is waited for, not timed (awht8w). TestRequestReachesCoordinator failed once in a niced full suite at load 95-115 with "crab's hook did not acknowledge", and passed alone. The hook answers on every path, so the failure meant only that it took more than the test's 20 s. From the delivery to the ack is about eight git processes in a row, and at load 137-500 one niced git took up to 14 s (traced: `git remote` 14.069 s, `git diff` 10.913 s). With every git started 7 s late, the test failed the same way on c186f78f. It and TestDeliveryInstructionsMatchCode now wait on the hook itself; a hook that never answers stops at `go test -timeout` with its stack. hug's own code is unchanged.

- `hug task done`'s reflection line and `hug task strike`'s note are dated by the board clock, as `done_at` already was (jsyksd). A done that ran across midnight could write `done_at` from one clock and the reflection date from another; on CI 44424953831 TestDoneWithoutAnswerIsCounted read today before midnight and found the line dated after it. The tests that compare these dates (TestDoneWithoutAnswerIsCounted, TestDoneReflectAndReflections, TestLifecycleJSONContract, TestStrikeThroughCLI, TestStrikeOutCheck) pin the clock at 23:59:59.9 on 2026-01-31 instead of reading the wall clock: all 5 failed with the wall-clock dates and pass with the board clock.

## v0.7.73 - 2026-11-19

- `hug tui` marks tasks in lists (rxr4z8). In the overview, the list and the board, Space marks the task under the cursor and steps to the next row; Space again unmarks it, and Space on a grouped heading still folds. A marked row shows `◆` in its gutter, in dim accent, and nothing else on the row changes; the cursor on a marked row shows `⋗`. While tasks are marked the footer reads `N marked ·` the actions valid for every one of them `· Esc unmark` (narrower than 70 columns, `N marked · ? actions · Esc unmark`, and `?` lists them in words), and an overview section heading counts its marks. `d`, `x` and `w` act on every marked task after one confirm naming the count and the ids (`x` and `w` ask their reason once); each is its own write and commit, a refused task is named with its `fix:` line and stays marked, and the tasks that went through are unmarked. `t` takes the marked tasks after one confirm naming them: the first plainly, the rest with it (`hug task take B --with A`, byj479), one commit each; with a task that is not todo among several, it is not offered. `e` opens every marked task in one editor. `m` grabs the marked tasks as one block: they gather at the first marked row, `j/k` move the block, Enter writes it in order with one commit, Esc puts every row back. Esc unmarks all.

- TestTmuxReload no longer fails on a busy machine (hfkt1n). Under load the tmux server reads hug's frame in pieces, and the wait after Esc took the first capture whose top line was the list, with the detail's lower rows and its "reloaded hug" line still below it. It now waits for the list's own key bar too. The wait after R also waits until the reloaded hug's first load is done, so "refreshing" in the header no longer fails "the same place". TestReloadFramesSettle holds both waits to frames captured under load.

- A withdrawn question reads as withdrawn: `hug task show` prints `withdrawn <at> (<why>)` under it, and the TUI detail draws it dim with its time; ASKED never counts it (nanvnt).

## v0.7.72 - 2026-11-18

- The README leads with `hug`: one command, named for Old Norse *hugr* (mind), with Odin's ravens as mascots, Huginn for tasks and Muninn for memory (p277bp check 1). Its commands table lists only `hug` commands, adds `hug apps` and `hug upgrade`, and says memories sync on `hug/memory`.

- Lint reads a glob citation as the pattern it is (ydwamb). 911j70's note asks to run `open -a Preview .hug/tasks/9/911j70/compare-*.png`; lint cut it at the `*` and reported `.hug/tasks/9/911j70/compare-: file not found`, with `fix: hug task attach 911j70 FILE --name compare-`, which would attach a junk file. A citation whose last path part holds `*`, `?` or `[...]` (`file:`, `task:REF/` and `.hug/tasks/` forms) is now matched against the task's files: no finding when one matches, else one missing-file finding naming the whole pattern, `no attached file matches`, with `fix: hug task attach ID FILE`. A plain missing file is still `file not found`. Prose after a whole name stays out of it, as before (`file:a.md?`, `**file:a.md**`, `[file:a.md]`). Lint only reads. On this board, lint on 911j70 now reports no findings. Cache schema 28.

- `scripts/ci-goldens.sh RUNID [JOB]` rewrites the TUI goldens a failed CI run reported as drifted, from that run's own screens, for when a local `HUG_UPDATE_GOLDEN=1` run cannot finish (a0gwvg). It writes nothing unless every WANT block in the log matches the committed golden byte for byte. First used on CI 44403710185 (4d7eec2d). A repository tool, not part of hug.

- Help and docs say only `hug` (p277bp). `hug help` no longer says "The commands huginn (= hug task) and muninn (= hug mem) work too", and every help text names `hug task` and `hug mem`: `hug mcp`, `hug setup`, `hug task help install`, `uninstall` and `import`, and the `hug mem` descriptions of `import`, `uninstall`, `install --bin`, `jcode instances` and `model status`. A test walks every help text (`hug help`, `hug task help` and each command it lists, `hug mem --help` and each command it lists) and fails on huginn or muninn as a word: 27 lines before, 0 after. The part READMEs (`huginn/`, `muninn/`), their docs and the TUI READMEs show `hug task ...` and `hug mem ...` (111 lines), with the current layout (`hug/tasks` at `.hug/tasks/`, project memories on `hug/memory` only when the repository opts in). The `huginn` and `muninn` links still work.
- `hug mem completion` is gone: it completed the old command name. Scripts already installed keep working.

## v0.7.71 - 2026-11-18

- `hug mem` no longer panics on a machine without `~/.jcode` (h2yn55). With only Claude Code or Codex installed, every `hug mem` command stopped with `slice bounds out of range [1:0]`; now `hug mem remember` and `recall` work there and create no `~/.jcode`.

- TUI rows show a task's tags right after its ID, dim, at most 22 columns (txjkrm checks 3 and 6, Jørgen: pie, id, tags, type, priority, stage, text). The tmux goldens, the palette included, carry the column at 80 and 140.

- hug names an MCP server too old to reload itself (z77pem). A `hug mcp` started before v0.7.54 never picks up a newer hug (mbdfzs reloads only servers started on v0.7.54 or later), so its calls run old rules and fail in ways an agent cannot read: on Statbus, `memory_remember` failed with `cannot modify graph_state because it is a view` from a 0.7.2 server started on 15 Nov. Once per session, at its first turn, hug looks for `hug mcp` servers whose binary is a Homebrew Cellar hug older than v0.7.54, with `ps`, plus `lsof` on macOS or `/proc/PID/exe` on Linux (about 0.1 s, once; tool calls never scan). The agent hears only of the server serving its own session, the one it can restart. For jcode that is the server of the session's jcode home; for Claude Code and Codex it is the server whose app the hook runs under. jcode shows it with that turn's recall at the first tool result; Claude Code and Codex show it in the prompt hook. It is one line, `hug: your hug mcp server (pid 3175) runs hug 0.7.2, older than v0.7.54: ...`, with `fix: mcp action=reload server=hug` (in Claude: quit and reopen it). Other apps' stale servers, or a session whose server hug cannot identify, get no line in the agent's turn. They go to hook.log, and `hug mem stats` lists every stale server on the machine (`stale_mcp`) for a person. hug never stops another app's process. Builds with no version in their path are not judged. Windows prints nothing.

- A hug MCP server too old to reload itself now says how to fix its failed memory write (mbdfzs check 3). Servers started before v0.7.54 never reload, and since v0.7.59 their `memory_remember` failed with `cannot modify graph_state because it is a view`, which names no fix (Statbus, 2026-11-18). The memory store's `graph_state` view now fails the read such a hug makes first, so it reports: `hug: this hug mcp server is older than v0.7.59 and cannot write this memory store; fix: restart it (jcode: mcp action=reload server=hug; Claude/Codex: quit and reopen the app). The note was saved: the next current hug that writes or indexes (hug mem index) makes it searchable.` Its searches still work. Checked with a v0.7.2 build: CLI and MCP `memory_search` answer, and CLI and MCP `memory_remember` print that line. Every store gets the view the next time a current hug opens it.

- TUI: in a task, j and k mirror each other (ykta5g, Jørgen: "J goes down but K goes to top again"). After j steps to the last check, dependency, link or file and then scrolls the page past it, k first scrolls back one line at a time with that item still marked, and steps to the previous one only once it is in view. Before, the first k re-centred the page, to the top on zhvkmj (offset 23 to 0 at 100x24). A step that leaves the view now scrolls the page just far enough to show the new item, never re-centring. And when a check gets focus, on open or by j/k, the whole check shows, its question and who asked included, not only its first line (Jørgen, gav9nw request 1); a check taller than the page shows from its first line, and j reads it down a line at a time before stepping on.

- A git that finished is no longer reported as failed on a busy machine (ad62aa). hug read git's output through pipes, and after git exited os/exec gave the copying goroutines 2 s (WaitDelay). At load 300-700 they did not run in time, so a git that had exited 0 came back as "exec: WaitDelay expired before I/O complete" with its output lost. A commit that landed was then reported failed, and a lost `rev-parse` read as "hug task needs a Git checkout as its working directory" (TestDeduceTwoClonesOneLine and five more in one niced suite). git now writes its output to temporary files, which are read once it exits. A process that outlives git, such as ssh or a hook's background job, can no longer hold the answer open either. The network deadlines are unchanged.

- Tasks that ship together are both in progress (byj479, Jørgen with a screenshot of sql_saga: sabertooth worked dee3fq and bqtzce for one release, hug refused the second take, and bqtzce sat in NEXT UP as ready with 2/2 checks ticked). `hug task take B --with A` takes B alongside A, the taker's active task: both are doing, B records `doing_with: A`, History says `take with A`, `hug task show B` says `with A (title)` and `show --json` has `doing_with`. B does not count as a second active task, so a third take is still refused; each closes on its own, and once A is parked or done, B is the work in hand. `--with` anything but the taker's active task is refused with a fix naming the active one (or a plain take when nothing is active). The busy refusal now offers it as a fifth fix after the detour, saying when: if they ship together, take it with the active one; if it is a detour, the active one waits on it. Ticking a check on a todo task you have not taken still ticks it, and the reply says the task is not taken with `fix: hug task take B --with A --model M --as NAME`. MCP `task_take` takes `with`. In `hug tui` the row in IN PROGRESS says `· with A` and the section counts both. For the TUI's marked-tasks take, `Store.TakeWith(ref, with, user)` is the one call. Help and the AGENTS.md section say it in one line.

- A check is a person's when it opens on that person, whatever the verb (ebben8). `Jørgen attaches a screenshot ...` (5z4zt4 check 5) was not asked of him after a release because "attaches" was not in a list of verbs. Now a check, wait or ask that opens on a person in hug.toml `[people]` by any of their names (short name, alias, long name such as `Jørgen H. Fjeld`, or first name), followed by a lower-case word, is theirs: asked of them after the release that ships it, refused by `hug task wait` with the `hug task ask` that says it, and flagged by lint as `wait-on-person`. His words quoted as a spec (`Jørgen 19:35Z: ...`), a thing of his (`Jørgen's machine`) and `owner` as a plain word (`the owner table`) are still not. Measured over every open check and wait on hug's board: 13 checks read as his before, 20 now, none lost, and 1 wait; Seid's 3177 are unchanged (it calls him "the King", which an `alias` in its `[people]` would cover).

- `hug mem sync` has a placement gate beside the secret check (rnpdnm B, gav9nw: Jørgen chose A+ and B together). A project memory stays on this machine when it names a home folder (`/Users/`, `~/`, `/home/`, `C:\Users\`), another project or client, or a person, or is a preference or correction. Names match case-insensitively as whole words, with ø/æ/å and accents folded ("Jørgen's", "Jorgen", "jhf:" all hold), and a project's own name never holds its memories. The names live in the local config only (`sync_gate.terms` in muninn's config.json, an additive key), never in a repository. The first sync seeds them once from the places this store holds memories for (other projects' and clients' folder names, client names, jcode instance names) and from hug.toml `[people]`, and says so naming the file, never the names. Sync prints one line per reason (`held 40: names a home folder`) and `fix: hug mem sync --held`, a read-only list of each held memory with its reason. Letting one push is a person's call after reading it: `hug mem sync --release ID` (a local exception) or `hug mem rescope ID --to personal`. A held line that was pushed before the gate drops from the branch's tip, and sync says once that the remote's history still has it. Measured on hug's 868 public memories (free, the gav9nw labels): 13 of 705 pushed don't belong (1.8%, Wilson 95% [1.1, 3.1]), against 58 today; 118 of the 810 that belong are held (14.6% [12.3, 17.2]). Paid calls: 0. This applies only to a repository that shares its memories (34r1ft).

- The capture extractor can place a memory as personal or as about another project, behind a switch that is off (rnpdnm A+, `extract.wide_places`, an additive config key). Off, the capture prompt is byte-identical to before: a test pins its sha256 with and without the session's places. On, a session that may write the personal scope (never a client session) offers two more places: personal (the person: preferences, habits, machine, home paths, accounts, work outside this project) and other (a different project, client or product, with `place_name`). Both are stored in the local `personal` scope, which no project sync writes; an other memory is tagged `about:NAME` so `hug mem tidy place` can move it deliberately later. Nothing is written into another project's scope automatically, and the model is sent meanings only, never a scope or another place's name. It turns on only after rnpdnm check 3 passes on fresh captures, judged blind.

- S in hug tui asks where the screen goes (hjbfzd, Jørgen: "attach it to ... ticket so-and-so or S for select or C for create"). The copy is made at once, as before; the input line then reads `Attach this screen to: ID → its title · select · create`. The field offers the task in view, so Enter alone attaches there as before; typing replaces it with another task's id (a prefix is enough, Tab completes it, an ambiguous or unknown one says so and attaches nothing). While the field is untouched, `s` opens the rows to pick the task with the list's own keys (`#` and `/` too; the writing keys only say what Enter does), and Enter on a row attaches there and returns to where S was pressed. `c` asks for a title and creates the task with the capture in its Files, in one commit, then opens it. With no task in view S now asks too, with an empty field (it said "no task in view"). Esc keeps only the copy and writes nothing. The prompt follows the calm rules: no background fill (the old confirm bar had one), `s` and `c` underlined in the accent and not bold, and the field's cursor is the theme's still underline.

- `hug mem sync --migrate` no longer runs the code repository's git hooks (9vp9by). The migration commit removes the old `.muninn/memories.jsonl` from the code branch, and it ran the project's hooks, unlike hug's other commits. A commit-msg gate, such as hug's own `Model:` gate in a repository that installed it, refused it, and a hook that left a background process running made the migration wait for that process. The commit now runs with `core.hooksPath=/dev/null`, as hug's task and memory branch commits do.

- `hug task ask` works on a todo task without taking it (yphbra). Before, asking a design question on a todo task that waited on another needed the dependency removed, a take, the ask, and the dependency added back (ec6z6j). Now the question goes in `## Questions` as on a doing task; the task stays todo with its assignee and dependencies, `design_*` show the oldest open question, and it is in the asked person's ASKED at once, even while a dependency is open, because the asker chose to ask now. A `hug task design` question keeps the old order and waits for the task's dependencies. Answers (`answer ID N`, or ticking the check) go to the asker. Taking a task with open questions keeps them: it becomes doing/asked, nothing is withdrawn. `--after` still needs your own doing task. MCP `task_update` ask does the same.

## v0.7.70 - 2026-11-18

- `TestBackgroundSyncOfflineCatchesUp` no longer sits out a fixed hang (0n4rnx). Its fake remote slept 5 s per network call, and the passes after the timing check paid it again, 32 s for the test on Windows. The remote now hangs until the test releases it (at most 10 s), then fails at once, still unreachable: 11.67 s in CI 44370638232. This changes a test only, not hug.

- A catch-up replay reads the board's objects through one `git cat-file --batch` per call (0n4rnx), not one git process per file, tree and parent. A catch-up of 12 commits started 221 git processes and now starts 66 (`TestCatchUpProcessesPerCommit`, at most 5 per commit plus 20). The reader is closed on every path, a stopped replay included (`TestCatchUpStopLeavesNoReader`). The replayed commits are unchanged: on the three recorded catch-ups (across a layout move, both clones moved, a first-parent merge resolution), old and new hug gave the same trees, authors, dates, messages, status and upgrade output.

- Project memories stay local unless the repository opts in (34r1ft, wxsyc5). Before, `hug mem sync` and its background passes pushed every project memory to a branch on the remote, public repositories included: public veridit/hug holds 868 of them on `hug/memory`. Now the setting is committed with the tasks, `[memory] share = true` in `hug.toml` on `hug/tasks` (Jørgen: a property of the repository, not the machine), so every clone follows it, a fresh one too (it reads `origin/hug/tasks` until it has its own). Absent means local, and local is not an error: `hug mem sync` there says `project memories stay on this machine (hug.toml [memory] share is not set)`, exits 0, and makes no worktree, fetch or push; a remember or recall starts no background pass; and `hug upgrade` renames a local `muninn-notes` to `hug/memory` on this machine only, leaving the remote's branch as it was (`remote muninn-notes kept; this repo keeps memories local`). Nothing hug prints on a local repository steers toward sharing (no `fix:` or `next:` line names it), since an agent runs those lines and in a public repository that would publish. A person opts a private repository in with `hug task memory share`; `hug task memory local` goes back, and bare `hug task memory` says which. Memories already on a remote are not removed. hug's own repository does not opt in.

- An answer's outcome reaches the coordinator (41hhgn check 11). When a person answers a question or review (typed, or by ticking the asked check), the asker's next write on that task is recorded under the answer as its outcome (`NAME read the answer: WRITE`), and hug now hands that outcome to the board coordinator the same way it hands the answer, in one line: `hug: nautilus acted on jhf's answer on ID (Title), check 2, on board NAME (PATH): hug task note ID "..." (the answer: "..."). hug task show ID [token]`. The writing command's reply says `outcome_receipt` (delivered or not, with the reason); the coordinator's hook acknowledges it on receipt. With no live session it waits for the coordinator's next session start, once. An answer still pending there is no longer owed once its asker has acted, so that session gets the outcome line alone, which carries the answer. The recall hook's board cue no longer repeats an outcome already delivered to that coordinator. No agent polls. The delivery text in AGENTS.md (hug setup) and the MCP tool descriptions says so.

- `hug tui` looks at a picture properly (33v71x). In a task's file frame, `+` (or `=`) and `-` zoom in steps from fit to 150, 200, 300, 400, 600 and 800 %, `h/j/k/l` scroll the zoomed picture, `0` fits it again, and the crumb shows `zoom N%`. `o` (and Enter), on a picture in the frame or marked in Files, hands it to the system viewer (`open` on macOS, Preview; `xdg-open` on Linux; `explorer` on Windows) and stays on the same screen; hug does not wait for the viewer. A picture never opens in the text editor; Enter on a text file still edits it. The footer and `?` help list these keys for pictures only, and zoom keys only where the terminal draws images.

- Shipped work leaves the board clean (yf39ya, Jørgen 2026-11-18: after a release hug's board showed 40 findings, 37 broken, mostly bookkeeping). No finding is silenced or downgraded; the flow now acts when each thing happens.

- Several questions on one task, each answered on its own (q2adr8, Jørgen on rxr4z8: "If you ask three questions, I should have three questions to answer in the ticket"). Every `hug task ask` adds a numbered question to a `## Questions` section of the task, never reusing a number, and its answer is written under it. The task stays asked until every question is answered; the stage's `asked_*` fields show the oldest open one. `hug task answer ID N "..."` answers question N; with one open, `hug task answer ID "..."` answers it as before; with several open, a bare answer is refused with `fix: hug task answer ID N "..."`. Ticking the check a question is about answers that question only. History says `answered asked (question N, check C)` when a task has more than one. `hug task show` lists the questions with their answers; `show --json` has `task.questions` and the asker's `answer.question_n`; MCP `task_update` answer takes `item`. In `hug tui` each open question asked of you has its own `a answer`: those naming no check are blocks at the top, each its own Tab stop, one naming a check sits under it; the answer box opens under the question marked `>`, and after it the `>` moves to the next one. The ASKED row says `N questions`; the record of answered ones is under QUESTIONS. An ask whose text holds a numbered list of two or more questions is refused with `fix: ask each question on its own: hug task ask ID --to NAME "<first>"`; replayed over hug's own board history (34 distinct asks) it refuses one, rxr4z8's. Questions still open when the task leaves asked (resume, wait, done, drop) are marked withdrawn under them, so a later ask does not bring them back. A task asked before this keeps working: its question becomes question 1 when a second is asked. The ask and answer help and the AGENTS.md section (`hug setup`) say one question per ask.

- In a task's detail, Tab to Files brings Files into view (zhvkmj check 5). A task that shows a screen capture above its page (an asked picture) scrolled to the capture's own `> ` rows instead, so Files had the focus but was never on screen; the focused row is now looked for under the focused section's heading only. Depends is a calm indigo in the count line, the rows and the board's DEPENDS heading (zhvkmj check 6); red now means only broken lint.

- `:` opens a palette under what it acts on (05r3pg). The task or list stays on screen above it; the open task's actions come first under `TASK id` (with `CHECK n` first when a check has the focus), then the global commands as a tree, each row with its key. An action with a key runs that key's path; one without asks `Run hug task ...` and goes through the same code as the MCP tools. Esc goes up one level, then closes.

## v0.7.69 - 2026-11-18

- Windows CI runs the `huginn/internal/cli` tests in two shards on two runners (0n4rnx): shard sync takes the sync, catch-up and layout-upgrade tests (`-run`), shard rest the others (`-skip`, same pattern), each one `go test` run under its 10-minute limit. One run of the package took 387-403 s on one runner and crossed 600 s on slow ones. A new job, `test-windows-cli-shards`, fails when a test ran in neither shard or in both.

- Memories can say when they apply (jrj1vq check 1, format and storage only). `hug mem remember` takes `--when` (the situation, as an activity), `--when-not` (look-alikes where it does not help) and `--case-scope`, `--case-path`, `--case-tool` (repeatable); MCP `memory_remember` takes `when`, `when_not` and `case {scope, paths, tools}`. The memory text is the note shown (the THEN); there is no `--then`. A plain memory stays valid and byte-identical on disk. The trigger lives as one `"trigger"` object in the native file and the project's `memories.jsonl`, and in a new index column `units.mtrigger` (additive, added on open; `trigger` is an SQL keyword). `hug mem show` and `list/search --json` show it. Remembering the same text with a different trigger is refused (`NOT STORED`, exit 3, `fix: hug mem remember --supersedes ID ...`): a changed trigger is a correction. `--supersedes ID` with the same text replaces the trigger in place, same id, with a supersede row in the fix trail; `--supersedes ID --no-trigger` clears it, stored as `"trigger": {}` so sync carries the clear. A line without the key, from a plain memory or an older hug's rewrite, never clears a trigger, and the next sync restores it. Dream does not merge twins whose triggers differ; an owner correction keeps the trigger. Recall is unchanged: nothing reads the trigger yet (checks 3 and 4).

- Capture can propose a trigger for rule and preference memories (jrj1vq check 2), behind `extract.triggers` in the muninn config, off by default and left off for now, measured blind on 40 fresh captures against a bar set in advance. With the switch off, the capture prompt is byte-identical to before (a test pins its sha256), and no trigger is stored. With it on, the existing extraction call also asks for `when`, an optional `when_not` and a `case` holding only entries the transcript states. A trigger on any other type, or a malformed one, is dropped and the memory is kept. Measured (DeepSeek Flash, 40 extraction plus 40 judge calls): durable share ON 0.852 against OFF 0.872, -0.020 [95% CI -0.091, +0.041], which passes. The judge rated 15/20 triggers good and 0/20 case invented. The hand check rated 20/20 good. Judge and hand agreed on 15/20 against a required 16, which failed: in 4 of the 5 disagreements the judge's own reason called the when_not a real look-alike. A judge-only rerun (11 calls), with a when_not definition fixed in advance and the same 20 triggers, agreed on 19/20, rated 19/20 good and 0/20 case invented. It marked all 5 controls with an unrelated when_not false. The switch stays off until the owner decides. The `memory_remember` description and `hug mem remember --help` ask for when/when_not on a rule or pattern, and tests hold the wording.

- `hug upgrade --tags` cleans a board's labels against its tags list (txjkrm step 5). It previews by default and writes nothing: one count line per kind of change, one line per task (`ID: labels [old] -> [new]`), exit 5 and `fix: hug upgrade --tags --apply`. `--apply` writes it in one commit, `hug upgrade --tags: ...`, after which lint finds no unlisted tag. Variants of a listed tag (case, `-` and `_`, `[tag_merges]` lines) merge on every task. Done and dropped tasks are history and get only those merges; on open tasks, a type or priority label that the field already says goes, and one that disagrees stays and is listed with its fix. A todo task labelled parked, paused or deferred becomes deferred with `deferred_reason "label parked (hug upgrade --tags)"`, draft goes, and any other unlisted label goes. design is an ordinary tag: the preview counts the open tasks that keep it and names `hug task design ID "question" --for NAME` for one that should become a design stage. A board with no list gets one proposed, the tags on `--min` (default 3) or more tasks after variant merges, and `--apply` writes it to `hug.toml`. On a copy of seid-lang's board (2531 tasks, 38 tags plus design, 50 merges), the preview listed 1451 tasks and 1754 unlisted-tag findings; after `--apply`, lint found none.

- The view line is quiet, one cue per meaning (qh9ep1 check 9, 911j70; Jørgen: "something more subtle"). An underline marks only a key you can press, and colour tells the keys apart: `v` in the accent, `o l b !` in violet, none bold. The view you are in is told by bold alone on its word; the other words are dim and nothing has a background. After `v` the line no longer lights up as a whole: one underlined cursor blinks after `view:`, where the second key goes, the same cursor as the answer box's, and goes when the view is chosen or `v` is cancelled.

- A detour from a doing task no longer loses it (4tycwd, crab: "next means what do I work on now"). A take refused for one ticket in doing gets a fourth `fix:` line (and a fourth entry in `fixes` over JSON and MCP), the detour: `hug task dep A --add B && hug task note A "where I was"`. A then depends on B, so it does not count as work in hand; when B is done, A is doing/active again by itself and first in `hug task next --mine`. `hug task next --mine` no longer offers my doing tasks that are waiting: during a wait, the detour or the next task comes first (before, the parked task came back with `next: hug task check A 1`). A wait still ends by `hug task resume`, `--until` or its condition, and a woken wait is active and next as before. With nothing else for me, `next --mine` exits 4 with `nothing for NAME to work on now; waiting:` and one line per wait (`ID waiting on WHAT`, `· until DATE` for a timed one), and a `waiting` list in JSON and MCP. `hug task resume` prints `was waiting on: ...` once, since afterwards that text is only in `## History`, which `show` does not print.

## v0.7.68 - 2026-11-18

- Windows CLI tests start no `git maintenance` (0n4rnx). Every commit, fetch and push in the `huginn/internal/cli` tests ran `git maintenance run --auto`, which Windows git runs in the foreground: 31 in one catch-up test, 4 for one commit and one push. The test binary now sets `maintenance.auto=false` for itself and every git it starts, and the fixture remotes set it in their own config, since receive-pack does not read the environment. This changes tests only, not hug.

- Three test flakes fixed at their cause (ybbfsa). A background pass that opens the board while another pass rebases it no longer fails with "not a working checkout of hug/tasks": the checkout is the branch's while `rebase-merge/head-name` names it (CI 44316906824, TestBackgroundSyncParallelNotesAllLand; the new TestOpenDuringRebase failed first with the CI's error). Test fixtures set `maintenance.auto=false` so git's detached auto maintenance cannot outlive or race a test: a fixture remote's receive-pack wrote `objects/maintenance.lock` after a push killed at NetTimeout (TempDir cleanup "directory not empty"; `gc.auto=0` alone still starts it), and a commit's worktree-prune deleted `.git/worktrees/linked` while `git worktree add` was filling it in (CI 44328375780). hug's own pushes and commits are unchanged.

- Tags listed per board (txjkrm, step 1 of 5). `hug.toml` takes a top-level `tags = ["compiler", "parser", ...]`, and optionally `[tag_merges]` with `OLD = "LISTED"` lines. With a list, lint flags an open task's tag not on it as `unlisted-tag` (advice). A variant of a listed tag (the same word ignoring case, `-` and `_`, or a stated merge) gets one fix: `hug task set ID --tag OLD=LISTED`. Any other tag gets two: `hug task tags add NAME`, or `hug task set ID --tag NAME=`, which removes it. A board with no list is not judged. `hug task tags` lists the board's tags and `hug task tags add NAME` adds one. `hug task set ID --tag` takes `NAME` (add), `OLD=NEW` (replace) or `OLD=` (remove); it is repeatable, and running it again changes nothing. MCP `task_update` set takes `tag` too. A lint finding may carry `fixes`, its alternative commands; text output prints the others as `or:` lines. The task field is still `labels`; showing tags on rows and `hug upgrade --tags` come next.

- TUI tag filter and grouping by tag (txjkrm step 2). The `label` filter is now `tag`: `: tag compiler` (or `f l`, then the tag) keeps the tasks tagged compiler, the chip and footer read `tag:compiler`, and the palette offers the board's tags. `g` cycles flat, group, epic and now tag (`: grouping tag`). By tag, each task sits under one heading: the first of its tags that hug.toml lists, in the list's order (the board's order of areas), else its own first tag, else `no tag`, so no row repeats. The help line reads `g group rows: flat, group, epic, tag (tag: first listed tag)`.

- One request for several tasks (ec6z6j, the non-visual part; marking rows in the TUI waits on rxr4z8). `hug task request ID1,ID2 "text"`, or `--task ID` (repeatable), and MCP `task_update` request with `ids`, write one request for the exact selection. The text is written once, verbatim, on the first task (its home), whose head names every target: `- STAMP request 3 by jhf (for a, b, c):`. Each other task gets a link with no copy of the text: `- STAMP request 1 by jhf (see a request 3):`. It is one commit, delivered once to the board coordinator; the message names the targets and gives one `hug task resolve ID N` line per target, also when it waits for the coordinator's next session. Each target resolves its own part, in any order, the home included. The home's History records `request 3 target b resolved by NAME`, and the last resolve adds `request 3 closed: all targets resolved`; only then is the receipt `answered`. Until then, `hug task show` and `hug task requests` print `targets: a resolved, b pending, c pending`, and the home stays listed. A link shows the home's text and targets. An unknown ID writes nothing: exit 2, `no task zzzzzz: nothing was written`, with the request on the known IDs as the fix.

## v0.7.67 - 2026-11-18

- A question to a person is asked, never waited on, and only of someone hug knows (agxqax, Jørgen on Statbus: five questions for him sat as waits, one asked of `jorgen`, so none was in his ASKED). On a board with hug.toml [people], `hug task wait` refuses a text that opens with `owner:` or a person's name (`jhf:`, `Jørgen:`), with `the owner's decision` (or answer, go-ahead, approval), or with `owner confirms ...`, and its fix is the ask that says it (`hug task ask ID --to jhf --check 4 "..."`); a person named in passing (`on Jørgen's machine`, `PR approved by jhf`) or an agent (`crab: ...`) is still a wait. `hug task ask --to` takes a person by any name: `jorgen`, `Jørgen`, `Jørgen H. Fjeld` or an `alias = ["..."]` in their [people] entry write `jhf`; a name that is nobody there and no agent on the board is refused, naming the people. Lint: new `wait-on-person`, and `unknown-person` on a person's other name now fixes with the ask to their short name. On a copy of Statbus's board it finds the 6 (cgtb4y, hmgadt, hzyejk, j2g6wd, mtw92d waits; ezp3h0 asked of jorgen) and nothing on hug's or sql_saga's; over 141 historical waits of hug and Statbus it flags 24, all questions to Jørgen.

- Sync with the same task changed in two clones (8zwy6r): notes, a different check ticked on each side, an answer and requests merge and both syncs succeed, as since the shared replay (798d91c8); now tested on a plain board, which on a13d7c74 stopped with `sync conflicts: ... (rebase aborted)`. When state was set on both sides, the board's state stays whole: the local move's node fields (`dropped_reason`, `dropped_at` of a drop) go to `## History` instead of staying in the front matter, where they linted broken `[stray-node-field]`; the local state is still in its note and the sync still prints `sync: ID state: kept the board's ...`.

- TUI header names the project (0sw0dd): `HUG · <project> · list` replaces `HUG TASKS · list`, in lists and in a task's breadcrumb. The project is the checkout's folder name, so two checkouts of one repository read differently (`hug`, `hug-testing`); `project = "NAME"` under `[board]` in hug.toml names it on every checkout. / search takes field terms beside words: `state:done` (also the old `status:done`), `type:bug`, `priority:`, `group:`, `tag:` (also `label:` until the CLI field is renamed, txjkrm), `assignee:`, `epic:`; `state:done memory` is the done tasks with the word memory. A term with any other field (`http://x`) stays a word.

- Sync recovers from an interrupted rebase it started itself (fcf4fg). hug marks each rebase it runs on the tasks branch and holds a lock while it runs; when a sync dies mid-rebase (as in Seid on 2026-11-16, which left `rebase-merge` holding only an autostash and failed every agent's write until someone ran git by hand), the next write, sync or background pass ends it under the board turn, puts the autostashed edits back, prints one `sync: ended an interrupted hug rebase ...` line and goes on. A rebase hug did not start (no marker, a marker from another rebase, a different head) still stops with `worktree busy` and its `git rebase --abort` fix; edits the rebase did not make are kept in `git stash list` first.

## v0.7.66 - 2026-11-18

- `hug task lint` stale-owner sees owners whose sessions are gone (9ahvgt). It used to say "no findings with code stale-owner" while 22 doing tasks on hug's board belonged to sessions jcode had reaped or closed hours earlier, because only a session file older than a day counted. Now a jcode session that says `"status":"Closed"` (ended, or reaped after 30 min idle) is gone an hour after its last write. The journal (`.journal.jsonl`) counts as a write. A name taken with `--as` by a session told "You are NAME" is matched to that session (`(kestrel was session calf)`). A name no session knows is gone once its task has been doing for a day. Sessions are read from `JCODE_HOME` and every jcode instance in the zsh registry, but only those that worked in this repository, since names repeat across instances. A person (`--model human`, or a hug.toml `[people]` name) is never judged. When a sessions folder exists but cannot be read, lint says so once with `fix: ls -ld DIR`. `hug task take` on a task whose owner is gone (the same test) is still refused, but its fix now takes the task over in one runnable line, `fix: hug task set ID --assignee ME && hug task take ID --as ME --model MODEL`. Before, it ended with `fix: hug task next` (mizaru on txjkrm). A live owner's task keeps `hug task next`.

- TUI messages and question captures (tz9z14): every message on the bottom line belongs to the screen that made it. Opening a task, another file or another view clears it, so `reloaded hug vX.Y.Z` from the overview no longer stays under a task (z82qsy did this for action results only). A screen capture a question shows (`--show`, an `.ans` file) now sits between dim rules under a dim label, `▸ NAME · SIZE · capture, as it was taken`, with runs of blank rows folded to one, so a captured overview no longer reads as the live one; Space in FILES still shows it exactly as taken.

- TUI task ids in text are Tab stops (0ga6m0). Every task id the detail draws is now a stop that Tab and h/l reach in reading order, and Enter opens it. Before, only ids in Links and dependencies were stops; now ids in a check, the asked question, the description sections, the finding, the answer, dependency notes, conditions, requests, Notes and History are stops too, marked where they stand. An id counts once per place it appears, and a `task:ID` already listed under Links is not a second stop. On a copy of hug's real board the slowest focus-ring build went from 38.3 ms to 6.9 ms, because a cached set of board ids and aliases screens out words that cannot be ids.

- Ticking the check a question is about answers the question (qsqhpe). When the person a question was asked of (`hug task ask --to NAME --check N`) ticks check N, with `hug task check ID N` (with or without evidence), `task_update` check over MCP or Space in `hug tui`, that tick is their answer. The task leaves asked for active, with one answer block in History: `answered asked (check N), by NAME, for ASKER` and the text `answered by ticking check N`. The asker sees it as it sees a typed answer, and it goes to the board coordinator like a typed answer (gmzd87). A tick by anyone else, or of another check, leaves the question open. From the other side, a ticked check cannot be asked about: `hug task ask --check N` on a ticked check is refused (exit 2, `check N is already ticked: ask about an open check, or add one with hug task set ID --accept ...`, with a fix naming the first open check, or `--accept` when every check is ticked). The TUI never draws a question on a ticked check (no `?` on its row, no `check N` in the overview): one asked before the tick shows as its own block, answered with `a`. `hug task help check`, `hug task help ask`, `hug task states` and the delivery text in AGENTS.md and MCP say so.

- Recall hook board lint (agh2h7): a checks-all-hold finding ("every check holds, but doing/waiting ...", fix `hug task done ID`) goes only to the task's own session, as a missing section does; when every assignee has gone by the rule `hug task lint`'s stale-owner uses (9ahvgt: the newest session in this repository closed over an hour ago or silent for a day, a "You are NAME" alias counted), the board's coordinator gets it instead ("Its owner NAME has no live session, so it is yours as coordinator"), and nobody else; on the live hug board the hook and stale-owner name the same 6 of 16 doing assignees as gone. Replayed on 2026-11-17 14:00 to 11-18 12:00 (hug, 618 sessions): 44w1ev's line no longer reached calf and crab while wyvern was live (both judged wrong twice); the six board4 sets are unchanged (217 lines, byte-identical).

- Recall hook (rk5q7r): at a break, the board's coordinator hears once a day how many open reflections are older than 3 days, with the command that reads them: "Task board (return): 22 open reflections are older than 3 days; read them grouped by the instruction they name: hug task reflections --by-instruction". No other session hears it, and it never takes the place of another board line. Replays: the six board4 sets and 2026-11-17 14:00 to 11-18 12:00 are byte-identical (no reflection there was 3 days old); with every reflection aged 4 days, one new line, crab's, at a break where nothing else was said.

- TUI detail is calmer where focus sits (kg07bt, Jørgen on 0gztsy): a focused section heading (ACCEPTANCE, DEPENDENCIES, LINKS, FILES, REQUESTS) is drawn like the others, bold accent, with `▸` before it and its keys dim after, no longer a filled block. A focused check asked of you, and a focused question that names no check, read as normal rows in their own colour, no fill and no bold; the `>` before them is drawn as the cursor, an underlined accent, the one mark of where you are. What has focus and what each key does are unchanged.

- A request or answer the coordinator had no live session for now reaches its next jcode session when it starts, once (bbxdd7). It waits in every known jcode home (`hug-inbox/pending/NAME/`), hug's `turn_start` hook delivers it and writes `delivered to NAME (SESSION)`, and History keeps a second session or home from delivering it again. A coordinator renamed before its next session gets it under the old name. A muninn/run test drives the jcode hook end to end: delivery at turn_start, acknowledgement at the next tool call.

- `hug task detach` no longer leaves a missing-file finding (bqfrda): a file cited before its "detached file:NAME" note is gone on purpose, and lint reports a missing cited file once however often it is cited (was twice after attach and detach).

- Lint no longer reads a quoted `.hug/tasks/ID/...` path of another board's task as a broken link (1phmre: w40cmk quotes Statbus's `.hug/tasks/37bryg/README.md`); only `task:REF/NAME` links to a task. A `task:` link to no task keeps `missing-task-ref`, and its fix is now `hug task note ID "task:REF/NAME (lost)"`, run without an editor (was `hug task edit ID`): a citation marked (lost) anywhere in the task settles every citation of it.

## v0.7.65 - 2026-11-18

- A waiting task with nothing left is asked to finish (bg1rd8). `hug task lint` reports `checks-all-hold` on a doing/waiting task whose every check is ticked or struck and whose every child is done, once the wait is over 2 hours old: "every check holds, but doing/waiting on landing slot for 25 hours: ...; wyvern: is it done?", fix `hug task done ID`. Before, a stale wait hid such a task (44w1ev sat a day). Not while an open dependency or board condition would refuse done, and not for `hug task ask --after` waits.

- TUI answer and request box (qpp30y): `a` and `r` draw the empty box with its label and cursor at once, not after the first key. Its cursor is `theme.cursor`, an underlined bold accent (`#00bfff` dark, `#006c9c` light) that blinks at 530 ms, instead of a steady reversed block. It is the only thing on screen that moves: the one-line inputs keep a steady cursor, and a closed box stops blinking.

- `hug task reflections` reads reflections the way they were written (2k68eg). The instruction quote opens only after a non-letter and closes only before one, so the apostrophe in AGENTS.md's or Jørgen's no longer starts a quote: 10 of the 28 instruction fields on hug's board were fragments such as `'s help says '`. `--by-instruction` groups open reflections by what they name (the hug command in the quote, else the quote, else a hug command in the text), the largest group first, so the three reports of `hug task check ID N` taking no evidence show as one group. A `hug task done` without `--reflect` now leaves one `Reflection (unanswered, DATE, WHO)` line, which `reflections` counts (`unanswered: N done without an answer: IDS`) and never lists as open; a later answer, or "nothing", takes its place.

- Dream starts at the beginning of your quiet stretch (vk9td0). hug counts the turns you type per hour of the week (168 buckets, last 4 weeks, seeded once from `hook.log`). Swarm DMs and harness notices are not counted. Once it holds a week, the daily pass waits for that day's longest quiet stretch, where an hour is quiet if it had a turn in at most a quarter of the weeks. Until then dream keeps the old rule of 30 minutes away. `hug mem dream --week` shows the histogram, the longest stretch and each day's start. `hug mem dream --status` names the next stretch, or how many of 7 days are still to learn. Dream items are still handed out at the next break, which after a night pass is your first message back. The rule follows the defrag scheduler in Jørgen's syncthing fork (`internal/activityhist`).

- Memory fixes leave a local trail (rjk44w): every forget (CLI, MCP, web, TUI) and every remember that names an old memory (`--reinforce`, or the new `--supersedes ID` / MCP `supersedes`) writes a row to a new `memory_fixes` table in muninn.db, with the session, harness, jcode instance, surface, an optional `--reason` (MCP `reason`), and whether recall showed that memory to the session first and how long before. `--supersedes` forgets nothing (forget the old one if it no longer holds), and naming the near-duplicate it corrects answers the duplicate question. `hug mem stats --fixes [--since DATE] [--json]` prints, per UTC day, recall blocks, fixes, fixes of a shown memory and of an unshown one, forget/reinforce/supersede, and forgets undone within 7 days, then per instance and the last 20 fixes. Recording never blocks the write and makes no model call. Recall reads nothing from it: the frozen recall4 replay is byte-identical. The schema change is additive only (`CREATE TABLE IF NOT EXISTS`).

- Background sync: a pass that cannot open the board (its checkout pruned, no remote) removes the pending marker its start wrote, so `hug task ls`'s sync state and waiters stop counting a pass that will never run (jj7jpz, CI 44230245084).

- TUI images in the file frame (2qndsd): Space on a PNG, JPEG or GIF in FILES opens it in the same frame as any other file, crumb (`HUG TASKS · list › ID › path · size · n of m`), note, rule, the image, rule, keys, instead of full screen until any key. iTerm2 and kitty (also inside tmux) draw it over the frame's rows, fitted to their height and width; elsewhere the frame holds one line, `image, 67 B: this terminal draws no images (iTerm2 and kitty do): Enter opens it`. `n` and `p` step between images and text files in that frame, and leaving an image repaints the screen once so no piece of it stays. `S` captures the frame as text, without the image bytes.

- `hug task check ID N "evidence"` ticks check N and notes `check N: evidence` in the same write, as the installed instructions promise (pw0t19); MCP `task_update` action `check` takes the same optional `text`. `hug task note` and `check` accept `--as NAME`. A wrong number of arguments on any task command names its usage line and ends `fix: hug task COMMAND --help` (was `fix: hug task help`).

## v0.7.64 - 2026-11-18

- Recall hook board lines (6fqhg9, rzfrgp): a hug command quoted inside another command's argument (a `--goal`, a note, a quoted lint line) no longer counts as run, so quoting `hug task take ID` does not make ID yours; `sh -c '...'` and `$(...)` still count. The "this looks like ID" take line no longer names a task you only opened with `hug task show`. The board4 judge now states that a missing-section or owner question is right only on the agent's own task; two judgings agree on 134 of 145 lines (was 7 of 15 on the lines that flipped). Replayed under it: six sets false 9 -> 7, right 106 -> 107; fresh hug false 4 -> 3 (muninn/docs/BOARD4.md).

- `hug task ask ID --to NAME` on a shipped task is no longer refused by the `shipped-not-done` finding that the ask settles (gpyqq9). A shipped task with every check ticked ("is it done?") can now be asked of its owner: the stage becomes asked and lint no longer reports it. Every other broken finding still holds the ask back, with its fix.

- Requests and answers reach the board coordinator by design (gmzd87). When a person writes `hug task request` or `hug task answer` (also `task_update` request/answer over MCP, and `r`/`a` in `hug tui`, all through one function), hug sends it at once to the newest live jcode session named by `hug.toml [board] coordinator`. It uses the jcode daemon's own `notify_session` request: an idle session starts a turn, and a busy one gets it as a soft interrupt at its next step. The message carries who wrote it, the task, the text verbatim, for an answer the question and who asked, and the command that closes it (`hug task resolve ID N "what changed"`). Receipt lives in History, each line written by the side that saw the step. `request N delivered to crab (SESSION)` or `not delivered: REASON` comes from the writer, using the daemon's reply. `acknowledged by crab` comes from hug's hook inside crab's own session once the message is in its transcript, never from saving alone. `in progress by crab: WRITE` is crab's first write on the task after that, and `answered` is resolve (for an answer, the asker's next write). `hug task requests` prints `receipt: STATE` per request, show/MCP give `request.receipt`, and the TUI's REQUESTS label shows it. With no coordinator, no live session, or a Claude Code/Codex board, a request stays `pending, not delivered: REASON` and still comes first in `hug task next`. Nothing polls. The AGENTS.md section from `hug setup` and the `task_requests`/`task_update` descriptions carry the same text (`cli.DeliveryFlow`), and a test drives every claim in it.

## v0.7.63 - 2026-11-18

- Recall hook board lint (4vxqp4): a missing Purpose or Principle no longer goes to a session whose message only mentions the task: quoted (a prompt or output it quotes), as a dependency ("depends on n8zmjw"), or as a task not to start. A task the session owns, acted on (take, check, note, set, edit), or its message gives as the work still gets it. Reading another session's doing task with `hug task show` is not acting on it. On the six board4 replay sets (blind judge, muninn/docs/BOARD4.md), false board lines fell from 12 to 8 and right lines rose from 101 to 104; re-judged with the corrected judge prompt (6fqhg9, two judgings agreeing), the shipped rule's sets read 115 lines, 106 right, 9 false. Fresh sessions kept their right lint lines (the same count before and after).
- `hug tui`: the `/` search and the `:` palette keep a history, like a shell (xfv9pw). In an empty prompt Up recalls the last entry, Up again goes further back, Down comes forward and past the newest is the empty prompt; Enter runs it. A recalled entry that is edited becomes a new entry; a repeat of the newest is kept once. A find in a task or file recalls the same searches. Once text is typed, Up/Down pick rows, matches and completions as before; ctrl-n/p never recall. The history is per board, in `<git-common-dir>/huginn/tui-history.json` beside the task cache (200 entries per list), never on the task branch.
- TUI file viewer (z82qsy): a shown file is all the screen holds. The crumb names it (`HUG TASKS · list › ID › path · size · n of m`), the note it was attached with follows dim (none, no line), then a full-width rule, the file, a full-width rule, and outside the frame the message line and the keys. The task's chips, dates, lint and title no longer show above a file, so a screenshot's own lines read as inside the frame. An action's result now reads as done, with the id (`took ID`, `marked done ID`, `answered ID`, `parked ID`, `gave back ID`, ...), not as the command (`answer vzghj8`), and it clears once you open another task, a file or another view, so it never shows under something it is not about.

## v0.7.62 - 2026-11-18

- The TUI sort picker (`s`) names the default order (nch9d4): `r  rank` is a row, "board order, the default: moves work only in it", and `r` goes back to it at once from any sort, clearing every field. The picker shows the order in force (`now: priority ↓`, or `now: rank (board order)`) apart from the one being built (`new: type · updated ↓`, or `new: nothing picked yet, Enter or r gives rank`), so an empty draft no longer reads as `sort: rank` while the rows are sorted by priority. With no column marked, the `c  current column` row is dim as a whole, since c cannot be picked then.

## v0.7.61 - 2026-11-18

- Project memories use branch `hug/memory` at `.hug/memory/` (w40cmk). `hug upgrade` previews the old `muninn-notes` branch and `.muninn-notes/` checkout move under this board's layout, and `hug upgrade --apply` moves them without changing the memory commit, JSONL bytes or this machine's native store/hooks/MCP configs. Linked worktrees share the main checkout. Ordinary use refuses detected old layouts instead of opening an empty replacement. The shared branch-name migration checks destination, refs, ownership, dirty state and remote reachability before changing anything, rolls back failed local moves and rejected pushes, and renames remote refs atomically with leases. A lost acknowledgement is verified against actual remote refs; if both acknowledgement and probe fail, the error distinguishes verified local restoration from an UNKNOWN remote outcome and gives a nonmutating ref check before retry. Retry never rewinds an advanced remote. Old data known only through a remote-tracking ref is explicitly attached through a quoted repair command, then previewed, never implicitly bootstrapped or discarded. Memory JSONL retains its existing merger, separate from the task resolver.
- Board layout 3 (w40cmk): every task lives at `C/ID.md` (C its id's first character) and keeps that name for good; its rank and group are `rank:` and `group:` lines in the file, its files are in `C/ID/`. A move (`hug task move`, rank or group) changes those two lines and nothing else, no rename and no `updated:` bump, so `git log --follow` and every link to a task file stay right. `hug upgrade --apply` moves a board in one commit (hug's own 163 tasks: 233 renames plus hug.toml; ls, next, lint and the TUI's first screen read the same before and after). Until a board is moved, this hug refuses to write it, with `fix: hug upgrade`. Old file names still resolve as refs, and a differing old bare slug is retained in `aliases:`. A clone that has unpushed commits from before the move is not rebased: sync stops with `sync stopped: the remote board is layout 3, this clone's N unpushed commits are layout 2; nothing was rebased, local commits are kept` and `fix: hug upgrade`, and `hug upgrade --apply` replays those commits onto the moved board one by one, each with its own message and author (a rank move becomes a `rank:` line). Notes, History, new requests and new checks from both sides are kept. Explicit times compare actual instants; untimed prose keeps deterministic board-before-local order without inventing chronology. A value both sides changed (a state, a priority, a section's text) keeps the board's, puts this clone's in a note on the task (`kestrel set state doing here at T (commit C "msg"); the board had done; kept done`), and prints one line for it. Competing resolutions keep the board's canonical resolution and retain this clone's verbatim resolver, time and text in a labelled note linked to the request's stable identity. Conflicting attachment and config bytes keep the board's active path and save this clone's exact original bytes and mode at `.local-FULLCOMMIT`, naming both paths. Ordinary local migration works offline; only known cross-layout catch-up fetches. An unreadable file, a differing recovery collision, or an edited task deleted remotely stops safely rather than discarding or resurrecting content. Replay happens in a private worktree before the finished result is installed, so a failure leaves HEAD and all local commits untouched and names the commit and original file. Replay uses each first-parent delta, including merge-only conflict-resolution edits, once. After migration, ordinary foreground and background sync use that same resolver for divergent changes: independent same-task appends reach both clones once, competing single values retain the board's and keep the local text visibly in a note, with one summary line per value only after the complete replay succeeds.

## v0.7.60 - 2026-11-18

- Background sync's `Busy` observation checks the pending child marker before the queue and run locks (rt80x4). A deterministic real-child test reproduced a false-idle handoff in 10 of 10 runs: a child could take the queue and remove its pending marker between observations, before its pass had run. The pending-first check closes that demonstrated window. The five-second expiry used to retry a dead startup is unchanged. The slow-child test now retains phase and state diagnostics if its remote-head assertion fails. This demonstrates and fixes an admissible race, not the established cause of the earlier release failure.
- Recovery release after v0.7.59 failed its test step before build or publication. The v0.7.59 tag is retained unchanged; this version also contains the faster-forget changes described below. Layout 3 is not included.

## v0.7.59 - 2026-11-18

- `hug mem forget` updates the memories it changed instead of running two full index passes (4vd1mz), including a batch of ids. Concept adjacency loads and writes only for the changed frontier, neighbor selection keeps a bounded top list, and a successful save is not read again unless another writer changed the file. On a 30,794-unit store copy, four paired runs after preparation fell from 3.19-4.26 s (median 3.415 s) to 0.81-1.07 s (median 0.920 s), with new-binary max RSS 218-230 MB. Reading the database before each run gave 2.48-3.06 s (median 2.545 s) before and 0.61-0.91 s (median 0.645 s) after, with new-binary max RSS 215-226 MB. MB are decimal. Seven of eight new-binary samples were below one second, with a 1.07 s tail: the goal of well under one second on every run is not achieved. These measurements did not flush the OS page cache and exclude preparation and one-time migration costs. The old whole-graph gob and quantized vectors move into narrow tables once on first use. Graph conversion is inside the writer's transaction, needs no upgrade command, and leaves one committed receipt in `hook.log` and `hug mem stats`; `hug mem index` reports it, and `hug mem upgrade --apply` can do it ahead of a hook. A rolled-back pass leaves no success receipt. Production Louvain traversal remains in Go map order, with no sorting added. Across the migration, all 277 replayed responses matched after excluding only `latency_ms` and `response.elapsed`. This is migration equality, not a recall-quality gain claim.

## v0.7.58 - 2026-11-18

- `hug tui`: showing a `.md` or `.markdown` file formats it with the task detail's quiet Markdown renderer (tmz060): accent headings, blue-grey inline code and code blocks, and lists, wrapped to the pane. `M` switches that file between formatted text and raw source without rereading or writing it. Other file types stay plain text, and the binary and 1 MiB limits still apply. Controls are escaped before Markdown is rendered, so a file cannot clear the terminal or set its clipboard. Formatting runs off the UI thread with a spinner, including large files; a late result from a file you left, or an old pane width, cannot replace the page. `/` and `n`/`N` find the text actually shown, and an existing find is recomputed when `M` changes the view.

- Background sync records its progress before waits in `.git/hug/sync/hug-tasks.log` (6ep7y2): queue taken, settling finished, run lock taken, fetch, board turn, push and deduction, each with a timestamp and pid. A stalled pass can now be located even when it never finishes. New phase lines contain status flags rather than full errors or remote URLs. Sync tests report the run lock independently of the queue when they time out, and retain the phase log in failed-test output before temporary files are removed. A subprocess regression test checks that output without waiting for a real timeout. This adds diagnostics, not a claimed fix for the Windows stall in CI 44237364634.

## v0.7.57 - 2026-11-18

- `hug task attach` no longer refuses a file over `sk-` inside a word (4c7327). On 2026-11-18 a seid report was refused over the file name `task-900-task979-ownership-reconciliation-2026-08-17.md` (`credential-shaped text near sk-900...`). A key now has to start at a word boundary, the rule hug's memory check and the CI secrets job already used, and real keys after `=`, `(` or a quote are still refused. A test now runs the CI gate script itself, so the three checks stay on one rule.
- `hug mem upgrade` records the paths and commands every memory names, the one pass tool-call recall needs (n7jbmx): the preview says `memory: would record the paths and commands memories name (tool-call recall)`, `--apply` does it, so it rarely lands in a hook. `hug mem index` does it too and says `anchors recorded for N memories (once)`. Wherever it runs, `hook.log` gets one line.
- Tool-call recall, off by default (n7jbmx). With `"tool_recall": true` in config.json, jcode's post_tool_feedback hook adds at most 2 memories to a tool result. They must be rules, facts or preferences that name the file, folder or path the call touches and embed close to the call, and each is shown once per session. Indexing now records the paths, file names and commands each memory names (once, 2.2 s on a 30k-memory store). On 300 replayed tool calls of 2026-11-16..18, judged blind: 10.7 useful and 9.0 noise memories per 100 calls, against 57.3 and 135.3 for message recall on the same calls. The hook's p95 was 150 ms on a copy of the live store at load 15. It stays off until fresh calls from 2026-11-19..20 pass the same bar (muninn/docs/RECALL-TOOL.md).
- `hug tui`: / in a task, or in a file it shows, finds text on that page, as in less (4s3g5q, Jørgen 23:29). It no longer runs a task search that lands you back on the task you were reading. Every match is tinted, including a match that a wrap splits across two lines. The first match at or after the top of the page scrolls into view, and the footer reads `match 2 of 5 · n next · N previous · Esc clear`. n and N step through the matches and wrap around. Esc clears the tints and leaves the page where it was, and u goes up. Case is smart, as in vim: all lower case ignores case. With no match the footer reads `0 matches in ID · Enter searches all tasks`, and Enter runs the task search. A find is never a list filter: after u the list shows every task. In the list, board and overview, / still finds tasks. A hidden task's message names `vl`, not the old i.
- `hug tui`: a column that every task of a section shares moves into the section's heading (1bbe96, Jørgen 00:11). `IN PROGRESS · assigned to [13] · all active · all feature` draws no STAGE or TYPE column, and the title gets the width. This applies to stage, type, priority and owner in the overview, each value in its own colour. The list says it after TITLE in its column header. It is decided on every task of the section after filters, never on the rows on screen, so scrolling cannot make a column come or go. A section of one task keeps its columns. H/L skip a hidden column, and o (and sc) never sort by one: a marker left on it stands on the next drawn column.
## v0.7.56 - 2026-11-18

- `hug tui`: the line under the view line reads `filter: none · sort: rank` (ahejza), with f and s marked, and shows the chips and the order once set. The order no longer sits in the top line. s opens the sort picker, which looks like the f filter picker: each field letter has a hint, picks chain as "then by", the same letter flips, ⌫ drops the last, Enter applies and Esc keeps the order. In the picker, c is the marked column, the same as o, and created moves to d. The digit quick filters 1-4 are gone (`: todo`, `: doing` … remain), and s no longer opens `: state` (`: state` still does). Help lists f and s with their picker keys.
- `go test ./...` fails on a leftover merge conflict marker in any tracked text file (55a7xx): `<<<<<<<`, `|||||||` or `>>>>>>>` at a line start, and `=======` in a file that has one of those. It names the file and line and ends with a `fix:` line. Twice on 2026-11-18 a rebase left `||||||| parent of ...` in CHANGELOG.md on main. The test makes one `git ls-files` call and takes 0.07 s.
- `hug upgrade --apply` changes only this board; this machine's files wait for `hug upgrade --apply --machine` (f5vx4x). On 2026-11-18 an upgrade run in a scratch clone to test a board migration also rewrote the hook in all six of Jørgen's live jcode configs, a change still waiting for his yes. The preview now prints two headings, `this board (PATH):` with layout and tasks, and `this machine (hook and MCP configs in ~/.jcode, ~/.claude, ~/.codex; the memory store):` with variables and memory, and ends `next: hug upgrade --apply` then `then: hug upgrade --apply --machine`. After `--apply`, machine changes still pending are listed as `would:` and the command exits 5 with `fix: hug upgrade --apply --machine`, so a script never reads half an upgrade as done. `--apply --machine` does both, each machine file backed up as before. `--machine` without `--apply` is refused with that fix.
- `hug mem dream` runs by itself without `hug mem serve` (h00cs2). On Jørgen's machine it had never run (serve never ran; the dry run on 2026-11-18 would retire 3 duplicates and queue 111 items). When a pass is due (daily), the capture hook starts a waiter. It runs the pass once you have been away 30 minutes and the 5-minute load is below 120, and gives up after 12 hours, saying why in `hook.log`. A waiter that slept through an upgrade hands the pass to the installed hug. Every pass leaves one line naming its undo receipt in `hug mem stats` (`"dream"`), the TUI's Needs a look pane and `hook.log`. `hug mem dream --status` and the `--dry-run` header say when it last ran, or never.
- `hug tui` picks views with the v prefix (qh9ep1). Line 2 reads `view: overview  list  board  !lint`, with v and each view's letter marked, and vo, vl, vb and v! switch views from a list or a task. Bare i, b and ! no longer switch views. While v waits, the view line lights up and the footer reads `v: o overview · l list · b board · ! lint · Esc cancel`. Esc or an unknown key cancels and changes nothing. The state counts moved onto the view line, right-aligned after a dim `│`, and drop from the right when the line is tight. The old counts line is gone, so lists show one row more. Help lists `v o`, `v l`, `v b` and `v !` as two-key rows, and its find matches `vo`.

## v0.7.55 - 2026-11-18

- hug refuses a board checkout that belongs to another repository (ygay9c). A copied or moved repo (cp -R, an APFS clone) keeps its `.hug/tasks` link to the original, so hug in the copy read and committed the original's board: wyvern's bench copy put 37 notes there. Now every command stops before it reads or writes: `open: the board checkout ORIG/.hug/tasks belongs to another repository: its .git names ORIG/.git/worktrees/tasks, outside this repository's COPY/.git ...; hug writes nothing to it`, then `fix: git -C COPY worktree repair COPY/.hug/tasks`, which gives the copy its own board and leaves the original's alone. The check reads one file and starts no git process.
- Background sync no longer drops a pass now and then (mmm8c8). Checking whether a syncer is queued (`hug task ls`, every write's nudge) takes the queue lock for a moment, and a syncer starting in that moment took the check for a queued syncer and exited, so the commit it was started for waited for the next write to be pushed. macOS CI saw it once (`settled returned before the pass pushed`); a stress test lost 4 to 6 of 300 syncers on every run. A refused syncer now waits for any check to finish and tries once more, so only a real queued syncer turns it away; the stress test loses 0 of 300 in 10 of 10 runs.
- `hug tui` has a request box (hbgybn check 4): `r` in a task, or on its row, opens a box under a REQUESTS heading for asking the agents for a change in your own words. Enter sends, Alt-Enter adds a line, and a paste keeps its lines. The row shows `req N` while requests are open. The task lists each request with who sent it, when, open or resolved, its text verbatim, and the account once resolved. `r` no longer refreshes, since the board updates by itself. In the task's key bar, `S screen` and `E edit files` now sit with the other keys that write, so a narrow bar cuts navigation first.
- A task can wait for a date (z2vmte, Jørgen 02:28: "waiting for time to pass is a sensible state and when the time has passed the system brings it back"). `hug task wait ID --until 2026-11-24 "why"` parks it in WAITING (`· until 2026-11-24`); on the first hug read that day (ls, next, show, the TUI, MCP), hug ends the wait itself in one commit (`hug task wake: ID (the date came)`, one History line), and the task is doing/active again, first in its owner's `hug task next --mine` until they write on it. `hug task ask ID --to NAME --after 2026-11-24 "question"` waits the same way and asks the question that day, verbatim with its `--check`; before the date it is in nobody's ASKED. A date that has come, or is no YYYY-MM-DD, is refused with the fix. MCP: `task_update` wait takes `until`, ask takes `ask_after`.
- hug mem: a forget that lands while a background project sync pushes stays forgotten; the sync no longer marks the forgotten memory as matching the pushed (still active) line, which the next pass read back over it. A project memory's short id is final when remember prints it (no more yx7vxv becoming erf4tk after sync), and forget and remember read the store back and fail with a fix: line if another writer undid them (tn8bmn).
- A `HUG_*` variable hug does not know stops the command before it does anything (a0p5kt): it exits 2, names the variable and the nearest real one, and the `fix:` line unsets it and sets that one, e.g. `hug mem forget: not a hug variable: HUG_NO_SYNC (nearest: HUG_SYNC)` / `fix: unset HUG_NO_SYNC; export HUG_SYNC=off`. A misspelt safety switch used to be ignored: on 2026-11-18 a benchmark set `HUG_NO_SYNC=1` against a store copy and its 11 forgets synced to the project's notes branch (repaired by hand). `hug help` and `hug version` still answer; a harness hook (`hug mem hook`, `hug mem capture --hook`) does nothing, writes the same problem and fix as one line in the hook log, and exits 0, so the harness never breaks. `hug help` now lists every variable a person sets, one line each. The known list lives in one place, and a test fails when the repository names a `HUG_` variable that is not on it.

## v0.7.54 - 2026-11-18

- `hug mcp` serves every request with the hug installed now (mbdfzs). Before it passes on a request, it checks the hug file the app started; when an upgrade has replaced it, it waits until no request is running (up to 30 s), then on macOS and Linux it execs the new hug on the same process and connection. The new hug replays the client's initialize and serves the held request, so the app sees no disconnect, and each tool result names the hug that served it in `_meta` (`hug/version`). On Windows, which has no exec, it answers each request read so far with an error that says to run it again, then exits so the app starts the installed hug. A hug older than this one is never exec'ed (it fails `hug mcp --can-resume`); the old server serves on and logs `fix: restart the app`. Servers started before this release do not have it: restart the app once.
- Lint judges open tasks only (t01kqm, Jørgen 2026-11-18: "I don't think linting should show the done tasks"). Done and dropped tasks are history: `hug task lint`, the TUI LINT section and the header counts neither show nor count their findings, broken or advice. The "N advice findings on closed tasks hidden" line, `hug task lint --all` and the JSON `hidden` and `hidden_codes` fields are gone. `hug task lint ID` on a closed task prints `ID is done: lint judges open tasks only`. A task is checked when it closes (done refuses an open check), so the `done-unchecked` code is gone too. A file hug cannot read as a task still shows, since nothing says it is closed. On Statbus's board, `hug task lint --summary` goes from 446 findings (314 broken) plus 170 hidden to 156 findings (24 broken) and no hidden line. A bounded list ends with `every one: hug task lint --json`.
- `hug tui` colours have one meaning each (zhvkmj, Jørgen 02:32: waiting and lint should not share a colour). Amber is only what waits on you: the ASKED title, `? N for you`, the `[asked]` badge, the question and the stage word of a question to you, the confirm bar. Red is only what is wrong: broken lint's `!`, the LINT title and the detail's `⚠` lines, depends, a picture that cannot be read. A wait on a thing (CI, a date), a question asked of someone else, the WAITING title and advice-only lint are dim. Types and priorities move off both: high is khaki, critical magenta (priority is no problem), bug pink. The table is in header-spec.md 'Colours', and a test holds the palette to it.

## v0.7.53 - 2026-11-18

- `hug tui`: what a task asks you shows once (0gztsy, 41hhgn). The overview opens with `ASKED · for you [N]`, above IN PROGRESS: every task asked of you, oldest question first, each row its title; only the focused row adds `└─ asked: check 9 · <question> · nautilus`. WAITING keeps waits on a thing and questions asked of others, and `? N for you` is the ASKED count. An empty section is its title line alone, so six sections fit 80x24. Opened, such a task's header says the stage in one badge (`[asked]`, `[design]`, `[review]`) and no stage line. A question about check N is drawn under that check, which carries `?` (`> ? ☐ 9. ...`), and the task opens with focus on it, so Space ticks and `a` answers. A question about no check is one block at the top of the details, styled like a check row, whole and wrapped, never cut, focused on open. The answer box is a textarea: Enter sends, Alt-Enter (or ctrl-j) starts a new line, a paste keeps its lines (CRLF too), and it grows up to half the screen. The asker's notice prints every line of the answer.
- `hug tui`: a task's detail no longer shows the list's column hint (`H/L column · order by owner (▲)`) above its footer (qtj177, Jørgen 02:31). The hint is drawn from the marked column on the rows only, while `H`, `L` or `o` was the last key, instead of being kept as a footer message that followed into the task. The file viewer names `j/k scroll` only when the file is longer than the screen. A test now presses every key each key bar names, in every place (list, overview, board, lint, a task in each focus, a file, each prompt), and fails if one does nothing there.
- `hug tui`: in a text field every printable key is text, `?` too (t64v7d, Jørgen 2026-11-18 01:52: the `?` ending his question opened help). In the answer, wait, strike, drop and new-title prompts, the filter's text, `/` find, the `:` palette and `#` jump, help is `F1`, which the footer names (`F1 help`); help opens over the field and `Esc` or `F1` returns to it with the text intact. Outside text, and in the sort builder and the filter's field chooser, `?` is help as before.
- On jcode, recall no longer goes into your input box (z9q42h). Before, `hug mem hook jcode` delivered the block with `jcode transcript --mode send`, which landed in the composer while you typed and showed as `[transcription]` messages that were not yours. Now `turn_start` computes the block and parks it for the turn, keyed by the id of the message that started it. `post_tool_feedback` prints it into the turn's first successful tool result, once, and `turn_end` drops it if no tool call took it. A block is never shown in a later turn: one parked for a message that is no longer the latest, or a turn that has answered, is dropped. A turn without a tool call gets no block (the agent can still `hug mem search`). A tool call with nothing parked costs one file check. `hug mem install jcode` puts the hook on all three keys next to your other hooks, including multi-line arrays. `hug mem upgrade` (and so `hug upgrade`) previews the move for installs on the old hook (`memory: would deliver jcode[NAME] recall in a tool result, not the input box: ...`, exit 5), and `--apply` makes it with a backup.
- One stage at a time: `hug task wait`, `ask`, `review` and `design` take the other stages' fields off the task (to History), so a wait on an asked task no longer keeps `asked_*`; `hug task set ID --state STATE` on a task already in STATE now settles a task holding two stages, and `hug task lint` names it as stage-disagrees with that fix (3nhp67, s6ff5k).
- `hug upgrade` moves a wait on a person (`waiting_for`) to `asked` only when it asks something: a `?`, a check of the task named, or a request in its first clause (approve, answer, confirm, decide, review, look, verify ..., also as `confirms`, `reviews`, `verifies`: "owner confirms time frames on Norway" asks the owner). Any other wait, such as a park, stays a wait. Its `waiting_for` goes, History says so, and the preview lists each one with the `hug task ask` that would make it a question. v0.7.52 moved every one: on hug's own board, the park "parked per crab ruling, no fourth arm" became a question for crab (41hhgn).
- A person is never handed a broken task (wdkqzx, Jørgen 23:51). `hug task ask ID --to NAME`, `review --by NAME` and `design --for NAME` (CLI and MCP) refuse with exit 2 while the task's own file has a broken finding, and write nothing: `hug task ask: 72fkzy is not ready to hand to jhf: 1 broken finding: [missing-goal] missing Goal`, then `fix: hug task set 72fkzy --goal "..."`. Once the finding is fixed, the same command goes through. A design question is not held back by a missing Goal or acceptance, since its answer is often what fills them. A wait on a thing (`hug task wait ID "CI run 123"`) is never held back. An ask about the one open check of a shipped task (`--check 9` on 6ephqk) is not held back by `shipped-not-done`, because the ask is that question.
- A git process that dies of a signal (or, on Windows, an exception) is reported as such: `the git process died before it finished (... signal: segmentation fault): this is no answer about the board; run it again`, with a fix that re-runs it. If it left `index.lock`, the message names the lock and the fix removes it first (3z65k1). The SIGSEGVs seen on 2026-11-17 were Go's own bug in -race test binaries on macOS (golang/go#79804): the forked child died before it ran git. hug now builds with `toolchain go1.26.11`, which has the fix. Release binaries were never affected.

## v0.7.52 - 2026-11-18

- Every hug task write ends with that task's own lint, so the agent that writes a task sees what is wrong with it right then, not the person it later asks (wdkqzx, Jørgen 23:51). check, strike, promote, note, attach, detach, dep, wait, resume, review, design and answer now print the block that show, take, new and set already printed: `lint: N findings`, one `line L: [CODE] ...` per finding, then one `fix:` line. A clean task prints nothing extra. `--json` and MCP `task_update` replies carry it as `lint`. done and drop print none. `shipped-not-done` no longer offers a fix hug refuses. With checks still open it says `... (its note says so) and check 9 is open; nautilus: does it hold, or strike it?` with `fix: hug task check 6ephqk 9`. Before, the fix was `hug task done 6ephqk`, which done refused because check 9 was unticked. With every check ticked or struck, the fix is still `hug task done ID`. A task whose stage already says why it is not done (doing/waiting, asked, review) gets no `shipped-not-done` at all; doing/active and todo still do. The task cache is rebuilt once (schema 21).
- `hug tui` keys read the same everywhere (e3sevr). `h`/`l` and `Tab`/`Shift-Tab` are one key pair: in the lists they change section, in a task they step one focus ring in reading order (the parent and the stage's task id, the task itself, its checks, each dependency, child, link and file, the ids in notes); `Enter` opens the id under focus. `H`/`L` move a column marker in list, overview and board: `>` stands before the marked cell on the cursor row, the column is highlighted and the header carries no marker; `o` orders by it. `J`/`K` page on rows as in a task; moving a task is `m` (grab), then `j`/`k` and `Enter`. On the task itself the footer lists the keys its state allows (`j/k` scroll, `e` edit, `t` take, `d` done, `w` wait, `x` drop, `y` yank) and each works from there; `w` asks what the task waits on, `T` is the type palette. A task is drawn in named bands (header-spec.md): the breadcrumb `HUG TASKS · list › <parents> › <id>` with `? N for you` on the right, the context (badges, dates, the stage's line), a dim rule, the bold title, a dim rule, then the details; the view line, counts and filter line are not drawn there. Column 1 holds only the pointer, grab or yank mark and the scroll arrows; the lint mark `!` has its own cell before the id, and `?` is gone from the rows. An empty section draws no column header. The footer says `h/l Tab pane` and `H/L o column/order`. Footers, key bars and status lines mark a letter key inside its word instead of repeating it (Jørgen: "*n*ext, not a repeat but a highlight"): `edit · take · done`, with the letter in the key colour and underlined; an upper-case key marks its letter in upper case so Shift shows (`New`, `Screen`, `Paste above`, `Edit files`); a combined hint is split so each key marks its own word (`H/L column · order`, `find · # id`); keys whose letter is not in the word keep their label (`Enter`, `Esc`, `Space`, `j/k`, `x drop`). Neighbouring hints that share their last word are one item, the earlier ones ending in `-` and joined by `, `: the file viewer reads `next-, previous file · Enter edit`.

- Requests (hbgybn). A person writes what they want changed on a task in their own words: `hug task request ID "..."`, or `-` for stdin. It is kept verbatim under `## Requests` with who and when, and numbered per task. It does not change the task's stage. An agent turns it into changes to the ticket and closes it with `hug task resolve ID N "what changed"`; the account is kept verbatim under the request. `hug task requests` lists the open ones on the board, each with its resolve command. An open request is the first thing `hug task next` prints for whoever it routes to. The recall hook tells that session once (`Task board: jhf asks on ID: "..." (request 1). Act on it, then: hug task resolve ID 1 "what changed"`). It routes to the task's assignee. On a task nobody holds, it goes to the board's coordinator, set with `hug task coordinator NAME` (hug.toml `[board]`, listed by `hug task setup`). With no coordinator, every session hears it. MCP: `task_update` actions `request` and `resolve`, and `task_requests`.
- After a person answers a question, the asker's first write on the task is recorded under the answer (`nautilus read the answer: hug task note ID "..."`, and `outcome` in `show --json`). The board's coordinator hears it once from the recall hook (41hhgn).
- A question to a person is now its own doing stage, `asked`, set with `hug task ask ID --to NAME [--check N] [--show FILE] "question"` (MCP `task_update` action `ask`). The question is kept verbatim, line breaks too. `hug task wait` is for things with no person to answer: `--on NAME` is refused with a fix that runs `hug task ask`. Answers may span lines (`hug task answer ID -` reads stdin) and go to History as one block, the question quoted with `> ` and the answer indented. `hug upgrade --apply` moves each wait on a person to `asked`, and a leading `check N:` becomes `asked_check`. Until then, `hug task lint` reports a leftover `waiting_for:` as old-layout, with the fix `hug upgrade`. `hug task show --json` and MCP `task_show` put each stage's own fields in an object named after the stage, next to `stage` (`"asked": {"what", "for", "check", "since", "by"}`, and likewise for `waiting`, `design`, `review` and `deferred`). The flat `fields` map no longer repeats them (41hhgn).

## v0.7.51 - 2026-11-18

- `hug tui` rows say where a task belongs, with no extra line (kx2ndm). A dim cell before the text names its group and epic: `later·↳gem59b Launch posts`. The overview footer explains `↳`. g (or `: grouping`) cycles the rows flat, by group, then by epic, and the header says `grouped by epic`. gg and G are gone (Jørgen: "never used it, vi specific"): top and bottom are `<` `>` Home End, in rows, the detail, help and a grab. Each value gets a heading with its count (`▾ later  [3]`). Space or Enter on a heading folds it to `▸ later  [3]`, and the count stays. No group is special and nothing starts folded. A task you jump or find to opens its heading. Actions on a heading are refused, not guessed.

- `hug tui` file viewer: the footer spells out `n next file · p previous file` instead of `n/p file` (scrxak, Jørgen: nobody guesses n/p).
- `hug task help` lists every command, grouped on five lines (work, people, shape, find, repo) under a dozen common examples, so it still fits on one screen. 16 were missing, among them `attach`, `detach`, `wait`, `answer`, `review`, `design`, `promote` and `upgrade`, and `hug task help attach` and `help detach` printed no usage. `hug task help upgrade` now names `--apply` instead of the removed `--dry-run`.
- In `hug tui`, `/` followed by Enter with nothing typed no longer reports "found ID" for the task under the cursor. It says nothing and leaves the cursor where it was.

- When git crashes while hug checks whether the task branch has its first commit yet, the error names the git call that crashed and how it died (`show-ref --verify --quiet refs/heads/hug/tasks: signal: segmentation fault`), with a `fix:` line that re-runs it (xv39b9). Before, it showed `rev-parse --verify HEAD: fatal: Needed a single revision`, which reads as a broken repository.

## v0.7.50 - 2026-11-17

- The background task sync (`hug task sync --background-pass`, started by every write) now runs niced, with bounded parallelism, as `hug mem`'s background commands do; its git calls inherit the lower priority (b6jgpn). Before, it ran at full priority. Each pass's line in the sync log ends with the priority it ran at (`nice 10`, or `class below-normal` on Windows). `HUG_FULL_SPEED=1` keeps full priority.

## v0.7.49 - 2026-11-17

- A memory captured in a repository project exists once on this machine, so `hug mem forget ID` retires it fully (5sc5rw). The project's notes keep each memory with its id, and before, every sync read this machine's own captures back from the notes as new `imp_` memories: same scope, same text, origin `repo:SCOPE/cap_ID`. On Jørgen's machine 2651 memories had such a copy. Search showed one of the two, so after a forget the other took its place: `hug mem forget xgxqs6` left `49ckwr` active, and search still returned it first. Now sync knows a note row as the memory it came from and updates that one. `hug mem forget` also retires any copy from before. `hug mem upgrade` folds the copies already in the store: it keeps this machine's memory and takes the copy's state and text when the copy is newer, along with the higher strength. It previews first (`memory: would fold 2651 copies of this machine's own memories in project:...`, exit 5), and `--apply` writes a backup (`memories.json.bak-upgrade-<time>`) before it folds and reindexes. `hug upgrade` runs it as its `memory:` part.
- `hug setup` no longer loses what it remembered when two hug commands write it at once, or on Windows while another reads it (557s7n). Each write has a temp file of its own and replaces the old one the way task files are replaced; reads let a writer replace the file under them. When a check still cannot be written, setup says `hug setup: could not remember this check (...)` once, with a `fix:` line that removes that one file; before, the failure was silent and every setup ran the slow check again.
- `hug setup` stays warm across hug releases, and says when an import only adds fields (3mavpt, follow-up). The remembered backlog answer is now keyed on the importer's own version (`importer.ConverterVersion`, pinned by a test against the converter's output) and the board layout, not on the hug binary, so an upgrade that leaves the importer alone no longer makes the next setup cold. A cold run prints its progress on stderr once a second: `hug setup: comparing 995 of 2494 backlog files`. When every update only adds frontmatter fields the board copies lack, setup and `--cleanup` say so. On Seid that is `every one of the 282 updates only adds fields the board copies lack (parent: on 282); nothing already on the board changes or goes`, and the fix becomes `hug task import backlog .backlog (only adds those fields)`. On a copy of Seid, that import changed only those 282 `parent:` lines, after which setup reported the import complete and offered the retirement.

## v0.7.48 - 2026-11-17

- `hug task take` on a task held back by open dependencies names them: `s4g46t is blocked by ybqdhq Scratch check reads hug (todo)`, exit 3, with one `fix:` line per way out (`hug task take ybqdhq --model MODEL`, or `hug task show` when the blocker is already taken, and `hug task dep s4g46t --remove ybqdhq`) (r5r8c1). An imported `assignee: [""]` (13 tasks in Seid) is nobody: such a task said `taken by  (ID)` with an empty name instead of why it was refused.
- Writing a front-matter field replaces its whole old value when that value runs over several lines and one of them starts with `#` (t3fqaw). Such a line was taken for a YAML comment, so `hug task set ID --title` on Seid's t6j13s (a 3-line folded title whose third line is `  #return/#break clauses ...`) left that line under the new title. The same held for every field a command writes: `--type`, `--priority`, `--assignee`, `--label`, `wait`, `design`. A comment at the key's column is still not part of the value.
- `hug tui` help (`?`) leads with the keys for where you are, then the keys that work everywhere (jmx10s). Each place has its own heading: `IN THE LIST`, `IN THE OVERVIEW`, `ON THE BOARD`, `IN LINT`, `IN THIS TASK · ID`, `IN THIS FILE`, `IN THE : PALETTE`, `IN THE FILTER PICKER`, `IN / FIND`, `IN # JUMP`, `IN THIS CONFIRM`. One key per row, with the key column aligned. On a screen 120 wide or more, the place's keys and `EVERYWHERE` stand side by side. Only keys that work there are listed. A task's help follows its focus: the checks show Space and -, a task that shows ids shows Tab, and the row keys (o, m, J/K moves) are not listed in a task. A prompt lists only its own keys, since letters are typed text there. Every key a row names is pressed in that place by a test. `/` in help still finds rows by key or word, and Esc clears the find and then closes help. Before, help was one global list in which keys from every place stood side by side.
- `hug setup` answers in about half a second on Seid, not 34 s (3mavpt). It used to run the whole backlog import as a dry run on every call (2,677 files converted, 28 of 32 s). Now `hug task import backlog` and setup's own dry run remember their answer on this machine (in the git common dir), keyed on the hug binary and the backlog's file listing: name, type, size and mtime, with no file read. While nothing changed, and every board task the run matched is still on the board, setup reports the remembered numbers and when they were taken. Otherwise it says `hug setup: comparing .backlog/ with the board ...` and runs the dry run again. The task-files step (`.backlog/evidence/`) is remembered too, keyed on that folder's listing and the board's commit. `--cleanup` still runs its own full dry run before it deletes anything. The import itself is faster: citations are only searched in files that contain `file:`, `task:` or `.hug/tasks/` (2.5 ms to 26 µs on a 22 KB note without any), so a cold setup on Seid takes 9.5 s instead of 33 s. An interrupted setup (^C) now says `hug task setup was interrupted` with `fix: hug task setup`. Before, it printed a checklist with steps that failed because their git calls were killed, such as `the commit-msg gate needs a Git checkout` in seid-lang's own checkout.

## v0.7.47 - 2026-11-17

- A hug older than its board says so and writes nothing to it (dx9vw2). The board names its layout in hug.toml on the hug/tasks branch (`layout = N`), and each hug knows the newest layout it reads. The `state:` rename is layout 2. A board without the key is layout 1, from before the key: `hug upgrade --apply` moves its tasks to `state:` and then writes `layout = 2` in its own commit (`hug upgrade: board layout 1 -> 2`), and `hug task init` writes `layout = 2` on a new board. Run `hug upgrade --apply` once on each board; a board left without the key is layout 1, which every hug writes. On a newer board every write (note, take, wait, set, new, sync, tui, upgrade, the background deduce pass) stops before touching it with `this board is newer than this hug: the board has layout 3 (hug.toml on branch hug/tasks), this hug knows layout 2` and `fix: brew upgrade veridit/tap/hug`. A read still prints and says it once on stderr; `--json` and MCP replies carry it as `board_newer`. `hug upgrade` writes the layout when a later hug raises it. This guards the next change of layout, not the last: on 2026-11-17 a hug from before `state:` read a task as having no state and sent its agent to `hug task take`, and a hug that old has no such check.
- In a `hug tui` task's detail, Tab and Shift-Tab step through every task id the detail shows, in reading order, and Enter opens the one marked (4bwcgp). The stops are:
  - the parent epic: a new `epic:ID` badge in the header;
  - an id or alias named in the stage line: `└─ waiting on after ▸46ek1f`, or a design question or a deferral;
  - the Dependencies;
  - an epic's children: a new CHILDREN section;
  - the task:ID Links;
  - an id named in the Notes.

  The marked id carries ▸ where it stands, and Esc or u comes back. Only a word that resolves exactly to a task is a stop: no prefixes, slugs or unknown ids. h/l now move between the detail's sections, as in the rows (bejd89); Tab did that before. The key bar shows `Tab next id` when there is one. Before, an id inside a wait reason could not be reached from the detail. Jørgen hit this on jr55s8's "waiting on after 46ek1f".

- The Model: trailer names the effort too (jr55s8): `Model: claude-opus-5-5/high`, or `Model: human`, since effort can change a model's behaviour entirely. `hug task install commit-msg` refuses a missing effort (`fix: ... --trailer 'Model: claude-opus-5-5/<effort>'`) and checks the effort against the model's own levels in hug.toml `[models]` on the hug/tasks branch, one line per model (`"claude-opus-5-5" = ["low", "medium", "high", "max"]`). A model not listed is refused with `fix: hug task models add ID EFFORT && git ... commit ...`. Without a `[models]` table, any of none, minimal, low, medium, high, xhigh, max goes, and a typo (`hihg`) is refused. `hug task models` lists the roster; `hug task models add ID LEVEL...` writes a model's line and commits it. `hug setup` has a `models` step. The gate now starts three git processes and one awk per commit (it reads the roster with `git cat-file`). Reinstall the gate to get it: `hug task install commit-msg`. `hug task take ID --model claude-opus-5-5/high` records the model and the effort apart (`model:` stays the id, `effort: high`), and show, ls, board and the TUI row and detail name both (`poodle · claude-opus-5-5/high`).
- `hug mem` capture also rejects a memory that names an agent its own transcript uses as one (typq1e). The memory "In Brovs DevOps, spider pushes commits" was stored before, since it has no "coordinator" or "DM" of its own. It is now rejected with `names an agent: spider` when the chunk it came from says "coordinator spider". "The spider", "a house spider" and "Spider-2" are still kept. On a held-out sample of 40 replayed capture calls that name agents, judged blind, memories naming an agent fell from 0.175 to 0.050 per transcript (−0.125 [−0.250, −0.025]), with 0 durable memories lost and 0 rejections among 13 controls. On the two earlier samples it went from 0.087 to 0.025 and from 0.163 to 0.050. On a fourth sample (fresh3) it went from 0.237 to 0.100, and it rejected 1 memory in 80 run-items that the judge rated durable in one run and agent-naming in the other: a work order naming a gone agent ("tiger's norename probe"). The 10 memories still left say "via fox", "operator squid" or "the foreman" (muninn/docs/CAPTURE4.md). `capture4 -rematch` re-runs the matcher on judged outputs without a model call.
- `hug tui` orders rows by a column like a spreadsheet header (bejd89). In the list, the overview sections and the board, Tab and Shift-Tab move a column cursor over the headers on screen: ✓ (checks ticked), ID, TYPE, PRIORITY, STAGE, TITLE, and OWNER in IN PROGRESS and WAITING. The marked header starts with ▸ and is underlined, so it shows without colour too. `o` on it cycles ▲ ascending, ▼ descending, and off (rank order again). The header shows the arrow (`TYPE ▲`; a label too long for its cell gives way to it, `PRIOR ▲`), the line under the rows says `o order by priority (▼)`, and in the overview the rows reorder within each section. The order survives `R` reload and a filter change. Pressing `o` with no column marked says `Tab picks a column, then o orders by it`. h/l still change section, and Tab no longer does. STAGE orders by the stage tree, ✓ by the share of checks ticked. The views move to the `v` prefix: `v o` overview, `v i` (or `v l`, `v v`) list, `v b` board, `v !` lint. `i`, `b` and `!` alone still work, and `o` no longer switches to the overview.
- A key in `hug tui` costs the same on a big board as on a small one (46ek1f). Each frame computed every task's stage several times over (O(n²) on the overview: 145,000 stage computations per frame on 2,000 tasks), read every task's checks for the progress column, and summed the row heights of whole sections to learn they do not fit. Stages and the progress width are now computed once per load, and row heights are summed only as far as the screen reaches. Seid copy (2,520 tasks), 30 presses of `j`: 2.70 s CPU on main (f24080c), 1.57 s with this change. The sections (IN PROGRESS, WAITING, NEXT UP, the board columns) are now found once per set of shown rows, not by scanning every task for each question a key and its frame ask (about 40 scans of the board per key), and so is the header's `? N for you`: 100 presses of `j` cost 8.63 s CPU on main (f24080c) and 0.66 s, 6.6 ms a key.

## v0.7.46 - 2026-11-17

- The `hug tui` overview has a WAITING section between IN PROGRESS and NEXT UP (tje4sx). IN PROGRESS shows only work someone is on now (doing/active). WAITING, titled `WAITING · on you first`, holds every open task that waits on a person or a thing: stage waiting, design or review, todo or doing alike. It also holds a doing task held by an open dependency. Rows waiting on you come first and are the only rows marked `?`. `? N for you` stays in the top header, the one count seen in every view. A todo task with an open dependency stays in NEXT UP with the stage `depends`. WAITING rows keep the owner column. Before, Jørgen's hug overview showed 12 of 12 IN PROGRESS rows waiting, 11 of them on him, and no active work at all. Every row is now one line, in the overview, the list and the board. Only the row under the cursor opens its second line (`└─ waiting on you: ... since ...`), and j/k and scrolling count that one extra line. Before, every waiting row took two lines (89y0b7).
- An open `hug tui` reloads the board once per write, not twice (46ek1f). A `hug task note` changes the task file and then, a commit later, the branch. On Seid's 2,520-task board those land about 300 ms apart, longer than the 200 ms debounce, so each write cost two full loads (about 200 ms CPU each). The watch now waits while a writer holds the board turn and signals once it is free; a turn held longer than 5 s does not hold the reload back.
- A key in `hug tui` costs the same on a big board as on a small one (46ek1f). Each frame computed every task's stage several times over (O(n²) on the overview: 145,000 stage computations per frame on 2,000 tasks), read every task's checks for the progress column, and summed the row heights of whole sections to learn they do not fit. Stages and the progress width are now computed once per load, and row heights are summed only as far as the screen reaches. Seid copy (2,520 tasks), 30 presses of `j`: 2.70 s CPU on main (f24080c), 1.57 s with this change.
- `hug task review ID --by NAME [--show FILE]` asks for a review without a hand edit (6w0gve). Your doing task becomes doing/review and you keep it: `review_by` is the reviewer (required; a person in hug.toml [people] or an agent), `review_since` the time, and `--show` names one of the task's files to look at (`review_show`), which the TUI detail draws under the review question, as for wait and design. The reviewer answers with `hug task answer ID "looks right"`; the review fields go to History and the task is active again. A waiting task is refused with `fix: hug task resume ID`, a todo task with the take. MCP `task_update` action `review` with `by` and `show`. `hug task states` lists the move.
- `hug mem` starts fewer git processes to find where a memory goes (ce3dmq). In a main checkout, the project lookup starts 2 git processes, where it started 4: the top level is the main checkout, so it no longer asks git for the worktree list. In a plain folder with no `.git` above it, the lookup starts none, where it started 1 (a failing `rev-parse`). Placing a memory looks the project up once, where it looked it up twice. The muninn app tests started 1656 git processes and now start 422, at about 45 ms each on the Windows runner. That package had hit go test's 10-minute limit on Windows (main 9f3e38f, run 44142356523). Most of that time was one test's setup, not git: TestCaptureEightCandidatesOnTwentyThousandUnits seeded 20000 units one autocommit at a time, 116.6 s of the package's 162.5 s on the Windows runner. It now seeds them in one transaction and takes 6.1 s; the package alone takes 57.0 s on Windows (run 44156932321). The capture it times, 8 candidates against 20000 units, still takes about 0.5 s.

## v0.7.45 - 2026-11-17

- `hug task show` prints the stage as its second line, in the TUI row's words (456vyt): `stage: waiting on jhf: approve the diff since 2026-11-17 17:47`, `stage: in review by crab since ...`, `stage: question for jhf: ...`, or the stage word alone (`stage: active`). Before, a recorded wait did not show in text `show` at all, and an agent ran `hug task wait` again thinking the first had failed.
- `hug tui` and agents writing at the same moment no longer race (95ahtk). Every TUI write (check, strike, answer, take back, done, drop, new, move, the `S` screen attach) now holds the board's write turn from changing the task file to its commit, the same as every CLI write. So do the commit after the editor closes and the commit when the TUI quits. Before, a `hug task note` beside the TUI could stage the TUI's temp file and fail with `unable to stat '.huginn-...'`, or meet it on `index.lock`, as an agent's `hug task attach` did in Seid at 16:49:23 on 2026-11-17. Readers (`ls`, `show`, `next`, `search`) never wait for the turn.
- A task's state is `state:`, everywhere (6ephqk step 4). The task file says `state: doing`, where it said `status: doing`. The flag is `--state` (`hug task ls --state doing`, `hug task search --state todo`, `hug task set ID --state todo`), the MCP parameter is `state`, and `--json` gives `"state"`. The TUI palette filter is `: state doing`, and the board's columns are the state columns. A file from before reads as before, and lint names it once with the new code `old-layout` (broken), "old layout: status: is now state:", fix `hug upgrade`. The lint code `unknown-status` is now `unknown-state`. New tasks and imports write `state:`. The cache schema goes to 18, so the first command after the upgrade rebuilds the cache.
- `hug upgrade` runs every migration and says what it did, part by part (asr4t2): `layout` (the board branch and worktree, the settings file), `tasks` (status: to state:, backlog leftovers and section order, waits on a person to `waiting_for`, each committed on hug/tasks), `variables` (HUG_* in hook and MCP configs) and `memory`. A part with nothing to change says `nothing to do`. `hug upgrade` (and `hug task upgrade`) previews: it changes nothing, starts each change with `would: `, prints the per-task diff (`ID: status: doing -> state: doing`), ends with `fix: hug upgrade --apply`, and exits 5 when changes are pending (0 when there is nothing to do; `--json` carries `pending` and `exit_code`). `hug upgrade --apply` makes the changes, each step its own commit on hug/tasks, and then `next:` is `hug setup`. `--dry-run` is gone.

## v0.7.44 - 2026-11-17

- States only a ticket's owner can settle become one question on the ticket (at3qep), in `hug task lint`, `hug task show` and the owner's recall board line at its next break (nobody else's): `landed-unchecked` (a commit naming a doing ticket reached main and no check is ticked or struck: "landed, but no check is ticked: commit 61bfb21 naming 0hkaa5 reached main (its note says so) and 0 of 6 checks are ticked or struck; kestrel: which checks hold?", fix `hug task check ID 1`), `shipped-not-done` (a release contains it and it is open, fix `hug task done ID`), `todo-has-commit` (todo with a commit on main, fix `hug task take ID`), `doing-idle` (doing and active with no note, check or commit naming it for 2 hours, fix `hug task note ID "where it stands"`), and the existing `branch-gone`; all broken. One question per ticket, in that order (landed, shipped, branch gone, todo, idle); fixing the state clears it, nothing is stored. The first three read the `landed`/`shipped in` notes `hug task deduce` writes. Replayed on hug's board as it was at 12:34 today: 5 questions, among them 0hkaa5 (shipped in v0.7.38, still doing), m1c28r and t709he (landed, 0 checks ticked), the three Jørgen found by hand; now: 1 (gjmqhn).
- `hug task lint`: every finding is `broken` (hug cannot read or trust the task: done with an open check, a link to nothing, a cycle, a missing Goal) or `advice` (below the house style: long title, no acceptance, missing section, section order, Backlog.md leftovers). Broken findings always show, first, on every task; advice shows on open tasks and is hidden and counted on closed ones; `--all` shows everything. JSON adds `severity` and `summary.broken`. `hug tui` counts broken findings in the header (`! 4 broken · 5 advice`) and marks advice-only rows with a dim dot (pbh9xq).
- Import is a one-time move (ggydj2). Once every task is in, `hug task import backlog DIR --cleanup` ends it in one commit on your branch: `git rm -r DIR` (history keeps it), backlog's MCP server out of `.mcp.json` (and `.cursor/`, `.vscode/`, `.jcode/` configs), and the BACKLOG.MD block out of AGENTS.md / CLAUDE.md. The import is recorded in `hug.toml` (source, commit, date), and a later import of DIR is refused, naming the commit that holds the old copy. `--dry-run` shows the diff first, plus `agent:` lines for what hug never edits: hooks, scripts and settings that still run backlog or read DIR. A commit refused by a hook changes nothing. `hug setup` offers the retirement once the import is complete, stops offering the import once it is recorded, and flags a leftover DIR with the `--cleanup` fix.
- A task carries pictures of what it is about (5z4zt4). In `hug tui`, `S` copies the screen as you see it to the clipboard as plain text (OSC 52, which works in iTerm2, kitty and over ssh, plus pbcopy on a local Mac), then asks to attach it to the task in view as `screen-HHMM.ans`, an ANSI capture that keeps the colours; Esc keeps only the copy. An `.ans` file shows with its colours in any terminal (Space in FILES); its other escapes are shown escaped. `hug task attach` takes PNG, JPEG and GIF images up to 2 MiB when the name matches the bytes. Space on an image draws it in iTerm2 and kitty (also inside tmux), any key comes back, and elsewhere it gets one line with the size. `hug task wait ID "look at the rows" --on jhf --show FILE` and `hug task design ... --show FILE` (MCP `show`) name the file to look at (`waiting_show`, `design_show`). The detail shows it right under the question; without `--show`, a task file named in the question is used. A question with a picture opens on the page, and nothing is read until it is shown.

## v0.7.43 - 2026-11-17

- `hug task lint` owns the evidence rules Seid kept in `tools/evidence.py` (cvbpdy). A citation of a path under a `forbid_cite` prefix is one finding per citation, naming the path and its line (before: one finding per task naming only the prefix, line 0); a citation followed by ` (lost)` or ` (not kept: ...)` is a known gap and not reported; a path that only contains the prefix (`/home/x/.jcode/scratch/...`) is another folder. A cited `.hug/tasks/.../ID/NAME` path is checked like `file:NAME` and `task:ID/NAME`: missing is `missing-file`, and ` (lost)` after any of the three accepts it. `hug task setup` in a repo with `.jcode/` offers `forbid_cite = [".jcode/scratch/"]` with a command that writes and commits it. On a copy of Seid's board: 530 findings on 464 lines, the same lines and count `evidence.py`'s rule gives (0 on one side only; before: 817 on 707 lines).
- `hug tui` opens at once (vnyz8z, np2kk2). The first frame is the board the task cache last saw, with a spinner in the header while the cache catches up behind it; quitting no longer waits for that or for the file watch. A first start without a cache counts the files it reads (`reading the board: ... 1234 of 3630 files`). Setting up the watch reads each folder once (it read the parent folder twice per subfolder). On a copy of the Seid board (2,520 tasks): first frame 9.08 s -> 0.17 s, median of 5. A `#ID Enter` typed before the board arrives opens the task when it does.
- Two hug versions on one machine no longer wipe each other's task cache (gjmqhn). The cache is one file per cache schema, `.git/huginn/cache-<N>.db`. Before, a `hug tui` left open on v0.7.41 and agents on v0.7.42 dropped and rebuilt the shared `cache.db` on every open. A cache file no hug has opened for 7 days is removed, the old `cache.db` included.
- `hug mem remember` never reads as success when it stored nothing (p29rbz). Its three refusals (a near-duplicate, a possible secret, a project-specific note sent global) start `NOT STORED:` and exit 3 (other errors exit 1; `hug mem remember --help` says so); the MCP `memory_remember` reply is an error result that starts `NOT STORED:`. Before, the CLI said `hug mem: not stored yet: ...`, and a Seid agent took a near-duplicate refusal for a stored memory, forgot the original and lost both.
- hug tui: inside a task, J and K scroll half a page; they no longer move the task in the rank, which happened unseen (x1ra0t). Moving with J/K, m, y and P/p works on list, overview and board rows, where the rows are seen to move. The task's key bar now names every key that writes (`t/d/x/e take/done/drop/edit`, `N new`), right after the focused section's keys, so a narrow bar drops navigation hints first; each still asks before it writes.
- A question to a person is answered where it is asked (3jtfw9). In `hug tui` the detail of a task that asks you puts the question at the top with who asked and how long ago, and `a` opens the input right under it; Enter answers. From the command line: `hug task answer ID "the answer"` (MCP `task_update` action `answer`). It answers what the stage asks: a design question (todo/design to ready), a wait (doing/waiting to active), or a review (back to ready or active; `done` stays its own step). The answer writes no new field: one `## History` line keeps the question, the answer, who answered and for whom, `answered design: "q" -> "a", by jhf, for crab`. The asker sees it first in `hug task show` and `hug task next --mine` until it next writes on the task, which adds `crab read the answer`.
- Rows that wait on something take two lines (89y0b7). Line 1 is the task as always, with its title; line 2 hangs under it and says what it waits on, the reason right after the colon, and since when: `└─ waiting on you: approve the design ▸ bxqtkt.md   since 14:42`, `└─ in review by wyvern`, `└─ question for jhf: …`, `└─ depends on 44w1ev  ▱ 0/1`. Active and ready rows stay one line; j/k move by task, and section heights and scrolling count lines. A person is named by a short name, "you" for the TUI user, and the detail adds the long name. `hug task wait ID --on NAME "what"` writes the new node field `waiting_for: NAME` (it was "NAME: what" in waiting_on); `hug upgrade` moves an old "NAME: what" there when NAME is a listed person. People live in hug.toml as `[people.jhf]` with `name` and `email = [...]`; `hug task people` lists them and offers an entry per commit author no entry covers (emails grouped by author name), `hug task people add SHORT EMAIL [--name NAME]` writes one, and `hug task setup` has a "people" step. In `hug tui` you are your agent name (--as, HUG_USER, the session) or, as a person, whoever hug.toml lists with your `git config user.email` (else the email before the @). New lint code `unknown-person`: a waiting_for, design_for or review_by that names no listed person and no agent on the board, fix `hug task people add NAME EMAIL`.
- `hug tui` shows a task's file in place (scrxak). In a task's Files, Space shows the marked file in the detail pane, read only, under `path · size · 3 of 26`. j/k and PgUp/PgDn scroll, n/p step to the next and previous file (stopping at the ends with a note), Enter still edits it, and Esc or u go back to Files with that file marked. Plain text only. Control characters and escape sequences are shown escaped (`\x1b[2J`), never sent to the terminal. A binary file or one over 1 MiB shows one line with its size and `Enter opens it`. A file is read only when it is shown, never when the list or the task opens.

## v0.7.42 - 2026-11-17

- Faster commands on big boards (44w1ev): the board walk reads each folder once and checks files on 4 threads, and `show`, `take` and `lint` read only what they need. On the Seid board (2,519 tasks): `hug task lint` 7.1 s -> 0.55 s, `show` 3.4 s -> 0.15 s, and quitting `hug tui` 2.9 s -> 0.3 s.
- On Windows, parallel `hug task` commands on one board no longer fail with "The process cannot access the file because it is being used by another process": task files are read with delete sharing and replaced with a POSIX rename, so a read and a write of the same task never fail each other (before: 6 of 11646 reads and 165 of 169 replaces failed in a 2 s race on Windows CI) (kbc99j).
- The README states how fast changes reach another clone: at its next command, at most about 7 s later (1.5 s quiet before the push, a fetch once the last is 5 s old). The two-clone sync test now asserts that bound, computed from the same constants, and polls the note and the project memory together; it used to wait for the memory only after the note, which left the memory 1.2 s on a slow macOS runner (4vcwxk).
- `hug mem` capture no longer loses its batch on Windows while another process (a recall hook, the MCP server) reads the memory store: `memories.json` is read so a save may replace it, and replaced with POSIX semantics, through the same `internal/sharefile` as task files (zd3x24). Before, a Windows run of 2 s with 4 readers and 1 writer failed 92 of 2745 loads and 145 of 429 saves.
- One ticket in doing per agent. While you have a doing task that is active, `hug task take` of another (CLI, MCP `task_take`, TUI `t`) is refused with exit 3, naming it: `not taking B: kestrel is on A (title): one ticket in doing at a time`, then three `fix:` lines: `hug task done A`, `hug task wait A "on what"`, `hug task set A --status todo && hug task note A "why it goes back"`. `--json` and MCP give `fix` (the first) and `fixes` (all three). A task waiting, in review or depending on another does not count, and taking a review is allowed beside your own work. `hug task resume` is held to the same rule. People (`--model human`) too (n2hq8p).
- hug notes what git says about tickets (g9dg32). A commit whose subject ends with ticket ids, `Fix both (t709he, 44w1ev)`, reaching main adds `landed <sha> on main <date>: <subject>` to each ticket (the subject cut at a word to 80 characters, with …); a release tag `vX.Y.Z` that first contains such a commit adds `shipped in vX.Y.Z (<sha>)`, and sets `done_release` on a done ticket. It runs at the end of `hug task sync` and of the background pass, and as `hug task deduce [--since REV]` (`--since v0.7.30` replays old history; the first pass starts at the current tip). Each clone keeps what it has read in `refs/hug/seen/main` and `refs/hug/seen/tags/*` of the code repository, and checks the ticket's Notes before writing, so a line is written once per ticket and commit however many clones and passes see it. Each write is reported as `observed: commit abc1234 naming m1c28r reached main; marked: landed abc1234; if wrong: hug task note m1c28r "not landed: abc1234 (why)"` (in the reply and the pass log). It never ticks a check and never marks done.
- Every check reports every problem at once, with one fix (fkpyg3). `hug task new "x"` names each missing part, `incomplete-task: missing Goal; missing section Purpose (why); missing section Principle (what we do and do not do); Acceptance requires at least one item`, and gives one `fix: hug task new "x" --goal "..." --purpose "..." --principle "..." --accept "observable check"`; `--purpose` and `--principle` are now required, as lint asks for them, and MCP `task_new` needs only `title` in its schema so the reply can list everything (`findings` plus `fix`). `hug task set` writes `--goal`, `--purpose`, `--principle` and `--details` and appends `--accept` items, so every fix line runs without an editor (MCP `task_update` set takes the same fields). `hug task lint` is the board overview: findings per code, then one line per task (`ID  missing-section, long-title  -> hug task lint ID`), and `next:` is the first task's whole fix. `hug task lint ID` lists every finding of that task and one combined fix (`hug task set ID --title "..." --purpose "..." --principle "..."; then hug upgrade`); `show`, `take`, `new` and `set` end with the same block, and their JSON carries it as `lint` {findings, fix}. Rows in `ls`, the board and the TUI keep the `!` marker only. Every open task is held to the description pattern: the 2026-11-16 cutoff is gone (done and dropped tasks stay exempt). New code `backlog-leftover`: Backlog.md `<!-- SECTION:... -->` / `<!-- AC:... -->` marker lines and the unused fields `created_date`, `ordinal`, `backlog_status` and `import_type_inference`; `hug upgrade` removes them, prints each removed line, and never moves text between sections.
- The recall hook gives no board line in a turn that tells the agent to stop (`stop now`, `do not run any`, `end your session`, `stand down`); a scope rule such as "do not start any other task" still gets one. Replayed on 6 session sets (muninn/docs/BOARD4.md): the one false lint line given under a stop order (hog, "STOP NOW") is gone, and fresh data keeps 13 of 14 lint lines. Holding unrelated lint findings until an agent is between tasks was measured and not kept: it cut fresh lint lines 14 -> 8 (x325pp).
- Background work (`hug mem` capture, indexing, the watcher) is niced on every thread on Linux, not only the one that asked (1rca5g). Linux keeps niceness per thread, so 11 of 12 threads still ran at nice 0; now each thread in `/proc/self/task` is lowered, and threads started later inherit it.
- The recall hook hands a missing Purpose or Principle only to a session that touched the task (its own, named, or worked this turn); other sessions get the board's other findings, such as a long title. An unrelated finding also waits while the session's own doing task has one. Holding every open task to the description pattern (fkpyg3, above) had raised false board lines from 7 to 23 on the 6 replayed session sets: its missing-section findings went to whoever was at a break. With this change, blind-judged (muninn/docs/BOARD4.md): false lines 23 -> 12, right lines 99 -> 101; on fresh hug sessions 4 lint lines (3 right) -> 3 (3 right). The 5 false lines still above the pre-fkpyg3 7 are not new real findings: 9 false lines are new and 4 old ones are gone. Of the 9, 5 are a missing section, at the session's start, on a task its message only mentions (a quoted prompt, a dependency); 4 are on the session's own or noted task at a moment the judge calls wrong, one of them mtzydx with every check ticked (tpnd3n; the first kind is 4vxqp4).
- Fewer git processes per command, about 45 ms each on Windows: `take` 19 -> 15, `wait` 13 -> 9, `note`, `check`, `done` and `resume` 10 -> 9. Where a take happens (branch, worktree) is asked of git for `take` only, a diff names the commits' trees directly instead of resolving them first, and the busy check reads the git directory from the `.git` file. A test counts git processes per command and fails when one gains a call. The background-sync tests run in parallel: `huginn/internal/cli` on Windows was 289 s in the full CI run (0sbvnq).
- `hug task search "words"` and MCP `task_search`: the tasks that best match, ranked by BM25 over id, aliases, title, goal, purpose, principle, details, acceptance and notes (id, aliases and title count most), each with the field that matched and a snippet; the `ls` filters narrow it. The index lives in the task cache and follows every write. At 10,000 tasks the query itself takes 31-39 ms for a word in every task (texts are read only for the results shown), and a cold cache builds in about 4 s, as before. `hug tui` `/` search ranks by it too (49n9kw).

## v0.7.41 - 2026-11-17

- An acceptance check that no longer applies can be struck out with a reason instead of being ticked or left open: `hug task strike ID N --reason "why"` (MCP `task_update` action `strike` with `reason`; in `hug tui` press `-` on the focused check, type the reason, Enter). The file keeps the text, crossed through, with the decision: `- [ ] ~~text~~ (not relevant: why, NAME, 2026-11-17)`, which any GitHub-flavoured Markdown viewer draws crossed out. `done` counts a struck check as settled and still refuses an open one; `show` prints it as `2. [-] ~~text~~ (not relevant: ...)`, `--json`/`--jsonl` and MCP give each check a `state` (open, ticked, struck) with `reason`, `struck_by`, `struck_on`, and the TUI draws it ☒, dim and crossed through. A reason is required; `--undo` (or `-` again in the TUI) restores it; a ticked check cannot be struck, and a struck one must be restored before it is ticked. A done task with an open check is now reported as broken by lint, naming both ways out (sfa8w4).
- `hug tui`: `/` search finds a task by an imported id (an alias such as `STATBUS-402`, as `hug task show` accepts), case-insensitively; a row found only that way names the alias, dim, after the title. `#` jump already did (vzghj8).
- `hug tui` shows what waits on you. A task whose stage names the TUI user is marked `?`, sorts to the top of its section, and reads `review: you`, `design: you · the question` or `waiting: you · what`. The title line counts them (`? 2 for you`). Who a question is for is written, never guessed. `hug task design ID "question" --for NAME` records `design_for`, and it is required. `hug task wait ID --on NAME "what"` writes `waiting_on: "NAME: what"`. A review names its reviewer in `review_by`. In `hug tui`, `t` on a task waiting for review takes it for review as you (`review_by` is you, and the commit says `Model: human`). `t` on a task you have gives it back to todo, keeping its stage: a review stays waiting for review, and the assignee moves to `## History` (6ephqk).
- `hug tui` rows show the stage, not the state: the section is the state, so a row reads pie, id, type, priority, stage, then the text. A stage with something to say shows it in place of the title (`waiting:  CI run 44063552584 · 12 min`, `review:   Jørgen reviewing · 5 min`, `design:   should type leave the rows?`, `deferred: after the release`). `active` and `ready` show the title. `depends` shows a bar of dependencies done and then the title (`depends   ▰▰▱ 2/3  One set of variable names`). The BLOCKED section is gone: a task with an open dependency stays in IN PROGRESS or NEXT UP with that stage, and the header, filters (`4`, `: depends`) and board column say `depends` where they said `blocked`. Rows still fit one line; at narrow widths type goes first, then priority, then the stage (6ephqk).
- Tasks move along the state tree only: `hug task wait ID "CI run 123"` and `hug task resume ID` for a doing task waiting outside the board, `hug task design ID "question"` (and `--clear` once answered) for a todo task with an open question, and `hug task drop ID "reason"`, which now needs its reason. Each move writes its node's fields (`doing_since`, `waiting_on`, `waiting_asked_by`, `design_question`, `design_asked_by`, `done_at`, `dropped_reason` ...) and moves the old ones to `## History` with the time. `hug task done` and the TUI `d` need a doing task: a todo task is refused with `fix: hug task take ID --model MODEL` (TUI: `t takes it, then d`). `hug task set ID --status todo` reopens a done or dropped task. The TUI `x` asks why before it drops. The stage is the first one that holds, in order (todo: deferred, depends, design, draft, review, ready; doing: waiting, depends, review, active), and when a task's state changes, the tasks that depend on it directly get their stage again. That is one level and no deeper: a stage reads only its own dependencies' state, and a stage change never changes a state, so a chain of 10 is right after every write (finishing link 1 frees link 2; link 3 still depends on link 2, which is still todo). A file without stages, as every board has until `hug upgrade` writes them, gets its stage on its first move (6ephqk).
- `hug task states` prints the task state tree from the one table hug holds it in: the states todo, doing, done and dropped, each state's stages (todo: deferred, depends, design, draft, review, ready, in that precedence; doing: active, waiting, depends, review), what each means, who sets it, the fields that belong to it (`doing_since`, `waiting_on`, `design_question`, `review_by`, `dropped_reason` ...) and the moves the commands make; `--json` gives the same. `hug task lint` holds a task's `stage:` and node fields to that tree with four new codes: `unknown-stage`, `stray-node-field` (a field outside its node, such as `waiting_on` on an active task), `missing-node-field` (a waiting task without `waiting_on`) and `stage-disagrees` (a `ready` task whose dependency is still open). Nothing writes stages yet: today's boards lint the same as before (6ephqk).
- The board says who is working (v3k6cp). The recall hook gives one `owner` line when a session works a doing task (took it, works in its worktree or branch, ticks its checks, or notes it while editing) that the board assigns to another name: `Task board: you are working on 44w1ev but the board says humpback. If it is yours now: hug task set 44w1ev --assignee wyvern`. The session name comes from the harness, never the model. `hug task take --as NAME` with a name other than the session's still takes and prints `warning: assignee NAME is not this session (...)` with that fix. `hug task lint` reports `stale-owner`: a doing task whose assignee has had no live jcode session for over 24 h. Replayed on the six BOARD4 sets: 14 owner lines, 13 judged right; all kinds 0.926 -> 0.934 precision (muninn/docs/BOARD4.md).
- `hug tui` gives each status one subtle colour, used in the count line, the status word on rows (closed rows too) and the overview section titles: IN PROGRESS in the doing colour, NEXT UP todo, BLOCKED red, RECENTLY DONE green, LINT the warning colour of `!`. The progress cell shows a fraction only for partial work: a done task shows its check count, `● 5`, in green instead of `● 5/5`, a task with nothing ticked shows `○ 3` dim, a task with some ticked keeps `◑ 2/4` (resumable work), and a dropped task shows none. Without colour everything still reads by its text (zhvkmj).
- `hug tui`: `u` inside a task goes up one level, as Esc does: back from a followed link, else to the list, board or overview it came from, with the task selected. The key bar in a task shows `u up`. In `: sort`, `u` is still the updated field (9xfnwr).
- `hug task done ID` asks one question: did the instructions (AGENTS.md section, tool descriptions, hug messages) guide you well, or did one mislead you or leave a gap? `--reflect "..."` (MCP `task_done` `reflection`) stores the answer on the task as `Reflection (open, DATE, NAME): ...` once done succeeds; "nothing" is a fine answer and stores no note. `hug task reflections` lists open ones across the board with the task, date, who and the quoted instruction; `hug task reflections resolve ID N applied|dismissed` closes one; `--all` shows closed ones too. The `hug setup` section asks agents to answer honestly (tb0dra).
- A task can wait on a whole epic, a group, or a condition hug checks itself. A dependency on an epic now waits until all its children are done or dropped, not for the epic's own status (`show`: `[todo, epic: 2 open]`). `hug task dep ID --add group:auth` waits until every task in folder `auth` (and its subfolders) is closed, and `--add "when PREDICATE"` waits for `date 2026-12-01`, `load below 120` (5-minute average), `branch feat/x merged` (the tip is recorded, so it still reads as merged once the branch is deleted), `release v0.7.40 exists`, `task:ID closed` or `group NAME closed`; add `wakes NAME` to say who it is for. `next` skips the task and `take` refuses it (exit 3, `fix: hug task show ID`) while it does not hold; a task or group condition also holds back `done`. `show` prints `waits: when load below 120  [waiting: load is 140.0]  wakes kestrel`, `--json` and MCP carry `waits` (text, met, seen, wakes), the TUI detail lists `⧗ when ... WAITING · what hug saw`, and the board marks the task `!`. Nothing is polled and nothing goes to the network: hug checks when it reads the board. A condition hug cannot read, or cannot check here (a branch this clone lacks, no load average on Windows), never holds, and lint reports it as `[unknown-condition]` with `fix: hug task dep ID --remove 'when ...'`. Memory conditions (`--holds`) use the same evaluator (16dxsr).
- `hug task note`, `check`, `set` and the other writes walk the board once instead of three times: the read before the board turn is gone (the write reads under the turn), and after writing a task file the cache rereads that file only, not all 10,000. At 10,000 tasks `note` drops from 0.52-0.56 s to 0.39-0.42 s; `show` and `next` are unchanged (44w1ev, step 8).

## v0.7.40 - 2026-11-17

- One set of variable names: every setting hug reads is `HUG_*` (`HUG_HOME`, `HUG_USER`, `HUG_MODEL`, `HUG_SYNC`, ...). A variable still set under its old `HUGINN_*` or `MUNINN_*` name is ignored, and every `hug` command says which one, with a `fix:` line naming the new name. `hug upgrade` renames the old names where hug wrote them (its own hook commands and MCP server entries in the Claude Code, Claude Desktop, Codex and Jcode configs), keeping a `.bak-hug-upgrade` copy, and never touches anyone else's entries (m1c28r).
- The 10,000-task benchmark is out of the normal tests: it runs only with `HUG_BENCH=1`, in its own CI job (ubuntu, no race detector), and fails when a warm command costs more than half of the same command without the cache, measured side by side in one process, instead of against a fixed time that slow runners crossed (ubuntu 4.93 s over 4.8 s; on Windows it took ~290 s of the package's 10 minutes). The tree before the cache work fails it (ls 0.70, lint 0.79, note 0.56 of uncached); today's commands sit at 0.07-0.21 (44w1ev).
- Todos are your own next steps on the task you took; details, evidence, waits and dependencies live in the task. `hug task take` now says so in two lines (`now: todos: your own next steps on ID ... a wait, a dependency or another agent's work is not a todo: hug task note ID "...", hug task dep ID --add OTHER`, then check, note, `hug task done ID, then hug task next`) and ends with `next: hug task show ID`; MCP `task_take` returns the same `guide` and `next`. The AGENTS.md section from `hug setup` states the loop, and `hug setup` now flags a `## Tasks: hug` section from an older hug with the replacement text (tzc9hb).
- `hug tui` search keeps your place: after `/` finds a task, Enter goes to it and Esc clears the search, and either way the full view shows with that same task selected and in view, ready for `m`, `J` and `K`. The footer says where it is (`found ID, row 50 of 60 in the list`) and, when sorted, that moving needs `: sort rank`. `↑↓` pick another match while typing. Tab keeps the search on as a filter, which Enter used to do. Clearing a kept search, a chip or all filters keeps the task too. A task the filter hides (opened with `#`) is never swapped for another: no row is selected and the footer says `ID is hidden by the filter type:bug: F clears it` (mcvfbz).
- `hug tui` rows fit one line at any terminal width: list, overview, IN PROGRESS and board rows share one layout. When the width runs short the title is cut first, then the IN PROGRESS owner cell (its branch before the name), then whole columns drop in a fixed order: type, priority, status. The progress pie and the id always stay. Widths count display cells, so CJK and emoji titles are cut cleanly with `…`. Before, at 60 columns an IN PROGRESS row lost its whole title behind the owner column (102bc0).
- Capture no longer stores a memory that names a swarm agent ("report to coordinator fox"): after the model call, a candidate naming an agent of this machine in agent context is rejected with `names an agent: NAME` in `hug mem capture log`. No extra call. Replayed on 40 + 40 real capture calls that name agents (two disjoint samples, two runs per arm) and blind-judged: agent-naming memories per transcript 0.31 -> 0.19 and 0.42 -> 0.25 (CI-clear), durable agent-free memories unchanged (+0.000), controls unchanged; all 24 rejections were judged agent memories. A prompt rule saying the same cut more but lost durable memories (-0.54 per transcript, CI-clear) and is not shipped (muninn/docs/CAPTURE4.md, ez50rb).
- Every commit can name what made it: `hug task install commit-msg` refuses a commit without exactly one `Model:` trailer (`Model: claude-opus-5-5`, or `Model: human` when the git author wrote it), with a `fix:` line whose `<model-id-or-human>` slot is refused until replaced. A hook already in `.git/hooks` keeps running first (as `commit-msg.pre-hug`); with `core.hooksPath` hug never edits the tracked hook and `hug setup` shows the two lines to add. Merges, rebases, cherry-picks, reverts, fixup!/squash! and the `hug/tasks` and `muninn-notes` branches are exempt; no bypass variable (`git commit --no-verify` is the emergency exit). hug's own commits on `hug/tasks` and `muninn-notes` add `Model:` from `--model` or `HUG_MODEL`, the TUI `Model: human`; never a guess. README: list commits by model with `git log --format='%h %(trailers:key=Model,valueonly,separator=%x2C) %s'` (t709he).

## v0.7.39 - 2026-11-17

- The task cache opens in one read instead of decoding 10,000 rows, a write stores only the tasks it changed, and `ls` takes notes and description texts from the cache instead of reading every file: at 10,000 tasks `ls` drops from 1.08 s to 0.45 s, `next`, `show` and `board` from 0.33 s to 0.24-0.26 s, `note` from 0.60 s to 0.51 s. An edit that keeps a file's size and lands in the same clock tick as the last read is now always seen: files modified within 2 s of the last sync are hashed even when size and time match (44w1ev, step 3).
- `hug task lint` takes file references from the cached parse instead of reading every task file (it reads them only when `forbid_cite` is set): at 10,000 tasks lint drops from 2.0 s to 0.36 s, on seid's 2,508 tasks from 1.50 s to 0.22 s (44w1ev, step 4).
- Every `hug task` command starts git 5 fewer times (8 instead of 13 for `next`): the repository is looked up once for the checkout check, the old-layout check and the board's checkout, the old refs are read in one call, the Git version is checked only before creating a checkout, and the remote comes from the repository's config when it defines `origin`. On a one-task board `next` and `show` drop from 0.14 s to 0.07 s, `ls` from 0.18 s to 0.10 s; at 10,000 tasks `next` and `show` drop from 0.25 s to 0.19-0.20 s (44w1ev, step 5).
- Commands about one task use half the memory at 10,000 tasks: the cache keeps notes and description texts in their own table instead of in the board it decodes on every command (21.6 MB -> 7.1 MB), and only `ls` reads them back. Peak memory for `next` and `show` drops from 165 MB to 95 MB, `board` from 160 MB to 85 MB, `lint` from 165 MB to 117 MB; `ls` takes about 35 ms longer (0.43 s -> 0.47 s) (44w1ev, step 6).
- A test builds a 10,000-task board and fails CI when `ls`, `next`, `show`, `board`, `lint` or `note` take more than 1.2 s once the cache is warm (4.8 s under the race detector); the tree before this work (e5cb389, v0.7.33) fails it with ls 2.0 s, lint 3.6 s, note 1.24 s (44w1ev).
- Parallel `hug task` commands no longer fail with `database is locked (5) (SQLITE_BUSY)` on the task cache: every cache transaction takes SQLite's write lock up front and waits for it (up to 30 s per try, retried), instead of failing at once when another command committed first. 20 parallel notes all land again (44w1ev on m8shdj).
- `hug tui` opens a 10,000-task board in about 0.5 s instead of 3 to 5 s, its heap 75-81 MB instead of 170-190 MB: it takes lint findings from the task cache instead of parsing every task file on each reload, reads a task's file when its detail opens, and finds task folders with one listing per group instead of one walk per task. Filtering and moving at 10,000 tasks take at most 53-127 ms per key (44w1ev).

## v0.7.38 - 2026-11-17

- `hug tui` IN PROGRESS rows name the branch a task was taken on after the owner, e.g. `alice · feat/x`, unless it is the main branch; on a wide terminal the owner column grows to fit it, up to 30 cells, and on a narrow one the branch is cut, never the owner (x1t1t1).
- Task and project-memory changes reach the other clones by themselves, within seconds: every `hug task` write commits locally and starts one detached syncer per board, which waits until a burst of writes is quiet for 1.5 s, then fetches, rebases and pushes (20 notes in 2 s are one or two pushes). Reads (`ls`, `show`, `next`, board) start a fetch when the last one is over 5 s old. `hug mem remember`/`forget` of project memories, `hug mem search` and the recall hook do the same on the `muninn-notes` branch. Nothing waits on the network: offline, a note costs what it costs online, and `hug task ls` says `sync: N unpushed: ...` with `fix: hug task sync`; they go out at the next write once the remote is back. A rebase conflict keeps every local commit and says so the same way. `take` keeps its synchronous push (the race lock); `HUG_SYNC=off` turns background sync off; passes log to `.git/hug/sync/` (0hkaa5).
- Every git network call (fetch, push, ls-remote) is non-interactive and has a deadline: no password or passphrase prompt (ssh BatchMode, no askpass, Git Credential Manager non-interactive; a user's own ssh command is kept), ssh gives up connecting after 10 s, a stalled transfer after 15 s, the whole call after 60 s, with a `fix:` line. An unreachable remote is named as such, not as a key problem (0hkaa5).
- Writes to one board take turns (a board lock from reading a task to committing it), so a background rebase never moves files under a command, and 20 parallel `hug task note` commands all land (0hkaa5).
- The board turn holds on a board without a remote too: before, 5 of 20 parallel `hug task note` commands there failed with `fatal: cannot lock ref 'HEAD'` or `index already contains staged changes`; now 20 of 20 land. `hug mem capture` takes the write turn for its saves and checkpoints (never across the model call), so it no longer interleaves with a remember, forget or index; its audit record too (m8shdj).

## v0.7.37 - 2026-11-16

- A memory can say how long it is true: `hug mem remember --holds "until 2026-12-01"` (or `"while task:ID open"`, `"while branch NAME unmerged"`; MCP `holds`). When the condition ends, checked without a model call, recall stops showing it and dream offers it once to retire or keep; nothing is deleted. A memory with only a recorded branch is never hidden: a merged or deleted branch raises one dream question. Recall output is byte-identical with no conditions stored (277 frozen + 147 fresh replayed messages, 0 differing). Dream also offers memories that name a stopped swarm agent, 10 per pass, with the tickets that agent's session worked on: on the local store 291 of 29,118 memories name an agent (precision 50/50 on a hand-checked sample), 159 name one that has stopped, 119 of those with a ticket. Capture is unchanged (ez50rb).

## v0.7.36 - 2026-11-16

- `hug tui` offers to reload when a newer hug is installed where it was started from (`brew upgrade` moves the `hug` symlink to a new Cellar folder; `go install` replaces the file): only the version corner changes, to `hug v0.7.35 → v0.7.36 · R reloads`, and nothing else happens until R. R becomes the new hug in the same terminal, on the same view, task, filters, sort, focus and scroll, and says `reloaded hug vX.Y.Z`; inside a prompt, the palette or help R is just a key there. A new binary that does not start, or an older one, leaves the running TUI as it is, with the reason and a `fix:` line. Not on Windows, which has no exec (j2vwk0).

## v0.7.35 - 2026-11-16

- The recall hook keeps the task board current: when the work and the board disagree it adds one line with the command that fixes it, e.g. `Task board: you are working on p277bp but it is todo. hug task take p277bp --model claude-opus-5-5`. It says so when code changes with no task doing (`hug task take ID`, or `hug task next` / `hug task new`), when the session's task has every check ticked (`hug task done ID`), and at a break hands out one lint finding with its `hug task ... ID` fix (one per 6 h; each finding goes to one agent, then rests 7 days). At most one line per turn, one per kind per hour. A task is the session's when it took it, its name is the assignee, or the task's recorded worktree or branch is where it works, never on model alone. `"nudge_board": false` in muninn config.json turns it off. Replayed on 46 real sessions (378 session-hours): precision 0.88 to 1.00 per set, 0.30 false lines per 10 session-hours at most; 3 lines on the 15-16 Nov overnight run where no task was taken; board read p95 27 ms on hug's board, 144 ms on seid-lang's 2,508 tasks (muninn/docs/BOARD4.md) (qawmnb).
- `hug task lint` and `board` at 10,000 tasks: a dependency or parent is looked up in an index, not by scanning the board, and lint lists each group folder once instead of checking every task path part by part: lint 2.65 s -> 2.18 s, board 0.42 s -> 0.35 s, ls 1.42 s -> 1.17 s (44w1ev, step 2).

## v0.7.34 - 2026-11-16

- `hug task` reads the board once per command instead of up to six times: at 10,000 tasks `next` and `show` drop from 0.59 s / 0.71 s to 0.34 s / 0.33 s, `note` from 0.99 s to 0.60 s, `ls` from 1.63 s to 1.41 s (44w1ev, step 1).

## v0.7.33 - 2026-11-16

- `hug tui` rows show acceptance progress before the id, a pie that fills by quarters and ticked/total, e.g. `◑ 2/4 8bsx4y`: green when all are ticked, yellow when some are, dim when none are; a task without checks leaves the cell blank. `hug task ls` prints ticked/total after the type, e.g. `1/2`, so the id stays the first field.
- A task reads top down: Goal, Purpose, Principle, Relevant details, Depends on, Acceptance, any other section, then Notes and History. `hug tui` shows Purpose, Principle and Relevant details under the goal, above the checks (they used to come after History). `hug task show` prints them after `goal:` and adds `purpose`, `principle` and `details` to `--json`. A new note on a task with History goes above it. `hug task lint` flags a file out of order (`section-order`, fix: `hug upgrade`), and `hug upgrade` puts every task file in order without changing any section's text, then commits (jnaspy).

## v0.7.32 - 2026-11-16

- `hug tui` ends the key bar with the version, `hug vX.Y.Z`, right-aligned and dim, the same string `hug version` prints; when the line is too narrow the version goes first and no key hint is pushed off (8bsx4y, from Jørgen).
- `hug tui` overview: rows the focused section does not need are no longer left blank; they go to IN PROGRESS until it shows all its tasks, then to NEXT UP, then the rest. With IN PROGRESS focused holding 2 tasks and NEXT UP 32, a 140x40 terminal now shows 21 of NEXT UP instead of 3 (z6kz20).
- `hug tui` shows a task's attached files: `⎘2` before the title on its row in overview, list and board, and a FILES section inside the task. Enter on a file opens it in the editor, and `E` opens the task's whole folder; the key shows only for a task with files. When the editor closes, the change is committed (`hug task edit ID file NAME`, `hug task edit ID files`) (zgr01t).

## v0.7.31 - 2026-11-16

- `hug task take` records the branch and worktree it runs in (`branch`, `worktree` on the task, `--branch`/`--worktree` override). `hug task show` prints `taken by: dragon · claude-opus-5-5 · feat/x`; `hug task ls` rows add the branch when it is not the main one; `ls --json` carries both (x1t1t1).
- MCP `task_take` requires `cwd`, the folder the agent works in: without it, with a relative path, or outside git it fails with a `fix:` line; the branch and worktree are read from that folder, not the server's (x1t1t1).
- `hug mem` records the branch on memories remembered or captured in a linked worktree or on a branch other than the main one (x1t1t1).
- `hug task lint` flags a doing task whose branch is merged into main or deleted (`[branch-gone]`, `fix: hug task done ID`); nothing is closed automatically. `hug tui` names the branch in the detail header (x1t1t1).

## v0.7.30 - 2026-11-16

- `hug tui` sorts by fields: `: sort` opens a builder where a letter adds a field (p priority, t type, s status, u updated, c created, g group, a assignee, n title), the same letter again flips its direction, ⌫ removes the last, Enter applies and Esc keeps the order there was. The header shows the order, e.g. `sort: priority ↓ · type`, and `: sort rank` goes back to rank order. While sorted, J/K, m, y and P/p move nothing and say `moving needs rank order: : sort rank` (mkzcqr).

## v0.7.29 - 2026-11-16

- `hug mem` writes wait their turn instead of failing when many agents write at once. Every write (remember, forget, rescope, correct, tidy, index) queues for one write turn, a lock file waited on in the kernel; a waiter fails only when one holder keeps it 20 minutes, and then names the holder (pid, command, since) with a `fix:` line. A waiter prints `hug mem: waiting 12s for the write turn, held by pid N (hug mem forget ...)` on stderr after 2 s and every 15 s after. Before, waiters shared SQLite's 30 s busy_timeout as one budget for the whole queue: on a copy of the real store 19 of 20 parallel forgets failed at 31 s with `database is locked (5) (SQLITE_BUSY)` behind one 32 s index pass; now 20 of 20 succeed (me51r2).
- `hug mem forget ID ID ...` and `hug mem forget --stdin` (one id per line; `hug mem list` lines work) forget many memories in one turn, with one save and one index pass; one unknown id fails the call and forgets nothing. `--json` prints `{"forgotten": [...]}` (me51r2).
- `hug mem list --json` and `hug mem search --json` print one object, `{"memories": [...]}`, each with id, kind, scope, category, type, tags, text and updated (search adds score) (me51r2).
- A recall hook's skipped index pass no longer leaves its connection's busy_timeout at 5 s instead of 30 s (me51r2).
- `hug tui` shows inline code and code blocks in a quiet blue-grey on a faint background (`#c9d6e3` on `#1a2630` dark, `#2b3a48` on `#e8eef3` light), without padding and without red; red is left for blocked and lint (8bsx4y).

## v0.7.28 - 2026-11-16

- `hug tui` overview: the section you are in takes every spare row and the others keep 3; RECENTLY DONE lists every closed task newest first and scrolls through them when focused; each section and board column keeps its scroll and row when you Tab away and back; an empty section or column is one line, `None` on its title (h9zghj).

## v0.7.27 - 2026-11-16

- CI runs the race detector once, on ubuntu; macOS and Windows run plain
  tests, so the release gate takes about 8 minutes instead of 14.
- `hug tui` shows task ids bare in their colour, without brackets, in rows, the detail header and the trail; an open task lists every `task:ID` its text names under Links, and Enter follows one (Esc comes back) (xxvm6t).

## v0.7.26 - 2026-11-16

- `hug tui` shows the model next to the assignee in a task's detail:
  `assigned:poodle · claude-opus-5-5` (et85h1).
- Every command and MCP tool that takes a task ref accepts `task:ID`, the form notes and goals use, as well as the bare id; printed commands keep the bare id (xxvm6t).
- CI is green on Windows again: the lint, auto-refresh and memory worktree tests no longer assume a shell, a fast `git` or forward-slash paths (4emme8).

## v0.7.25 - 2026-11-16

### Fixed

- `hug task promote ID N` gives the new task the source's Purpose,
  Principle and Relevant details, so it starts complete and `hug task lint`
  reports it clean (it used to get `missing section Purpose, Principle`).
  A source without them gives a child without them. (heype5)
- `hug task tui` shows text in angle brackets as written in the task
  detail: a note like `git worktree add --detach <dir> <rev>` no longer
  loses `<dir>` and `<rev>`, which the markdown renderer read as HTML.
  Code spans and fenced blocks are unchanged. (heype5)

## v0.7.24 - 2026-11-16

### Changed

- `hug tui` moves a task far in one go, one commit per move (3xcesq).
  `m` grabs the selected task: `j`/`k` (with a count such as `5k`), `gg`
  and `G` move it on screen, nothing is written until `Enter` drops it
  with one commit, and `Esc` puts it back. The footer says `Moving ID · row
  N of M in SECTION (was K) · nothing written until Enter` and the key bar
  shows only the keys that work. `y` yanks the selected task, then `P`
  pastes it above the selected row and `p` below. In lists `P` and `p` only
  paste: with nothing yanked they are not shown and say "nothing yanked: y
  on a task first"; `k` moves up. `J` and `K` move one row with no confirm, since the other key
  undoes it. Moves stay within the rows shown in one section; RECENTLY
  DONE, ordered by close time, refuses them and says where to move. In
  lists, the type palette moved from `y` to `:type` and `f t` (`y` inside
  a task still opens it).

## v0.7.23 - 2026-11-16

### Changed

- `hug task take` needs the model doing the work: `--model claude-opus-5-5`
  (or `HUG_MODEL`; a person gives `--model human`). Without it, take changes
  nothing and ends with `fix: hug task take ID --model MODEL`. The agent or
  session name is optional: `--as poodle`, otherwise `HUG_USER`, otherwise
  the jcode session name (from `JCODE_SESSION_ID`); with no name the
  assignee is the model. hug never uses `git config user.name` and no
  longer reads `HUGINN_USER`. Both are recorded on the task (`assignee`,
  and a new `model` field), and `hug task show` (`taken by: poodle ·
  claude-opus-5-5`), `ls` (doing rows), `board --wide` and JSON show them.
  The MCP tool `task_take` takes `model` and `agent`. `--mine` means the
  same name take records. `hug setup` and the AGENTS.md section tell
  agents to name their model on take. (et85h1)

### Fixed

- `hug task tui` puts every task in one place. A task in progress that waits
  on a dependency stays in IN PROGRESS, drawn blocked, and no longer shows a
  second time in BLOCKED; BLOCKED holds only tasks nobody started that
  cannot start. The header counts no longer overlap and add up to the tasks
  shown, and doing says how many of its tasks are blocked: `todo 8  ·  doing
  15 (1 blocked)  ·  blocked 7  ·  done 15  ·  dropped 15` (it used to read
  `todo 15 · doing 15 · blocked 8` on the same 60 tasks). The board columns
  and the status filter (`blocked`, `todo`, keys 1 to 4) follow the same
  rule. (zaec08)

### Added

- The overview has a RECENTLY DONE section, between BLOCKED and LINT: the 3
  tasks closed last, newest first, by the commit that closed them (`hug
  task done` or `drop`, or a `set` or edit that made the status done or
  dropped), with how many are closed in all. On a short terminal it gives
  way first and keeps one row. (zaec08)

## v0.7.22 - 2026-11-16

### Fixed

- `hug mem sync` in a linked Git worktree (`git worktree add`) writes and
  commits project memories in the main worktree's one `.muninn-notes/`
  checkout, the same rule as tasks. It used to check `muninn-notes` out
  under the linked worktree, which git refuses ("already used by worktree")
  when the main worktree has it, or else made a second, separate checkout
  there. `hug mem project` shows the main worktree's file. (065wwn)

### Added

- `hug task tui` refreshes by itself when tasks change on disk: a
  `hug task note`, `set` or `take` in another terminal or by an agent shows
  within a fraction of a second, with no key pressed. The selected task, the
  list's scroll and the open task's scroll stay put. An open search, palette,
  confirm question or editor is never interrupted; the change shows once it
  closes. The TUI asks the OS to tell it when the task branch's ref, the
  `packed-refs` file or the tasks folder changes, so it does nothing while
  nothing changes; where the OS cannot do that, it checks the ref's
  modification time every 2 seconds. `r` still refreshes at once. (nfv18p)

## v0.7.21 - 2026-11-16

### Fixed

- `hug task` works in a linked Git worktree (`git worktree add`). It used
  to try a second checkout of `hug/tasks`, which git refuses ("already used
  by worktree"), and its `fix:` line repeated the same failing command. Now
  every linked worktree uses the one checkout at the main worktree's
  `.hug/tasks/`, so `hug task ls`, `new`, `take`, the MCP tools and the TUI
  read and write the same board from any worktree. `hug task init` in a
  linked worktree of a fresh repository creates the board in the main
  worktree. A checkout folder that was deleted or that git no longer knows
  now gets a `fix:` that works (`git worktree prune`, or move it aside and
  check the branch out again). (065wwn)

## v0.7.20 - 2026-11-16

### Fixed

- In `hug task tui`, J and K move a task past the row shown below or
  above it, in the section on screen (NEXT UP, a board column, the list)
  and after filters. They used to move it past the next task in the full
  order, often an epic, a closed task or a filtered-out one, so a press
  made a commit and nothing on screen changed. The confirm line names both
  tasks (`Move a1dx7s below x1t1t1 in NEXT UP: ...?`), the result says
  where it went (`moved a1dx7s below x1t1t1, row 2 of 3 in NEXT UP`), and
  at the first or last row shown the message says so and nothing is
  committed.

## v0.7.19 - 2026-11-16

v0.7.18 (tagged at 1fe040f) carried everything below except the hiding of
findings on closed tasks.

### Fixed

- `hug task lint` is never silent: a clean board prints `no findings` and a
  `next:` line. On a board with more than 30 findings, plain lint prints a
  summary by code, the first 10 findings and how to see the rest
  (`--all`), instead of every one. On a 2,478-finding board the old
  318 KB output came back as nothing in an agent's tool panel.
- `hug task lint help` and `hug task lint help --code` print the lint help
  (they said "expected 0 positional arguments" and "missing value for
  --code"). The same goes for other commands without arguments (`ls help`,
  `next help`, ...). `new help` still makes a task titled "help".
- The `not-a-dependency` finding starts with what to do when a task has
  no dependencies: "(for no dependencies, leave the section empty or
  remove it; ...". It used to suggest `none` was missing a form.

### Added

- `hug task lint --summary`: one line per code with its count, its meaning
  and the command that lists it, aligned with spaces. JSON lint replies
  carry `summary` (`total`, `codes`) and still list every finding.
- `hug task lint --code` with no value (or `--code help`) lists every code
  with its count and meaning. An unknown code says so, names the closest,
  lists the valid ones and ends in `fix: hug task lint --code`.
- `hug task lint --all` prints every finding, including those on done
  and dropped tasks.

### Changed

- `hug task lint` (CLI, MCP `task_lint` and the TUI) leaves out findings
  on done and dropped tasks and ends with how many it left out:
  `986 findings on closed tasks hidden (432 done-unchecked, ...): hug task lint --all`.
  JSON replies carry `hidden` and `summary.hidden_codes`. Opening a closed
  task in the TUI still shows its own findings.
- A mistyped flag's `fix:` is that command's help (`hug task lint --help`),
  not the general index.

## v0.7.17 - 2026-11-16

### Fixed

- A line in `## Depends on` that is not a dependency (prose, a question) no
  longer becomes one. It used to take the first word as a task id: a
  question showed the task as blocked on "Is". A dependency line is
  `- <task id>` (or an alias like `TASK-123`, optionally `task:ID`, then a
  note); anything else is reported by `hug task lint` with the new code
  `not-a-dependency`, quoting the line and showing the form to write. It
  creates no dependency and never blocks the task.
- `hug task lint`'s `next:` repairs the first finding (its fix command)
  instead of naming an unrelated task to take. Same in `--json`. A clean
  board keeps the usual next.

### Changed

- TUI shortcut letters are picked by one rule shared by the header, palette
  and help: each choice gets its first letter no earlier choice took (todo,
  doing, blocked, done, dropped become t, d, b, o, r), and a label that
  starts with a sigil, like `!lint`, gets that sigil.
- Every lint finding carries a fix: text output prints `fix: COMMAND` under
  each finding, JSON findings have a `fix` field. A dependency on no task
  now reads "depends on ID, but no task has that id or alias; remove the
  line, or write an existing task's id" with `fix: hug task dep TASK
  --remove ID` (code `dangling-dependency` unchanged).
- TUI: after an edit that leaves the task with a new finding, the footer
  shows it (task, line, problem) and a bar with its fix and `e edit again ·
  Esc dismiss`; `e` reopens the editor on that task at once. The key shows
  only while the finding is there. The task detail header shows the
  finding's line and, on its own line, the fix.
- `hug task edit`'s `next:` is the fix for the edited task's first finding.
- The task cache schema is 4, so lint stops serving findings cached under
  the old dependency reading.

## v0.7.16 - 2026-11-16

### Fixed

- TUI palette: after `f`, a field can be shortened, so `: f g ` lists the
  groups and `: f t spike` filters by type. It said "No completions".

### Changed

- TUI task detail has three focus states: page, acceptance and dependencies.
  Tab and Shift-Tab cycle them and skip empty sections; a task opens on its
  checks when it has any. Only the focused heading is highlighted (with a
  `▸`); with page focus no check shows the `>` pointer, j/k scroll the page
  and the task id in the tab bar carries the focus.
- Inside a task the tab bar marks no view; it shows the trail instead, for
  example `list › 000001`, on the same line.
- Key hints follow focus and show only keys that work there. Keys are never
  written as `[ / ]` (read as the `/` key): headings say `j/k or [ ] step`.
  Space ticks only with the checks focused.
- Every confirm prompt (take, done, drop, move, edit, new, tick) is one
  full-width bold warning-coloured bar: `▶ …? Enter yes · Esc no`, the
  question and its keys surviving truncation. The tick wording stays
  `Tick: ☐ 2. text`; the edit prompt names the editor it will run
  (`Edit qawmnb in subl --wait?`), and help says how it is chosen ($VISUAL,
  else $EDITOR, git core.editor, vi).
- Board legend names its columns: `Type: F feature … · Priority: C critical
  …` with T and P emphasised; narrow boards show `T: … · P: …`.
- The task header shows no empty `[]` badge for an unset type or priority.
- In a task, `n` and `p` open the next and previous task (Ctrl-N and Ctrl-P
  still work); `{` and `}` are gone, being hard to reach on a Norwegian
  keyboard. In lists `n` and `p` move like `j` and `k`. New task is `N` in
  every view.

## v0.7.15 - 2026-11-16

### Added

- State-change reminders: `hug task take`, `check`, `done`, `drop` and
  `promote` print `now:` lines saying what to do at that moment (MCP: a
  `guide` field). Take says to keep one todo named with the task id, not one
  per check, and that the task stays doing through detours. Done on an epic's
  last child says to close the epic.
- TUI detail: `{` and `}` (also Ctrl-P and Ctrl-N) open the previous and next
  task of the list the detail was opened from. The footer shows "Task 3 of 22".
- TUI palette is a tree and uses the full height: `filter`, then the field,
  then its values with task counts. `: f done` finds a value without naming
  its field. The short forms (`status todo`, `done`) still run when typed.

### Changed

- TUI rows: the lint mark `!` sits in the gutter left of the id instead of
  its own column. Board rows show type and priority as one coloured letter,
  with a legend under the board, so titles get the width.
- TUI: Space in a list no longer asks to tick a check you cannot see. It
  points to Enter. Inside a task the prompt reads `Tick: ☐ 1. text`, the
  check drawn as the ticket shows it.
- TUI detail: open acceptance checks use the plain text colour, so they no
  longer look like the blue headings. Ticked checks stay green.

### Fixed

- `hug mem --help` and `hug mem` errors say `hug mem` (was `muninn`), and an
  unknown command ends with a `fix:` line.

## v0.7.14 - 2026-11-16

v0.7.13 was tagged but not published: its own test fixtures tripped the
release's no-secrets gate. v0.7.14 carries the same changes plus:

- TUI: the help bar is limited to the terminal width.

## v0.7.13 - 2026-11-16 (not published)

### Fixed

- The secret check (memory writes and task attach) catches keys stored as
  `name=value` under a secret-like name, such as `aws_secret_access_key=...`
  or `DB_PASSWORD: ...`, when the value looks like a key (12+ characters
  mixing letters and digits). On the 28,476 stored memories here it flags
  nothing new; sudoers `NOPASSWD:` lines and variable names stay unflagged.

## v0.7.12 - 2026-11-16

### Fixed

- `hug task show ID --file NAME` prints the file. Since v0.7.4 it printed
  only the next hint unless `--json` was given.
- `hug task take` commits as `hug task take ID` (was "Update hug/tasks"),
  and its check for unpushed tasks looks at the current task branch
  (it still named the old `huginn` branch).
- TUI help lists the filter keys s/t/p/g/l/a/e and the `epic` command.

## v0.7.11 - 2026-11-16

### Fixed

- On a machine without a git identity, task commands say so and give the
  `git config --global user.name/user.email` fix (found in a clean Linux
  container), instead of suggesting to re-run the failed commit.
- TUI commits read `hug task check ID` (not `huginn: check ID`), and every
  version line says `hug vX.Y.Z`.

## v0.7.10 - 2026-11-16

### Fixed

- Speed with epics: parents are resolved once per command. On 2,498
  seid-lang tasks `hug task ls` went from 19.1 s to 0.8 s and the TUI starts
  in 1.4 s instead of 18.2 s.
- TUI: NEXT UP shows what `hug task next` would pick (unassigned, not an
  epic); the header says HUG TASKS; the Space prompt no longer ends in "??".

## v0.7.9 - 2026-11-16

### Changed

- `hug mem` commands end with a `next:` hint, like `hug task` (on stderr
  for commands that print JSON, so stdout stays parseable; hooks, mcp,
  serve and tui stay silent).

## v0.7.8 - 2026-11-16

### Changed

- Memory TUI: places read as folder names ("hug" instead of
  "project:rc-0ca4bf6be5d989c3") and plain words for shared scopes
  ("all agents", "jcode, all projects"). The names come from the folders
  jcode sessions ran in, scanned in the background and cached for a day.

## v0.7.7 - 2026-11-16

### Added

- Task description pattern: `hug task new --purpose --principle --details`
  (and the same `task_new` fields) write Purpose, Principle and Relevant
  details after Goal. Lint (`missing-section`) asks for Purpose and
  Principle on open tasks created from today; older tasks are not flagged.
  The names come from a blind test: named sections scored +0.49 [+0.28,
  +0.71] over none (docs/task-vocabulary-test.md).

## v0.7.6 - 2026-11-16

### Added

- Lint findings carry a code, shown as `[code]` and as `code` in JSON
  (long-title, no-acceptance, parent-too-deep, ...). `hug task lint --code
  CODE` keeps one kind, so problems can be fixed one by one.
- `hug task board --wide`: one section per column with full titles.
- This repository has an AGENTS.md; setup's agent section now covers the
  task description pattern, epics and promote.

### Changed

- `hug task init` makes the first commit and says what it set up: branch,
  worktree, head and remote.

## v0.7.5 - 2026-11-16

### Added

- Epics. `hug task set ID --parent EPIC` puts a task under an epic (epic >
  task > acceptance item; an epic has no parent, `--parent none` clears).
  `show` lists the epic and its children; `done` on an epic refuses while a
  child is open (exit 3); lint reports a missing parent or one that is too
  deep, with the exact fix.
- `hug task import backlog` maps `parent_task_id` to the parent epic. On a
  copy of seid-lang 282 of 294 links came back; rerun to restore them.
- `hug task promote ID N`: acceptance item N becomes its own task, ranked
  right after ID under the same epic. The item then reads "done when
  task:NEW is done" and is ticked when NEW is done. MCP: `task_update`
  action `promote`.
- `--epic EPIC` on `ls`, `next` and `board` (and MCP list/next); `next`
  never picks an epic itself. TUI: filter field `e` and `:epic ID`.
- `hug apps [install] [--dry-run]`: Claude Desktop, Claude Code and
  ChatGPT/Codex get the hug MCP server (backup kept, other servers kept).
  `hug setup` lists it.
- Lint: open tasks with titles over 80 characters, with the fix.

### Changed

- When the remote refuses SSH (apps start MCP servers without your SSH
  agent), the error says so and gives a fix that works; reading tasks never
  needs the remote.
- `hug mcp` notes in its replies when Homebrew upgraded hug underneath it,
  so the app gets restarted.
- TUI: the selected row is tinted across its full width.

## v0.7.4 - 2026-11-16

### Added

- Task files: `hug task attach ID FILE`, `hug task show ID --file NAME`,
  `hug task detach ID NAME`. A task's files live in a folder named by its id
  next to the task file. Text only, 256 KiB each, secrets refused. Notes cite
  them as `file:NAME` (own task) or `task:REF/NAME`; lint flags missing ones.
- `hug task import files DIR [--cite PREFIX]` moves an existing evidence
  folder into task files and rewrites citations (seid-lang:
  `hug task import files .backlog/evidence --cite .backlog/evidence`).
- `hug task ls --jsonl`: one JSON task per line.

### Changed

- Settings file is `hug.toml` (`huginn.toml` still read; `hug upgrade`
  renames it).
- Every message, fix and help line names `hug task ...` / `hug mem ...`.
  `hug version` prints one version.
- `hug task ls` columns: id, status, type, slug, title.
- New slugs: the first four meaningful words of the title, at most 32
  characters. Existing file names are unchanged.
- `hug task rebalance` prints one summary line, and its help explains it
  only renames filenames (order and contents unchanged).
- Recall: `skip_records_above` defaults to 600. On 147 fresh real messages,
  judged blind, long messages got 2.56 fewer irrelevant memories each
  [CI -3.00, -2.08] with relevant unchanged (-0.08 [-0.26, +0.08]).
- TUI: acceptance items wrap with a hanging indent, the selected item is
  highlighted, j/k step through items; Space asks "Tick item N of ID: ...?";
  o/i/b/! work in detail; overview sizes sections by need with ↑/↓ marks
  where rows are hidden; "owner" is now "assigned".

## v0.7.3 - 2026-11-15

### Fixed

- The jcode recall hook no longer delivers a memory after the turn it was
  recalled for has ended. Before, jcode started a new turn with it, which
  looked like an empty "[transcription]" message from the user.
- `hug task import backlog .` explains that the repository root holds the task
  folder and suggests `huginn import backlog .backlog --dry-run`.
- `hug setup` lists the `task_*` MCP tool names that `hug mcp` serves.

### Added

- Memory type: rule, preference, fact or record. `memory_remember` and
  `muninn remember --type` take it, capture's existing model call sets it, and
  `muninn types` fills it in for older memories (resumable, `--dry-run`).
- `skip_records_above` (off by default): automatic recall can leave "record"
  memories out for long messages; `memory_search` still finds them.
- `hug task import github [OWNER/REPO] [--dry-run]`: issues become tasks
  through the gh CLI, with alias GH-<number>; re-running syncs, huginn edits win.
- `hug upgrade`: tasks move from branch `huginn` at `.huginn/` to branch
  `hug/tasks` at `.hug/tasks/` (branch rename and worktree move, history
  unchanged, remote branch renamed). Task commands on an old repository stop
  with `fix: hug upgrade`, and `hug setup` lists it first. Safe to run again.

## v0.7.2 - 2026-11-14

### Changed

- Recall scores keyword matches with BM25 (`bm25_scale: 2.0`): rare words such
  as ticket ids and branch names count more, common words less. On 277 replayed
  real messages, judged blind, long messages got 0.39 more relevant memories
  each [95% CI +0.25, +0.55] with no more irrelevant ones; short messages are
  unchanged. `bm25_scale: 0` restores the old scoring. Details:
  muninn/docs/RECALL4.md.

## v0.7.1 - 2026-11-13

### Added

- `hug mcp`: one MCP server with the memory tools (`memory_*`) and the task
  tools (`task_list`, `task_next`, `task_take`, ...). If memory cannot open, the
  task tools are still served. `muninn mcp` and `huginn mcp` are unchanged.
- The installers leave a configured `hug` server alone instead of adding
  separate muninn/huginn servers; `hug setup` suggests `hug mcp`.

### Fixed

- Hooks record the path the program was started from (e.g.
  `/opt/homebrew/bin/muninn`), not the versioned Cellar path, which broke on
  every Homebrew upgrade.

## v0.7.0 - 2026-11-13

### Added

- One binary, `hug`: `hug task ...` (huginn), `hug mem ...` (muninn), `hug setup`,
  `hug tui`, `hug version`. Called as `muninn` or `huginn` it acts as that part,
  so existing hooks and MCP configs keep working.
- One release per tag: `hug_<version>_<os>_<arch>` archives with hug, muninn and
  huginn; Homebrew `veridit/tap/hug` (the `muninn` and `huginn` formulas now
  install hug); Scoop `hug`; one root `install.sh`.
- `muninn dream --summary`, recall counts in `muninn show`, and dream items that
  ask the driving agent to vet memories unrecalled for 60 days (never archived
  automatically).

### Fixed

- Hooks and MCP configs installed from hug name the `muninn`/`huginn` link, not
  `hug`, which does not know `hook` or `mcp` at the top level.
