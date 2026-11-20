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
asked_by: "crab"
asked_check: "2"
asked_what: "Decision 1, which client owns an imported memory: 1A keep today (you rescope by hand), 1B keep the scope the source states and otherwise place by the folder it was written in (recommended, ~1 day), or 1C hold client-folder rows for your review (~2 days)? Trade-offs are under Decision 1 in Details."
asked_for: "jhf"
asked_since: "2026-11-20T09:57"
---

## Goal
Jørgen decides which client owns an imported memory, and whether jcode memories are imported once or read live.

## Purpose
Imported memories can land in the wrong client's scope, and jcode memories live in two places, so forget and sync have no single source of truth.

## Principle
Decide first, build later; imports and sync never cross a client boundary.

## Relevant details
Decision 1: which client owns an imported memory.
- 1A Keep today: the owner rescopes by hand. Cost none. Risk: engram project notes written in a client folder are visible to every session until someone notices.
- 1B Recommended. Keep the scope the source states (engram personal stays personal); when it states none, place by the folder it was written in, client included. Cost about 1 day. Risk: a folder that became a client later narrows old notes; fails closed, rescope undoes it.
- 1C Hold for review: client-folder rows go to a scope nobody recalls until the owner places them. Cost about 2 days plus a review screen. Risk: unseen memories and one more chore for the owner.

Decision 2: jcode memories imported once, or read live.
- 2A Keep reading live. Cost none. Risk: two sources of truth for good; forget is a suppression list; muninn depends on jcode file format.
- 2B Recommended. Import each instance once (dry-run must match the live index exactly), then stop reading it live; files stay on disk, setup can switch live reading back on. Cost about 2 days. Risk: scope mapping is the one place a client row could leak; the zero-difference dry-run is the gate.
- 2C Import on touch (copy a row when edited, forgotten or synced). Cost about 2 days across every write path. Risk: two sources of truth for an unknown time.

Facts (2026-11-18, read-only): 17,089 memories are read live from 128 jcode files frozen since 2026-11-05 (jcode memory is off in all 6 instances); 8,758 of them sit under a client root. Engram: 3,703 observations, 1,526 written under a client root, all but one already in a client scope. Full evidence: design.md in Files.

After the decision, only the decision goes into docs/DESIGN.md.


Earlier text:
The design proposal is attached as design.md (moved out of docs/DESIGN.md on branch hpqt99 at 32ed02dd; it never reached main). Once Jørgen decides, the decision, not the proposal, goes into docs/DESIGN.md.

Today: hug setup turns jcode's own memory off per instance and routes saves to muninn's MCP, while jcode's existing memory files stay and are indexed read-only (muninn/internal/harness/jcode_takeover.go). A jcode memory both read live and imported shares one short id; recall shows it once and forget removes both (1tz1yn). Only admin 'rescope <id> --to <scope> --cross-client' moves memories across client boundaries.

Background (the former Purpose): Jørgen 2026-11-05 17:00Z (crab session), in the same breath as personal vs shared (gav9nw): 'Imported data. Okay, we have to do that again later and write into the to-dos' (memories 5zwpx7, 1c49sh). Two points are open. (1) Imported data (e.g. Engram) does not record client origin, so it becomes client-only only through the explicit cross-client admin rescope (fqdv6k, 2026-11-06; muninn/docs/ROADMAP.md). (2) jcode memories are both read live and imported, and Jørgen wants muninn to own all memories eventually: import jcode memories once, stop jcode writing, stop reading jcode files live, so forget and sync work on one source of truth (b97h8c).

Former Principle: A design decision first, written down, no code in this task. Keep the client boundary that shipped 2026-11-05: imports and sync can never bypass it, and harness files stay read-only (ROADMAP.md). The direction is Jørgen's (one source of truth in muninn); the task decides how and when to get there, and he makes the call.

## Depends on

## Acceptance
- [ ] ~~decision in DESIGN.md~~ (not relevant: split into checks 2 and 3, one decision each, asked on the ticket (Jørgen request 3, 2026-11-20), crab, 2026-11-20)
- [ ] Decision 1 (client of an imported memory): Jørgen picks 1A, 1B or 1C
- [ ] Decision 2 (jcode memories): Jørgen picks 2A, 2B or 2C
- [ ] The decision is written in docs/DESIGN.md and the build tasks are filed

