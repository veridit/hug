---
id: gav9nw
title: "Design: personal vs shared memories"
state: "doing"
type: spike
priority: "low"
assignee: ["piglet"]
labels: ["design"]
created: 2026-11-10
updated: 2026-11-10
parent: "9taa9d"
group: "design"
rank: "zy"
aliases: ["design-personal-vs-shared-memories"]
doing_since: "2026-11-18T18:53"
stage: "waiting"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
waiting_asked_by: "piglet"
waiting_on: "crab review and landing of 8a1b0c05 (CI 44386929067)"
waiting_since: "2026-11-18T21:03"
---

## Goal
Decide how muninn tells personal memories from shared ones

## Purpose
Jørgen 2026-11-05 17:00Z (crab session): 'how do you know that something is personal or not? How is that judged? We have to look at that later ... write into the to-dos' (memories 5zwpx7, 1c49sh). It is open question 2 in docs/DESIGN.md (2026-11-11). It matters because only shared project memories may be pushed to a repo branch, while personal ones never go into a repo (hbzc5e), and he wants an in-between team-shared scope as well (873tqd).

## Principle
A design decision first, written in DESIGN.md, no code in this task. Start from the scopes that exist today (personal, global, project, client) and the rules already decided (below); do not reopen them unless the decision needs it. Jørgen makes the call; the task brings him options with their consequences.

## Relevant details
Decided so far: new Engram memories with no project default to personal, and global is reserved for working rules shared with all non-client views (s755vd; 9s4v6g: 9 personal memories moved to personal, global kept 2 rules). Placement rule 2026-11-06 (s5tse0): a session sees only its top-level (global or client root) memories plus its own project's. Related: 34r1ft (per-repo setting whether project memories go on a branch). Epic 9taa9d moves memory to one file per memory, so the decision should hold for that layout too.

## Depends on

## Acceptance
- [x] decision in DESIGN.md

## Requests
- 2026-11-18T20:36 request 1 by jhf:
  When I'm looking at this the zoom the thing I'm going to respond to the first line is displayed the whole section should be displayed otherwise I think I didn't scroll correctly
  resolved 2026-11-18T20:37 by crab:
    The detail now has to show a focused check whole, the question with it, not just its first line. If the check is taller than the screen, it shows from its top. It is check 4 on ykta5g (j/k in the detail): piglet is changing the same scroll code now, so it ships with that fix. Until then, j scrolls the rest of the question into view.

## Notes


