# 0pr03h dry run: one-sentence Goals for open tasks

Board: hug/tasks at 69c43e54 (2026-11-20 04:00 +01:00), read from a copy; nothing was written to the board.

Rule (hug task lint goal-too-long, branch 0pr03h): the Goal is one sentence of at most 140 characters; detail goes in Details.

Open tasks (todo, doing) whose Goal fails: 47 of the board's open tasks (36 doing, 11 todo). All 47 are over 140 characters; 19 are also more than one sentence. Every proposed Goal below passes the same rule (checked with task.GoalProblem).

Proposal per task: the new Goal, and Relevant details becomes the old Goal word for word, then a blank line, then the old Relevant details unchanged. No text is lost; nothing else on the task changes.

Apply (after approval): for each entry in goals-dry-run.json, hug task set ID --goal GOAL --details DETAILS (JSON fields goal and details hold the exact text).

## 1. tb0dra (doing): Agents reflect on the instructions and propose fixes
Now: Goal is 476 characters in 3 sentences

Proposed Goal (130 characters):

> Agents reflect at done on whether the instructions guided them, and each reflection becomes an instruction change or is dismissed.

Moves to Relevant details (the old Goal, first):

> Reflection is part of the loop. When an agent closes a ticket, hug task done asks it one question: did the instructions (AGENTS.md section, tool descriptions, hug messages) guide you well, or did one mislead you or leave a gap? The answer is stored with the ticket as a Reflection note naming the exact instruction. hug task reflections collects open ones across the board, and a coordinator or Jørgen turns them into an instruction change (shown as a diff) or dismisses them.

Then the existing Relevant details (206 characters), unchanged.

## 2. 9taa9d (todo): Merge muninn and huginn into one hug binary
Now: Goal is 199 characters

Proposed Goal (111 characters):

> One hug binary carries tasks, memories and messages, with muninn and huginn kept only as compatibility aliases.

Moves to Relevant details (the old Goal, first):

> A unified, easy-to-navigate, non-confusing CLI and system: one hug binary with tasks, memories and messages; .hug/<kind>/ on branches hug/<kind>; muninn and huginn kept only as compatibility aliases.

Then the existing Relevant details (703 characters), unchanged.

## 3. p277bp (doing): Brand: hug as the tool, Huginn and Muninn as mascots
Now: Goal is 150 characters

Proposed Goal (88 characters):

> Users meet one tool, hug, with the ravens Huginn and Muninn as mascots, not as commands.

Moves to Relevant details (the old Goal, first):

> Users meet one tool, hug (Old Norse hugr, mind), with the ravens Huginn (thought, tasks) and Muninn (memory) as mascots and etymology, not as commands

Then the existing Relevant details (260 characters), unchanged.

## 4. 6assqk (todo): Recall: learned 'not when' triggers and a local candidate rule
Now: Goal is 174 characters

Proposed Goal (108 characters):

> Recall shows more of the relevant memories search finds and learns when not to show one, all local and free.

Moves to Relevant details (the old Goal, first):

> Recall shows more of the relevant memories search already finds, and stops showing a memory in situations like those where it was judged irrelevant before, all local and free

Then the existing Relevant details (849 characters), unchanged.

## 5. n7jbmx (doing): Recall on the agent's tool calls (post_tool_feedback)
Now: Goal is 218 characters

Proposed Goal (110 characters):

> Memories reach the agent mid-turn when its tool calls match them, if a fresh held-out test says to turn it on.

Moves to Relevant details (the old Goal, first):

> Memories reach the agent mid-turn when what it is doing (files touched, commands run) matches them, with a stricter bar than message recall; done when a fresh held-out test decides whether tool-call recall is turned on

Then the existing Relevant details (1594 characters), unchanged.

## 6. bxqtkt (doing): Contributions: agent feedback and GitHub intake with minimal bureaucracy
Now: Goal is 244 characters

Proposed Goal (112 characters):

