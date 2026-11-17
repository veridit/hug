# Preview: tags on rows (txjkrm), Seid overview at ~170 columns

Real tasks from ~/Veridit/src/seid-lang. Tags shown are what remains AFTER the
proposed cleanup (hug upgrade --tags): type labels (feature), stage labels
(design, paused), priority labels (urgent) and one-offs move out; variants merge
(compiler-lang -> compiler, perf -> performance). Tags are dim on screen.

## Before (today, v0.7.39)

```
  IN PROGRESS · assigned to  [2]
◔ 1/4  n8zmjw  blocked feature medium   TCC gate memcheck: rt coverage, exit-reachable leak policy and stack false re…  hibiscus
● 3/3  n0fzz1  blocked bug     medium   Test runner calls any exit 137 a memory-limit stop                              daisy

> NEXT UP · rank order  [1071]
○ 0/5  q92ye5  todo    feature medium   Compile-time word when @build misses TRMC (split from TASK-1782 AC#3)
○ 0/2  mkvb34  todo    feature medium   C-backend TCO / trampoline (root fix for TASK-027)
○ 0/2  wfwwwy  todo    feature medium   Stdlib Namespace: trie-of-HAMTs implementation
!      h05n4x  todo    feature low      Refactor: TSum canonical-id vs zonk-use-chain disambiguation in typecheck.ml
○ 0/4  xxfq4w  todo    feature high     Compile-time bounds elision via Nat type-level index calculations, steps A-D
!      zjedy0  todo    feature medium   Unified ability/handler surface tightenings (.Refactor .Compiler.Lang)
○ 0/2  gbwtn3  todo    feature low      .Feature .Compiler.Staging — EWith quote full support (Expr/With variant)
!      tmt5pk  todo    feature high     Principal axis: make all .sd tests run and pass (.Foundation .Compiler.Test)
○ 0/2  09khnp  todo    feature medium   Inline record-rename sugar (r .old:.new)
```

## After (tags before the title, dim)

```
  IN PROGRESS · assigned to  [2]
◔ 1/4  n8zmjw  doing  depends  medium   memcheck tcc          TCC gate memcheck: rt coverage, exit-reachable leak po…  hibiscus
● 3/3  n0fzz1  doing  depends  medium   test runner           Test runner calls any exit 137 a memory-limit stop       daisy

> NEXT UP · rank order  [1071]
○ 0/5  q92ye5  todo   ready    medium   compiler staging      Compile-time word when @build misses TRMC (split from TASK-1782 AC#3)
○ 0/2  mkvb34  todo   ready    medium   codegen runtime       C-backend TCO / trampoline (root fix for TASK-027)
○ 0/2  wfwwwy  todo   design   medium   stdlib namespace      Stdlib Namespace: trie-of-HAMTs implementation
!      h05n4x  todo   draft    low      typecheck compiler    Refactor: TSum canonical-id vs zonk-use-chain disambiguation in typecheck.ml
○ 0/4  xxfq4w  todo   design   high     typecheck performance Compile-time bounds elision via Nat type-level index calculations, steps A-D
!      zjedy0  todo   draft    medium   parser compiler       Unified ability/handler surface tightenings
○ 0/2  gbwtn3  todo   ready    low      compiler staging      EWith quote full support (Expr/With variant)
!      tmt5pk  todo   draft    high     compiler test         Principal axis: make all .sd tests run and pass
○ 0/2  09khnp  todo   design   medium   parser format         Inline record-rename sugar (r .old:.new)
```

What changed:
- The type column (`feature` on every row) gives way to the stage (6ephqk); type shows in the detail and when you filter by it.
- Tags are one dim column, up to ~22 cells, cut after the title and before the owner.
- Title prefixes that were really tags (`.Feature .Compiler.Staging —`, `(.Refactor .Compiler.Lang)`) are moved into tags by the cleanup, so titles get shorter.
- Stages shown are guesses for the preview (q92ye5 ready, wfwwwy design, ...); the real ones come from 6ephqk's rules.

Open: should the type column stay as well (one more column), or move to the detail view as above?

## Revision (Jørgen 2026-11-17 10:28): state in the header, stage before the text

The state is already the section (IN PROGRESS = doing), so the row drops it.
Order: pie, id, type, priority, tags, stage, then the text. When the stage has
extra data the text is `stage: reason` instead of the title; without it, the title.

```
> IN PROGRESS · assigned to  [12]
○ 0/4  zhvkmj  feature         tui          active    Status colors: rows and sections share one subtle scheme
◕ 3/4  tzc9hb  feature high    setup        review:   release, then hug setup here and in seid-lang
◑ 3/5  44w1ev  chore   medium  cache perf   waiting:  CI run 44063552584 · 12 min
○ 0/4  m1c28r  chore   medium  env          depends:  44w1ev (doing)
○ 0/4  16dxsr  feature high    deps         active    Depend on a group or epic, and on a stated condition
```

Ruled (Jørgen 10:56): the reason replaces the title fully. The colon is how you
know which one you are reading: `stage:` then the reason, a bare stage then the title.
