# When AI needs a human: the task says so, and you answer in it

Jørgen 2026-11-17 11:39: AI drives most work; sometimes a task waits on him.
He should see that at once, and the task should hold the question and
everything needed to answer it.

## Three stages already in 6ephqk carry exactly this

| stage           | field             | means                                   |
|-----------------|-------------------|-----------------------------------------|
| todo/design     | design_question   | a decision is needed before work starts |
| doing/review    | review_by         | work is done, someone must look at it   |
| doing/waiting   | waiting_on        | the agent waits on something outside    |

When the field names a person (`waiting_on: jhf`, `review_by: jhf`, or a
design question with no agent able to rule), the task is **yours**.

## Revision (Jørgen 11:42): a stage, not a section

No new section. A task waiting on you stays in its own section (IN PROGRESS or
NEXT UP), its stage names you, it is marked `?` and sorts to the top of that
section. The header counts them and one key filters to them.

```
HUG TASKS · overview   631 tasks · ! 76 findings · ? 2 for you
  IN PROGRESS · assigned to  [11]
? ◔ 3/4  zhvkmj  high    review: you        status colors, approve the look              nautilus
? ◑ 3/5  44w1ev  medium  waiting: you       accept check-3 design (walk every file)?     wyvern
  ◔ 1/6  sdxbdr  high    active             defer-checkout-procure: move the upgrade...  architect
  ○ 0/7  ezp3h0  high    active             Time frames: one live frame, frozen frames...
```

## (superseded) Overview: a section on top, only when something waits on you

```
  NEEDS YOU · 3
? ◔ 3/4  tzc9hb  high    review:   hug setup diff in seid-lang, approve or say what to change    crab
? ○ 0/5  txjkrm          design:   should type leave the rows, or stay as a column?              crab
? ◑ 3/5  44w1ev  medium  waiting:  accept check-3 design (walk every file, ~45 ms of 200 ms)?   wyvern

  IN PROGRESS · assigned to  [9]
...
```

The colon rule from 10:56 gives the question instead of the title. The last
column is who asked. The section is absent when nothing waits on you.

## Detail: the question at the top, then what you need to answer

```
txjkrm  Tags: listed per board in hug.toml, shown on rows
todo · design: should type leave the rows, or stay as a column?        asked by crab, 11:18

  Context   Type shows `feature` on 29 of 29 rows in Statbus; dropping it gives
            titles 8 cells. Bugs then show only via type:bug.
  Options   1 drop type from rows   2 keep it   3 show it only when not feature
  See       file:compact-overview.md

  a answer · 1/2/3 pick · Enter open the file
```

`a` (or a number) writes your answer as a note with your name, clears the
question, and the task re-derives its stage (ready, or back to active). The
agent that asked is woken by the change (5dxzxe, our own harness spike, until
then: the coordinator watches the board).

## What the agent must write when it asks

`hug task design ID "question" --context "..." --option "..." --option "..."`
(and the same for wait/review): lint refuses a question without context, so
you never get a bare "approve?".
