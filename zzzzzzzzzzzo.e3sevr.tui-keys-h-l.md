---
id: e3sevr
title: "TUI keys: H/L move the column marker; J/K page; marker sits on the cursor row"
state: "doing"
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
doing_since: "2026-11-17T23:27"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
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
- [ ] H/L move the column marker in list, overview and board; Tab/Shift-Tab in the overview change section like l/h; J/K page; m grabs to move (tests)
- [ ] the header row has no marker; the cursor row shows > before the marked column's cell and the marked column is highlighted (goldens at 80 and 140)
- [ ] help lists the new keys and TestHelpKeysAreBound passes
- [ ] Jørgen moves the marker with H/L, orders with o, and pages with J/K, and says it reads clearly
- [ ] ~~in a task's detail a dim full-width rule stands between the filter line and the task's title (goldens at 80 and 140)~~ (not relevant: replaced: the filter line left the detail; the rule now sits between the ticket header and the details (header-spec.md 00:03), crab, 2026-11-18)
- [ ] Tab = l and Shift-Tab = h in every view; in a task's detail both step one focus ring in reading order (stage line, each section, the task ids inside a section as their own stops); Enter opens the focused id (tests; help and TestHelpKeysAreBound follow)
- [ ] Column 1 holds only the pointer, grab mark and ↑/↓; '?' and '!' sit right before the id in a fixed 2-cell slot, so a scroll arrow and a '?' never share a cell (golden with a waiting-on-you task as the last shown row)
- [ ] ~~The detail's title line starts with the task id in the id colour, then the bold title; the id is no longer in the header trail or the badge row; page focus ▸ sits before the id on the title line (golden at 80 and 140)~~ (not relevant: superseded by Jørgen 00:02: the id goes in the breadcrumb, not the title line (header-spec.md), crab, 2026-11-18)
- [ ] Focus on the task id (the page stop) lists the task-wide keys in the footer (j/k scroll, e edit, t take, d done, x drop, w wait, y yank) and each of them works from there; no focus stop shows an empty footer (key test per listed key)
- [ ] ~~In a detail the header is one line, 'HUG TASKS · <view> › <stacked parents> ›' with '? N for you' on the right; no view line, counts or filter line; the title line with the id follows (golden at 80 and 140, one task open and two stacked)~~ (not relevant: replaced by the next check (id in the breadcrumb, Jørgen 00:02), crab, 2026-11-18)
- [ ] In a detail, line 1 is 'HUG TASKS · <view> › <stacked parents> › <id>' with the id in the accent colour and '? N for you' on the right; then the bold title alone, then badges (no id badge) and dates, then a dim full-width rule, then the details; no view line, counts or filter line (golden at 80 and 140, one task open and two stacked)

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
## History
- 2026-11-17T23:27 todo -> doing/active (take)