> Outside agents and people contribute to hug through one triage step, and we can always see who contributed what.

Moves to Relevant details (the old Goal, first):

> Settle how outside agents and people contribute to hug with the least process: agent feedback, tickets written on forks, and GitHub issues all reach our board through one triage step, and we can always see who contributed what and what to pull.

Then the existing Relevant details (894 characters), unchanged.

## 7. 5dxzxe (todo): Own agent harness: persistent workers, messages and wakes on the board
Now: Goal is 279 characters

Proposed Goal (117 characters):

> We decide whether hug grows its own swarm layer: persistent workers, messages never lost, and waits that wake agents.

Moves to Relevant details (the old Goal, first):

> Decide whether hug grows its own swarm layer, likely on a Go agent harness such as a Go port of Pi: workers that persist across idle time and restarts, messages that are never lost, and waits that are real conditions on the board which wake the right agent when they become true.

Then the existing Relevant details (464 characters), unchanged.

## 8. txjkrm (todo): Tags: listed per board in hug.toml, shown on rows, Seid's labels cleaned up
Now: Goal is 375 characters

Proposed Goal (118 characters):

> Each board lists its tags in hug.toml, rows show them, lint holds tasks to the list and hug upgrade cleans old labels.

Moves to Relevant details (the old Goal, first):

> Labels become tags: each board lists its tags in hug.toml, every task's tags show dimmed on its row before the title (compiler parser  Compile-time bounds…), lint holds every task to the list, and hug upgrade cleans an existing board (merges spelling variants, moves labels that duplicate type, stage or priority into those fields, drops one-off tags) after showing the diff.

Then the existing Relevant details (399 characters), unchanged.

## 9. 47sxn2 (doing): The board keeps itself true: deduce what happened, ask when it can't
Now: Goal is 583 characters in 2 sentences

Proposed Goal (116 characters):

> Tickets match what really happened: hug writes what it can deduce from git and asks the owner what it cannot settle.

Moves to Relevant details (the old Goal, first):

> Tickets match what really happened without a person reminding anyone. Two levels: (1) when hug can deduce an event with certainty from git or the board (a commit naming the ticket reached main, a release tag contains it, a branch recorded on take is merged), it writes that onto the ticket itself; (2) when it observes something it cannot settle (landed but checks unticked, a doing ticket with no activity, a commit on main naming no ticket, a merged branch whose ticket is still doing), it raises a clarification request on the ticket, addressed to its owner, who answers in place.

Then the existing Relevant details (392 characters), unchanged.

## 10. ggydj2 (doing): Import is a one-time move: --cleanup retires the source, re-import is refused
Now: Goal is 435 characters

Proposed Goal (106 characters):

> An import is a one-time move: --cleanup retires the source directory and a second import of it is refused.

Moves to Relevant details (the old Goal, first):

> After hug task import backlog DIR, hug task import backlog DIR --cleanup ends the move: it removes DIR with git rm in one commit (git history keeps it), records the import in hug.toml (source, commit, date), and from then on a new import of that source is refused with a fix: line naming the commit where the old copy lives. hug setup stops offering the import once it is recorded, and flags a leftover DIR with the --cleanup fix line.

Then the existing Relevant details (420 characters), unchanged.

## 11. 5z4zt4 (doing): Attach screenshots and screen captures to a task; the detail shows them inline
Now: Goal is 557 characters in 2 sentences

Proposed Goal (118 characters):

> A task carries screenshots and ANSI captures that its detail shows inline, and a question can name the one to look at.

Moves to Relevant details (the old Goal, first):

> A task can carry pictures of what it is about: PNG/JPEG screenshots and ANSI screen captures (a TUI frame with its colours). hug task attach ID FILE stores them with the task; the detail view shows them inline where the terminal can draw images (iTerm2 inline images, kitty graphics, sixel) and ANSI captures with their colours in any terminal; otherwise one line with the size and 'Enter opens it'. A question to a person (waiting_on, review_by, design_for) can name the attachment to look at, and the detail shows that attachment right under the question.

