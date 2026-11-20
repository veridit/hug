---
id: qh9ep1
title: "Key prefixes: v picks the view (vo vl vb v!), a pending prefix shows its choices"
state: "done"
type: feature
priority: ""
assignee: ["crab"]
labels: []
group: ""
rank: "zzzzzzzzzzlh"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-20T09:30"
done_release: "v0.7.69"
---

## Goal
Views are chosen with v and a letter (vo, vl, vb, v!), and every pending prefix key shows its choices in the footer.

## Purpose
Jørgen 23:35 local, on the header 'overview list board !lint': it should become 'view: overview ...' so that vo selects it, the trie idea from 0sw0dd (header tree). Single letters are scarce: o is now order and l is right (e3sevr), so the views move under one prefix.

## Principle
One prefix, one family, shown where it lives. Every key shown works, and every pending key shows what comes next. No timeout guessing: a prefix waits until the next key or Esc. No old single-key aliases kept (no backward compatibility). Subtle colours.

## Relevant details
The view line reads 'view: overview  list  board  !lint' with v and each view's letter marked, and the views are chosen with two keys: vo, vl, vb, v!. While a prefix key is pending, the line it belongs to lights up and the footer lists its choices ('v: o overview · l list · b board · ! lint · Esc cancel'). Every prefix in the TUI works the same way (a key trie), so g can be 'g: g top · r grouping' (kx2ndm).

Today: the tab line underlines o, l, b and ! as single keys. After: line 2 is 'view: overview  list  board  !lint', with 'v' and the first letter of each name marked. Pressing v highlights that line and sets the footer to its choices; the next key picks, any other key cancels with a footer note. Same mechanism for g (gg top, gr grouping, kx2ndm) and any later prefix; context help (?) lists them as 'v o  overview' rows. Depends on e3sevr (the key map) and lands with or after kx2ndm's g. Owner: nautilus.

## Depends on

## Acceptance
- [x] The view line renders 'view: overview  list  board  !lint' with v and each view letter marked (golden at 80 and 140 columns)
- [x] vo, vl, vb and v! switch views from every view; bare o, l, b and ! no longer switch views (key test)
- [x] After v alone the view line is highlighted and the footer reads 'v: o overview · l list · b board · ! lint · Esc cancel'; Esc or an unknown key cancels and changes nothing (key test, golden)
- [ ] ~~g uses the same prefix mechanism: gg top, the grouping key from kx2ndm, footer shows the choices while pending (key test)~~ (not relevant: Jørgen 23:38 local: gg and G dropped, so g is no prefix (it groups rows, kx2ndm); v is the only prefix, and check 8 covers the gg/G removal, nautilus, 2026-11-18)
- [x] Context help (?) lists prefix keys as two-key rows and its search finds them by either key or name (help test)
- [x] Jørgen confirms on his board after release
- [x] The state counts sit on the view line, right aligned after a dim separator, and the old counts line is gone (golden at 80 and 140; at 80 counts drop from the right first)
- [x] gg and G are gone (keys, hints, help); < Home go to top and > End to bottom as before (key test)
- [x] The view line is subtle, one cue per meaning (Jørgen 19:54 on check 6): an underline only marks a letter you can press, colour marks the keys (v, then o l b !), the active view is told by bold OR colour alone on its word, and nothing has a background fill; no letter carries underline+bold+colour together (golden at 80 and 140, colour)

## Notes

Jørgen 23:36 local: the state counts line 'todo 31 · doing 26 · depends 0 · done 79 · dropped 3' moves onto the view line, right of it, after a separator, probably right aligned. One line instead of two:
  view: overview  list  board  !lint            │ todo 31 · doing 26 · depends 0 · done 79 · dropped 3
When tight, counts drop from the right (dropped, done) before the view names shorten.

Jørgen 23:38 local: drop gg and G ('I never used it', 'vi specific'). Top and bottom stay on < > Home End. With gg gone, g is no prefix: the g item of this ticket is void; v is the only prefix so far.

Header picture for all of tonight's rulings is attached to e3sevr as header-spec.md (crab 00:00 local). Build to it. Where it and this ticket differ, tell crab.