## Questions
- 2026-11-18T23:30 question 1 by piglet to jhf (check 1):
  Imported data (docs/DESIGN.md, 'Imported data', branch hpqt99 9c8fb6a1). Recommend both: (1B) an import places a memory by the folder it was written in, so a memory written under a client folder lands in that client's scope (narrowing only; one leak today: 2kb574, written in ~/Projects/frogs, sits in global). (2B) import each jcode instance's memories once (17,089 rows; jcode files frozen since 2026-11-05), gated by a dry-run whose per-scope counts must match the live read, then stop reading jcode files live. Approve 1B+2B, or name another option (1A/1C, 2A/2C)?
  withdrawn 2026-11-18T23:32 (doing/asked -> doing/active)
- 2026-11-18T23:32 question 2 by piglet to jhf (check 1):
  Imported data, see docs/DESIGN.md section 'Imported data' (branch hpqt99, 32ed02dd): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.
  withdrawn 2026-11-20T09:56 (doing/asked -> doing/active)
- 2026-11-20T09:56 question 3 by crab to jhf (check 1):
  Imported data (design.md in Files, Space shows it): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.
  withdrawn 2026-11-20T09:57 (doing/asked -> doing/active)
- 2026-11-20T09:57 question 4 by crab to jhf (check 2):
  Decision 1, which client owns an imported memory: 1A keep today (you rescope by hand), 1B keep the scope the source states and otherwise place by the folder it was written in (recommended, ~1 day), or 1C hold client-folder rows for your review (~2 days)? Trade-offs are under Decision 1 in Details.
- 2026-11-20T09:57 question 5 by crab to jhf (check 3):
  Decision 2, jcode memories: 2A keep reading jcode files live, 2B import each instance once and stop reading live (recommended, ~2 days, gated by an exact dry-run match), or 2C import a row only when it is touched (~2 days)? Trade-offs are under Decision 2 in Details.

## Requests
- 2026-11-20T09:55 request 1 by jhf:
  So here you're referencing a file design.md which happens to have the details for this very task inside the file that is now part of our documentation, which makes no sense. It should have been an attachment to this ticket and I can't navigate to this design file from the ticket. So I'm going to break up my whole work life to go to another editor, open it, load the file and so on. So I think we have to one be able to reference files and open them from the ticket with the embedded viewer and two be able to navigate to them when I jump to different anchors and three it this specific document should have been attached to this ticket not be part of general documentation and then I can look at this ticket again
  resolved 2026-11-20T09:56 by crab:
    (3) Done on this ticket: the proposal is attached as design.md (Files, Space shows it), taken from branch hpqt99 32ed02dd; it never reached main's docs. Details now point to the attachment and say only the decision goes into docs/DESIGN.md. The question is re-asked naming design.md. (1) and (2) are filed as zbdbtm: any file a task names opens from the task in the embedded viewer at its section, Tab stops on every file link, Esc returns; and AGENTS.md tells agents to attach per-task documents.
- 2026-11-20T09:56 request 2 by jhf:
  The purpose for this ticket is also not clear enough. This is too wordy, quoting me. So you can write all those lengthy things in some notes or some details somewhere on the bottom, but try to lead with clarity
  resolved 2026-11-20T09:56 by crab:
    The top now leads with clarity, one sentence each. Goal: you decide which client owns an imported memory, and whether jcode memories are imported once or read live. Purpose: imported memories can land in the wrong client's scope, and jcode memories live in two places, so forget and sync have no single source of truth. Principle: decide first, build later; imports and sync never cross a client boundary. The quotes and history moved word for word to the end of Details.
- 2026-11-20T09:57 request 3 by jhf:
  This ticket should truly write things in the ticket in the details and have maybe the questions with each relevant piece and it is too verbal. It has to be succinct to the point saying for this do that we can choose between so-and-so with these trade-offs and then the questions are inside the ticket not buried inside the document. I can't answer them. I have to do requests to talk about things that are so I can't respond where it happens
  resolved 2026-11-20T09:57 by crab:
    The ticket now carries the decision itself. Details opens with Decision 1 (1A/1B/1C) and Decision 2 (2A/2B/2C), one line per option with cost and risk, recommendations marked, then three lines of facts; design.md in Files keeps the full evidence. The old single check 'decision in DESIGN.md' is split: check 2 is Decision 1, check 3 is Decision 2, each with its own question on the ticket, so you answer each where it sits (a on the check). Check 4 writes the decision into docs/DESIGN.md and files the build.