Then the existing Relevant details (399 characters), unchanged.

## 12. xgwwhn (todo): Seid switches its pre-commit to hug task lint
Now: Goal is 151 characters in 2 sentences

Proposed Goal (110 characters):

> On Seid, hug task lint reports what tools/evidence.py check reports, so its pre-commit hook can switch to hug.

Moves to Relevant details (the old Goal, first):

> on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch
>
> Promoted from item 4 of task:cvbpdy.

Then the existing Relevant details (439 characters), unchanged.

## 13. h00cs2 (doing): hug mem: dream never runs on its own unless hug mem serve is running
Now: Goal is 170 characters

Proposed Goal (116 characters):

> The daily dream pass runs on every machine that captures, with or without hug mem serve, and shows when it last ran.

Moves to Relevant details (the old Goal, first):

> The daily dream pass (merge and retire duplicates) runs on every machine that captures, whether or not a hug mem serve process is up, and a user can see when it last ran.

Then the existing Relevant details (367 characters), unchanged.

## 14. 0gztsy (doing): Detail: one place for what a task asks of you; open with focus on it
Now: Goal is 582 characters in 5 sentences

Proposed Goal (82 characters):

> A task's detail shows what it asks of you in one place and opens with focus on it.

Moves to Relevant details (the old Goal, first):

> A task's detail shows what it waits on you for once, not three times. The header keeps one stage line ('└─ waiting on you: ... since ...'). The separate '? waiting:' question block goes when the wait names a check: that check in the acceptance list carries the question marker instead, looks exactly as it does in the list (same box, number, text), and the detail opens with focus on it, so Space ticks it and 'a' answers. A wait that names no check keeps one question block, styled like an acceptance row, and gets focus on open. The lint line for a person-wait is gone via wdkqzx.

Then the existing Relevant details (643 characters), unchanged.

## 15. qh9ep1 (doing): Key prefixes: v picks the view (vo vl vb v!), a pending prefix shows its choices
Now: Goal is 410 characters in 3 sentences

Proposed Goal (116 characters):

> Views are chosen with v and a letter (vo, vl, vb, v!), and every pending prefix key shows its choices in the footer.

Moves to Relevant details (the old Goal, first):

> The view line reads 'view: overview  list  board  !lint' with v and each view's letter marked, and the views are chosen with two keys: vo, vl, vb, v!. While a prefix key is pending, the line it belongs to lights up and the footer lists its choices ('v: o overview · l list · b board · ! lint · Esc cancel'). Every prefix in the TUI works the same way (a key trie), so g can be 'g: g top · r grouping' (kx2ndm).

Then the existing Relevant details (507 characters), unchanged.

## 16. rxr4z8 (doing): In lists, Space marks tasks and the footer offers actions for the marked ones
Now: Goal is 366 characters in 3 sentences

Proposed Goal (90 characters):

> In lists, Space marks tasks and the footer offers actions that apply to every marked task.

Moves to Relevant details (the old Goal, first):

> In the overview, list and board, Space toggles a mark on the focused task and steps to the next row. While one or more tasks are marked, the footer becomes an action bar for them, e.g. '2 marked · m move · t take · d done · x drop · w wait · e edit · Esc unmark', and each action applies to every marked task, with one confirm naming the count. Esc clears the marks.

Then the existing Relevant details (535 characters), unchanged.

## 17. 41hhgn (doing): Questions to a person get their own doing stage, 'asked'
Now: Goal is 598 characters in 4 sentences

Proposed Goal (92 characters):

> A question to a named person is its own doing stage, asked, separate from waiting on things.

Moves to Relevant details (the old Goal, first):

> doing gets a stage 'asked': the owner asked a named person something and can't go on until they answer. Interaction with a person is its own thing, not a kind of waiting. Fields: asked_what (the question), asked_for (who), asked_check (the check it is about, optional), asked_since, asked_by, asked_show. 'waiting' keeps only waits on things with no person (CI, a release). The TUI drops the '?' glyph: the row reads 'asked: you · check 9 · <question>', and questions for the viewer come first. hug upgrade moves every waiting with waiting_for into asked, and a leading 'check N:' into asked_check.

