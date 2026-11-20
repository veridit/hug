# Imported data: design proposal (hpqt99, 2026-11-18; amended 32ed02dd)

Two points stayed open on 2026-11-05: (1) what client an imported memory
belongs to, and (2) whether jcode memories are imported once or read live.
The direction is Jørgen's: muninn owns every memory, one source of truth.
The shipped rules stay: imports and sync never cross a client boundary, and
harness files stay read-only (muninn/docs/ROADMAP.md).

**Recommendation.** 1B and 2B below, in that order: they share one function.

### Facts on this machine (read-only queries, 2026-11-18 22:25Z)

- 31,347 memory rows: 17,089 read live from 128 jcode files, 14,258 native.
- jcode memory is off in all 6 instances (`[features] memory = false` and
  `memory` in `[tools] disabled`). The newest jcode memory file was written
  2026-11-05 22:04: none of the 128 files on disk is newer. The files have
  been frozen for 13 days.
- jcode rows per instance: seid 6,708, brovs 5,992, statbus 2,766, global
  1,001, truth_tribe_and_treasure 315, GenderedGames 307. Of the 17,089,
  16,484 are active and 658 are hidden by forget (`suppressed`), which leaves
  15,832 that recall can show.
- Imported twins are rare. No row comes from `hug mem import jcode`
  (0 native rows with a `jcode@` origin outside tidy). 428 native rows are
  tidy copies of jcode rows (403 `tidy:jcode@...` moves, 25 keepers), and
  only the 25 keepers share a short id with a live row.
- 8,758 live jcode rows (51%) sit under a client root: brovs 5,992 and
  ssb 2,766. They get their client scope at read time, per project
  (`ClientSourceScope`, internal/app/binding.go), 22 and 21 scopes.
- Engram: 3,703 observations, newest 2026-11-15, so engram is still in use.
  1,526 were written under a client root (ssb 903, frogs 342, brovs 282).
  1,525 of them are in a client scope today (how each got there is not
  recorded, only that import alone cannot do it). Split by the scope engram
  itself gives them: 1,525 say `project` and are all in a client scope
  (client:ssb 903, client:frogs 341, client:brovs 282). 1 says `personal`:
  `2kb574`, written in `~/Projects/frogs` on 2026-06-29, sits in `global`.
  It is not a leak. It is Jørgen's rule for how he writes ("lead with the
  positive"), injected into 6 sessions (`injections`), and it belongs
  everywhere.
  Of all 3,703, engram marks 11 `personal` (9 stored `personal`, 2
  `global`) and 3,692 `project`. None has an empty project.
  No engram observation has been written under a client root since
  2026-11-05.

### 1. Client provenance for imported data

Today an Engram import never resolves a client root
(`ImportScopeForDir`, internal/app/importscope.go:37). The row falls back to
`engram:<project>` or `personal`, both visible outside the client, until the
owner runs `rescope --cross-client`. A jcode import does get a client, from
the memory folder's path (internal/app/rescope.go:89-97), but it flattens
every project to `client:<name>`, while the live read keeps one scope per
project.

- **1A. Keep it: the owner rescopes by hand.** Cost: none. Risk: every
  engram `project` observation written in a client folder is visible to
  every other session until someone rescopes it.
- **1B. The source's own scope first, then the directory it was written
  in, client included.** (Amended 2026-11-18 after crab's review: the first
  draft placed every row by directory, which would have narrowed `2kb574`
  into client:frogs and hidden it from every other session.)
  - When the source states a scope, the import keeps it and never narrows
    it by directory. Engram `personal` stays `personal` (or `global` where
    it already is), wherever it was written. The source said who it is for.
  - When the source states no place, the directory is the provenance:
    engram `project` (it means the project it was written in) and a jcode
    project file. If that directory is under a client root, the new row
    gets the scope a live read there would give it (`ClientSourceScope`,
    per project). jcode's own `global.json` under a client root is the
    client's top level, as the live read has it.
  - A row already stored outside the client is not moved: that is still a
    crossing, and stays the admin `rescope --cross-client`. Import lists
    such rows with the exact rescope command, for the owner to judge.
  - A row whose directory is gone or unknown keeps today's fallback.

  What it does with the 1,526 engram rows written under a client root:
  the 1,525 `project` rows are in a client scope already and stay there
  (an unchanged source row is never rewritten). The 1 `personal` row,
  `2kb574`, keeps its scope and stays in `global`. New observations
  follow the same split.

  Cost: about one day. The change is in ImportScopeForDir and
  Binding.MergeImport, plus rows in the alpha/global/client matrix test.
  Risk: a folder that became a client after the memory was written also
  narrows it. That fails closed, and rescope undoes it.
- **1C. Hold for review.** Rows from a client root, or from no known
  directory, go to a scope nobody recalls until the owner places them.
  Cost: about two days, plus a review screen. Risk: they go unseen, and the
  review is one more chore for the owner.

**Recommend 1B.** The evidence is already in the source data: the scope
it states, and otherwise the folder it was written in. 1A leaves every new
client `project` row visible until someone notices. 1C asks the owner to do
by hand what the source already says.

### 2. Import jcode memories once, or read them live

- **2A. Keep reading live.** Cost: none. Risk: two sources of truth for good.
  Forget on a jcode row is a suppression list, not an edit. jcode rows never
  reach `hug/memories` or the private folder in Storage above. muninn keeps
  depending on jcode's file format (`graph_version`, `memories`, `edges`).
- **2B. Import once per instance, then stop reading it live.** For each
  instance: `hug mem import jcode --dry-run` prints row counts per effective
  scope, for the live index and for the import, and they must match exactly.
  Then the import writes native rows with the same short ids and the
  per-project client scopes of 1B. Suppressed rows import as retired
  (active false), and tidy copies are matched by origin and not duplicated.
  After that the indexer skips that instance's memory folder. The files stay
  on disk untouched, and `hug setup` can switch live reading back on per
  instance.
  Cost: about two days, including a migration test on a copy of these
  128 files. The native store grows from 14,258 to about 31,000 rows. The
  jcode text is 2.8 MB, so about 6 MB more JSON (estimate, not measured).
  Risk: the scope mapping is the one place a client row could leak. The
  zero-difference dry-run is the gate, and 1B makes import and live read use
  one function.
- **2C. Import on touch.** Keep reading live, and copy a row into the native
  store only when it is edited, forgotten or synced. Cost: about two days,
  spread over every write path. Risk: two sources of truth stay, for an
  unknown time. Every action has to handle both kinds.

**Recommend 2B.** The files have been frozen since 2026-11-05 because
`hug setup` turned jcode memory off everywhere, so one import is a true
snapshot and nothing is lost after it. Imported twins barely exist (25), so
the twin handling in tidy is not a reason to wait. The later move of native
rows to files (Storage above) carries these rows along with the rest.
>>>>>>> Stashed changes

