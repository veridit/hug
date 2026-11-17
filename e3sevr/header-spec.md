# TUI header: all of Jørgen's 2026-11-17 evening rulings in one picture

crab, 00:00 local. Tickets: e3sevr (keys, trail, title line), qh9ep1 (view prefix and counts),
ahejza (filter and sort line), 41hhgn (asked stage). Each later ticket builds on this picture.
Nothing here may contradict another line.

## List views (overview, list, board, lint)

```
HUG TASKS · overview   1/138 tasks · ! 14 broken · 29 advice                       hug v0.7.50
view: overview  list  board  !lint        │ todo 31 · doing 26 · depends 0 · done 79 · dropped 3
filter: none · sort: rank
```

- Line 1: the product, the current view, the task count and the lint counts. Unchanged.
- Line 2 (qh9ep1): `view:` with v and each view letter marked; vo, vl, vb and v! switch views.
  The state counts sit right-aligned after a dim `│`, replacing the old counts line.
  At 80 columns the counts drop from the right first (dropped, then done).
- Line 3 (ahejza): `filter: … · sort: …`, with f and s marked. The `1-4` hint is gone.
- That is one line fewer than today.

## Detail of a task

Jørgen 00:01 local: most of the list header is not relevant inside a task. The only thing that
matters is where u takes him back, as a breadcrumb, at the start of line 1.

```
HUG TASKS · overview › 6ephqk ›                                   ? 16 for you · hug v0.7.50
▸ kx2ndm  TUI: groups and epics visible and collapsible            ← id leads the title (e3sevr)
[doing] [feature] [high]  group:root  assigned:nautilus · claude-opus-5-5/high · main
└─ asked: you · check 9 · look at the rows on your board        ← 41hhgn: no '?' glyph
```

- Line 1 is the breadcrumb: `HUG TASKS ·`, then the view u returns to, then any parent tasks
  on the detail stack, each ending in `›`. The rightmost crumb is where u goes. The word
  `detail` is gone.
- The task count, the lint counts, the view line, the state counts and the filter/sort line
  all leave the detail. They describe lists. Only `? N for you` stays, right-aligned, since it
  is about him, not the list.
- vo, vl, vb and v! still work from a detail. They are listed in help, not drawn.
- The title line follows the breadcrumb: the id in the accent colour, then the bold title.
  With page focus, `▸` sits before the id. The id badge leaves the badge row.
- That frees three lines for the task itself.
- Page focus on the id lists the task-wide keys in the footer (e3sevr check).

## Footer while a prefix is pending (qh9ep1)

```
v: o overview · l list · b board · ! lint · Esc cancel
```

The view line lights up while `v` waits.
