---
id: e3sevr
title: "TUI keys: H/L move the column marker; J/K page; marker sits on the cursor row"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzzzzyh"
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T23:27"
stage: "asked"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
asked_what: "e3sevr is on main (f85c01f, next release). Please move the column marker with H/L, order with o, and page with J/K, and say whether it reads clearly. Also look at the footers: a key letter is now marked inside its word ('next-, previous file', 'edit · take · done', 'H/L column · order')."
asked_for: "jhf"
asked_since: "2026-11-18T01:39"
asked_by: "nautilus"
asked_check: "4"
---

## Goal
Keys in list, overview and board: j/k up/down, J/K page down/up, h/l left/right (in the overview: previous/next section, the same as Shift-Tab/Tab), H/L move the column marker left/right, o orders by the marked column. Moving a task in rank is m (grab) only; J/K no longer move tasks. The column marker is no longer drawn in the header row: on the cursor row a > stands between the columns, just before the marked column's cell, and the marked column is highlighted. So the cursor shows by the row's > sigil, the in-row marker and the column highlight.

## Purpose
Jørgen 2026-11-17 23:20 local, on the v0.7.48 overview: the header marker '▸✓' over the pie column should move into the cursor row between the columns; 'j/k up down, J/K page up/down, h/l left right, overlapping Tab/Shift-Tab for the overview navigation, H/L for column left right moving the marker. The marker plus the column highlight is the visual clue in addition to the > sigil.' Reverses bejd89's Tab for the column cursor and x1ra0t-era J/K as row moves outside the detail.

## Principle
One key family per axis: lower case moves the cursor, upper case moves it further (page, column). Nothing writes on a cursor key; moving a task in rank stays an explicit grab (m). Help (jmx10s) lists exactly the bound keys; its key test must keep passing.

