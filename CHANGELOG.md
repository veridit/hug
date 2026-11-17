# Changelog

Releases are tagged `vX.Y.Z` and ship one `hug` binary. Earlier part releases
are in [muninn/CHANGELOG.md](muninn/CHANGELOG.md) and the `muninn-v*` and
`huginn-v*` tags.

## Unreleased

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