OPTIONS at /Users/jhf/.jcode/scratch/gav9nw/OPTIONS.md (scripts, labels, logs alongside). 868 memories on public hug/memory 1abcc497, blind Flash labeler, 30 paid calls (cap 100); hand check: random 30 agree 30/30, all 90 non-repo labels read (34 overturned), 63 regex suspects read (0 overturned). Corrected truth: 58 of 868 (6.7%) don't belong on hug's branch (29 personal, 16 other project, 13 team). Correction to brief: the extractor already places every capture (here/client/global, extract.go:280) since v0.4.0-31; 'here' is pushed, and it has no personal/other-project place, so 21 of 570 placed captures leaked with reasons like 'Specific local repo path'. Leak/withheld (Wilson 95%): A0 today 6.7%/0.2%; A+ extractor with personal+other places (labeler proxy, hypothesis) 0.1% [0,0.7]/4.1%; B rule gate at sync 2.4% [1.5,3.8]/11.1%; C agent-explicit only (mem_ ids) 18.4% [9.2,33.4]/96.2% - agents' deliberate remembers leaked most; A+ and B together 0/699 observed, 13.7% withheld. Undo after push is impossible: forget keeps text (22 retired lines still carry it), history keeps it, MergeImport does not propagate deletions. DESIGN.md not edited; Jørgen picks.
file:OPTIONS.md (9890 bytes, sha256:adf62c91)
crab 18:55 local, review: accepted as the options packet for Jørgen. Key facts it redeems: the extractor already places (extract.go:280: here|client|global, 'when unsure choose here'); 58/868 (6.7%) on public 1abcc497 don't belong (29 personal, 16 other, 13 team) after reading all 90 non-repo + 63 suspects; leak/withheld A0 6.7%/0.2%, A+ 0.1%/4.1% (hypothesis), B 2.4%/11.1%, C 18.4%/96.2%, A+&B 0/699/13.7% (hypothesis). Push cannot be undone (forget keeps text; history; MergeImport copies). 30 paid Flash calls. Recommendation to Jørgen: A+ and B together, then verify A+ on fresh captures before any repo opts in.
file:screen-2036.ans (12706 bytes, sha256:f159358e) screen capture from hug tui
crab 20:40 local: Jørgen 19:40Z chose A+ and B together ('unless someone clever can dream up something better'). Rest of check 1: write the decision into DESIGN.md (piglet, after ykta5g), then file the build task. He didn't answer the 58 memories already on public hug/memory, so it gets its own task and question (one per ask).
check 1: 8a1b0c05 on main 939636ff (branch gav9nw-design, ci/gav9nw CI 44386929067): docs/DESIGN.md gains '## Personal vs shared memories (gav9nw, decided 2026-11-18)': Decision (A+ and B together, Jørgen 19:40Z, confirm A+ on fresh captures before any repo opts in), Reason (push cannot be undone; 58/868 on 1abcc497 don't belong; extractor already places, no personal/other place), How (A+ places; B gate beside LooksSecret, term list in local config; 0/699 leak, 13.7% withheld, marked as hypothesis since the labeler stands in for the extractor; confirmation = rnpdnm check 3), Not taken (C 7/38 18.4% [9.2, 33.4] leak and 96.2% withheld; A+ alone 1/778; B alone 18/738 2.4% [1.5, 3.8] and 11.1% withheld; A0 58/866). Open question 2 removed; the 58 are q0a8rx, the build rnpdnm.
check 1: DESIGN.md '## Personal vs shared memories (gav9nw, decided 2026-11-18)' in 8a1b0c05 (CI 44386929067 green): decision A+ and B, reason, how, the 0/699 marked as hypothesis, fresh-capture confirmation before any opt-in, not taken C / A+ alone / B alone / A0 with numbers
crab 22:24: landed on main 4d7eec2d (CI 44409320072 8/8), released in v0.7.71 (tag on edc97cab, release.yml 44411153821)
## History
- 2026-11-18T18:53 todo -> doing/active (take)
- 2026-11-18T18:53 doing/active -> doing/waiting (wait)
- 2026-11-18T18:54 doing/waiting -> doing/asked (ask); was waiting_asked_by: piglet; waiting_on: crab review, then Jørgen picks an option; waiting_since: 2026-11-18T18:53
- 2026-11-18T20:36 request 1 by jhf
- 2026-11-18T20:36 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:37 request 1 acknowledged by crab
- 2026-11-18T20:37 resolved request 1 by crab
- 2026-11-18T20:40 answered asked (check 1), by jhf, for crab (doing/asked -> doing/active):
  > Which way do memories get placed before any repo shares? Measured on hug's 868 public memories (OPTIONS.md attached): 58 (6.7%) don't belong (29 personal, 16 other projects, 13 team). Leak / hug-memories-kept-local: today 6.7% / 0.2%; A+ extractor gets personal + other-project places 0.1% / 4.1% (estimate); B fixed rule at sync 2.4% / 11.1%; C only deliberate remembers 18.4% / 96% (agents' own remembers leaked most). A pushed memory can't be taken back. I recommend A+ and B together (0 of 699 leaked, 13.7% kept local), confirmed on fresh captures before any repo opts in. And the 58 already on the public branch: leave them, or rewrite the branch without them (backup exists)?
  So it seems that A plus and B is the right choice unless someone clever can dream up something better at this moment.
- 2026-11-18T20:40 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-18T20:40 answer 1 acknowledged by crab
- 2026-11-18T20:40 crab read the answer: hug task note gav9nw "crab 20:40 local: Jørgen 19:40Z chose A+ and B together ('unless someone clever can dream up something better'). Rest o…"
- 2026-11-18T21:03 doing/active -> doing/waiting (wait)
