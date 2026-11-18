# Rows that wait on someone: two lines (Jørgen 14:55)

Today (one line, the subject is gone, the colon floats, "jhf" is typed text):

```
○ 3    bxqtkt  spike     waiting:  jhf: approve design tmp/lanes/bxqtkt.md (check 1) …   kestrel
```

Proposal: line 1 is the task as always, line 2 hangs under it and says what it waits for.

```
? ○ 3    bxqtkt  spike    Contributions: agent feedback and GitHub intake            kestrel
         └─ waiting on you: approve the design  ▸ bxqtkt.md                      since 14:42

  ○ 3/5  44w1ev  chore    Scale: stay fast and safe at 10,000 tasks                  wyvern
         └─ waiting on landing turn: 5bd9cb5 with 49n9kw                         since 14:30

? ◔ 3/4  zhvkmj  feature  Status colors: rows and sections share one scheme          nautilus
         └─ waiting on you: look at the colours  ▸ screen-1452.ans          since 14:30
```

- `└─` ties the reason to its task; the reason starts right after the colon.
- `waiting on you` when it is you, `waiting on CI`, `waiting on wyvern`, never initials.
- `▸ name` is an attachment: Tab lands on it in the detail, Space shows it, Enter edits it.
- Rows that are active or ready stay one line.

How hug knows it is you: today the TUI's "you" is --as / HUG_USER / the agent session /
the model, and HUG_USER is unset on your machine, so nothing matches and no `?` shows.
Proposal: `hug.toml` lists the people (`people = ["jhf"]`), `hug setup` asks once and
sets HUG_USER, and `--on NAME` must be a listed person or agent (lint flags free text).
