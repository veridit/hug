# gav9nw: personal vs shared memories, three options

For docs/DESIGN.md open question 2. Jørgen picks first. DESIGN.md is not edited here.

Data: the 868 memories on public `origin/hug/memory` at 1abcc497 (822 `cap_` captured,
38 `mem_` written by an agent with `hug mem remember`, 8 `imp_` imported; 846 active, 22 retired).
Scripts, labels and logs are in this folder: `label.py`, `measure.py`, `labels.json`,
`hand.json`, `hand90.json`, `measure.json`, `logs/`.

## What happens today (correcting "today nothing places a memory")

The capture extractor already places every candidate. Since muninn v0.4.0-31
(2026-11-07, capture run 979 of 7,932), each candidate gets `place` = here | client |
global and a `place_reason` in the same paid call (`muninn/internal/extract/extract.go:280`,
`muninn/internal/app/places.go`). In a git repo, `here` is the project scope, and
`SyncBranch` pushes every line in the project scope (`muninn/internal/project/sync.go:164`).
A personal scope exists (s755vd: Engram memories with no project default to it), but
the extractor is not offered it: the only way out of the repo is `global`, and the prompt
says "When unsure choose here". Across this machine's placed captures the extractor chose
here 5,224, client 265 and global 148 times.

So the real gap is not "nothing places a memory". The places are missing personal and
other-project, and the default pushes.

## How it was measured

- **Blind labeler:** DeepSeek Flash through jcode, no tools, credentials redacted,
  30 memories per call. It saw only the memory text. It did not see the category, the
  extractor's place, or which option was being scored. Labels: repo (about hug),
  personal (the person, his machine, paths, his other work), other (another
  project/client), team (cross-project agent process). **30 paid calls, cap 100**
  (`calls.json`). 29 batches plus 1 retry.
- **Hand check 1: random 30** (seed 20261118, `hand.json`). The labeler agrees with my
  labels on 30/30 (0 errors, 95% CI [0, 11.4]%). Correction: my first hand label for
  cap_3bd670 ("Hug worker protocol: DM the coordinator...") was team. I changed it to
  repo after seeing the disagreement, applying the rule I used everywhere else (a worker
  rule that names hug is repo). Unrevised, agreement is 29/30. The random 30 hold
  almost no non-repo memories, so they cannot measure leaks.
- **Hand check 2: every non-repo label plus suspects** (`hand90.json`). I read all
  90 memories the labeler kept off hug's branch, and the 63 repo-labeled memories a
  marker regex flagged (/Users, ~/, seid, statbus, Jørgen, "User wants", ...).
  - I overturned 34 of the 90: 33 to repo, 1 repo to other. Most of the 33 were hug
    worker rules that name hug, and hug facts measured on the Seid board. I overturned
    none of the 63 suspects. Borderline cases I kept as repo: hug facts measured on the
    private Seid board, and "seid-lang is a private repo, so the public README should
    not link to it". A stricter reader would move those few to other.
  - Corrected truth: repo 810, personal 29, other 16, team 13. So **58 memories (6.7%)
    on the public branch do not belong there**. The remaining 715 repo labels are
    unchecked. That biases option A+'s leak count downward (see below).
- Scores against the corrected labels. Leak = pushed but not hug's. Withheld = hug's
  but kept local. Intervals are Wilson 95%.

## Option A: the extractor places at write time (today's mechanism, widened)

**Who places:** the capture extractor, in the call it already makes (no extra paid call).
Agent-written memories place by the `--place` flag that `hug mem remember` already has.

**Scopes:** add `personal` (the person, his machine, home paths: never in a repo) and
`project:<other>` / `client:<name>` (resolved from the catalog `hug mem tidy place`
already builds, `tidy/place_catalog.go:34`). `here` stays the repo. Team rules go to
`global` until a team scope exists.

**Measured:**

| variant | pushed | local | leak | withheld |
|---|---|---|---|---|
| A0 as run (pre-placement captures pushed) | 866 | 2 | 58/866 = 6.7% [5.2, 8.6] (29 personal, 16 other, 13 team) | 2/810 = 0.2% |
| A as run, placed captures only | 570 | 2 (+296 never placed) | 21/570 = 3.7% [2.4, 5.6] | 2/551 = 0.4% |
| A+ with personal/other places (approximated by the labeler) | 778 | 90 | 1/778 = 0.1% [0.0, 0.7] | 33/810 = 4.1% [2.9, 5.7] |

The 21 "here" leaks show the cause in the extractor's own words. It recognised what
the memory was about and still picked here, because nothing else was offered:
- "Seid lives at /Users/jhf/Veridit/src/seid-lang..." gave "Specific local repo path and
  access mode for this project".
- "Statbus is live production: observe only..." gave "Applies to the Statbus production
  system".
- "muninn's dream has never run on this machine" gave "Observation about this machine's
  hug/muninn setup".

