# qccpka trace: why 15 captures were in personal scope with extract.wide_places unset

piglet (claude-opus-5-5), 2026-11-19 01:37-01:41 UTC. Read-only. No store, code or config change.

## Verdict

Capture did not put them there. All 15 were captured into hug's project scope, project:rc-0ca4bf6be5d989c3.
At 2026-11-18T19:49:51Z, a deliberate batch rescope for q0a8rx (mizaru, under crab's ruling B at 19:49Z) moved them to personal.
That batch moved 58 memories off hug's public hug/memory branch: 45 to personal, 13 to global.
Capture placement (places.go) is not involved. wide_places being off is consistent with what happened.
There is no code bug.

The real finding is the opposite. In the 66ex4y daily (01:26Z today), I undid 9 of those deliberate moves without knowing.
I rescoped dpvcdm, wwqfa8, 8z7757, ghtqen, y4061x, t4q0t0, 8e91jw, t2tp92 and a0yarc back into project:rc-0ca4bf6be5d989c3.
q0a8rx had decided they belong in personal, because they name private paths or machine state, or are about observe-only Seid and Statbus.
They must not be back in hug's project scope. That scope syncs to a public branch once sharing is turned on.
The live store still holds my wrong move, and crab's order forbids me to touch it now. Undoing it is for crab to order (see "Consequence").

## Worked example: dpvcdm (cap_802d0ca44e177044)

1. **Created by capture into the project.**
   - capture_runs id 7460 (2026-11-18 05:47:57Z, session_hatchling_1794905151586, trigger session_end, binding scope jcode@global) has 72 actions.
   - All 72 are `place: here | scope: project:rc-0ca4bf6be5d989c3 | action: add`. None are personal.
   - The dpvcdm row is capture_actions run_id 7460, position 1:
     `{"place":"here","place_reason":"Worker workflow for the hug repo.",...,"content":"hug workers use their own worktree under $JCODE_SCRATCH_DIR from origin/main, never ~/Projects/hug, ...","scope":"project:rc-0ca4bf6be5d989c3","action":"add","reason":"new memory in bound scope","origin":"capture:jcode:session_hatchling_1794905151586_536fb261929c9ac4:db6fc6b10bf4:1"}`
   - Code path: capture.go:437 `action.Scope = destination.Scope`. The destination comes from places.go:160 capturePlacement.
     wide is false, so it returns `b.Placement("", "here")`. In ~/Projects/hug, placementScopes (places.go:48) gives here = the project scope.
2. **Still in the project at 19:49:51Z.**
   - The rescope backup tidy/rescope-20261118T194951Z-4179401420/1-memories.json is the store just before the move.
   - All 15 are there in project:rc-0ca4bf6be5d989c3, with `updated` 05:48-12:01Z.
3. **Moved to personal by the q0a8rx batch.**
   - /Users/jhf/.jcode/scratch/q0a8rx/moves.tsv line 62-63:
     `# personal cap_802d0ca44e177044: private paths and machine setup` / `dpvcdm personal`.
   - q0a8rx/rescope.txt: `rescoped 45 to personal / rescoped 13 to global / rescoped 58 memories / backup: ...rescope-20261118T194951Z-4179401420`.
   - The only writer of `rescope-<UTC>-` backups is rescope_batch.go:154 (`hug mem rescope --from`).
4. **Comparison of the two backups.**
   - 19:49:51Z backup against my 01:26:55Z backup: 58 memories changed scope (45 project to personal, 13 project to global).
   - 42 of the 45 carry `updated` 2026-11-18T19:49:51.
5. **The personal row in today's capture_actions is a reinforce, not a placement.**
   - run 8355, 23:37:26Z, kikazaru, position 15: `"action":"reinforce","target":"dpvcdm","scope":"personal","place":"scope"`.
   - capture.go:438-446: a reinforce takes the target's existing scope (`action.Scope = target.Scope`, then `action.Place = b.PlaceForScope(...)`). That gives "scope" because personal is not one of the binding's places.
   - So it reports where dpvcdm already was. It does not choose personal.
   - Among capture_actions since 2026-11-18T02:31Z, this is the only row with scope personal.

## The 15, with q0a8rx's reason (moves.tsv) and what 66ex4y did

| sid | q0a8rx label: reason | 66ex4y (01:26Z) |
|---|---|---|
| dpvcdm | personal: private paths and machine setup | rescoped back to project (wrong) |
| wwqfa8 | personal: names home path working-tree notes file | rescoped back to project (wrong) |
| 8z7757 | personal: names home path ~/Projects/hug | rescoped back to project (wrong) |
| t4q0t0 | personal: names private ~/Projects path | rescoped back to project (wrong) |
| t2tp92 | personal: owner home scratch path | rescoped back to project (wrong) |
| a0yarc | personal: machine state: dream never ran | rescoped back to project (wrong) |
| ghtqen | other: Statbus is live production, observe only (crab 19:49Z: never into Statbus's scope) | rescoped back to project (wrong) |
| y4061x | other: Statbus needs hug v0.7.38+, sync log paths | rescoped back to project (wrong) |
| 8e91jw | other: where Seid lives, observe-only (crab 19:49Z: never into Seid's scope) | rescoped back to project (wrong) |
| d9afy7 | personal: private worktree path | forgotten as a duplicate of dpvcdm (fine either way) |
| 1z7e3x | other: which hug build Seid runs | forgotten as stale (fine either way) |
| nrd8m1 | other: H3 runner's load gate on this machine | left personal |
| tg629m | personal: rule about his machine's load | left personal |
| zq3faj | other: Seid board counts | left personal |
| 69q162 | personal: owner machine python/PIL setup | left personal |

## Against hpqt99's 1B rule (the scope the source states wins)

1B-amended is about imports: a source that states a scope keeps it.
The source here is a person's ruling, recorded in a moves file: Jørgen at 19:42Z, then crab's ruling B at 19:49Z.
That is a stronger statement of scope than a folder. The personal scope is correct and should stand.
The 66ex4y rescope placed these memories by folder (written in ~/Projects/hug, so hug's), which is exactly what 1B says not to do when the scope is stated.

## Consequence (for crab; I changed nothing)

- **Undo of my 9 moves:** `hug mem rescope --from FILE` with the 9 lines `<sid> personal`. Or restore them from the 01:26:55Z backup:
  tidy/rescope-20261119T012655Z-3457466500 holds the store just before my move, so for these 9 rows it holds personal.
- **Leak risk now:** none while share is off. `hug task memory` prints "project memories: local (hug.toml [memory] share is not set; hug mem sync pushes nothing)". `hug mem sync --held` says "155 project memories stay on this machine (placement gate ...)".
  Not checked: whether the gate would also hold these 9 if share were on.
- **qccpka's acceptance:**
  - Check 1 is answered by this file.
  - Check 2 (a test that wide_places off never lands in personal) has no bug to pin. Capture never placed them in personal, so a test would pass on today's main.
  - qccpka can be dropped or reworded. That is crab's call.
- **For the daily method (66ex4y):** before moving a memory's scope, read memory_fixes, the tidy/rescope-* backups and the scratch move files.
  A scope that is not where capture put it was probably moved there on purpose.
  hug mem stats --fixes could show rescopes, but neither batch rescope (19:49:51Z with 58 moves, nor mine at 01:26:55Z with 9) wrote a fix row.
  memory_fixes holds 25 rows: forget 24 (2026-11-18 19:09:17 to 2026-11-19 01:26:50) and supersede 1 (01:27:07). None is a rescope.
  - Not verified: whether rjk44w meant to cover rescope (its CHANGELOG names forget, remember, reinforce and supersede).