## Relevant details
Mock, cursor on 3mavpt with the TYPE column marked:
    ●  3    3mavpt >bug      high    done   hug setup takes 34 s on Seid ...
 (the marked column's cells are highlighted down the section; the header row shows plain column names). Code: huginn/internal/tui columns (bejd89), actions.go J/K move (actionNames), tui.go key switch, help sections. Detail keys are unchanged: Tab steps ids, h/l sections (4bwcgp).

## Depends on

## Acceptance
- [x] H/L move the column marker in list, overview and board; Tab/Shift-Tab in the overview change section like l/h; J/K page; m grabs to move (tests)
- [x] the header row has no marker; the cursor row shows > before the marked column's cell and the marked column is highlighted (goldens at 80 and 140)
- [x] help lists the new keys and TestHelpKeysAreBound passes
- [ ] Jørgen moves the marker with H/L, orders with o, and pages with J/K, and says it reads clearly
- [ ] ~~in a task's detail a dim full-width rule stands between the filter line and the task's title (goldens at 80 and 140)~~ (not relevant: replaced: the filter line left the detail; the rule now sits between the ticket header and the details (header-spec.md 00:03), crab, 2026-11-18)
- [x] Tab = l and Shift-Tab = h in every view; in a task's detail both step one focus ring in reading order (stage line, each section, the task ids inside a section as their own stops); Enter opens the focused id (tests; help and TestHelpKeysAreBound follow)
- [x] Column 1 holds only the pointer, grab mark and ↑/↓; '?' and '!' sit right before the id in a fixed 2-cell slot, so a scroll arrow and a '?' never share a cell (golden with a waiting-on-you task as the last shown row)
- [ ] ~~The detail's title line starts with the task id in the id colour, then the bold title; the id is no longer in the header trail or the badge row; page focus ▸ sits before the id on the title line (golden at 80 and 140)~~ (not relevant: superseded by Jørgen 00:02: the id goes in the breadcrumb, not the title line (header-spec.md), crab, 2026-11-18)
- [x] Focus on the task id (the page stop) lists the task-wide keys in the footer (j/k scroll, e edit, t take, d done, x drop, w wait, y yank) and each of them works from there; no focus stop shows an empty footer (key test per listed key)
- [ ] ~~In a detail the header is one line, 'HUG TASKS · <view> › <stacked parents> ›' with '? N for you' on the right; no view line, counts or filter line; the title line with the id follows (golden at 80 and 140, one task open and two stacked)~~ (not relevant: replaced by the next check (id in the breadcrumb, Jørgen 00:02), crab, 2026-11-18)
- [ ] ~~In a detail, line 1 is 'HUG TASKS · <view> › <stacked parents> › <id>' with the id in the accent colour and '? N for you' on the right; then the bold title alone, then badges (no id badge) and dates, then a dim full-width rule, then the details; no view line, counts or filter line (golden at 80 and 140, one task open and two stacked)~~ (not relevant: reordered by Jørgen 00:03: breadcrumb, context, subject, details (next check), crab, 2026-11-18)
- [ ] ~~The detail is drawn in named bands per header-spec.md: breadcrumb 'HUG TASKS · <view> › <parents> › <id>' (id accent, '? N for you' right), context (badges without id, dates), subject (bold title, ▸ on page focus), a dim rule, then details; the global band (view line, counts, filter/sort) is not drawn (golden at 80 and 140, one task open and two stacked)~~ (not relevant: Jørgen 00:04 adds a rule above the subject (next check), crab, 2026-11-18)
- [x] The detail is drawn in named bands per header-spec.md: breadcrumb 'HUG TASKS · <view> › <parents> › <id>' (id accent, '? N for you' right), context (badges without id, dates), a dim rule, subject (bold title, ▸ on page focus), a dim rule, then details; the global band is not drawn (golden at 80 and 140, one task open and two stacked)
- [x] Footers and hint lines mark a letter key inside its action word instead of repeating it ('next file · previous file' with n and p highlighted; 'edit · take · done · drop'); keys that are not a letter of the word keep their label (Enter, Esc, Space, Tab, j/k, ?, /) (golden of the file viewer footer and the overview footer; a test that no footer shows a letter key followed by a word starting with that letter)
- [ ] ~~Neighbouring footer hints that share their last word write it once with a '-' on the earlier ones: the file viewer footer reads 'next- · previous file' with n and p highlighted (golden)~~ (not relevant: Jørgen 00:45:54: comma inside a group (next check), crab, 2026-11-18)
- [x] Neighbouring footer hints that share their last word form one item: the earlier ones end in '-', they are joined by ', ', and ' · ' separates items; the file viewer footer reads 'next-, previous file · …' with n and p highlighted (golden)

## Notes

Jørgen 23:23 local, detail view: a thin horizontal rule between the filter line ('No filters · f picker · 1 todo ...') and the task's title, so the title stands out and the task starts clearly below the headings. Same rule style as the one under the detail header (dim, full width).


Jørgen 23:27 local: Tab/Shift-Tab and h/l must agree. Today they differ in the detail (Tab steps task ids since 4bwcgp, h/l step sections) and must not. Ruling (crab, from his 23:20 key map 'h/l left right, overlapping Tab/Shift-Tab'): everywhere, Tab = l and Shift-Tab = h. In the detail both step through one focus ring in reading order: the header's stage line, each section (checks, deps, files, links, notes), and the task ids inside them as their own stops. No key has a meaning that its partner lacks.
Jørgen's 23:30 screenshot, overview footer: 'h/l pane · … · Tab/o column/order' gives Tab and h/l different jobs, which is the split he asked to remove. Also the empty focused WAITING section draws a column header row under 'None'; show the header only above rows.
Jørgen 23:43 local, screenshot: the '?' (waiting on you) sits in the first gutter column, which ↑/↓ scroll arrows also use, so on the first/last shown row one hides the other. Ruling: column 1 holds only the pointer, grab mark and scroll arrows; the '?' and '!' marks move right before the ticket id:
  before: ?!◐ 3/4  89y0b7  feature high ...
  after:  ↓ ◐ 3/4  ?!89y0b7  feature high ...
The id column widens by 2 so ids stay aligned whether or not a row has marks.
Jørgen 23:58 local: the task id sits in the header's trail ('overview › kx2ndm') and should instead lead the detail's title line, in the id colour (accent), e.g. 'kx2ndm  TUI: groups and epics visible and collapsible'. Ruling (crab): the title line becomes '<id in accent> <title in bold>'; the id leaves the header trail (the trail keeps the view and any parent ids you came through, 'overview › 6ephqk ›' only when stacked); the ▸ page-focus mark moves with the id to the title line; the id badge in the badge row goes too (one fact in one place).
Jørgen 23:59 local, screenshot of 'overview › ▸ kx2ndm' focused: 'this receives focus but I can't do anything with it?' Today that stop is focusPage: j/k scroll the whole page and nothing else, and the footer doesn't say so. Ruling (crab): with the id on the title line (23:58), focusing it means 'the task as a whole'. The footer lists what works there: 'j/k scroll · e edit · t take · d done · x drop · w wait · y yank · R request (hbgybn) · Enter edit'. Every key it lists works, and a key that doesn't apply (t on a doing task) is left off. No focus stop exists that offers nothing.
file:header-spec.md (2485 bytes, sha256:d853c110)
Jørgen 00:01 local, screenshot of the detail header: 'most of this seems not relevant inside a task; the only relevant thing is if I press up, where do I come back? More like a breadcrumb, at the start of HUG TASKS · detail.' header-spec.md updated: in a detail, line 1 is 'HUG TASKS · overview › <parents> ›' (where u goes), with '? N for you' on the right. The counts, view line, state counts and filter line leave the detail. 3 lines freed.
file:header-spec.md (2605 bytes, sha256:6c7fafa8) replaces sha256:d853c110
file:header-spec.md (3566 bytes, sha256:3e513627) replaces sha256:6c7fafa8
Jørgen 00:02 local: the badge/dates block is the ticket header; after the breadcrumb comes the id and title, then that header, then the details. header-spec.md updated: three bands (breadcrumb; id+title, badges, dates; dim rule; details). The doubled wait lines (orange '└ waiting on you…' plus blue '? waiting… a answer') go to 0gztsy's one-place rendering.
file:header-spec.md (3932 bytes, sha256:da65f983) replaces sha256:3e513627
Jørgen 00:02 local, replacing 23:58: 'put the ticket ID in the breadcrumb, meaning the header gets a prominent space.' The id is the last crumb ('HUG TASKS · overview › kx2ndm', id in accent, crumbs before it dim); the title line is the bold title alone, ▸ before it on page focus; no id badge. header-spec.md updated.
file:header-spec.md (4548 bytes, sha256:c0b38735) replaces sha256:da65f983
Jørgen 00:03 local: 'clear ideas: a global header, and it goes away, then the breadcrumb, then the context of the ticket, then the subject, then the details.' header-spec.md rewritten around six named bands (global, breadcrumb, context, subject, details, footer), with a table of which show in lists vs a task. In a detail: breadcrumb with the id last, context (badges, dates), subject (bold title), dim rule, details. Code and goldens use the band names.
file:header-spec.md (4894 bytes, sha256:cda72b68) replaces sha256:c0b38735
Jørgen 00:04 local: keeps context before subject, and wants 'a line right above the subject so it stands out'. Spec: breadcrumb, context, dim rule, subject, dim rule, details.
file:header-spec.md (5528 bytes, sha256:daa22937) replaces sha256:cda72b68
file:header-spec.md (5952 bytes, sha256:32713c2e) replaces sha256:daa22937
Jørgen 00:44 local, on the file viewer footer 'n next file · p previous file': 'it should be *n*ext etc., not a repeat but a highlight.' Rule for every footer and hint line: a key that is a letter of its action word is shown as that letter highlighted inside the word ('next file · previous file', with n and p in the key colour, underlined), the same way the view line marks 'view: overview list board'. A key that is not in the word keeps its own label: Enter, Esc, Space, Tab, j/k, arrows, '?', '/'. If the letter isn't in the word, the label stays too ('y yank' stays only if y isn't marked, it is: 'yank').
Jørgen 00:45 local: 'next- · previous file' (n and p highlighted). Adjacent hints that share their last word write it once, on the last of them; the earlier ones end in '-' (suspended hyphen, like 'pre- and post-war'). E.g. 'next- · previous file', 'previous- · next task'. Only for neighbours in the same footer with the same last word.
Jørgen 00:45:54 local: hints that share a word are joined with ', ' not ' · ': 'next-, previous file'. The ' · ' stays between unrelated hints, so a group reads as one item: 'next-, previous file · edit · Esc back'.
Landed on main 4b8681e0..f85c01f2 (12 commits, 385f774..f85c01f). Evidence per check:
1: TestColumnCursorMarksHeader (H/L move the marker in list/overview/board), design_test TestPaneKeysAndBoardSelection (Tab = l, Shift-Tab = h change section), detailkeys TestRowJKPageNeverMove (J/K page, m grabs), move/sort/grab tests move with 'm j enter'.
2: TestColumnCursorMarksHeader + testdata/columns/cursor.txt at 80 and 140 (header has no marker; '>' before the marked cell; column highlighted).
3: helpkeys TestHelpKeysAreBound, TestHelpFitsAndGolden (testdata/help/contexts.txt, tmux help goldens).
6: idstops TestTabCyclesEveryIDInReadingOrder (ring: parent, stage id, page, checks, deps, links, note ids; Tab/l forward, Shift-Tab/h back, Enter opens), focus_test TestFocusTabCyclesAndHighlightsOneHeading.
7: TestGutterHoldsOnlyPointerAndArrows + testdata/rows/gutter.txt (80 and 140, the question for jhf as WAITING's last shown row under ↓; ! in its own cell before the id; no ? in rows). Fails on 62a68c1: 'at 80 the gutter holds ?'.
9: TestPageFocusListsTaskKeys (doing and todo: footer lists edit/take/done/wait/x drop/yank and each acts; w parks it waiting; no ring stop has an empty footer). Fails on 62a68c1.
13: TestDetailBandsInOrder + testdata/detail/bands.txt (one task and two stacked at 80 and 140: breadcrumb, context, rule, subject, rule, details; no global band). Fails on 62a68c1 (breadcrumb 'HUG TASKS · detail …').
14: TestFooterMarksTheKeyInItsWord (every help context at 80 and 140: no footer/key bar repeats a letter key before its word; no one-letter label beside words holding its letter; mark is accent+underline). Fails on 270a519 ('overview key bar repeats the key "q"'). Rulings crab 00:10: upper case marks upper case (New, Screen, Paste above, Edit files); combined hints split (H/L column · order, find · # id); x drop keeps its label.
16: TestFooterGroupsHintsSharingAWord (grouped 'next-, previous file', apart stays apart, narrow drops whole items) + testdata/detail/glance.txt. Fails on 270a519.
Gates: CI 44207423678 at b368f92 success on all 5 (ubuntu -race + tmux, macos, windows, bench, secrets); after the rebase onto 4b8681e local go test ./huginn/... all ok, gofmt/vet clean; earlier at cbacbef local -race tui race-exit 0 and go test ./... all-exit 0. Check 4 is Jørgen's. (nautilus)
landed 385f774 on main 2026-11-18: hug tui: one focus ring (h/l Tab), H/L mark a column, J/K page, the breadcrumb…
landed 30c0db7 on main 2026-11-18: gofmt (e3sevr)
landed 8f7c529 on main 2026-11-18: hug tui: tests for the gutter, page-stop keys, detail bands, an empty section's…
landed df7c424 on main 2026-11-18: hug tui: the empty-section test marks a column and matches the header line…
landed 87dc80a on main 2026-11-18: hug tui tmux tests wait for the breadcrumb, not 'HUG TASKS · detail' (e3sevr)
landed fa07466 on main 2026-11-18: hug tui: column hint drops '· h/l section'; the bands test pins both clocks…
landed be165b3 on main 2026-11-18: hug tui: footers mark a letter key inside its word; neighbours sharing a last…
landed be784f6 on main 2026-11-18: hug tui: split combined hints so each key marks its own word (H/L column ·…
landed f85c01f on main 2026-11-18: hug tui: tmux goldens for the marked footers (e3sevr checks 14, 16)
shipped in v0.7.52 (385f774)
file:header-spec.md (7778 bytes, sha256:b44592e4) replaces sha256:32713c2e zhvkmj: 'Colours: one meaning each' table (crab 02:52, 03:01)
## History
- 2026-11-17T23:27 todo -> doing/active (take)
- 2026-11-18T01:39 doing/active -> doing/asked (ask)
