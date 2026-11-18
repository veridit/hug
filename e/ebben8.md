---
id: ebben8
title: "After-release: a check is a person's when a [people] person is its subject"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-18
updated: 2026-11-18
---

## Goal
personChecks (afterrelease.go) recognises an open check as a person's when its subject is a person in hug.toml [people] (short name, alias, long or first name, or the owner), whatever the verb: 'Jørgen attaches a screenshot ...' (5z4zt4 check 5) as well as 'Jørgen confirms ...'. A person named elsewhere in the check (object, possessive thing, 'on Jørgen's machine') is still not the subject.

## Purpose
bonehound's count for yf39ya check 5: 5z4zt4 check 5 is not asked of Jørgen after release because 'attaches' is not in AddressesPerson's subject verb list (agxqax), so the shipped task stays shipped-not-done instead of in his ASKED. A verb list misses every verb nobody listed.

## Principle
Decide by who the subject is, not by the verb; no verbs added one at a time. Measure the old and new matcher on every open check of hug's and Seid's boards (read-only copies) and get crab's ruling on the diff before changing code. Seid is observe-only. Wait texts (lint wait-on-person) keep their own reading unless crab rules otherwise.

## Relevant details
personq.go: subject regexp has a fixed verb list (approve|answer|confirm|...|tick). afterrelease.go personChecks calls AddressesPerson on each open check. Reported by bonehound for yf39ya check 5; crab assigned to mizaru 2026-11-18 20:49.

## Depends on

## Acceptance
- [ ] a table of every open check on hug's and Seid's boards with the old and new matcher's result, every difference listed, sent to crab before any code change
- [ ] 5z4zt4 check 5 ('Jørgen attaches a screenshot ...') is a person's check for jhf; a test that failed first shows it, and a check naming a person only as object or owner of a thing is not
- [ ] mutants of the new subject rule are killed, CI passes on all jobs

## Notes