**A+ is a hypothesis, not a measurement of the extractor.** Its row is the blind
labeler's verdict with a four-way choice. The real extractor sees the transcript but
also has the "when unsure choose here" bias. Its leak is scored against labels I
corrected only where they were suspect, so 0.1% is a lower bound. Confirming it means
building the places and measuring fresh captures. That cost is zero extra calls per
capture, plus one blind check like this one (~30 calls).

**Wrong placement:** a memory placed `here` wrongly is pushed at the next sync. See
"Undo after push" below: it cannot be taken back. A memory placed personal wrongly only
costs recall in other clones. `hug mem rescope` moves it into the project later, and
the next sync pushes it.

## Option B: a deterministic rule decides what may leave the machine

**Who places:** nobody new. A local rule (no model) refuses to push a line that names
a home path (`/Users/`, `~/`), another known project or client (Seid, Statbus, Brovs,
...), the person (Jørgen, jhf, "the owner"), or has category preference/correction.
Everything else in the project scope pushes.

**Scopes:** unchanged. The rule is a gate at sync, like today's secret gate
(`sync.go:169`, `LooksSecret`), so it also covers memories written before the rule.

**Measured:** pushed 738, local 130. Leak 18/738 = 2.4% [1.5, 3.8] (7 team, 6
personal, 5 other). Withheld 90/810 = 11.1% [9.1, 13.5].
- Its misses are memories with no marker word: "The background shell's python3 lacks
  PIL", "Hug workers must not use desktop input or Chrome" (team, by the labeler).
- Its false holds are hug decisions credited to Jørgen ("Jørgen's rule for hug: many
  agents writing at once is normal...").

**Wrong placement:** same as A. A held line can be released by rescope or a rule
exception. A leaked line cannot be taken back.

**Cost:** zero paid calls. The term list must be maintained (new clients, new names),
and a list that names clients is itself not something to commit to a public repo. It
would live in local config.

## Option C: only an agent's deliberate remember is shared

**Who places:** the agent, explicitly: `hug mem remember --project` shares, and
everything captured stays local (personal store) until an agent or Jørgen promotes it
(`hug mem rescope ID --to project:...`).

**Scopes:** local by default (personal), project only by an explicit act. Team stays
out of scope.

**Measured (approximated by id prefix: `mem_` = agent-written):** pushed 38, local
830. Leak 7/38 = 18.4% [9.2, 33.4] (5 team, 1 personal, 1 other). Withheld 779/810 =
96.2% [94.6, 97.3].
- The deliberate writes leak *more* often than captures. The 7 are coordinator rules
  such as "Coordinator (crab) todo cleanup removes worker lane items..." and
  "Jørgen's global jcode swarm default is swarm_model claude-opus-5-5...
  (~/.jcode/config.toml)".
- This is the measured answer to Jørgen's condition ("the agent placing a memory
  makes the proper personal vs repository call"). On this branch the agents that
  placed deliberately did not.

**Cost:** zero paid calls. Nearly nothing is shared, so clones and other agents lose
96% of hug's project memory unless someone promotes it by hand.

## Combination measured

A+ and B together, pushing only when both say push: 699 pushed, **0 leaks** in the
corrected labels, 111/810 = 13.7% withheld. This is the only arrangement with no
observed leak, and it remains a hypothesis for the same reason as A+.

## Undo after push (all options)

A pushed memory cannot be taken back without rewriting public history:
- `hug mem forget` keeps the line with its full text and `active: false`. All 22
  retired lines on 1abcc497 still carry their text.
- Rescoping out of the project drops the line from the next `memories.jsonl`, but git
  history keeps every earlier commit (46 on this branch).
- Other clones keep their copy: `MergeImport` does not propagate deletions
  (`ingest/importer.go:70`, "an import is a copy, not a mirror").

So every option is judged by its leak rate, and withheld memories are the cheap
error. Removing what is already on 1abcc497 (58 lines by the corrected count) is a
branch-history decision for Jørgen, already raised on 34r1ft.

## Relation to 34r1ft

34r1ft (Jørgen chose A: committed `hug.toml [memory] share`, absent = local) decides
*whether* a repo shares at all. This question decides *which* memories a sharing repo
pushes. With 34r1ft as decided, a public repo that never opts in leaks nothing under
any option above.

## Cost summary

| option | paid calls | leak | withheld |
|---|---|---|---|
| A0 today | 0 extra | 6.7% | 0.2% |
| A+ extractor, wider places | 0 extra (same call) | 0.1% (hypothesis, lower bound) | 4.1% |
| B rule gate | 0 | 2.4% | 11.1% |
| C agent explicit | 0 | 18.4% | 96.2% |
| A+ and B | 0 extra | 0/699 observed (hypothesis) | 13.7% |

This measurement: 30 paid Flash calls (cap 100). Hand checks: 30 random plus 90 flagged
plus 63 regex suspects, all read.
