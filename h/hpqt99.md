---
id: hpqt99
title: "Design: imported data pass"
state: "doing"
type: spike
priority: "low"
assignee: ["crab"]
labels: ["design"]
group: "design"
rank: "zz"
created: 2026-11-10
updated: 2026-11-10
doing_since: "2026-11-18T23:18"
stage: "asked"
model: "claude-opus-5-5"
branch: "hpqt99"
worktree: "/Users/jhf/.jcode/scratch/hpqt99"
asked_since: "2026-11-20T09:56"
asked_by: "crab"
asked_check: "1"
asked_what: "Imported data (design.md in Files, Space shows it): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B."
asked_for: "jhf"
---

## Goal
Decide client provenance for imports and live-read vs import of jcode memories

## Purpose
Jørgen 2026-11-05 17:00Z (crab session), in the same breath as personal vs shared (gav9nw): 'Imported data. Okay, we have to do that again later and write into the to-dos' (memories 5zwpx7, 1c49sh). Two points are open. (1) Imported data (e.g. Engram) does not record client origin, so it becomes client-only only through the explicit cross-client admin rescope (fqdv6k, 2026-11-06; muninn/docs/ROADMAP.md). (2) jcode memories are both read live and imported, and Jørgen wants muninn to own all memories eventually: import jcode memories once, stop jcode writing, stop reading jcode files live, so forget and sync work on one source of truth (b97h8c).

## Principle
A design decision first, written down, no code in this task. Keep the client boundary that shipped 2026-11-05: imports and sync can never bypass it, and harness files stay read-only (ROADMAP.md). The direction is Jørgen's (one source of truth in muninn); the task decides how and when to get there, and he makes the call.

## Relevant details
The design proposal is attached as design.md (moved out of docs/DESIGN.md on branch hpqt99 at 32ed02dd; it never reached main). Once Jørgen decides, the decision, not the proposal, goes into docs/DESIGN.md.

Today: hug setup turns jcode's own memory off per instance and routes saves to muninn's MCP, while jcode's existing memory files stay and are indexed read-only (muninn/internal/harness/jcode_takeover.go). A jcode memory both read live and imported shares one short id; recall shows it once and forget removes both (1tz1yn). Only admin 'rescope <id> --to <scope> --cross-client' moves memories across client boundaries.

## Depends on

## Acceptance
- [ ] decision in DESIGN.md

## Questions
- 2026-11-18T23:30 question 1 by piglet to jhf (check 1):
  Imported data (docs/DESIGN.md, 'Imported data', branch hpqt99 9c8fb6a1). Recommend both: (1B) an import places a memory by the folder it was written in, so a memory written under a client folder lands in that client's scope (narrowing only; one leak today: 2kb574, written in ~/Projects/frogs, sits in global). (2B) import each jcode instance's memories once (17,089 rows; jcode files frozen since 2026-11-05), gated by a dry-run whose per-scope counts must match the live read, then stop reading jcode files live. Approve 1B+2B, or name another option (1A/1C, 2A/2C)?
  withdrawn 2026-11-18T23:32 (doing/asked -> doing/active)
- 2026-11-18T23:32 question 2 by piglet to jhf (check 1):
  Imported data, see docs/DESIGN.md section 'Imported data' (branch hpqt99, 32ed02dd): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.
  withdrawn 2026-11-20T09:56 (doing/asked -> doing/active)
- 2026-11-20T09:56 question 3 by crab to jhf (check 1):
  Imported data (design.md in Files, Space shows it): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.

## Requests
- 2026-11-20T09:55 request 1 by jhf:
  So here you're referencing a file design.md which happens to have the details for this very task inside the file that is now part of our documentation, which makes no sense. It should have been an attachment to this ticket and I can't navigate to this design file from the ticket. So I'm going to break up my whole work life to go to another editor, open it, load the file and so on. So I think we have to one be able to reference files and open them from the ticket with the embedded viewer and two be able to navigate to them when I jump to different anchors and three it this specific document should have been attached to this ticket not be part of general documentation and then I can look at this ticket again

## Notes


Design pushed: 9c8fb6a1 on hpqt99 + ci/hpqt99 (docs/DESIGN.md, 'Imported data'). Read-only counts on this machine 2026-11-18 22:25Z: 17,089 live jcode rows / 128 files (none newer than 2026-11-05 22:04), 14,258 native; 0 native rows from 'import jcode'; 25 tidy keepers share a sid with a live row; 8,758 jcode rows under client roots (brovs 5,992, ssb 2,766); engram 3,703 obs, 1,526 written under a client root, 1,525 in client scope, 1 leak (2kb574, frogs -> global). Recommend 1B + 2B; asked jhf on check 1.
Amended before answer (crab review): 1B now keeps a scope the source states (engram personal stays personal/global, never narrowed by directory); directory placement only when the source states no place (engram project, jcode project files). 2kb574 is NOT a leak: Jørgen's writing rule ('lead with the positive'), engram personal, injected into 6 sessions, stays global. Counts of the 1,526 client-root engram rows: 1,525 project (all already in client scope: ssb 903, frogs 341, brovs 282; 1B-amended leaves them), 1 personal (2kb574; 1B-amended keeps global). All engram: 11 personal, 3,692 project. Commit: see hpqt99 branch head.
file:design.md (7506 bytes, sha256:2480b561)
## History
- 2026-11-18T23:18 todo -> doing/active (take)
- 2026-11-18T23:30 doing/active -> doing/asked (ask)
- 2026-11-18T23:32 doing/asked -> doing/active (resume); was asked_by: piglet; asked_check: 1; asked_for: jhf; asked_since: 2026-11-18T23:30; asked_what: Imported data (docs/DESIGN.md, 'Imported data', branch hpqt99 9c8fb6a1). Recommend both: (1B) an import places a memory by the folder it was written in, so a memory written under a client folder lands in that client's scope (narrowing only; one leak today: 2kb574, written in ~/Projects/frogs, sits in global). (2B) import each jcode instance's memories once (17,089 rows; jcode files frozen since 2026-11-05), gated by a dry-run whose per-scope counts must match the live read, then stop reading jcode files live. Approve 1B+2B, or name another option (1A/1C, 2A/2C)?
- 2026-11-18T23:32 doing/active -> doing/asked (ask)
- 2026-11-20T02:45 handed to crab: piglet's session went silent (last seen 2026-11-19 02:41)
- 2026-11-20T09:55 request 1 by jhf
- 2026-11-20T09:55 request 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T09:55 request 1 acknowledged by crab
- 2026-11-20T09:55 request 1 in progress by crab: hug task attach hpqt99 /Users/jhf/.jcode/scratch/hpqt99-design.md
- 2026-11-20T09:56 doing/asked -> doing/active (resume); was asked_by: piglet; asked_check: 1; asked_for: jhf; asked_since: 2026-11-18T23:32; asked_what: Imported data, see docs/DESIGN.md section 'Imported data' (branch hpqt99, 32ed02dd): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.
- 2026-11-20T09:56 doing/active -> doing/asked (ask)
