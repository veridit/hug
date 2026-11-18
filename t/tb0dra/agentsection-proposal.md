# Three additions to the "## Tasks: hug" section

`hug setup` writes this section into every repo's AGENTS.md (agentSection,
huginn/internal/cli/setup.go). Each addition comes from agents' reflections
at `hug task done` (tb0dra), and crab has reviewed them. Each "before" is the
text on main today (v0.7.72), quoted verbatim.

## 1. A later ruling is written on the earlier ticket too

Why: reflections h176dp, t64v7d, bejd89. A check contradicted an earlier
ruling and neither ticket linked the other. A reversed rule stayed pinned by
an old test. A ticket still named a key that a later ticket had changed.
(This repo's own AGENTS.md got the same rule in commit a0939668.)

before:
```
Requests: a person's request on a task (hug task requests) comes first in hug
task next. Turn it into changes to the ticket, then hug task resolve ID N
"what changed"; if it is unclear, hug task ask instead.
```
after:
```
Requests: a person's request on a task (hug task requests) comes first in hug
task next. Turn it into changes to the ticket, then hug task resolve ID N
"what changed"; if it is unclear, hug task ask instead.
A ruling that changes an earlier ticket goes on that ticket too (hug task note
OLD "superseded by NEW: ..."), so show on either one finds the other.
```

## 2. A premise is reproduced first; the principle wins

Why: reflections 95ahtk, 46ek1f, typq1e, x325pp, kx2ndm, scrxak.
- A ticket said "huginn has no file lock today" after one had been added.
- A slowness premise turned out to be a cold start.
- A check ("0 per transcript") pulled against the principle (lose no durable
  memory), and nothing said which wins.

before:
```
Describe a task with Goal (where we aim), Purpose (why), Principle (what we do
and do not do), Relevant details, then Acceptance. Titles stay under 80
characters; detail goes in the Goal.
```
after:
```
Describe a task with Goal (where we aim), Purpose (why), Principle (what we do
and do not do), Relevant details, then Acceptance. Titles stay under 80
characters; detail goes in the Goal. A claim in a task about the code or a
number is a premise: reproduce it before building on it. When a check and the
principle pull apart, the principle wins; note it on the task.
```

## 3. A check only someone else can tick is its own task

Why: reflections rt80x4, 44w1ev, cvbpdy.
- A shipped task stayed open for a day on a check only another repo's owner
  could tick.
- Tasks with every check ticked sat with nobody to close them.

Lint already flags the second case (checks-all-hold). This addition prevents
the first.

before:
```
Two tasks that ship together (one change or release): take the second with the
first, hug task take B --with A --model MODEL; both are doing.
```
after:
```
Two tasks that ship together (one change or release): take the second with the
first, hug task take B --with A --model MODEL; both are doing.
A check only someone else can tick (another repo's owner, a release) is its own
task, linked with hug task dep, so yours is done when your part is.
```
