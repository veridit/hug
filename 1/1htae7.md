---
id: 1htae7
title: "Retired memories are deleted after 30 days"
state: todo
type: feature
priority: ""
assignee: []
labels: []
group: ""
rank: "zzzzzzzzzzzx"
created: 2026-11-27
updated: 2026-11-27
parent: "9taa9d"
---

## Goal
A memory that was forgotten or replaced is deleted for good 30 days later, by dream, with no command needed.

## Purpose
Jørgen 2026-11-27: why do we store the trash? Nothing reads a retired memory except an undo; b8mk9s decided 30 days (check 6) but no code deletes anything.

## Principle
Keep a retired memory only as long as it can be needed: an undo window of 30 days. Dream deletes, never recall or a write. The days are a setting.

## Details
Today (count on a copy of memories.json, 2026-11-27): 23,960 memories, 551 retired (0.5 MB of 20.3 MB), 89 retired more than 30 days ago; 36 of them point to the memory that replaced them (retired_into). Retired memories are kept forever: no code deletes them (grep for purge/prune in muninn/internal finds none). Recall and tidy read only active ones (tidy.go:156 'active=1'); tidy undo has its own snapshot folders. fr2205 (one file per memory) carries this over as the _retired/ folder.

## Depends on

## Acceptance
- [ ] Dream deletes retired memories older than the setting (default 30 days) and keeps younger ones (test)
- [ ] On a copy of the live store, the 89 retired before 2026-10-28 are gone and every active memory is unchanged (counts noted)
- [ ] The days are a setting; hug mem config shows it
- [ ] No index work for a retired memory: EmbedDirty (indexer.go:356) embeds only active units, and no neighbour list holds a retired one (test)

## Notes

crab 2026-11-27 12:48Z, read-only on the live index: units active=0: 1156 (active 39948). Dream (dream.go:376), recall/search (search.go:310) and neighbour rebuild (pipeline.go:355) filter active=1, and a retire clears concepts and anchors (indexer.go:174-185). Not filtered: EmbedDirty's query (indexer.go:356-358) has no active filter, so retired units are embedded too: all 1156 retired units have an embedding, which costs on every model switch (dt02yw) and ~4 MB of vectors. 1 retired unit is still in a neighbour list.
