---
id: scrxak
title: "Files in a task: Space shows a file inline, n/p step, Esc or u back"
state: "done"
type: feature
priority: "medium"
assignee: ["nautilus"]
labels: []
group: ""
rank: "zzzzzzzzzz4"
created: 2026-11-17
updated: 2026-11-17
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
done_at: "2026-11-18T00:40"
done_release: "v0.7.51"
---

## Goal
In a task's FILES section, Space shows the selected file in the detail pane, read-only, with a header 'path · size · 3 of 26'. j/k and PgUp/PgDn scroll, n/p show the next and previous file of the task, and Esc or u go back to the FILES list with the file you last saw selected. Enter still edits and E still opens the folder. A binary file, or a file over 1 MiB, shows one line instead of content ('binary, 32.6 KiB: Enter opens it'). Long lines wrap, tabs expand, control characters are escaped.

## Purpose
Jørgen 2026-11-17 11:05, looking at 26 evidence files under backlog-evidence/TASK-2034.02/ in a Seid task: to see what an evidence file says, he now leaves hug for the editor. That costs focus. Glancing through a few files is the common case, and editing them is rare.

## Principle
A glance, not a viewer: plain text only, no syntax highlighting, no Markdown rendering, no search, no editing. Same keys as elsewhere (j/k scroll, n/p step, Esc/u up one level, as 9xfnwr). Read the file only when it is shown, never on list or detail load. Never let a file's bytes reach the terminal raw.

## Relevant details
FILES section: huginn/internal/tui/detail.go:421 (hint 'j/k or [ ] step · Enter edits · E folder'). Done task zgr01t added the files section, the icons and Enter to the editor. Detail footer today: 'j/k scroll · n/p task · Esc back'. Inside the file view, n/p step files; outside it they step tasks. File reads go through the existing file-content checks (binary, too large).

## Depends on

## Acceptance
- [x] Space on a file shows its text in the detail pane with the header 'path · size · i of N'; Esc and u return to FILES with that file selected (golden at 80 and 140 columns)
- [x] n/p step to the next and previous file, stopping at the ends with a footer note; j/k and PgUp/PgDn scroll (test)
- [x] a binary file and a file over 1 MiB show one line with the size and 'Enter opens it', not their bytes; ESC sequences and control characters in a text file are shown escaped (test with a file holding \x1b[2J)
- [x] opening the detail and the list reads no file contents; showing a file reads only that file (test counts reads)
- [x] Jørgen looks through the TASK-2034.02 evidence on Seid with Space and n/p, and says it works

## Notes


nautilus 14:08: built on nautilus/scrxak, commit 0ea1611 from main 5b80e87, CI on ci/scrxak. Not on main. Check 1: TestGlanceShowsFile, golden huginn/internal/tui/testdata/detail/glance.txt at 80 and 140 (header 'compiler/000001/bench/README.md · 32 B · 1 of 5'; Esc and u back to Files with '> ⎘ evidence/escapes.log' marked). Check 2: TestGlanceStepsAndScrolls ('first file: 1 of 5', 'last file: 5 of 5', j/k/PgDn/PgUp offsets). Check 3: TestGlanceNeverPrintsRawBytes (no \x1b[2J, BEL or CR on screen; shown as \x1b[2J and \a; 'binary, 8 B: Enter opens it'; 1 MiB+1 file not read: 'too large to show here, 1.0 MiB: Enter opens it'). Check 4: readFile counted, 0 reads on list and detail, 1 per file shown.
nautilus 16:18: landed on main af51fc3 (9b27bf7..af51fc3, rebased from 0ea1611; CHANGELOG under Unreleased), local go test tui ok 81.4s, no golden changed. 1: TestGlanceShowsFile at 80 and 140 + golden detail/glance.txt (header path · size · i of N; Esc/u back to FILES with it marked). 2: TestGlanceStepsAndScrolls (n/p stop at the ends with a footer note; j/k and pages scroll). 3: TestGlanceNeverPrintsRawBytes (\x1b[2J escaped, binary and >1 MiB one line 'Enter opens it'). 4: TestGlanceShowsFile counts readFile calls: list and detail read none, showing reads only that file. ci/scrxak and wip/scrxak deleted. Check 5 is Jørgen's.
landed af51fc3 on main 2026-11-17: hug tui: Space on a task file shows it in place, read only (path · size · i of…
shipped in v0.7.43 (af51fc3)
landed 0cd10ba on main 2026-11-18: hug tui: the file viewer's footer spells out n next file · p previous file…
Check 5: Jørgen answered in the TUI 00:08: approved, asking that the footer spell out n/p. Done: landed f5576132..1fda4f82 (0cd10ba footer 'j/k scroll · n next file · p previous file · …' and status line 'n next file · p previous file · Esc back to files', asserted in TestGlanceStepsAndScrolls; golden testdata/detail/glance.txt reviewed; 28c656d stageBoard clock pin). Local tui ok 45.0s, -race tui ok 103.7s. (nautilus)
Reflection (dismissed, 2026-11-18, nautilus): Jørgen's answer carried a small request; doing it before closing kept the outcome whole. 'n/p step' in the goal read clearly to us and not to him: footers should spell keys out. (nautilus)
shipped in v0.7.51 (0cd10ba)
## History
- 2026-11-17T15:02 todo -> doing/active (take)
- 2026-11-17T15:08 doing/active -> doing/waiting (wait)
- 2026-11-18T00:08 answered waiting: "check 5: look through the TASK-2034.02 evidence on Seid with Space and n/p in a task's FILES (on main af51fc3)" -> "This worked well. I tried it inside of Seid I do think that the markdown formatting will make sense at a later point. The space view the file N and P switch between files and yeah I don't think people who doesn't know this would know that NNP are next and previous, so you might need to write it out to make the footer a little bit longer. This is good approved and I will file more changes later when I find the need", by jhf, for nautilus (doing/waiting -> doing/active)
- 2026-11-18T00:40 nautilus read the answer
- 2026-11-18T00:40 doing/active -> done (done); was doing_since: 2026-11-17T15:02
