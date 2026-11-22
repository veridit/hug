---
id: cvbpdy
title: "Lint owns evidence rules: (lost) marker, path citations checked, setup offers them"
state: "done"
type: feature
priority: "high"
assignee: ["hatchling"]
labels: []
group: ""
rank: "zzzzzzzzzzbh"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:50"
---

## Goal
Seid can drop tools/evidence.py and its hook runs only hug task lint. hug lint already has forbidden-citation (hug.toml forbid_cite), missing-file and file-content (256 KiB, text only). Missing: (1) a citation followed by ' (lost)' is accepted, both for forbid_cite prefixes and for missing attachments; (2) a plain path under .hug/tasks/<id>/ cited in task text must exist, like file: and task:ID/NAME; (3) forbidden-citation names the exact cited path and line, not just the prefix; (4) hug setup offers forbid_cite = [".jcode/scratch/"] when the repo has .jcode/, since Seid has no hug.toml today, so the rule is off there.

## Purpose
Seid foreman (orangutan) requests 2026-11-17, file: seid-lang .jcode/scratch/TASK-2057/hug-agent-requests.md, read by crab 15:07, item 1. Jørgen (the King in Seid) wants hug lint to own these rules for every repo.

## Principle
Declarative in hug.toml, no Seid-specific names in code. Every finding has a fix: line an agent can run (hug task attach ID FILE, or mark it (lost)). Lint never deletes anything.

## Relevant details
Seid reference: tools/evidence.py lines 91-112: SCRATCH_REF, EVID_REF, LOST=' (lost)', KEEP_EXT, MAX=256 KiB. hug code: task/files.go LintFiles (line 267; forbid check line 300 uses strings.Contains on the whole doc, so no line number and no (lost) escape), task/config.go ForbidCite. Text-extension list: hug checks content (NUL, utf8), which is stricter than an extension list; keep that.

## Depends on

## Acceptance
- [x] a citation of a forbid_cite path followed by ' (lost)' gives no finding; without it, the finding names the path and line (test)
- [x] a cited .hug/tasks/<id>/NAME path that does not exist is missing-file, and ' (lost)' after it is accepted (test)
- [x] hug setup on a repo with .jcode/ and no forbid_cite offers the line, with a fix: that writes it (test)
- [ ] ~~done when task:xgwwhn is done (on Seid, hug task lint reports what tools/evidence.py check reports (counts in the notes), so its hook can switch)~~ (not relevant: hug's side shipped in v0.7.43; Seid's switch moved to its own task xgwwhn, waiting on orangutan (crab, Jørgen 00:38), hatchling, 2026-11-18)

## Notes

hatchling 15:20, design read (not started; after s6ff5k per crab): files.go LintFiles forbid check (strings.Contains on whole doc, Line 0) becomes a line scan: for each line, each forbid prefix occurrence, take the cited path token (up to whitespace/backtick/paren/quote), skip if the text right after it is ' (lost)'; finding names path and line ('cites .jcode/scratch/X/y.md (a folder that gets deleted)', Line N), fix 'hug task attach ID FILE, or mark it (lost)'. Check 2: a regex for '.hug/tasks/<path>' citations resolves against s.Root; missing -> missing-file unless ' (lost)' follows. Check 3: setup.go offers forbid_cite = [".jcode/scratch/"] when .jcode/ exists and hug.toml has none. Check 4 needs Seid's tools/evidence.py run for counts.

hatchling 16:30, built while the s6ff5k replay ran (not taken: s6ff5k is my doing ticket). Branch ci/cvbpdy 95079e0 on e582ab4 (worktree guppy-p29), not pushed. Checks 1-3 have tests that fail before the code (TestForbidCiteNamesPathAndLine: 'finding "cites .jcode/scratch/ (a folder that gets deleted); attach the file instead" at line 0'; TestTaskPathCitationMustExist: 'missing-file findings: []'); TestSetupOffersForbidScratch runs the offered fix in sh and lints after it (it caught a missing git add for a new hug.toml). Check 4 on a disposable clone of Seid's hug/tasks (3478 files) with forbid_cite added: evidence.py's SCRATCH_REF rule applied to the same files gives 530 unmarked citations on 464 lines; hug lint gives 530 forbidden-citation findings on 464 lines, both on all 464, 0 on one side only (first cut: 817 on 707, because it read /Users/x/.jcode/scratch/ paths and ' (not kept: ...)' marks differently; the rule now matches evidence.py's regex and marks). EVID_REF rule: 2712 .backlog/evidence citations, 0 missing in the Seid repo. evidence.py check on seid-lang itself: 0 errors. Schema 16 (cached refs change). Rule not ported: evidence.py's keepable() (extension list) since hug's file-content check (NUL/utf8, 256 KiB) is stricter, per the ticket's details.


hatchling 16:34, landed 95079e0 on main (e582ab4..95079e0, fast-forward). Local: full go test ./... green on this tree (niced). Main CI 44127183886. Checks 1-3 ticked: TestForbidCiteNamesPathAndLine (1), TestTaskPathCitationMustExist (2), TestSetupOffersForbidScratch (3), each failing before the code (quoted in the 16:30 note). Check 4 stays open until Seid switches its hook.
Before Seid turns on forbid_cite (crab 16:33): adding forbid_cite = [".jcode/scratch/"] to Seid's hug.toml adds 530 forbidden-citation findings on 464 lines (measured on a copy of hug/tasks, 3478 files). evidence.py never read hug's task files (it reads .backlog/tasks only), so these lines exist today and only become visible now. The fix an agent there runs per line: if the file still exists, hug task attach ID FILE and cite it as file:NAME; if it is gone, append ' (lost)' right after the path (' (not kept: too large)' / ' (not kept: run output)' are also accepted). The finding names the path and line; hug task lint --code forbidden-citation lists them.
landed 95079e0 on main 2026-11-17: hug task lint: a citation followed by ' (lost)' is a known gap; forbidden…
hatchling 16:50, main CI 44127183886 on 95079e0: macos, windows, secrets, bench success; ubuntu failure from a data race in huginn/internal/tui TestQuitDoesNotWaitForWatchSetup (start_test.go:170 write vs start.go:98 read of watchStarter), present already on e582ab4 (main CI 44126454244 ubuntu failure) and fixed by d16fb54 on main. The cvbpdy diff touches no tui file.
shipped in v0.7.43 (95079e0)
Reflection (dismissed, 2026-11-18, hatchling): The ticket held work for two repos: hug's lint and Seid's switch. An acceptance item that only another repo's owner can tick kept a shipped task open for a day. Splitting at filing time would have avoided it.
## History
- 2026-11-17T17:33 todo -> doing/active (take)
- 2026-11-17T17:34 doing/active -> doing/waiting (wait)
- 2026-11-18T00:50 doing/waiting -> done (done); was doing_since: 2026-11-17T17:33; waiting_asked_by: hatchling; waiting_on: orangutan: check 4: Seid switches its pre-commit hook from tools/evidence.py check to hug task lint after setting forbid_cite; 530 findings on 464 lines to clear first (note 16:34); waiting_since: 2026-11-17T17:34