Then the existing Relevant details (760 characters), unchanged.

## 18. hbgybn (doing): Requests: a person writes a request in a task, an agent turns it into edits
Now: Goal is 579 characters in 3 sentences

Proposed Goal (110 characters):

> A person writes a request on a task in their own words, and an agent turns it into ticket edits and closes it.

Moves to Relevant details (the old Goal, first):

> In a task's detail, R opens a multi-line box where the viewer writes a request to the agents in their own words (short, dictated, rough). hug stores it on the task as an open request: who, when, the text verbatim. The task's owner, or whoever works the board, is told at once (recall hook and the write-command footer), reads it, turns it into systematic changes to the ticket (goal, checks, details, notes, state), and closes the request with a one-line account of what changed. The person sees open requests on the row ('req 1') and the account under the request in the detail.

Then the existing Relevant details (991 characters), unchanged.

## 19. w40cmk (doing): Layout 3: every task file is ID.md and never renamed; rank and group in the file
Now: Goal is 538 characters in 5 sentences

Proposed Goal (110 characters):

> Every task file is .hug/tasks/ID.md for its whole life, with rank and group in the file, so links never break.

Moves to Relevant details (the old Goal, first):

> Each task lives at .hug/tasks/ID.md for its whole life, so a path or URL to it never breaks (e.g. github.com/veridit/hug/blob/hug/tasks/6ephqk.md) and git log -- ID.md is its full history. The rank prefix, the slug and the group folder leave the file name: rank and group become front-matter fields. A move edits the moved task's rank: line; it never renames. The task's own folder stays .hug/tasks/ID/. Memories move to the matching branch name: muninn-notes becomes hug/memory (9taa9d check 2). hug upgrade migrates both, preview first.

Then the existing Relevant details (924 characters), unchanged.

## 20. z9q42h (doing): Recall reaches jcode through a tool hook, never the person's input box
Now: Goal is 275 characters

Proposed Goal (96 characters):

> On jcode, recall reaches the agent through a tool hook and never touches the person's input box.

Moves to Relevant details (the old Goal, first):

> On jcode, the recall block reaches the agent without touching the TUI's input box: turn_start computes it and parks it for the session, and the first tool call of that turn delivers it as post_tool_feedback (or pre_tool additionalContext). jcode transcript is no longer used.

Then the existing Relevant details (609 characters), unchanged.

## 21. z2vmte (doing): Wait for a date: wait --until and ask --after, woken by hug
Now: Goal is 342 characters in 2 sentences

Proposed Goal (92 characters):

> A doing task can wait until a date, then returns to its owner or asks its question that day.

Moves to Relevant details (the old Goal, first):

> A doing task can wait for time to pass: hug task wait ID --until DATE "why" parks it in WAITING until DATE, then it returns to its owner, first in hug task next; hug task ask ID --to NAME --after DATE "question" parks it in WAITING until DATE, then the question appears in that person's ASKED. Nothing about it shows in ASKED before the date.

Then the existing Relevant details (638 characters), unchanged.

## 22. mbdfzs (doing): hug mcp serves with the installed hug, never an old binary
Now: Goal is 225 characters

Proposed Goal (118 characters):

> A running hug mcp server serves every request with the hug installed now, re-executing itself when the binary changes.

Moves to Relevant details (the old Goal, first):

> A running hug mcp server serves every request with the hug that is installed now: when the binary on disk changes, it re-executes itself between requests, keeping its stdio connection, so no write comes from an older version.

Then the existing Relevant details (380 characters), unchanged.

## 23. 6ep7y2 (doing): Windows CI: a background syncer held the queue 30 s and no pass ran
Now: Goal is 263 characters

Proposed Goal (119 characters):

