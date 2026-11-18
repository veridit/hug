# Detail view: the stage first (Jørgen 14:47)

The overview row says `waiting: step 4 after hatchling lands; check 9 jhf look`.
Today the detail view drops that and shows `[doing]`, then the description.
The stage, what the task waits on, since when and who asked appear nowhere.

## Proposal: the header line says what the row said, then the question

```
Task state and stage: a hard-coded tree replaces status
doing · waiting: step 4 after hatchling lands; check 9 jhf look            since 14:30 (1 h) · asked by nautilus
feature · 6ephqk · assigned nautilus · claude-opus-5-5 · main · updated 2026-11-17
```

When it waits on you, it says so first and offers the answer:

```
Status colors: rows and sections share one subtle scheme; done shows a count
? doing · waiting on you: look at the colours after v0.7.41                since 14:30 (1 h) · asked by nautilus
  a answer · Enter on a file opens it
```

Then the open checks before the description, because they are what's left:

```
LEFT · 2 of 9 open
  4. hug upgrade turns every status: into state and stage on hug's own board
  9. Jørgen looks at the rows and stages on the hug and Statbus boards
```

DESCRIPTION, PURPOSE, PRINCIPLE and the rest follow below, unchanged.

Also seen: the rows show no `?` because `waiting_on: "jhf: ..."` is free text.
`wait --on jhf` writes the name as a field of its own, but agents wrote `jhf:`
into the text instead. Fix: the TUI user matches a waiting_on that starts with
their name, and lint asks for `--on NAME` where the text starts with a name.
