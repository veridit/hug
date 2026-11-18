# Grid mockup: the hug TUI lists as a sectioned spreadsheet

Prototype only (lane `grid mockups`, coordinator crab). Branch `mock/grid` in the worktree
`/Users/jhf/.jcode/scratch/grid-mock`, based on origin/main `4e776695` (v0.7.61). **Do not merge.**

## Verdict

All five design points are built and captured on a real board: a copy of hug's own
`.hug/tasks` with 173 tasks, read-only, at `/Users/jhf/.jcode/scratch/board-repo`.

- The captures use the real `hug` binary in tmux.
- Sizes: 100x30 and 160x45.
- Each view (overview, list, board) has the cursor on a row mid-section and a non-default column selected (PRIORITY).
- The view line is captured before and after `v`.
- Each screen exists as `.txt` (plain), `.ansi` (`tmux capture-pane -e`) and `.png`.

**Start here:** the side-by-side sheets `out/compare-*.png` (BEFORE | order A | order B) and `out/compare-viewline.png`.

## Before/after pairs (sheets)

All paths are under `/Users/jhf/.jcode/scratch/grid-mock/out/`.

| Sheet | Shows |
|---|---|
| `compare-overview-cursor-100x30.png` | Overview. NEXT UP is focused, cursor on 3rd row `66ex4y`, PRIORITY column selected. Before / A / B side by side |
| `compare-overview-cursor-160x45.png` | Same at 160x45 (stacked vertically), cursor on `66ex4y` mid NEXT UP |
| `compare-list-cursor-100x30.png` | List, cursor mid-screen (row 20 of 30), PRIORITY selected |
| `compare-list-cursor-160x45.png` | Same at 160x45 |
| `compare-board-cursor-100x30.png` | Board, DOING focused, cursor on `47sxn2` (7th row), P column selected. The selected row's wait note shows under it |
| `compare-board-cursor-160x45.png` | Same at 160x45 (4 status columns visible) |
| `compare-viewline.png` | View line, 8 states top to bottom (listed in the next table) |

The 8 states in `compare-viewline.png`:

| # | State |
|---|---|
| 1 | Before, idle |
| 2 | Before, `v` pressed (the whole line recolours) |
| 3 | Before, after `v o` |
| 4 | After, idle |
| 5 | After, `v` pressed, blink phase 1 |
| 6 | After, `v` pressed, blink phase 2 (+530 ms) |
| 7 | After, `v o` chosen (the cursor is gone) |
| 8 | Alternative "letters" variant |

## Individual captures

Directory layout: `out/<variant>/<size>/<name>.{txt,ansi,png}`.

- Variants: `before` (main), `after-A`, `after-B`, `after-A-letters` (view-line alternative).
- Sizes: `100x30`, `160x45`.

| Name | What it shows |
|---|---|
| `overview-initial` | The first screen after launch. The view line is idle |
| `viewline-pending` | Right after pressing `v` on the overview |
| `viewline-pending-blink2` (`.ansi`, `.png`) | The same, 530 ms later: the other blink phase |
| `overview-cursor` | Tab×3 to NEXT UP, `jjj`, then `L` to PRIORITY: a cursor mid-section and a non-default column selected |
| `list-cursor` | `vl`, `<`, then j×(H/2): cursor mid-screen, PRIORITY still selected |
| `board-cursor` | `vb`, Tab to DOING, j×9, k×3: cursor mid-column, P selected |
| `board-viewline-pending` | `v` pressed on the board |
| `after-vo` | `v o` back to the overview: the prefix cursor is gone |

All 64 PNGs exist (`ls out/*/*/*.png | wc -l` → `64`).

Blink evidence: `cmp` of `viewline-pending.ansi` against `viewline-pending-blink2.ansi`.

| Variant | 100x30 | 160x45 |
|---|---|---|
| before | SAME | SAME |
| after-A | differ | differ |
| after-B | differ | differ |
| after-A-letters | SAME | SAME |

The cursor blinks in the default prototype. Main and the static letters variant never move.

## The two column orders (choose one)

ID is first in both. OWNER is last in both, and only exists in the overview's ASKED, IN PROGRESS and WAITING sections.

- **A: attributes, then text.** `ID │ ✓ │ TYPE │ PRIORITY │ STAGE │ TITLE │ OWNER`
  - Today's order with ID moved to the front.
  - Short fixed-width cells line up at the left, and the title takes the rest of the line.
  - Scans like a table: the eye runs down TYPE and PRIORITY easily.
  - Board columns keep about 22 title cells at 100 wide.
- **B: read first, then attributes.** `ID │ TITLE │ STAGE │ ✓ │ TYPE │ PRIORITY │ OWNER`
  - The id and what it is sit together, so a row reads as a sentence: "66ex4y Capture: daily quality check … ready ● 1/1 chore medium".
  - Attributes form a right-hand block.
  - Costs:
    - The attribute block floats right in wide terminals, far from the id. At 160 wide in the overview, about 50 cells separate the title end from STAGE.
    - In narrow board columns the title is squeezed between ID and ✓ (about 9 cells at 160x45 with 4 columns).

To switch, run the prototype with `MOCKGRID_ORDER=A` (default) or `MOCKGRID_ORDER=B`.

## What changed (design points)

