# TUI screen bands: named once, used everywhere

crab, 2026-11-18 00:04 local, from Jørgen's rulings 2026-11-17 23:35 to 00:03. Tickets: e3sevr
(keys, detail bands), qh9ep1 (view prefix and counts), ahejza (filter and sort), 41hhgn (asked
stage), 0gztsy (the question on its check). Each builds on this file. Where a ticket and this file
differ, ask crab.

Jørgen 00:03: "we have to have some clear ideas: a global header, and it goes away, and then you
have the breadcrumb, then the context of the ticket, then the subject, and then the details."

## The bands, top to bottom

Every screen is built from these named bands. Code and goldens use these names.

| Band | What it is | List views | Task detail |
|---|---|---|---|
| **global** | the whole board: view tabs, counts, filter and sort | shown | **gone** |
| **breadcrumb** | where you are, and where u takes you back | `HUG TASKS · overview` | `HUG TASKS · overview › 6ephqk › kx2ndm` |
| **context** | facts about the ticket: state, type, priority, group, owner, model, branch, dates | (in rows) | shown |
| **subject** | what the ticket is: its title | (in rows) | shown, bold |
| **details** | the body: goal, acceptance, files, notes; scrolls | the rows | shown, under a dim rule |
| **footer** | the keys that work at the focus, or the message | shown | shown |

## List views (overview, list, board, lint)

```
HUG TASKS · overview                      1/138 tasks · ! 14 broken · 29 advice · ? 15 for you
view: overview  list  board  !lint        │ todo 31 · doing 26 · depends 0 · done 79 · dropped 3
filter: none · sort: rank
──────────────────────────────────────────────────────────────────────────────────────────────
  IN PROGRESS …                                                                  ← details (rows)
```

- breadcrumb (line 1): `HUG TASKS ·` and the view, with the board's counts right-aligned.
- global (lines 2-3):
  - Line 2 (qh9ep1): `view:` with v and each view letter marked; vo, vl, vb and v! switch
    views. The state counts sit right-aligned after a dim `│`. At 80 columns they drop from the
    right first.
  - Line 3 (ahejza): `filter: … · sort: …` with f and s marked. No `1-4` hint.

## Task detail

```
HUG TASKS · overview › 6ephqk › kx2ndm                                    ? 15 for you    ← breadcrumb
[doing] [feature] [high]  group:root  assigned:nautilus · claude-opus-5-5/high · main     ← context
created 2026-11-16 · updated 2026-11-16 · tui-groups-epics-visible                        ← context
▸ TUI: groups and epics visible and collapsible                                           ← subject
──────────────────────────────────────────────────────────────────────────────────────────────
GOAL …                                                                                    ← details
ACCEPTANCE …
```

- **global** goes away inside a task. vo, vl, vb and v! still work there, listed in help, not
  drawn.
- **breadcrumb**: the view u returns to, the parent tasks on the detail stack, then this task's
  id in the accent colour as the last crumb. The crumbs before it are dim. `? N for you` sits on
  the right, because it is about him, not the list.
- **context**: the badge row (state, type, priority, group, assignee, model, branch; no id
  badge) and the dates line, dim except the badges' own colours.
- **subject**: the title, bold, full width, right above the details, so it reads as the heading
  of what follows. `▸` before it marks page focus, and the footer then lists the task-wide keys.
- A dim full-width rule closes the header. Everything under it is details and scrolls.
- The asked or waiting question is not in the header. It shows once, on the check it is about
  (0gztsy), or as one block at the top of the details when it names no check.

Order question for Jørgen: context before subject (as he said it), or subject before context
(a title first, then its facts)? This file follows his words: context, then subject.

## Footer while a prefix is pending (qh9ep1)

```
v: o overview · l list · b board · ! lint · Esc cancel
```

The view line lights up while `v` waits.
