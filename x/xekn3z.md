---
id: xekn3z
title: "Release v0.7.88"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-20
updated: 2026-11-20
---

## Goal
v0.7.88 is published, installs via brew, and Jørgen says whether his board stays clean after a release.

## Purpose
Release-as-group trial (75mxf3): one ticket groups what ships and asks one question per user-facing change.

## Principle
Shipped tickets are children; one ask per user-facing change.

## Relevant details
main 1e4a3386 (CI 44710823480, 8/8), CHANGELOG 79a9ae52, merge 489684a6, release.yml 44712289344 success. Ships 72n8m4 (done refuses a task with an open question; the board cue stops offering it) and yf39ya checks 7-9 (todo task with an open question is quiet; a shipped task's agent-owned check is handed to its owner; a shipped task waiting on a dependency is quiet until it holds).

## Depends on

## Acceptance
- [x] release.yml green and brew upgrade gives hug version 0.7.88 with links into the new Cellar

## Notes

check 1: release.yml 44712289344 success; brew upgrade: hug v0.7.88, hug/muninn/huginn -> ../Cellar/hug/0.7.88/bin/*
