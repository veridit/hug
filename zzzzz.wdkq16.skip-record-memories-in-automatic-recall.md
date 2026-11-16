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

Fresh 147 replayed on the typed snapshot (recall4 default+typed vs norecords600+typed), judged blind (real/judge9.log). long n=64: irrelevant/msg 3.38 -> 0.81, delta -2.56 [CI -3.00, -2.08]; relevant 0.38 -> 0.30, delta -0.08 [CI -0.26, +0.08] (no significant fall). human n=83: identical. Default 600 shipped in v0.7.4 (03df210).
