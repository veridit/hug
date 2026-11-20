# Screen capture prompt: proposed wording (hjbfzd, request 1)

Jørgen 2026-11-20 08:43Z: the prompt reads "Attach this screen to:", then the
task's long title, then "select · create". It should read "Screen capture:",
attach to the pre-filled task, or select, or create.

## Today (v0.7.87, golden testdata/detail/capture-prompt.txt, 80 columns)

    screen copied (30 lines)
    Attach this screen to: 000002 → Bug task 02: make the termin… · select · create
    keys: Enter attach · Tab complete · Esc not attached · F1 help

## Proposed

The line names the thing first, then the three ways it can go. The offered
task shows its ID only; its title moves to the footer line, where it does not
push the choices off the edge.

    Screen capture: attach to 000002 · select · create
    keys: Enter attach to 000002 "Bug task 02: make the terminal…" · Esc keep only the copy

- `Screen capture:` is the label (bold accent, as the other prompts).
- `attach to 000002` is the default. Enter does it. Typing replaces the ID
  (a prefix completes, Tab fills it in), as today.
- `select` (s underlined) picks the task from the list.
- `create` (c underlined) asks for a title and creates the task with the
  capture attached. "new" is dropped: create already says it.
- The title of the offered task goes to the keys line, cut to fit.
- With no task in view (nothing offered): `Screen capture: attach to _ · select · create`.

The other two steps, same voice:

    Screen capture: new task, title: _
    keys: Enter create and attach · Esc back

    Screen capture: select the task · j/k move · Enter attach here · Esc back

The clipboard note ("screen copied (30 lines)") stays on the message line above,
as today.

See `capture-wording-100x30.ans` for the proposed line in the real theme.
