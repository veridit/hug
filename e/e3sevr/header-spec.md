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
──────────────────────────────────────────────────────────────────────────────────────────
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
- Two dim full-width rules frame the subject (Jørgen 00:04: "a line right above the subject so
  the subject stands out a little bit more"): one between context and subject, one between
  subject and details. Everything under the second rule is details and scrolls.
- The asked or waiting question is not in the header. It shows once, on the check it is about
  (0gztsy), or as one block at the top of the details when it names no check.

Order (Jørgen 00:04, "yes"): context, then subject, as he first said it.

## Overview sections (41hhgn, 1bbe96)

Jørgen 00:12: asked is "a separate section that I really need to address"; waiting is "just a
place to live for things that aren't active right now".

```
ASKED · for you [16]                         ← top: what needs his answer, oldest first
IN PROGRESS · assigned to [13] · all active
WAITING [2] · all waiting                    ← CI, releases, questions asked of others
NEXT UP · rank order [28]
RECENTLY DONE · last closed [82]
LINT · tasks needing attention [30]
```

A column that is the same on every task of a section is not drawn there; the heading says it
once (1bbe96).

ASKED rows show the title, like every row. Only the focused row adds the question on a second
line (Jørgen 00:13: the question needs the title as its context):

```
> ◐ 8/9  6ephqk  feature        Task state and stage: a hard-coded tree replaces status
         └ asked: check 9 · look at the state/stage rows on your board · nautilus · 3 h
  ◔ 4/5  3jtfw9  feature  high  Answer a ticket's question in place
```

## Footer while a prefix is pending (qh9ep1)

```
v: o overview · l list · b board · ! lint · Esc cancel
```

The view line lights up while `v` waits.

## Colours: one meaning each (zhvkmj)

crab 02:52 and 03:01 local, from Jørgen 02:32 ("it bugs me that waiting and lint are two
different issues" in one colour). Each meaning has one colour, and no two meanings share one.
TestOneColourPerMeaning (huginn/internal/tui) holds the palette to this table: each type and
priority sits at least 60 RGB away from amber and red, and the five types stay apart from each
other. The colours live in render.go (forYouColour, problemColour, typeColours,
priorityColours), dark / light.

| Meaning | Colour (dark / light) | Where |
|---|---|---|
| Waits on you | amber #ffb454 / #995200 | ASKED title, `? N for you`, a task's `[asked]` badge, the question block and line, the stage word of a question to you, the confirm bar (background) |
| Something is wrong | red #ff8585 / #ae2525 | broken lint: its `!`, the LINT title, the detail's `⚠` and `fix:` lines; depends; a picture that cannot be read |
| Calm, inactive | dim #8795a8 / #586474 | WAITING title, a wait on a thing (CI, a date), a question asked of someone else, advice-only lint (`·` and its detail lines), deferred |
| State | todo #75bfff / #215da1, doing #00bfff / #006c9c, done #8ac987 / #31732c, dropped dim | the count line, the status word, IN PROGRESS / NEXT UP / RECENTLY DONE titles, board columns |
| Type | feature #bc9dff / #7143a8, bug #ff9ecb / #a8327f, chore #b8bccb / #586474, spike #c4b48a / #6b5a2a, docs #7bd4c2 / #146f61 | the type cell and badge |
| Priority | critical #ff7ad9 / #a3246f, high #d8cc85 / #5f6b1a, medium #a7b4c4 / #586474, low #8795a8 / #586474 | the priority cell and badge |

Priority is no problem: critical is magenta, never red. A yanked row's `y` is the accent blue,
not amber. On a light terminal chore, medium and low share the dim grey: quiet values, no meaning
of their own.