> The Windows stall in TestBackgroundSyncBurstIsBatched is explained by the logs and fixed, with a test that fails first.

Moves to Relevant details (the old Goal, first):

> When a test's settled() times out, the pass log and the failure say which lock each syncer waited on and for how long, and the cause of TestBackgroundSyncBurstIsBatched's stall on Windows (CI 44237364634) is found and fixed, with a counting test that fails first.

Then the existing Relevant details (918 characters), unchanged.

## 24. gmzd87 (doing): Requests reach the board coordinator by design
Now: Goal is 224 characters

Proposed Goal (110 characters):

> A request on any task reaches the board coordinator through hug, even when it is idle, with a visible receipt.

Moves to Relevant details (the old Goal, first):

> A request a person writes on any task reaches the board coordinator (hug.toml [board] coordinator) through hug itself, even when that session is idle, and its receipt is visible: pending, acknowledged, in progress, answered.

Then the existing Relevant details (393 characters), unchanged.

## 25. qpp30y (doing): Answer box shows at once, with the only moving cursor on screen
Now: Goal is 201 characters

Proposed Goal (98 characters):

> Pressing a or r draws the answer box at once, with a blinking cursor as the only motion on screen.

Moves to Relevant details (the old Goal, first):

> Pressing a (answer) or r (request) draws the empty box under the question immediately, with a blinking underline cursor where the text goes; that cursor is the only motion on the otherwise calm screen.

Then the existing Relevant details (376 characters), unchanged.

## 26. 911j70 (doing): Lists as a sectioned spreadsheet: ID row header, Excel-like selection
Now: Goal is 288 characters

Proposed Goal (111 characters):

> hug tui lists read as a calm sectioned spreadsheet with the task ID as the row header and Excel-like selection.

Moves to Relevant details (the old Goal, first):

> hug tui lists read as a calm sectioned spreadsheet: the task ID is the leftmost row header, selection tints only the row's ID and the column's heading, cells are normal weight on the plain background, and the view line keeps stable two-step key colours with one blinking underline cursor.

Then the existing Relevant details (281 characters), unchanged.

## 27. kg07bt (doing): Detail: a calm ACCEPTANCE heading; only the cursor marks the question
Now: Goal is 362 characters

Proposed Goal (107 characters):

> In a task's detail the ACCEPTANCE heading is quiet and only the blinking cursor marks the focused question.

Moves to Relevant details (the old Goal, first):

> In a task's detail the ACCEPTANCE heading is quiet (no background fill; at most bold or dim, like the other section headings), and the focused asked check / question is not bold plus background plus strong colour at once: it reads as a normal row, and the one blinking underline cursor (accent #00bfff/#006c9c, 530 ms) is the only thing that says 'you are here'.

Then the existing Relevant details (329 characters), unchanged.

## 28. tz9z14 (doing): TUI: the reload message clears when you open a task
Now: Goal is 409 characters in 2 sentences

Proposed Goal (89 characters):

> A main-screen message such as 'reloaded hug vX.Y.Z' clears when you open a task's detail.

Moves to Relevant details (the old Goal, first):

> 'reloaded hug vX.Y.Z' (and any other main-screen message) is gone once you move to another screen: opening a task's detail starts with an empty message line, or the detail takes over that row. Also find why the zhvkmj detail at 1690x1340 px showed overview text ('All tasks · : filters and commands · ! lint finding · ↳ epic') and ~8 blank rows above DESCRIPTION, and fix it if it is a leak from the overview.

Then the existing Relevant details (483 characters), unchanged.

## 29. wxsyc5 (doing): Stop pushing hug's own memories to the public repo
Now: Goal is 228 characters

Proposed Goal (104 characters):

> No real memories go to a public remote, and Jørgen decides what happens to those already on veridit/hug.

Moves to Relevant details (the old Goal, first):

