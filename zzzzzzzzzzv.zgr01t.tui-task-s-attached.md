---
id: zgr01t
title: "TUI: a task's attached files show with an icon and open in the editor"
status: "doing"
type: feature
priority: "medium"
assignee: ["vole"]
labels: []
created: 2026-11-16
updated: 2026-11-16
model: "claude-opus-5-5"
---

## Goal
A task with attached files shows a small file mark with the count on its row in every list (e.g. '⎘2'), and inside the task its files are listed where they belong, each opening in the editor with one key; one key opens the task's whole file folder in the editor

## Purpose
Jørgen 2026-11-16: there are no general files, every file belongs to a task, so files are not a view of their own; they are attached to the ticket and should be visible on it. Today the TUI shows nothing but the 'file:NAME (bytes, sha256)' note line; reading one needs 'hug task show ID --file NAME'.

## Principle
No FILES view or global files list. The mark on the row only when a task has files. Inside the task, files are part of the task (a short list under the description, or the existing file: note lines made followable), opened with the editor the same way e opens the task, and hug commits what changed when the editor closes so history stays whole. Key letters by assignShortcuts, shown only when the task has files. Folder: .hug/tasks/<id>/.

## Relevant details
Storage: huginn/internal/task/files.go (Attach, text only, 256 KiB). Example: 44w1ev has gen-scale-board.py. Open question for Jørgen: commit edits made in the editor on close, or open view-only.

## Depends on

## Acceptance
- [x] a task with 2 files shows a file mark with 2 on its row in overview, list and board; a task without files shows none (golden)
- [x] inside the task, the files are listed and Enter on one opens it in $EDITOR; after the editor closes a change is committed (test with a fake editor)
- [ ] one key opens the task's folder in the editor; the key is not shown for a task without files (test)

## Notes

2026-11-16 decided (Jørgen): edits made in the editor are saved and committed when the editor closes, same as e for the task. Open question closed.

commit 60d3f4b. CI run 43931868360 on 60d3f4b (now origin/main): ubuntu, macos, windows success. Goldens testdata/tmux/140x40-no-color: list.txt ' !000001   todo    feature critical ⎘2 Feature task 01: ...'; board.txt '>!000001   F C ⎘2 Feature task 01: make the …'; overview.txt the same row under NEXT UP and LINT; ⎘ appears on no other row (grep -c: overview 2, list 1, board 1, all 000001). Fixture testfixture/seed.go gives 000001 bench/README.md and bench/results.tsv. TestFileMarkOnRows checks overview, list, board: every ⎘ line is 000001 with ⎘2.

Check 2: TestFilesOpenInEditorAndCommit (files_test.go): detail of 000001 lists 'FILES (2)', '⎘ bench/README.md', '⎘ bench/results.tsv'; Tab to Files, j marks results.tsv, Enter runs $VISUAL (the test binary as fake editor, appends '2\t40\n'); after editorMsg the file ends '1\t42\n2\t40\n' and Commit got exactly 'hug task edit 000001 file bench/results.tsv'; message 'editor closed; re-linted'. PASS on 3 OSes in CI.

Check 3: TestFolderKeyOnlyWithFiles: E on 000001 runs the editor on the folder .../000001 (a directory), commit 'hug task edit 000001 files'; on 000002 (no files) neither 'E files' in the key bar nor a FILES section shows, and E starts nothing and says 'this task has no files: hug task attach 000002 FILE'. Help lists 'E  in a task with files (⎘2): its folder in the editor'.