1. **ID is the leftmost column and the row header.**
   - The `✓` column moves right (A: second, B: fourth).
   - The id cell is ` 9taa9d ` with a pad on each side, so its selected tint reads as a cell.
   - The lint mark `!`/`·` and the gutter marks (`↑ ↓`, grab `≡`, yank `y`) stay in the two cells before it.
2. **Excel-style selection.**
   - Only two things are highlighted: the selected row's id cell, and the selected column's heading. Both use the same tint (`#2b5a78`, white text).
   - Their intersection, the current cell, has its text underlined in its own colour.
   - The row bar is gone, and so are the column band, the in-row `>`, and the row's `>` pointer.
3. **Toned down.**
   - Cells are normal weight on the plain background. The only bold left is the selected id and the section headings.
   - The focused section's heading keeps a `>` but no longer gets the selection background. Section headings stay bold, in their state colour, on the plain background.
   - The column heading row is dim and normal weight. Only the selected heading is tinted.
   - So a section heading (bold coloured text, no background) and the selected column heading (tinted background, small caps label) never look alike.
4. **View line.**
   - `v` is accent cyan and bold (step 1). `o l b !` are violet (step 2). The current view's name is normal text, the others dim.
   - The colours are stable: the line never recolours as a whole.
   - Pressing `v` shows it as entered (the selection tint, like a typed key). One blinking underline cursor then stands in the slot after `view:` where the second key goes.
   - The cursor disappears as soon as o/l/b/! is chosen (state 7 in `compare-viewline.png`).
5. **Two column orders**, as above.

## Choices made (for Jørgen)

- **One "you are here" cursor** (crab DM 11:44Z relaying Jørgen 11:43Z):
  - Style: an underline in accent cyan (`#00bfff` dark / `#006c9c` light), bold, blinking at **530 ms**. That is bubbles' `cursor` default blink speed, so macaque's answer-box cursor (qpp30y) in `CursorBlink` mode with the same underline style blinks in step and reads as the same thing.
  - It is the only motion on screen. I told crab the spec (`theme.cursor`).
- **The grid's current cell does not blink.** It is a static underline, because in a list the selection already shows on the id and the heading.
  - A blinking current cell can be seen with `MOCKGRID_CELLBLINK=1`, but it was not captured.
  - If Jørgen wants the cursor *everywhere*, this is the switch to flip. The pending-`v` cursor takes priority over it.
- **Where the next-key underline sits:** in a slot after `view:` (one underlined cell, like a text cursor).
  - Alternative captured as `after-A-letters`: the four second-step letters underlined, not blinking.
  - I chose the slot because it is one place, not four, and it matches the answer-box cursor.
- **Colour tiers:** amber means "waits on you" and red means "problem", so neither was free. The two tiers are cyan (v) and violet (o l b !).
- **Selection tint:** one tint (`#2b5a78` dark / `#bfe3f5` light) is used in exactly two places: the id cell and the column heading. It is brighter than main's row bar (`#16364a`) so that one cell still reads.
- **Current-cell marker:** the cell's text is underlined, not its padding. An empty current cell shows an underlined `·` so the cursor never vanishes.
- **Selected wait note** (the `└─ waiting on …` line under a selected row): it no longer gets the bar. It is indented under the id.
- **Selected group/epic heading row** (`▾ group [n]`): its fold arrow takes the id-cell tint, as its row header.
- **Board:** the selected column heading is tinted only in the focused status column. The other status columns show plain headings, so the selection exists once on screen.
- **Owner cell:** it is now a normal grid cell in a quiet slate colour (`#9fb3c8`). Before, it was bold accent cyan, which competed with the ids.
- **Closed tasks** (done/dropped): the whole row is dim except the id and the state word, as on main.
- **Not changed:**
  - `sort.go` (nch9d4's lane). Order-by still uses the existing `gridColumns` fields. Only their left-to-right order changed.
  - Key bindings (H/L column, Tab section, o order).
  - The detail view, the palette, and the pickers. Those still use main's `selected` bar style.

## How it was built and captured

- **Code:** `huginn/internal/tui/mockgrid.go` (new) plus small switches in `render.go`, `columns.go`, `viewline.go`, `group.go`, `waitrow.go` and `tui.go`. `MOCKGRID_OFF=1` restores most of main's drawing, but not all (see Tests).
- **Captures:**
  - `mock/capture.sh BINARY VARIANT`: its own tmux socket, `TERM=xterm-256color`, truecolor, keys as listed above.
  - `mock/ansi2png.py`: renders the PNGs with Menlo 15 on `#0b1620`, falling back to Apple Symbols for `⎘`.
  - `mock/sheets.py`: builds the comparison sheets.
- **Binaries:** `bin/hug-before` (origin/main) and `bin/hug-after` (prototype). Built with `GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19`.
- **Tests:**
  - The prototype breaks main's goldens on purpose: `go test -short ./huginn/internal/tui/` has 19 `--- FAIL` with the mock on.
  - With `MOCKGRID_OFF=1`, 10 tests still fail (e.g. `TestColumnCursorMarksHeader`, `TestPendingViewPrefixShowsChoices`). These are parts of the prototype not behind the switch: the reordered `gridColumns`, the removed row `>` pointer, and the blink tick `v` now schedules. I did not diagnose each test one by one.
  - Nothing here is meant to land. A real implementation would rewrite those tests.
- **Root checkout `/Users/jhf/Projects/hug`:** never edited, reset or committed. One patch addressed it by mistake through a relative path, failed to match, and wrote nothing. Every later edit used absolute scratch paths.