> No real memories go to a public remote: the installed hug stops adding to veridit/hug's hug/memory branch now, and Jørgen decides what happens to the 868 memories already there (46 commits since 2026-11-17 11:02, head 1abcc497).

Then the existing Relevant details (460 characters), unchanged.

## 30. z77pem (doing): hug names running MCP servers too old to reload themselves
Now: Goal is 257 characters

Proposed Goal (116 characters):

> Any hug run that talks to an agent names hug mcp servers too old to reload themselves, once per session, with a fix.

Moves to Relevant details (the old Goal, first):

> Any hug run that talks to an agent (hook, mcp, mem/task write) notices a hug mcp process on this machine running a binary older than v0.7.54 (no reload) or a removed Cellar path, and says once per session which pids and how to restart them, with a fix: line

Then the existing Relevant details (271 characters), unchanged.

## 31. 33v71x (doing): Image attachments: zoom with + and -, scroll with j/k, o opens Preview
Now: Goal is 314 characters

Proposed Goal (89 characters):

> An attached picture can be zoomed, scrolled and opened in the system viewer from hug tui.

Moves to Relevant details (the old Goal, first):

> A picture attached to a task (PNG/JPEG) can be looked at properly from hug tui: + and - zoom in and out inside the file frame, j/k (and h/l when wider) scroll the zoomed picture, 0 fits it again, and o opens the file in the system viewer (Preview on macOS, xdg-open on Linux, start on Windows) instead of an editor

Then the existing Relevant details (336 characters), unchanged.

## 32. ykta5g (doing): In a task, k after j scrolled past the last check jumps to the top
Now: Goal is 269 characters

Proposed Goal (100 characters):

> In a task's detail, j and k are mirror images: k after j scrolls back up and never jumps to the top.

Moves to Relevant details (the old Goal, first):

> j and k in the detail are mirror images: after j steps through the checks and then scrolls the page past the last one, k first scrolls the page back up one line at a time until the marked check is in view, and only then steps to the previous check; the view never jumps

Then the existing Relevant details (413 characters), unchanged.

## 33. q2adr8 (doing): Ask several questions at once, each answered on its own
Now: Goal is 229 characters in 2 sentences

Proposed Goal (116 characters):

> Several questions on a task are asked and answered each on its own, and the task stays asked until all are answered.

Moves to Relevant details (the old Goal, first):

> An agent with several open questions on a task asks each separately, and the person answers each in its own place (CLI, MCP, hug tui) without one answer covering all of them. The task stays asked until every question is answered.

Then the existing Relevant details (449 characters), unchanged.

## 34. rnpdnm (doing): Memory placement: extractor places personal/other projects, sync gate rule
Now: Goal is 492 characters in 4 sentences

Proposed Goal (111 characters):

> A sharing repo pushes only memories that pass both the extractor's placement and a local, model-free sync gate.

Moves to Relevant details (the old Goal, first):

> A sharing repo pushes only memories both A+ and B pass (gav9nw, Jørgen 19:40Z). A+: the capture extractor places a memory as personal, project:<other> or client:<name> in its existing call (no extra paid call), from the catalog hug mem tidy place builds. B: a local, model-free gate at sync, beside LooksSecret (sync.go), holds any line that names a home path, another known project or client, the person, or is a preference/correction. Its term list lives in local config, never in the repo.

Then the existing Relevant details (348 characters), unchanged.

## 35. yf39ya (doing): Shipped work leaves the board clean, not a pile of lint
Now: Goal is 456 characters in 3 sentences

Proposed Goal (104 characters):

> Work that ships leaves no lint behind: the flow closes, hands over or asks the moment each case happens.

Moves to Relevant details (the old Goal, first):

> Work that ships does not leave lint behind. Today after each release the board shows tasks that are done in fact but still open: checks-all-hold (11 tasks waiting on 'crab review/landing' long after they shipped), shipped-not-done (5 shipped tasks still open on a 'Jørgen confirms after release' check), and stale-owner (17 tasks whose worker session ended). The flow closes, hands over or asks at the moment each happens, so lint shows only real problems.

