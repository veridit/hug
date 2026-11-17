---
id: 5z4zt4
title: "Attach screenshots and screen captures to a task; the detail shows them inline"
status: todo
type: feature
priority: "high"
assignee: ["nautilus"]
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
A task can carry pictures of what it is about: PNG/JPEG screenshots and ANSI screen captures (a TUI frame with its colours). hug task attach ID FILE stores them with the task; the detail view shows them inline where the terminal can draw images (iTerm2 inline images, kitty graphics, sixel) and ANSI captures with their colours in any terminal; otherwise one line with the size and 'Enter opens it'. A question to a person (waiting_on, review_by, design_for) can name the attachment to look at, and the detail shows that attachment right under the question.

## Purpose
Jørgen 2026-11-17 14:51 on zhvkmj, which waits on him to 'look at the colours': 'I'm supposed to look at something but I don't know what. I can read the ticket but there's nothing to see.' A review of a look needs the look in the ticket. He asked for inline images in tickets.

## Principle
Show, don't describe: a question about a look carries the picture. Read only when shown (as scrxak). Never send raw bytes the terminal cannot draw; detect support, fall back to one line. ANSI captures are text, so they also diff in git and work over ssh.

## Relevant details
Builds on scrxak (Space shows a file inline, files read only when shown) and the existing hug task attach / task files. Jørgen's terminal reports TERM_PROGRAM=iTerm.app (iTerm2 inline images protocol: OSC 1337 File=). Agents can produce an ANSI capture from tmux (capture-pane -e -p) or from a TUI golden with colour. Asker side: hug task wait ID --on jhf "look at the IN PROGRESS rows" --show FILE.

## Depends on

## Acceptance
- [ ] hug task attach ID shot.png and an ANSI capture; the detail shows both inline in iTerm2 and kitty, ANSI with colours anywhere, and one line elsewhere (golden for the fallback, manual check for images)
- [ ] a wait or review that names an attachment shows it right under the question in the detail (test + golden)
- [ ] no image bytes are read until the attachment is shown (test counts reads)
- [ ] Jørgen answers zhvkmj's colour question from the picture in the ticket

## Notes

Jørgen 2026-11-17 14:51: prefer text pictures. An ASCII/ANSI capture is a text file, cheap to take, readable in any terminal, diffable in git, and an agent can read it too, while a PNG needs a vision model to check. Proposal: in hug tui a key dumps the current screen as a capture (frame text with colours, as the goldens already render), attached to the task in view or saved under tmp/ with the path shown in the footer. Format: a .ans file (ANSI escapes kept) shown inline in the detail with its colours, and in Markdown as a fenced block (```ansi) so it reads plainly in a viewer. PNG stays for things outside the TUI.

Jørgen 2026-11-17 14:51: the dump key also copies to the clipboard, so he can paste it straight into a chat with an agent. The screen text goes out through OSC 52, which works in iTerm2, kitty and over ssh, with pbcopy as a fallback on macOS. Plain text by default, so it pastes cleanly; the attached .ans file keeps the colours. The footer says 'screen copied (24 lines) · attached as screen-1452.ans to zhvkmj'.