- 2026-11-20T11:40 request 4 by jhf:
  So this ticket would have been so much better if we had the inline editing system because now you list something one place, you ask questions another place and things are kind of disconnected, don't you think?
  resolved 2026-11-20T11:40 by crab:
    Yes, agreed: options in Details and the questions under checks 2 and 3 are disconnected. That is what p87ngv (inline marks) fixes, and the look and mock are both approved now. This ticket becomes its first real use: p87ngv check 17 rewrites hpqt99 with each question marked on its Decision's options and answered there. If you want to decide before that ships, questions 4 and 5 (under checks 2 and 3) restate the options in full, so each can be answered on its own.
- 2026-11-20T11:41 request 5 by jhf:
  I attach two screenshots where you see all the listings and options A1, A2, A, B, and so on. And then below you see the questions which are so they're disconnected. And when there are multiple options, why wouldn't we just render them inline so I can go and pick the options like it's A, B, C, or other or something? I mean, we're doing a pretty nice interaction system that's in the works now, I think

## Notes


Design pushed: 9c8fb6a1 on hpqt99 + ci/hpqt99 (docs/DESIGN.md, 'Imported data'). Read-only counts on this machine 2026-11-18 22:25Z: 17,089 live jcode rows / 128 files (none newer than 2026-11-05 22:04), 14,258 native; 0 native rows from 'import jcode'; 25 tidy keepers share a sid with a live row; 8,758 jcode rows under client roots (brovs 5,992, ssb 2,766); engram 3,703 obs, 1,526 written under a client root, 1,525 in client scope, 1 leak (2kb574, frogs -> global). Recommend 1B + 2B; asked jhf on check 1.
Amended before answer (crab review): 1B now keeps a scope the source states (engram personal stays personal/global, never narrowed by directory); directory placement only when the source states no place (engram project, jcode project files). 2kb574 is NOT a leak: Jørgen's writing rule ('lead with the positive'), engram personal, injected into 6 sessions, stays global. Counts of the 1,526 client-root engram rows: 1,525 project (all already in client scope: ssb 903, frogs 341, brovs 282; 1B-amended leaves them), 1 personal (2kb574; 1B-amended keeps global). All engram: 11 personal, 3,692 project. Commit: see hpqt99 branch head.
file:design.md (7506 bytes, sha256:2480b561)
file:screen-1140.ans (53232 bytes, sha256:7c1b06a7) screen capture from hug tui
file:screen-1140-2.ans (47336 bytes, sha256:19bfc305) screen capture from hug tui
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
- 2026-11-20T09:56 request 2 by jhf
- 2026-11-20T09:56 request 2 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T09:56 resolved request 1 by crab
- 2026-11-20T09:56 request 2 acknowledged by crab
- 2026-11-20T09:56 request 2 in progress by crab: hug task set hpqt99
- 2026-11-20T09:56 resolved request 2 by crab
- 2026-11-20T09:57 request 3 by jhf
- 2026-11-20T09:57 request 3 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T09:57 request 3 acknowledged by crab
- 2026-11-20T09:57 request 3 in progress by crab: hug task set hpqt99
- 2026-11-20T09:57 doing/asked -> doing/active (resume); was asked_by: crab; asked_check: 1; asked_for: jhf; asked_since: 2026-11-20T09:56; asked_what: Imported data (design.md in Files, Space shows it): do you approve 1B-amended plus 2B? 1B-amended: an import keeps the scope the source states (engram personal stays personal) and places by the folder it was written in only when the source states none. 2B: import each jcode instance's memories once, then stop reading jcode files live. The alternatives are 1A (as today, the owner rescopes by hand) and 2A (keep reading jcode files live). I recommend 1B-amended plus 2B.
- 2026-11-20T09:57 doing/active -> doing/asked (ask)
- 2026-11-20T09:57 asked question 5 of jhf, 2 open (doing/asked -> doing/asked)
- 2026-11-20T09:57 resolved request 3 by crab
- 2026-11-20T11:40 request 4 by jhf
- 2026-11-20T11:40 request 4 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
- 2026-11-20T11:40 request 4 acknowledged by crab
- 2026-11-20T11:40 resolved request 4 by crab
- 2026-11-20T11:41 request 5 by jhf
