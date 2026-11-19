---
id: w7tq3k
title: "Lint warns when an acceptance check reads like a question"
state: "doing"
type: feature
priority: "medium"
assignee: ["daisy"]
labels: []
group: ""
rank: "zzzzzzzzzzzm"
created: 2026-11-19
updated: 2026-11-19
doing_since: "2026-11-19T09:20"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
hug task lint flags an acceptance check that is really a question to a person. Today a check like 'Should the ASKED row show the count?' sits among the checks, is easy to tick by mistake, and nobody is asked, so the question waits for someone to notice it and turn it into a hug task ask, which in practice happens late or never. .Request .by:jhf .at:2026-11-19T09:40Z .prev:.Sentence .msg:"Make it concise: say the harm in a few words." The finding names the check, and its fix: line runs hug task ask with the check's text.

## Purpose
Jørgen 2026-11-19 08:55Z: questions today look like acceptance criteria. A question belongs where a person is asked and answers it.

## Principle
Lint finds, it never rewrites a check. One finding per check, with one fix: line.

## Relevant details
Code: huginn/internal/task lint (codes.go for the code name). Sibling rules: q2adr8 (one question per ask).

## Depends on

## Acceptance
- [ ] A check that ends in '?' is a lint finding, with a fix: line that runs hug task ask (test)
- [ ] A check that starts with Should, Can or Do is a finding too (test) .Question .by:daisy .to:jhf .at:2026-11-19T09:52Z .prev:.Item .on:"Should, Can or Do" .msg:"English only, or also Norwegian (Skal, Kan, Bør)?" .Answer .by:jhf .at:2026-11-19T10:04Z .msg:"""English only, so "Should" counts and "Skal" does not, nor "Bør""""
- [ ] hug task lint --json gives the finding the code check-is-question (test)

## Notes

## History
- 2026-11-19T09:20 todo -> doing/active (take)
