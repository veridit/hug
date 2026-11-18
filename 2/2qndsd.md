---
id: 2qndsd
title: "PNG screenshots open inside the file frame"
state: todo
type: bug
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzv"
created: 2026-11-18
updated: 2026-11-18
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
- z82qsy

## Acceptance
- [ ] A PNG opens with the crumb, the note, a top rule, the image within the frame rows, a bottom rule and the keys (golden for the fallback; tmux capture in iTerm2 attached to this task).
- [ ] n/p moves between images and text files with the same frame (test).

## Notes
