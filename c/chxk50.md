---
id: chxk50
title: "Release v0.7.104: no stale lint finding after an upgrade"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzyq"
created: 2026-11-22
updated: 2026-11-22
stage: "ready"
---

## Goal
v0.7.104 rebuilds the board cache once, so lint stops serving findings an older hug cached.

## Purpose
One release ticket per release. v0.7.104 = main 142a5b90, landed a4e05130 on exact-SHA CI 8/8 (run 45013140613).

## Principle
Nothing for Jørgen to try beyond lint; the one question is about restarting old hug processes.

## Details
Shipped 6q1d1t checks 1-2: SchemaVersion 31; a fingerprint test fails when what the cache holds changes without a bump (13 mutants). After brew upgrade at 10:51Z, hug task lint on this board: 'no findings' (was section-order on 9s0vj3 and 6q1d1t). Old hug tui/mcp processes from 11-17..11-19 still run their own cache-30.db; one writing set --details can add a second Relevant details section (hxqqdp).

## Depends on

## Acceptance
- [ ] Jørgen restarts his long-running hug tui and hug mcp sessions, or says to leave them

## Questions
- 2026-11-22T11:52 question 1 by crab to jhf (check 1):
  v0.7.104 is installed and lint is clean. Ten hug processes from before v0.7.102 still run: hug tui pids 78631 (since 11-17), 49453 (11-18), 34088 (11-19), and hug mcp 1402, 8265, 56816, 65214, 71259, 96239. They keep their old rules; one writing set --details would add a second 'Relevant details' section (hxqqdp). Will you restart them (quit those hug tui windows; restart the jcode sessions that own the mcp ones), or should I leave them?
  answered 2026-11-25T10:46 by jhf:
    I don't understand this. So when I open up the old Hug Tui sessions, I will reload them. But more importantly, MCP should always reload automatically. Asking me about that means you're questioning, or I no longer have confidence that we did that implementation appropriately. Not for you, but by itself, that it detects a new version and the MCP automatically reloads. And then this is a question about caching and linting. Surely this is something you have automated tests for, not something I can do easily

## Notes

crab 10:52Z: shipped v0.7.104 (main 142a5b90, landed a4e05130, CI run 45013140613 8/8). release.yml success; brew-verified: Cellar/hug/0.7.104, hug version = v0.7.104. hug task lint: no findings.


## History
- 2026-11-22T11:52 todo -> todo/design (ask)
- 2026-11-25T10:46 answered design (check 1), by jhf, for crab (todo/design -> todo/ready):
  > v0.7.104 is installed and lint is clean. Ten hug processes from before v0.7.102 still run: hug tui pids 78631 (since 11-17), 49453 (11-18), 34088 (11-19), and hug mcp 1402, 8265, 56816, 65214, 71259, 96239. They keep their old rules; one writing set --details would add a second 'Relevant details' section (hxqqdp). Will you restart them (quit those hug tui windows; restart the jcode sessions that own the mcp ones), or should I leave them?
  I don't understand this. So when I open up the old Hug Tui sessions, I will reload them. But more importantly, MCP should always reload automatically. Asking me about that means you're questioning, or I no longer have confidence that we did that implementation appropriately. Not for you, but by itself, that it detects a new version and the MCP automatically reloads. And then this is a question about caching and linting. Surely this is something you have automated tests for, not something I can do easily
- 2026-11-25T10:46 answer 1 delivered to crab (session_crab_1793783281662_44ffaee927a8a9d5)
