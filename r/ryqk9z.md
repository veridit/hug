---
id: ryqk9z
title: "CI: macOS job cancelled when no hosted runner picks it up"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzy"
created: 2026-11-19
updated: 2026-11-19
---

## Goal
A ci.yml run fails only for a real test or build failure: today 18 of 118 failed runs in 48 h failed solely because test (macos-latest) was cancelled after 15-22 min with 'The job was not acquired by Runner of type hosted even after multiple attempts', every other job green

## Purpose
Each such run reads as red on a branch or on main and costs a person or agent a look and a rerun; it is GitHub runner capacity, not hug

## Principle
Decide with the facts in hand; no test retry and no test timeout change; a person decides whether macOS stays a required job or which runner label it uses

## Relevant details
CI census 2026-11-19 (iwazaru, scratch/ci-census.md). Runs: 44386677307 44379205716 44370619880 44368819470 44368801244 44360178493 44353109208 44325825385 44309649380 44309549053 (2026-11-18) and 44156212255 44151035876 44135891625 44132278678 44129616831 44127545147 44124111415 44123636231 (2026-11-17), 8 on main. Job durations 15m01s-21m49s, conclusion cancelled, no test step ran. Check-run annotations (3 sampled: 44386677307, 44156212255, 44353109208): 'The job was not acquired by Runner of type hosted even after multiple attempts'; 44386677307 also 'Due to capacity constraints, jobs targeting macOS arm64 runners may experience longer queue times.' ci.yml matrix os: [ubuntu-latest, macos-latest, windows-latest], no timeout-minutes.

## Depends on

## Acceptance
- [ ] The options (runner label, required or not, rerun of a cancelled job) are written with their cost and a person's choice is recorded
- [ ] No run in the following 48 h fails only because a macOS job was never picked up

## Notes
