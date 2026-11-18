# 5z4zt4 design (nautilus 16:52)

Pictures in a task, preferring text pictures (Jørgen 14:51).

## 1. What can be attached
- Text, as today (256 KiB, UTF-8). `.ans` is text: an ANSI capture with its colours.
- Images: PNG, JPEG and GIF, recognised by their first bytes and not by the name, up to 2 MiB.
  `hug task attach ID shot.png` now accepts them. Any other binary is still refused,
  as is an image whose bytes don't match its name. Lint applies the same rule to files on the board.

## 2. Showing it (Space in FILES, as in scrxak)
- `.ans`: shown with its colours. Only colour codes (SGR, `ESC[...m`) are kept; every other
  escape (cursor moves, clearing, titles, OSC) is shown escaped. Each line is cut to the width.
  This works in any terminal. Under NO_COLOR it is shown as plain text.
- Image in iTerm2 (TERM_PROGRAM=iTerm.app, WezTerm) or kitty (TERM=xterm-kitty or
  KITTY_WINDOW_ID): Space draws it full screen outside the frame (the TUI steps aside, like
  the editor does), and any key comes back to FILES. Drawing inside bubbletea's frame would be
  erased by the next repaint.
- An image elsewhere gets one line: `image, 120.3 KiB: Enter opens it`.
- A file is read only when it is shown (as scrxak). The list and the detail read no image bytes.

## 3. Capture the screen: `S`
- `S` takes the frame as you see it, before the key bar changes. The plain text goes to the
  clipboard: OSC 52 always (iTerm2 and kitty, and it works over ssh), plus pbcopy on a local
  macOS. Then the bar asks `▶ Attach this screen to ID as screen-1452.ans? Enter attaches · Esc
  keeps only the copy`. That follows the x1ra0t rule: every key that writes asks first.
- On Enter it attaches the coloured `.ans`, commits, and the footer says
  `screen copied (24 lines) · attached as screen-1452.ans to zhvkmj`.
- On a list row the task is the selected one; in a task it is that task.

## 4. A question names its picture
- `hug task wait ID "look at the rows" --on jhf --show screen-1452.ans` and
  `hug task design ID "?" --for jhf --show NAME` write `waiting_show` / `design_show`.
  MCP task_update has a `show` field. The name must be one of the task's files; otherwise it is
  refused with the fix `hug task attach ID FILE`.
- When there is no _show field, the first task file named in the question text is used
  (which already shows as ▸ NAME).
- In the detail, right under the question (`? waiting: …`, asked by, `a answer`), the picture
  is shown: an `.ans` inline with its colours, at most 30 lines and then `… N more lines: Space
  in FILES shows all`. An image gets one line `▸ shot.png · image 120 KiB · Space in FILES shows it`.
- Review: no command writes a review request's text today, so no review_show for now.
  Question to crab.
