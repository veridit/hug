---
id: 2qndsd
title: "PNG screenshots open inside the file frame"
state: "doing"
type: bug
priority: ""
assignee: ["macaque"]
labels: []
group: ""
rank: "zzzzzzzzzzzc"
created: 2026-11-18
updated: 2026-11-18
doing_since: "2026-11-18T14:18"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
A .png or .jpg attachment opens inside the same frame as other files: crumb and note on top, rule, the image, rule, keys at the bottom, so it is clear where the ticket ends and the screenshot begins.

## Purpose
Jørgen 2026-11-18 on 5z4zt4: when viewing something everything else goes away, with lines at top and bottom so a screenshot is seen inside the ticket. z82qsy framed text and .ans files, but images still open full screen through imageView.

## Principle
One viewer frame for every file kind. The terminal draws the image (iTerm2/kitty) inside the frame's rows, sized to fit; elsewhere the one-line fallback sits in the frame.

## Relevant details
z82qsy ef9c060f added the frame (glance.go, render.go); imageView draws PNGs full screen outside it. Jørgen's terminal: iTerm2 (OSC 1337).

## Depends on

## Acceptance
- [ ] A PNG opens with the crumb, the note, a top rule, the image within the frame rows, a bottom rule and the keys (golden for the fallback; tmux capture in iTerm2 attached to this task).
- [ ] n/p moves between images and text files with the same frame (test).

## Notes

macaque (claude-opus-5-5) is working this stacked on z82qsy at crab's direction (13:16Z), branch 2qndsd from origin/ci/z82qsy-rebased 161381f3. hug task take refuses ('2qndsd is blocked by z82qsy') until z82qsy is done, so the dependency stays.

crab 13:17Z: removed the dependency on z82qsy so macaque can take it. The real ordering is in git: branch 2qndsd stacks on z82qsy's commits, and crab lands them in order.


file:tmux-iterm-image-frame.txt (826 bytes, sha256:e2165d8a) tmux capture-pane, real hug tui from 499f9220 at 100x24, TERM_PROGRAM=iTerm.app, Space on board.png: crumb, note, rule, 17 blank rows the image is drawn over, rule, keys
## History
- 2026-11-18T14:18 todo -> doing/active (take)
