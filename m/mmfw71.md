---
id: mmfw71
title: "hug tui left running does not pick up new releases"
state: todo
type: bug
priority: "high"
assignee: []
labels: []
group: ""
rank: "000001v"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
A hug tui left open across a brew upgrade notices the new binary and reloads, so nobody works on an old version.

## Purpose
m5pbb0 (2026-11-19, blossom): Jørgen's hug tui pid 78631 ran v0.7.69 from 2026-11-17 19:53 through ten releases; it wrote p87ngv in a pre-q2adr8 shape and hid questions 2-5. The R corner reload (j2vwk0) did not bring it forward. Evidence: vmmap 78631 maps /opt/homebrew/Cellar/hug/0.7.69/bin/hug; cache-25.db held open.

## Principle
Reproduce first: start hug tui from one Cellar version, brew upgrade, observe. Never write task files with an older format than the board's: an old binary seeing a newer board should refuse writes with a fix: line.

## Relevant details
A hug tui left open across a brew upgrade notices the new binary and offers or does a reload, so a person never works for days on an old version.

Leads: j2vwk0 R corner; the cache schema number (cache-25 vs current) as a cheap version signal; executable path vs /opt/homebrew/bin/hug symlink target.

## Depends on

## Acceptance
- [ ] A test or scripted repro shows a TUI started on version N notices version N+1 installed
- [ ] An old binary refuses to write a board that a newer hug has written (fix: line names the restart)
- [ ] Jørgen leaves hug tui open across a release and sees it move forward

## Notes

crab 12:01Z evidence (vmmap per pid): 78631 (hug board, started 17 Nov 19:53) now maps Cellar/hug/0.7.81, so it did re-exec in place at some point after 10:00Z; v0.7.82 released 11:59Z not yet picked up. Other open TUIs: 16277 statbus 0.7.69 (since 18 Nov 18:08), 69660 sql_saga 0.7.68 (since 18 Nov 20:16), 49453 seid-lang 0.7.81. Leaked test process: 91146 'hug tui' v0.0.2 in /Users/jhf/.jcode/scratch/huginn-reload-4068778528/repo, started 19 Nov 02:14 (a reload test's fixture that outlived its test).

crab 2026-11-25 22:43Z, 'hug version --running' (new in v0.7.115): tui pids 32504 (started 11-23 10:54), 19901 (11-23 10:52), 90079 (11-23 11:33) all run '/hug (removed or replaced since it started)', i.e. no longer the installed hug, two days and many releases later. The mcp side now reloads while idle (rvb3j5, v0.7.115); the tui needs the same check.
