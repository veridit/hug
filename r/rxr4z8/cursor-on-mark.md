# rxr4z8 check 7: the sign for a marked row under the cursor

macaque, 2026-11-18. In the list, `◆` marks a row and `>` is the cursor.
When the cursor is on a marked row, the row needs one sign that says both.
Below are the same four rows drawn twice: once with `⋗` and once with `▸`.
In each one the cursor is on 000005, which is marked.

## A: ⋗ (greater-than with a dot: the cursor carrying a mark)

```
> NEXT UP · rank order  [15] · 1–4 shown · 3 marked
◆ ○ 2    000001   feature critical ready     compiler Feature task 01: make …
⋗ ○ 2    000005   docs    critical depends   runtime Docs task 05: make the ter…
◆ ○ 2    000009   spike   high     ready     docs Spike task 09: make the termi…
  ◑ 1/2  a1b2c3   feature high     ready     Launch (epic)
```

## B: ▸ (a filled pointer)

```
> NEXT UP · rank order  [15] · 1–4 shown · 3 marked
◆ ○ 2    000001   feature critical ready     compiler Feature task 01: make …
▸ ○ 2    000005   docs    critical depends   runtime Docs task 05: make the ter…
◆ ○ 2    000009   spike   high     ready     docs Spike task 09: make the termi…
  ◑ 1/2  a1b2c3   feature high     ready     Launch (epic)
```

For comparison, the cursor on a row that is not marked keeps `>`:

```
◆ ○ 2    000001   feature critical ready     compiler Feature task 01: make …
> ○ 2    000005   docs    critical depends   runtime Docs task 05: make the ter…
```
