# ec6z6j check 1: one request for the marked tasks

piglet, 2026-11-18. This is a mock, not built yet. The marks come from
rxr4z8 (macaque is building them): `◆` in the gutter, no tint, and a footer
that becomes an action bar while tasks are marked. ec6z6j adds one action
to that bar, `request`, which sends one request for exactly the marked set.
The request side is already on main (v0.7.68, `hug task request a,b,c`): the
text is stored once on the first task, a link goes on each of the others,
it is delivered once, and each target is resolved on its own.

The screens are the real 80x24 overview from main (d5c66a1f), with four
tasks that each lack a purpose. Only the marks, the footer, the box and the
targets line are drawn by hand.

## 1. Three tasks marked

Space on kfcejs, vrtnq2 and 1zesh9 marks each one and steps down, so the
cursor ends on thszdj, which is not marked. The cursor row is not part of
the request: only the marked rows are.

```
HUG · overview   4/4 tasks · ! 0 broken · 8 advice
view: overview  list  board  !lint       │ todo 4 · doing 0 · depends 0 · done 0
filter: none · sort: rank
  ASKED · for you  [0] · None
  IN PROGRESS · assigned to  [0] · None
  WAITING  [0] · None
> NEXT UP · rank order  [4] · all ready · all feature · 3 marked
◆ ○ 1   ·kfcejs            Docs task 05: make the terminal say why
◆ ○ 1   ·vrtnq2            Spike task 09: make the terminal fit
◆ ○ 1   ·1zesh9            Feature task 01: make the board quiet
> ○ 1   ·thszdj            Bug task 17: lint misses a missing purpose

  RECENTLY DONE · last closed  [0] · None
  LINT · tasks needing attention  [4] · 0 broken · 8 advice · all ready · all f…
◆ ○ 1   ·kfcejs            Docs task 05: make the terminal say why
◆ ○ 1   ·vrtnq2            Spike task 09: make the terminal fit
◆ ○ 1   ·1zesh9            Feature task 01: make the board quiet
  ○ 1   ·thszdj            Bug task 17: lint misses a missing purpose



3 marked · move · x drop · wait · edit · request · Esc unmark

j/k move · Space mark · Enter open · ? help · quit
```

- The footer is rxr4z8's, and ec6z6j adds only `request`, with r marked in
  the word as the other keys are. t is missing because crab ruled at 20:44
  that take works on one task at a time.
- r with nothing marked works as it does today: a request on the cursor
  row's task.

## 2. r: the box names the exact set (proposed, A)

The box opens at the bottom of the list, so the marked rows stay in view
while you write. Its heading counts the set and lists every id.

```
HUG · overview   4/4 tasks · ! 0 broken · 8 advice
view: overview  list  board  !lint       │ todo 4 · doing 0 · depends 0 · done 0
filter: none · sort: rank
  ASKED · for you  [0] · None
  IN PROGRESS · assigned to  [0] · None
  WAITING  [0] · None
> NEXT UP · rank order  [4] · all ready · all feature · 3 marked
◆ ○ 1   ·kfcejs            Docs task 05: make the terminal say why
◆ ○ 1   ·vrtnq2            Spike task 09: make the terminal fit
◆ ○ 1   ·1zesh9            Feature task 01: make the board quiet
> ○ 1   ·thszdj            Bug task 17: lint misses a missing purpose

  RECENTLY DONE · last closed  [0] · None
  LINT · tasks needing attention  [4] · 0 broken · 8 advice · all ready · all f…
◆ ○ 1   ·kfcejs            Docs task 05: make the terminal say why
◆ ○ 1   ·vrtnq2            Spike task 09: make the terminal fit

▸ REQUEST FOR 3 MARKED · kfcejs vrtnq2 1zesh9
  request: Each of these lacks a purpose: write one from the goal._

Enter send · Alt-Enter new line · Esc cancel · F1 help
```

- The box is the same multi-line box r opens today in a task's REQUESTS
  section, with the same keys. Esc cancels, writes nothing and keeps the
  marks.
- More ids than fit end in `…`, but the count is always shown in full:
  `REQUEST FOR 12 MARKED · kfcejs vrtnq2 1zesh9 …`.

### B, the alternative: the box in the first marked task's page

r opens kfcejs's detail, as r on a row does today, and the box carries a
line naming the others:

```
▸ REQUESTS · Enter sends · Alt-Enter a new line · Esc cancels
  for 3 marked: kfcejs vrtnq2 1zesh9
  request: Each of these lacks a purpose: write one from the goal._
```

In B, the other two tasks are off screen while you write.

## 3. After Enter

The marks clear because all three went through, as Jørgen ruled for
rxr4z8 (check 9). Each target shows `req 1`, as one request on one task
does today, and the footer line says where the request went.

```
> NEXT UP · rank order  [4] · all ready · all feature
  ○ 1   ·kfcejs            req 1 Docs task 05: make the terminal say why
  ○ 1   ·vrtnq2            req 1 Spike task 09: make the terminal fit
  ○ 1   ·1zesh9            req 1 Feature task 01: make the board quiet
> ○ 1   ·thszdj            Bug task 17: lint misses a missing purpose
...
request 1 for 3 tasks, on kfcejs: pending, delivered to crab
```

## 4. Pending and handled, in each target's page

After crab has fixed vrtnq2 and nothing else, the home task (kfcejs) shows
the text and every target:

```
REQUESTS
1. jhf · 4 min ago · acknowledged by crab
   Each of these lacks a purpose: write one from the goal.
   for 3: kfcejs pending · vrtnq2 resolved · 1zesh9 pending
```

A target that is not the home (vrtnq2) shows the same text and where it
lives:

```
REQUESTS
1. jhf · 4 min ago · resolved · request 1 on kfcejs, 1 of 3 resolved
   Each of these lacks a purpose: write one from the goal.
   └ crab: added a purpose
```

Today's TUI shows neither line. Measured on main d5c66a1f at 80x24 after
`RequestFor(kfcejs, vrtnq2, thszdj)`:
- kfcejs shows the text with no targets.
- vrtnq2, once resolved, shows only `1. jhf · 0 min ago · resolved` and
  `└ crab: added a purpose`, without the text.

The CLI shows both (`targets: a resolved, b pending`), and check 3 is ticked
for the CLI. Section 4 brings the same view to the board.

## Narrow (under 70 columns)

The footer follows rxr4z8 check 8 (`3 marked · ? actions · Esc unmark`),
and `?` lists `r request for the 3 marked`. The box heading still fits:

```
▸ REQUEST FOR 3 MARKED · kfcejs vrtnq2 1zesh9
```

## The question

Where does the box open when you press r with tasks marked:
**A** at the bottom of the list, with the marked rows in view (proposed),
or **B** in the first marked task's page?
