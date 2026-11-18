---
id: npda3r
title: "Task description vocabulary: goal, purpose, principle, zoom-in"
state: "done"
type: spike
priority: ""
assignee: []
labels: ["tasks", "vocabulary"]
group: ""
rank: "zzzzy"
created: 2026-11-16
updated: 2026-11-16
aliases: ["task-description-vocabulary-goal-purpose-principle"]
---

## Goal
Agents write every task zoomed out first (where, why, how) and zoomed in only where a decision depends on it, using section names measured to make cheap and strong models do this

## Purpose
Agents write focused tasks that a doer can act on without reading history; the names must make goal and reason separate.

## Principle
Choose names by a blind, paired test on real tasks, not by taste; enforce them gently (lint on new tasks), never retroactively.

## Relevant details
Result: V1 +0.49 [+0.28,+0.71] over no sections; Details equals Relevant details; Constraints worse (-0.28). Report: docs/task-vocabulary-test.md. Lint applies from 2026-11-16 (task.PatternSince).

## Depends on

## Acceptance
- [x] 6 vocabularies x 3 writer models x 5 real tasks rewritten; blind Opus judge scores zoom, grounding, relevance, fidelity, goal/purpose separation
- [x] winning names chosen from the report, with per-model results
- [x] hug task new/take/done/lint enforce the chosen sections; setup and MCP instructions teach them

## Notes

Complete (all 90 cells, 6 broken Flash cells rerun with a markup guard). Paired vs no sections: V1 +0.49 [+0.28,+0.71], V2 +0.52 [+0.36,+0.68], V3 +0.41, V5 +0.33 [+0.07,+0.59], V4 +0.21 [-0.01,+0.45]. V2 vs V1 +0.03 [-0.15,+0.23]; V4 vs V1 -0.28 [-0.43,-0.12]. Decision: keep V1 (Goal/Purpose/Principle/Relevant details/Acceptance). Separation score carries most of the gain (1.93 -> 3.73). Report: docs/task-vocabulary-test.md.

Item 3: hug task new --purpose/--principle/--details (and MCP task_new fields) write the sections after Goal; lint code missing-section flags open tasks created on or after 2026-11-16 that lack Purpose or Principle (older tasks untouched); setup's agent section and AGENTS.md teach the pattern. TestNewWithPatternAndLint.
