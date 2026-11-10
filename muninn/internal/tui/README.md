# Owner terminal browser

`muninn tui` opens a populated, Binding-scoped overview. Its four panes are Places,
Recent, Told agents lately and Needs a look. Nothing is written by browsing,
searching, rendering details or previewing recall. The CLI's existing startup
indexing still runs before the terminal browser starts.

## Navigation and finding

- `j/k` or arrows move, `h/l`, Tab and Shift-Tab change panes.
- `gg/G`, `<`/`>`, Home/End jump to the top/bottom. Ctrl-D/U and PgDn/Up page.
- Enter opens a full memory. Esc unwinds one level and never quits. `q` quits only
  outside input, Ctrl-C always quits, or waits for an already confirmed write.
- `/` searches live using the existing searcher. Enter retains the exact results
  and scores. The next Enter opens the selected memory. Esc clears the search.
- `#` completes visible stable-id prefixes and numeric row ids. Enter opens one.
- `f`, then `p`, `c` or `d`, picks a place, category or date from counted values.
  Type to narrow or use j/k and Enter. Chips remain visible at the top. Different
  fields intersect. Selecting the same field replaces its previous value.
- `1/2/3/4` choose here/client/global/all. `all` removes the place chip, not the
  Binding's visibility boundary or other chips. Backspace removes the last chip.
  `F` and `:clear` clear all chips and the text query.
- `?` opens help, including from an input without inserting a question mark.
- Ctrl-E toggles existing score explanations. Ctrl-R focuses related memories.
  Enter follows a related memory, with Esc returning to the previous detail.

The `:` palette completes commands and values. Tab or Enter inserts the selected
completion. When the input is already exact, Enter executes it. Ctrl-N/P and
arrows choose completions. Commands:

```
overview                 recent                    told
place <visible place>    category <category>        date today|yesterday|YYYY-MM-DD
search <query>           goto <visible id>          today
yesterday                dupes                     moves
clear                    recall <read-only preview query>
```

Dates use the memory's last-updated date in the user's local timezone. Place
aliases come from Binding.Places. Additional visible scopes remain selectable.
An alias collision is labelled `scope:<scope>` so it cannot silently select a
similarly named, different scope. An unavailable client shortcut explains why it
cannot be selected instead of widening the search.

## Details and owner actions

Details have a fixed normal-colour title and category/place/date/id/strength
badges, a scrollable Markdown body, origin/session metadata and a navigable
related pane. Glamour v0.10.0 is shared with huginn. HTML comments/storage markers
and inline HTML tags are removed before rendering. Long code and URL tokens wrap
inside the pane. Empty related results have one dim explanation.

Owner actions always re-resolve the memory and write policy at confirmation:

- `e` writes a correction in a multiline editor. Ctrl-S opens confirmation, then
  `y` appends a new correction and retires/suppresses its predecessor. Original
  text and dates are not rewritten. A private byte-exact backup precedes writes.
  Upstream note/session files are never edited. Failure restores the original
  native files and suppression state. Secret-looking corrections fail closed.
- `m` chooses another writable visible place, then Enter and `y` confirm. This
  uses Binding.Rescope and the same cross-client restrictions as the CLI.
- `x`, then `y`, retires through the existing Binding.Forget path.
- Esc or `n` cancels confirmation without writes.

Needs a look reads the newest valid duplicate and move plans from the data dir
or its `tidy` child. Only wholly visible duplicate clusters and visible-source,
permitted-target moves appear. No plan is generated automatically. Empty states
show `muninn tidy dupes` and `muninn tidy scan`. In review, j/k and paging scroll
long entries, Enter opens the current keeper/memory, `n` skips without modifying
plans or memories, `y` approves this one entry through App.ApplyTidy and its
existing backup/receipt path, and Esc stops. Unreviewed entries are never applied.
Stale fingerprints still fail closed. Skips are local to the current TUI session.

## Presentation

Dark accent is `#00bfff`, with readable adaptive light variants. Category colours
are stable, and places use a stable FNV-1a colour mapping. Ids/dates/scores are dim.
`NO_COLOR` suppresses foreground/background colours without changing the layout.
All clipping uses display-cell widths, not byte counts. Resize keeps fixed
headers and reflows detail Markdown.

## Tests and screenshots

Model tests exercise visibility, input routing, completion, filtering, stale
search responses, resizing, complete details, confirmations, correction rollback
and the existing tidy apply path. `go test ./...` also builds the real binary and
runs `scripts/tui-pty.py` when tmux, Python 3 and nice are installed. That test
checks 36 committed text goldens: nine screens at 80x24 and 140x40, each with and
without colour. It asserts actual ANSI colour presence/absence and clean exit.
It skips only in `-short` mode or when terminal prerequisites are unavailable.

Regenerate intentionally after inspecting the changed screens:

```
nice -n 19 env GOMAXPROCS=2 GOFLAGS=-p=2 go build -o /path/in/scratch/muninn ./cmd/muninn
nice -n 19 env GOMAXPROCS=2 GOFLAGS=-p=2 python3 scripts/tui-pty.py /path/in/scratch/muninn --update
```

All fixtures have a throwaway HOME/MUNINN_HOME and every tmux session is killed
in a finally block. No checked-in golden contains live memories.