Then the existing Relevant details (572 characters), unchanged.

## 36. hjbfzd (doing): Screen capture: attach to a chosen task, pick one, or create a new task
Now: Goal is 268 characters in 2 sentences

Proposed Goal (102 characters):

> After S captures the screen in hug tui, the person picks the task it attaches to or creates a new one.

Moves to Relevant details (the old Goal, first):

> After S captures the screen in hug tui, the prompt offers where it goes: type a task ID (completion), s to pick a task from the list, or c to create a new task with the capture attached; Enter attaches, Esc cancels. Today the capture only attaches to the focused task.

Then the existing Relevant details (319 characters), unchanged.

## 37. byj479 (doing): Take tasks that ship together: take B --with A keeps both in doing
Now: Goal is 390 characters

Proposed Goal (108 characters):

> An agent holds two tasks that ship together in doing with hug task take B --with A, and hug guides it there.

Moves to Relevant details (the old Goal, first):

> An agent working on two tasks at once, because they ship in the same change or release, holds both in doing: 'hug task take B --with A' is allowed while A is active, both show in IN PROGRESS together ('with A'), and each closes on its own. hug guides the agent there: the busy refusal offers it as a fix, and ticking a check on a task you do not hold says to take it with the one you're on.

Then the existing Relevant details (460 characters), unchanged.

## 38. ynh84t (doing): hug mem install writes 'muninn hook'; task install names the MCP entry huginn
Now: Goal is 311 characters

Proposed Goal (113 characters):

> Hooks and MCP entries hug writes into harness configs run and name hug, and hug upgrade moves existing ones over.

Moves to Relevant details (the old Goal, first):

> Hooks and MCP entries that hug writes into users' harness configs run and name hug: hug mem install writes 'hug mem hook claude|codex|jcode' (not '<path>/muninn hook ...'), hug task install names its jcode MCP entry hug (not huginn), and hug upgrade --apply --machine moves existing configs over with a preview.

Then the existing Relevant details (622 characters), unchanged.

## 39. nfe0h2 (todo): Long-message recall noise: design a pre-registered measurement
Now: Goal is 224 characters

Proposed Goal (120 characters):

> A pre-registered, blind-judged test on fresh data decides whether a situation filter cuts recall noise on long messages.

Moves to Relevant details (the old Goal, first):

> A pre-registered, blind-judged measurement decides whether a local situation filter cuts irrelevant recall on long (templated, machine-written) messages without losing relevant recall, on a fresh set no earlier test has seen

Then the existing Relevant details (630 characters), unchanged.

## 40. ryqk9z (todo): CI: macOS job cancelled when no hosted runner picks it up
Now: Goal is 274 characters

Proposed Goal (113 characters):

> A ci.yml run fails only for a real test or build failure, never because no hosted macOS runner picked up the job.

Moves to Relevant details (the old Goal, first):

> A ci.yml run fails only for a real test or build failure: today 18 of 118 failed runs in 48 h failed solely because test (macos-latest) was cancelled after 15-22 min with 'The job was not acquired by Runner of type hosted even after multiple attempts', every other job green

Then the existing Relevant details (745 characters), unchanged.

## 41. w4qmr1 (doing): Recall skips jcode image notices: exact prefixes from jcode's source
Now: Goal is 278 characters

Proposed Goal (106 characters):

> Recall injects nothing for jcode's own image notices, matched by exact prefixes taken from jcode's source.

Moves to Relevant details (the old Goal, first):

> Recall injects nothing for jcode's own image notices (screenshot-omitted, attached-image and image-omitted markers sent in the user role), matched by an exact listed prefix taken from jcode's source; every other message, including one that only starts with '[', recalls as today

Then the existing Relevant details (1230 characters), unchanged.

## 42. p87ngv (doing): Inline marks on a task's text; resolved dialogue moves to a history file
Now: Goal is 480 characters in 4 sentences

Proposed Goal (118 characters):

