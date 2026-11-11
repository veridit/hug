# Terminal task browser

`huginn tui` opens the overview: status counts, in-progress tasks with owners,
next tasks by global rank, dependency-blocked tasks and lint. Use `o`, `v`, `b`
and `!`, or `:overview`, `:list`, `:board`, `:lint`, to change views.

## Navigation and finding

| Key | Action |
| --- | --- |
| `j/k`, arrows | Select rows, or scroll the detail/help viewport |
| `gg`, `G`, `<`, `>`, Home, End | Top / bottom, including fast coalesced `gg` input |
| `ctrl-d/u`, PgDn/PgUp | Half / full page |
| `h/l`, left/right, Tab/Shift-Tab | Previous / next overview pane or board column; change view from the list; switch acceptance/dependency focus in detail |
| Enter | Open task; in the dependency section follow the selected dependency |
| Esc | Close one layer: help, confirmation, picker value, picker fields/input, followed dependency, detail, filters, then return to overview. Never quits |
| `q`, `ctrl-c` | Quit. `q` is text when an input has focus |
| `/` | Live case-insensitive title/slug/id/owner search. Enter retains it, Esc clears it |
| `#` | Complete an id prefix or imported alias, including `#TASK-1785` and `#1785`; Enter opens the choice |
| `f` | Quick-filter field picker: `s` status, `t` type, `p` priority, `g` group, `l` label, `o` owner |
| `1/2/3/4` | Filter todo / doing / done / blocked |
| Backspace | Remove the most recent filter chip when no input has focus |
| `F`, `c`, `:clear` | Clear chips and text search |
| `:` | Command palette; Tab or Enter completes; Enter executes a complete command; arrows or ctrl-n/p select a completion |
| `?` | Help overlay, even while typing; never inserted into an input |
| `r` | Refresh task files and lint |
| `s`, `y` | Open status/type palette completion |

Picker values display counts respecting all other filters and the live search.
Type to narrow, or use j/k on an empty query and arrows/ctrl-n/p at any time.
Filters combine across fields. Picking a second value for a field replaces its
previous chip and makes it the newest chip. Root groups and absent metadata are
available as `(root)`, `(none)` and `(unassigned)`. Jump resolves against all tasks,
retaining the active filters so Esc returns to the same filtered context.

Palette commands: `overview`, `list`, `board`, `lint`, `help`, `todo`, `doing`,
`done`, `dropped`, `blocked`, `status <s>`, `type <t>`, `priority <p>`, `group <g>`,
`label <l>`, `owner <o>`, `goto <id>`, `clear`. Values come from all task data,
not just the currently filtered result.

## Editing

Every edit requires Enter to confirm; Esc cancels without writing.

| Key | Action |
| --- | --- |
| `[` / `]` | Select acceptance item or dependency in the focused detail section |
| Space | Toggle selected acceptance item, defaulting to the first |
| `t`, `d`, `x` | Take, done, drop, retaining core acceptance/dependency checks |
| `J/K` | Move down/up in **global** rank order, also from filtered views |
| `n` | Enter title, then confirm. Goal equals title and one unchecked verification item is created |
| `e` | Confirm `$EDITOR` (default `vi`), then refresh and lint |

`Commit` is called after each successful changed local mutation. An optional
`Take func(ref string) (task.Result, error)` owns the entire distributed take
transaction and its commit, so the TUI never double-commits it. Error messages
remain inline with their first exact `fix:` command. Malformed files remain
editable by their confined path, even with no valid id. Symlink paths and paths
escaping the store are rejected.

## Layout and design choices

- Bubbles `textinput`, `viewport`, `key` and `help` own the relevant primitives.
- IDs are primary coloured six-character badges, following the user's later
  2026-11-10 clarification rather than the earlier dim-id design wording.
  Dates are dim. Status, type and priority have stable adaptive colours, with
  `#00bfff` accent on dark terminals. Completed/dropped row text is subdued while
  the id and lint marker retain their colours. `NO_COLOR` removes colour without changing layout.
- List columns are cell-aligned and titles truncate with `…`. Slugs are reserved
  for detail, where they add information rather than duplicate list titles.
- The board has five logical columns, horizontally paged to fit. Status lives in
  the column header, so card rows reserve columns for id, lint, type, priority and
  title. At 80 columns two status columns fit; at 140 three fit.
- `blocked` is a derived view of open tasks with unmet or missing dependencies,
  not a new persisted status. Overview counts overlap todo/doing; the board puts
  such tasks only in its blocked column. Malformed unknown-status tasks remain
  in the todo column with a lint marker so they are repairable.
- A task detail has a fixed title/badge/date/lint header and a scrollable body.
  Glamour renders Markdown with adaptive dark/light styles and the same accent.
  Storage comments are removed first. Sections appear in description, acceptance,
  dependencies, notes, history order, followed by any additional source sections.
  Empty acceptance/dependencies use a dim sentence; empty notes/history are hidden.
  A plain-words lint warning appears once in the header, without a path prefix.
- Search and completion are modal but help can overlay them without destroying
  their text. Filter and navigation keys deliberately replace the old `g` group
  and `l` label prompts; those filters remain in `f` and the palette.

## Validation and golden screenshots

Run all builds/tests with `nice -n 19 env GOMAXPROCS=2 GOFLAGS=-p=2`. On macOS check
`sysctl -n vm.loadavg` first and pause if the five-minute load exceeds 120.

`go test ./internal/tui` runs exhaustive model tests and real CLI tmux goldens
when tmux is installed. `-short` skips only the tmux acceptance. Goldens live in
`testdata/tmux/`: 60 deterministic synthetic tasks, ten screens at 80x24 and
140x40, both colour and NO_COLOR. `capture-pane -e` additionally asserts colour
presence/absence. Every started session uses an isolated tmux socket and is
cleaned up, and the real CLI must exit zero.

```sh
nice -n 19 env GOMAXPROCS=2 GOFLAGS=-p=2 go test ./internal/tui
# Deliberate golden update, followed by reviewing the text diffs:
nice -n 19 env GOMAXPROCS=2 GOFLAGS=-p=2 HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./internal/tui
```

Set `HUGINN_TUI_ARTIFACTS` to also save plain and ANSI captures outside the tree.
The reported done/lint/Markdown/empty-section shape has its own golden. Model
coverage includes every shared key, modal Esc hierarchy, completion, filter counts,
imported aliases, Markdown stripping, fixed headers, dependency follow/back,
confirmation cancellation, tied ranks, transactional take, malformed documents,
editor confinement and width bounds through 200x60.

`python3 scripts/tui-pty.py <artifact-directory>` retains real PTY mutation and
terminal-restoration acceptance: persisted acceptance toggles and global rank
moves, blocked completion, clean quit, alternate-screen exit, canonical input
and all terminal attributes after the next canonical read. Fuzz seeds exercise
arbitrary malformed task bytes through rendering and action handling.
