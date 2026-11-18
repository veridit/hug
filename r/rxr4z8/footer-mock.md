# rxr4z8 footer mock: Space marks, the footer acts on the marked

nautilus, 2026-11-18 05:20, drawn from the current main (52ff46f9) screen
with the 60-task fixture. Not built yet: this is the picture to agree on
first (the ticket says: mock before building).

## 80 columns, overview, 3 marked in NEXT UP (one of them an epic)

Space on 000001 marked it and stepped to 000005; Space there, Space on
000009, then k back to 000005. The cursor row keeps `>`; a marked row has
`◆` in the gutter (dim accent, never red) and a quiet tint across it.

```
HUG TASKS · overview   60/60 tasks · ! 2 broken · 5 advice
view: overview  list  board  !lint   │ todo 8 · doing 15 (1 depends) · depends 7
filter: none · sort: rank
  ASKED · for you  [0] · None
  IN PROGRESS · assigned to  [14] · 1–1 shown · all active
↓ ◑ 1/2  000002   bug     critical compiler Bug task 02: make…  alice · feat/te…

  WAITING  [1]
  ◑ 1/2 !000026   feature high     depends   docs Feature task 2…  carol · feat…

> NEXT UP · rank order  [15] · 1–4 shown · 3 marked
◆ ○ 2   ·000001   feature critical ready     compiler ⎘2 Feature task 01: make …
> ○ 2    000005   docs    critical depends   runtime Docs task 05: make the ter…
◆ ○ 2    000009   spike   high     ready     docs Spike task 09: make the termi…
◆ ◑ 1/2  a1b2c3   feature high     ready     Launch (epic)

  RECENTLY DONE · last closed  [30] · 1–1 shown
↓        000003   feature medium   done      compiler Render compiler-lang task…

3 marked · move · take · x drop · wait · edit · Esc unmark
j/k move · Space mark · Enter open · ? help · quit
```

- The footer line (where `All tasks · : filters and commands` sits today)
  becomes the action bar while anything is marked. Each key is marked in
  its word, as every footer now does (e3sevr): m in move, t in take, x
  before drop, w in wait, e in edit.
- `d done` is not offered: the epic a1b2c3 finishes through its children,
  so done is not valid for all three. Only actions valid for every marked
  task show.
- The key bar gains `Space mark` in lists.
- The section heading says `· 3 marked`, so a mark scrolled out of view is
  not forgotten.

## The confirm, after t

```
Take 3 tasks: 000001, 000009, a1b2c3?  Enter confirm · Esc cancel
```

One confirm names the count and the ids (cut with `…` when long).

## After Enter, one refused

```
took 2 of 3 · a1b2c3 refused: taken by bob
fix: hug task next
```

The refusal is the one `hug task take` gives today (store.go, Kind
"taken"), word for word. The refused task keeps its mark, so a second
try or Esc is one key; the others are unmarked once done.

## 140 columns

The same lines, wider; the footer has room for every word:

```
3 marked · move · take · x drop · wait · edit · Esc unmark
```

## Narrow (under 70)

```
3 marked · m t x w e · Esc
```

## m with marks

m grabs the marked tasks as one block (3xcesq): they gather at the
cursor's place in their order, j/k and < > move the block, Enter drops it
once (one commit), Esc puts them back. Marks can span sections only for
t/x/w/e; m needs them in one section and says so otherwise:

```
m moves marks within one section: 2 in NEXT UP, 1 in WAITING · Esc unmark
```

## Questions for Jørgen

1. `◆` in the gutter, or another sign (`•`, `+`, `✓`)?
2. After an action, unmark the tasks that went through (as drawn) or keep
   every mark?
3. Space on a heading still folds (kx2ndm); Space on a task marks. OK?
