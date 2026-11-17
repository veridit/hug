---
id: qh9ep1
title: "Key prefixes: v picks the view (vo vl vb v!), a pending prefix shows its choices"
state: todo
type: feature
priority: ""
assignee: []
labels: []
created: 2026-11-17
updated: 2026-11-17
---

## Goal
The view line reads 'view: overview  list  board  !lint' with v and each view's letter marked, and the views are chosen with two keys: vo, vl, vb, v!. While a prefix key is pending, the line it belongs to lights up and the footer lists its choices ('v: o overview · l list · b board · ! lint · Esc cancel'). Every prefix in the TUI works the same way (a key trie), so g can be 'g: g top · r grouping' (kx2ndm).

## Purpose
Jørgen 23:35 local, on the header 'overview list board !lint': it should become 'view: overview ...' so that vo selects it, the trie idea from 0sw0dd (header tree). Single letters are scarce: o is now order and l is right (e3sevr), so the views move under one prefix.

## Principle
One prefix, one family, shown where it lives. Every key shown works, and every pending key shows what comes next. No timeout guessing: a prefix waits until the next key or Esc. No old single-key aliases kept (no backward compatibility). Subtle colours.

## Relevant details
Today: the tab line underlines o, l, b and ! as single keys. After: line 2 is 'view: overview  list  board  !lint', with 'v' and the first letter of each name marked. Pressing v highlights that line and sets the footer to its choices; the next key picks, any other key cancels with a footer note. Same mechanism for g (gg top, gr grouping, kx2ndm) and any later prefix; context help (?) lists them as 'v o  overview' rows. Depends on e3sevr (the key map) and lands with or after kx2ndm's g. Owner: nautilus.

## Depends on

## Acceptance
- [ ] The view line renders 'view: overview  list  board  !lint' with v and each view letter marked (golden at 80 and 140 columns)
- [ ] vo, vl, vb and v! switch views from every view; bare o, l, b and ! no longer switch views (key test)
- [ ] After v alone the view line is highlighted and the footer reads 'v: o overview · l list · b board · ! lint · Esc cancel'; Esc or an unknown key cancels and changes nothing (key test, golden)
- [ ] g uses the same prefix mechanism: gg top, the grouping key from kx2ndm, footer shows the choices while pending (key test)
- [ ] Context help (?) lists prefix keys as two-key rows and its search finds them by either key or name (help test)
- [ ] Jørgen confirms on his board after release
- [ ] The state counts sit on the view line, right aligned after a dim separator, and the old counts line is gone (golden at 80 and 140; at 80 counts drop from the right first)
- [ ] gg and G are gone (keys, hints, help); < Home go to top and > End to bottom as before (key test)

## Notes

Jørgen 23:36 local: the state counts line 'todo 31 · doing 26 · depends 0 · done 79 · dropped 3' moves onto the view line, right of it, after a separator, probably right aligned. One line instead of two:
  view: overview  list  board  !lint            │ todo 31 · doing 26 · depends 0 · done 79 · dropped 3
When tight, counts drop from the right (dropped, done) before the view names shorten.

Jørgen 23:38 local: drop gg and G ('I never used it', 'vi specific'). Top and bottom stay on < > Home End. With gg gone, g is no prefix: the g item of this ticket is void; v is the only prefix so far.