> People and agents mark a task's text inline, and each mark resolves into clearer ticket text, its dialogue kept aside.

Moves to Relevant details (the old Goal, first):

> The ticket stays the crisp working document. A person or agent marks a piece of a task's text with a comment, question or request that sits inline in the text, visible as text, with a recognizable sigil. The agent picks marks up, acts or answers, and resolves each by rewriting the ticket text to the clarified result and removing the mark. The dialogue that led there (the mark, replies, who and when) is appended to a history file attached to the task, never kept in the ticket.

Then the existing Relevant details (4568 characters), unchanged.

## 43. fd0j4p (todo): Agent instructions: questions in tickets, no blocked to quiet a reminder
Now: Goal is 378 characters

Proposed Goal (114 characters):

> Agent instructions say the board is the truth, blocked names who must act, and questions for a person go in tasks.

Moves to Relevant details (the old Goal, first):

> The hug section in AGENTS.md and the help that agents read say: (1) the hug board is the source of truth, harness todo lists do not copy it; (2) a reminder to finish is never a reason to mark something blocked, and blocked names who must do what; (3) questions and discussion for a person go in a task (one ask each, or an anchored thread once p87ngv ships), never only in chat.

Then the existing Relevant details (411 characters), unchanged.

## 44. x8zzzn (doing): Prototype: recall memories delivered into the dialogue by harness notify
Now: Goal is 396 characters in 2 sentences

Proposed Goal (124 characters):

> A per-harness prototype delivers recalled memories into the dialogue by the harness's notify path, measured against today's.

Moves to Relevant details (the old Goal, first):

> A prototype, switchable per harness, delivers recalled memories straight into the agent's dialogue by the harness's own notify path (jcode notify_session, as hug task delivery gmzd87 does), instead of parking them for the first tool call. Measured against the current path; if it works it replaces it on harnesses that support it, with the hook path as the fallback where notify is not available.

Then the existing Relevant details (1535 characters), unchanged.

## 45. mmfw71 (todo): hug tui left running does not pick up new releases
Now: Goal is 145 characters

Proposed Goal (112 characters):

> A hug tui left open across a brew upgrade notices the new binary and reloads, so nobody works on an old version.

Moves to Relevant details (the old Goal, first):

> A hug tui left open across a brew upgrade notices the new binary and offers or does a reload, so a person never works for days on an old version.

Then the existing Relevant details (153 characters), unchanged.

## 46. 75mxf3 (todo): Releases group their tickets and carry the owner's feedback questions
Now: Goal is 458 characters in 4 sentences

Proposed Goal (117 characters):

> Each release is one ticket that groups what it ships and asks the owner, one question each, whether it works for him.

Moves to Relevant details (the old Goal, first):

> A ticket closes when its work ships; its checks hold only what the worker can prove (tests, CI, goldens). Each release is one ticket: the parent of every ticket it contains, linked by hug when the tag is cut, and the place where the owner is asked, one question each, whether the shipped changes work for him. It stays asked until he has answered, then is done. A no becomes a new bug linked to the original ticket and the release; the original stays closed.

Then the existing Relevant details (887 characters), unchanged.

## 47. 5xaxd6 (todo): Flaky on Linux: data race in bubbles textinput sanitizer across TUI tests
Now: Goal is 329 characters

Proposed Goal (106 characters):

> huginn/internal/cli TUI tests pass under -race on Linux without the bubbles textinput sanitizer data race.

Moves to Relevant details (the old Goal, first):

> huginn/internal/cli TUI tests pass under -race on ubuntu: TestTUIHistorySurvivesReopen, TestTUIPaletteNoteCommits, TestEveryWriteEndsWithTheTasksLint and TestEveryRefCommandAcceptsTaskColon no longer race in charmbracelet/bubbles runeutil.(*sanitizer).Sanitize (runeutil.go:65 read vs :93 write) when two TUI programs run at once

Then the existing Relevant details (293 characters), unchanged.
