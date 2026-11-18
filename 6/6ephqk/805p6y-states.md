# Task state and stage (805p6y draft, for Jørgen's review)

One tree, compiled into hug (huginn/internal/task/states.go) and printed by
`hug task states`. Lint checks every task against it. Nothing else defines states.

## The tree

Each node declares its own extra fields. A field is named after the node it
belongs to: `<state>_<name>` or `<stage>_<name>`.

    todo                          section NEXT UP
      draft                       set by hug   (Goal or Acceptance missing)
      design                      set by agent
        design_question  (required)   line: the question
        design_since     (by hug)
      depends                     set by hug   (a dependency not done; line: "depends on 16dxsr (doing)"; nobody assigned)
      ready                       set by hug   (complete, dependencies done; no line)
      review                      set by agent (work done, waits for a reviewer; nobody reviewing yet)
        review_by        (optional)   line: "review by Jørgen"
    doing                         section IN PROGRESS
      doing_since      (by hug)
      active                      set by agent (line: first unticked check)
      waiting                     set by agent
        waiting_on       (required)   line: "CI run 44057678727 · 12 min"
        waiting_since    (by hug)
      depends                     set by hug   (taken, but a dependency is not done; line: "depends on 16dxsr (doing)")
      review                      set by reviewer on take (someone is reviewing it now)
        review_by        (required)   line: "Jørgen reviewing · 5 min"
    done                          section RECENTLY DONE (lint: every check ticked, dependencies done)
      done_at          (by hug)
      done_release     (by hug, when tagged)
    dropped                       hidden
      dropped_reason   (required)
      dropped_at       (by hug)

## What each node means (one question per level)

States answer: does anyone have it?

| state | means |
|---|---|
| todo | nobody is working on it |
| doing | someone has it |
| done | finished: every check ticked |
| dropped | decided: never |

todo stages answer: why is nobody working on it?

| stage | means | set by |
|---|---|---|
| deferred | someone decided: not now (it may be complete or not) | person, `deferred_reason`, optional `deferred_until` |
| depends | another task must be done first | hug |
| design | a question must be answered first | agent or person, `design_question` |
| draft | not described well enough to start (Goal or Acceptance missing) | hug |
| review | the work is done and waits for someone to check it | agent |
| ready | nothing stands in the way; anyone may take it | hug |

When several hold, the first in this order wins: deferred, depends, design, draft, review, ready.
A decision beats a fact; a fact about other tasks beats a fact about this one.
The others still show in the detail view.

doing stages answer: what is happening with it now?

| stage | means | set by |
|---|---|---|
| active | the owner is working on it | agent |
| waiting | the owner waits on something outside the board (CI, a reply, a release) | agent, `waiting_on` |
| depends | the owner can't go on until another task is done | hug |
| review | someone is reviewing it | reviewer on take, `review_by` |

deferred vs dropped: deferred is "yes, not now"; dropped is "no".
waiting vs depends: waiting is outside the board; depends is another task.

## In the file

    state: "doing"
    stage: "waiting"
    doing_since: 2026-11-17T08:46
    waiting_on: "CI run 44057678727"
    waiting_since: 2026-11-17T09:41

## The rules lint checks

1. `state` is a root of the tree; `stage` is a child of that root (or empty where the root has none). The same stage name may sit under two states (depends and review under todo and doing); its meaning is the same, its section follows the state.
2. (Restated 2026-11-17 11:37, crab, Proposal A, following Jørgen's precedence ruling.) A node's fields exist only while its condition holds; the stage is the first node, in order, whose condition holds. The stage is therefore derived from the fields and the board alone, and lint can check it. todo: deferred (deferred_reason), depends (a dependency not done), design (design_question), draft (Goal or Acceptance missing), review (review_since or review_by), else ready. doing: waiting (waiting_on), depends (a dependency not done), review (review_by), else active. An outranked node keeps its fields (design_question stays on a todo task while depends wins). A field of another state is never allowed (doing_since on a todo task). Leaving a state, or clearing a condition (resume, design --clear), moves the fields to `## History`. hug rewrites the stage on every write, and when a task's state changes it rewrites the stage of the tasks that depend on it directly. That is one level, since a stage reads only its dependencies' state.
3. Required fields of the current node are present and non-empty.
4. Stages set by hug (draft, depends, ready) match their condition; hug rewrites them on every write.
5. Data hug can derive (dependencies, checks) is never copied into a field.

## Transitions (the only ones allowed)

    todo/*          -> doing/active    hug task take
    doing/active   <-> doing/waiting   hug task wait ID "on what" / hug task resume ID
    todo/*          -> todo/design     hug task design ID "question"
    doing/*         -> done            hug task done
    any             -> dropped         hug task drop ID "reason"

`hug upgrade` turns every `status:` into state/stage.

## In the TUI

- `t` (take) already exists and asks before taking. With stages it takes the task into the doing stage that matches where it is: todo/review becomes doing/review with review_by = you; todo/ready, design or depends becomes doing/active. The row moves to IN PROGRESS with your name at once.
- The user's name is the TUI user (`Model: human` in the commit), never an agent's.
- `t` on a task you already have offers to give it back to todo, keeping its stage (doing/review becomes todo/review).

## Open for Jørgen
(none)

## Decided
- depends is a stage of both todo (nobody has it) and doing (someone has it); the word matches ## Depends on, and the BLOCKED section becomes DEPENDS (Jørgen 2026-11-17 10:12, 10:14).
- review is a stage of both todo (waits for a reviewer) and doing (someone is reviewing) (Jørgen 2026-11-17 10:13).