landed a90f4d44..03e783d0 (nautilus). CI run 44233304533 green on all 6 jobs (ubuntu with tmux goldens, macos, windows, windows-cli, bench, secrets); -race on tui ok (119.6s); tui + gitbranch green after the final rebase. Tests: huginn/internal/tui/viewline_test.go, all four fail first on main 28791dea..a90f4d44 with the shim viewLine/stateCounts: TestViewLineCarriesCounts 'at 80 line 2 is " overview   [list]   board    !lint"', TestViewPrefixPicksEveryView 'from overview bare "!" switched to lint', TestPendingViewPrefixShowsChoices 'footer while v waits: "view: overview · list · board · ! lint"', TestHelpListsPrefixRows 'help lacks "v o view: overview"'. Check 1: golden testdata/header/view-line.txt at 80 and 140; v, o, l, b, ! each in the key style (raw render). Check 2: vo vl vb v! from each of the 4 views and from a task; bare o (orders), b, ! and the old i switch nothing; l stays the pane key (h/l Tab, e3sevr), so it is not a view key by itself. Check 3: footer 'v: o overview · l list · b board · ! lint · Esc cancel', the view line renders differently while pending (TrueColor), Esc/x/j/q/e/S leave state unchanged, x says 'v x is no view; nothing changed', ? opens help and cancels v, Esc in a task cancels only the prefix; golden testdata/header/view-pending.txt. Check 5: help rows 'v o view: overview' etc., help find hits for vo, vb, v !, !, 'view: board'. Check 7: counts right-aligned after │ on line 2 (TestHeaderCountsAddUp now reads them there), the old line gone; at 80 'dropped' and 'done' drop first. Check 8: gg/G were already gone (kx2ndm), TestKeyTableListNavigation in design_test.go: '>' then 'G' does not move or speak, g groups; < > Home End unchanged. Check 4 struck (g is no prefix, Jørgen 23:38).
landed 03e783d on main 2026-11-18: tui: v is the view prefix (vo vl vb v!), the view line reads 'view: …' and…
shipped in v0.7.56 (03e783d)
crab 19:55 local, on Jørgen's check 6 answer 'pretty good and works the way I hoped, but the colouring is too intense... underscore AND bold AND colour on the pickable letter, and background on the active one... subtle please': new check 9 spells his rule. It supersedes 911j70 check 3 as built in 89e280c9 (active view on #2b5a78 background, v bold accent): macaque reworks it before it lands. Check 6 stays open until he confirms the toned-down line.
file:view-line-before.ans (2994 bytes, sha256:66e6ccf8) check 9 before (main 6a6b6573) / after (dd1830f0): hug tui top 3 lines, tmux 100x24 truecolor, idle, after v, +530 ms, after v l. After: v accent and o l b ! violet, underlined, not bold; the active view in bold dim only; no background (48;2 count 0, before 2)
file:view-line-after.ans (3082 bytes, sha256:00c2ab26) check 9 before (main 6a6b6573) / after (dd1830f0): hug tui top 3 lines, tmux 100x24 truecolor, idle, after v, +530 ms, after v l. After: v accent and o l b ! violet, underlined, not bold; the active view in bold dim only; no background (48;2 count 0, before 2)
macaque, check 9 rework (crab 18:54): commit dd1830f0 on branch qh9ep1-subtle (911j70 89e280c9 rebased as 897d4810 onto main 6a6b6573, then this one), CI 44375969737. Cues: an underline on v and o l b ! only; colour by step (v #00bfff/#006c9c, o l b ! #c9b3ff/#6a3fb0), no key bold; active view = bold alone on dim #8795a8 (chose bold over the default foreground: the dim line stays one colour); no background anywhere (the 911j70 #2b5a78 tint and the entered-v tint are removed). TestViewLineMarksActiveView checks every cell at 80/140 in all 4 views; it failed first on 897d4810 '911j70_test.go:56: at 80 in overview v recoloured column 0: idle "\x1b[1;38;2;0;191;255mv" pending "\x1b[1;38;2;255;255;255;48;2;43;89;120mv"'. Golden testdata/header/view-cursor.txt (80/140, colour + NO_COLOR). Before/after .ans attached (48;2 background codes: before 2, after 0).
check 9: dd1830f0, CI 44375969737 6/6 success. TestViewLineMarksActiveView: every cell at 80 and 140 in overview/list/board/lint: no background, underline only on v o l b !, v 38;2;0;191;255 and o l b ! 38;2;201;179;255 not bold, the active word bold on 38;2;135;149;168, no cell underline+bold+colour. Golden testdata/header/view-cursor.txt (colour + NO_COLOR). Before/after file:view-line-before.ans / file:view-line-after.ans (48;2 count 2 -> 0). -short tui: 'ok github.com/veridit/hug/huginn/internal/tui 262.193s'.
crab 20:20 local: check 9 accepted on review, qh9ep1-subtle dd1830f0 on 897d4810 (911j70 c3 rebased): underline only on pressable keys, active view bold alone, no background (.ans 48;2 2->0); CI 44375969737 6/6. Lands in the next stack after the windows-cli shard.
landed f9ff43b on main 2026-11-18: tui: the view line is quiet, one cue per meaning (qh9ep1 check 9)
shipped in v0.7.69 (f9ff43b)
## History
- 2026-11-18T04:25 todo -> doing/active (take)
- 2026-11-18T05:00 doing/active -> doing/asked (ask)
- 2026-11-18T19:54 answered asked (check 6), by jhf, for nautilus (doing/asked -> doing/active):
  > after the release: line 2 now reads 'view: overview  list  board  !lint' with the counts at the right after │; press v (the line lights up, the footer lists o l b !), then o, l, b or ! to switch. Bare b, ! and i no longer switch views. Does it read clearly?
  It is pretty good and it works the way I hoped, except for the colouring is too intense and it's over the top. What do I mean by that? I mean that right now I had another ticket to reduce the or to improve the coloring, but I think it needs to be toned down. You have both underscore and you have bold and you have color on the letter that you can pick. It's too much and you have background on the one that is active. It's like it's a we have to be subtle. So the underscore tells you it can be picked, right? And then the coloring, maybe either the bold or the color tells you which one is active. And the color tells you what you can pick. Something more subtle, please
- 2026-11-18T19:54 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T19:54 answer 1 acknowledged by crab
- 2026-11-18T19:54 answer 1 in progress by crab: hug task set qh9ep1
- 2026-11-18T21:07 doing/active -> doing/asked (ask)
- 2026-11-20T09:30 answered asked (check 6), by jhf, for crab (doing/asked -> doing/active):
  > v0.7.69 is installed. Is the view line calm now? It should show an underline only on keys you can press, the active view in bold alone, and no background colour.
  answered by ticking check 6
- 2026-11-20T09:30 doing/active -> done (done after jhf ticked check 6 (release v0.7.69)); was doing_since: 2026-11-18T04:25
