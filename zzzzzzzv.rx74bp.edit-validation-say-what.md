---
id: rx74bp
title: "Edit validation: say what is wrong and how to fix it"
status: "doing"
type: bug
priority: "high"
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
After an edit that breaks a task file, the TUI and hug task lint say in plain words what is wrong, on which line, and what to do, and the task is not silently shown as blocked

## Purpose
Jørgen 2026-11-16 tested validation on purpose: he added a 'Depends on' section to qawmnb containing the sentence 'Is there a task:asdfasdf or ticket:asdlfkjasldkfj?'. The result was the header line 'dangling dependency Is', the task turned [blocked], and a dependency row read '↳ Is [blocked] BLOCKED there a task:asdfasdf or ticket:asdlfkjasldkfj?'. Nothing said what a dependency line must look like or how to fix it

## Principle
Every finding ends in something to do. Say it in the user's words, not the tool's ('dangling' means nothing to a reader). A line the tool cannot read is reported as unreadable, not guessed at: it took the first word 'Is' as a task reference and ignored the two task-like words that were there. A malformed line must not change the task's status. The editor flow shows problems right after the edit and offers to reopen the file

## Relevant details
Message source: huginn/internal/task/query.go:246 builds 'dangling dependency ' + dep.Ref; codes in document.go:47. Lint output today: 'zzzzzz.qawmnb.recall-nudges-keep-task.md:26: [dangling-dependency] dangling dependency Is' with a next: line about an unrelated task ('hug task take vk9td0'). hug task dep ID --add REF / --remove REF is the supported way to change dependencies; a hand edit writes '## Depends on' lines, whose expected form the message should show. Cases to handle: a reference to a task that does not exist (say which id, and that no task has it); prose in the section (say the line is not a dependency, show the expected form, do not block the task); the unknown prefix 'ticket:'. TUI: detail.go renders the header warning and the dependency row; after e the TUI refreshes and lints (keys.go help says 'edit in $EDITOR, then refresh + lint'). Wanted in the TUI: the warning names the line and the fix, and a key reopens the editor at that line when the editor supports it. Do after mv4vye lands (same files)

## Depends on

## Acceptance
- [x] a dependency line naming a missing task reports the id and says no task has it, with a fix line that removes it (hug task dep ID --remove REF) or names the form to write
- [x] prose in the Depends on section is reported as 'not a dependency' with the expected form shown, and does not mark the task blocked
- [x] hug task lint's next: line for a finding points at fixing that finding, not at an unrelated task
- [ ] after an edit in the TUI that introduces a finding, the finding is shown with its fix and one key reopens the editor
- [ ] tests cover the three cases above with the exact text from Jørgen's edit

## Notes

check 1 (297bb4b): go test -run TestMissingDependencyNamesIDAndFix ./huginn/internal/task/ passes: finding "depends on zz9zz9, but no task has that id or alias; remove the line, or write an existing task's id", code dangling-dependency, fix: hug task dep k7m2qx --remove zz9zz9

check 2 (297bb4b): /tmp/hugtest task lint on this board prints: zzzzzz.qawmnb.recall-nudges-keep-task.md:26: [not-a-dependency] not a dependency: "Is there a task:asdfasdf or ticket:asdlfkjasldkfj?" (a dependency line is "- <task id>", or run hug task dep qawmnb --add REF; prose belongs in Notes) / fix: hug task edit qawmnb. TestProseInDependsOnIsNotADependency: 0 dependencies, Blocked empty, DependencyDetails empty.

check 3 (b28480d): /tmp/hugtest task lint -> next: hug task edit qawmnb (was next: hug task take vk9td0); --json -> "next":"hug task edit qawmnb". TestLintNextFixesFirstFinding: clean board keeps next: hug task take ID.
