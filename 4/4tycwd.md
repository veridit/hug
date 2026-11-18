---
id: 4tycwd
title: "Detour scratch for in-progress work"
state: "doing"
type: spike
priority: ""
assignee: ["piglet"]
labels: ["needs-discussion"]
group: ""
rank: "zzzzzzv"
created: 2026-11-16
updated: 2026-11-16
doing_since: "2026-11-18T18:57"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
When an agent must leave an in-progress task for a detour, it parks where it was and picks it up again without loss

## Purpose
Detours (a bug found mid-task, a user question) make agents forget the half-done task; the board says doing but the thread is lost

## Principle
Light: no park command, no marker. A detour to another task is a dependency (hug task dep A --add B plus a note); a wait is for things outside the board and hug wakes it (--until or its condition). next means what to work on now, so a waiting task is never offered as next work during its wait (crab 17:57).

## Relevant details
Candidates: hug task park ID 'where I was' (a dated note plus a marker), shown by hug task next --mine and in the take guide; or reuse notes with a 'parked:' prefix. Decide after the nudge work (qawmnb) shows how often detours happen

## Depends on

## Acceptance
- [ ] an agent can park and resume a task with one command each, and next --mine shows parked work first
- [ ] hug task take refused for one-in-doing prints a fourth fix: hug task dep A --add B && hug task note A "where I was" (A the doing task, B the refused one), in text and in the JSON fixes array
- [ ] hug task next --mine skips my doing tasks in stage waiting; a woken wait (--until day reached or its condition holds) counts as active and comes first as today
- [ ] when nothing else is left, hug task next --mine says so and lists my waiting tasks one line each (id, what it waits on), exit 4
- [ ] hug task resume ID prints once what the task was waiting on ('was waiting on: ...')
- [ ] one CHANGELOG line; tests fail first on origin/main and pass after; CI green

## Notes

Proposal (piglet, probed on hug v0.7.67 in a throwaway repo; not taken: gav9nw is parked, nothing built):
Already covered, no new command: a detour to another task is a dependency, not a wait (the wait help says so). hug task dep A --add B && hug task note A "where I was". A becomes doing/depends, which does not count for one-in-doing, so take B succeeds. When B is done, A goes back to doing/active on its own (History: doing/depends -> doing/active (dependency B)), next --mine returns A first, and show prints the note. Probe: A dep C, take C ok, done C, next --mine = A, stage active.
The wait route (what crab used on wxsyc5) works less well:
(1) During the detour, next --mine returns the parked task A, not the detour, with next: hug task check A 1. Stage waiting is not skipped (store.go:734 skips only Waiting() = deps/conditions).
(2) resume A while the detour is still doing is refused (one in doing). That is right, but the fix lines don't name the detour.
(3) After resume, the 'where I was' text stays only in ## History, which show does not print; resume's output does not echo it.
Narrowed proposal (no park command, no marker):
a. The take refusal gains a fourth fix when the refused task is known: hug task dep A --add B && hug task note A "where I was", where B is the task being taken.
b. next --mine skips my doing/waiting tasks (they come back via resume or --until, as today), so the detour is next.
c. resume prints what the task waited on ('was waiting on: ...') once.
Check 1 becomes: one command to park (dep + note), none to resume (it returns by itself), and next --mine shows the detour during it and the parked task right after.
Open for crab: is (b) a bug fix or wanted behaviour (a CI wait showing in next --mine as a reminder)?


## History
- 2026-11-18T18:57 todo -> doing/active (take)
