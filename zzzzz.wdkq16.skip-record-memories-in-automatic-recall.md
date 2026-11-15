---
id: wdkq16
title: "Skip record memories in automatic recall"
status: "doing"
type: spike
priority: ""
assignee: ["Jørgen H. Fjeld"]
labels: ["recall"]
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Automatic recall stops injecting 'record' memories (what happened) for long messages when that cuts noise without losing useful memories, measured with the stored types

## Depends on

## Acceptance
- [ ] fresh 147 replayed with stored types: default vs default+norecords600, blind judge, session-cluster CIs
- [ ] skip_records_above default set only if long irrelevant/msg CI < 0 and relevant/msg does not fall

## Notes
