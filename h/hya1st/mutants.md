# hya1st / 32ym01 mutants (branch hya1st; tests at c8b22551; each mutant applied by hand, tests run, file restored with git checkout)

| # | rule broken | first failing test | failure line |
|---|---|---|---|
| 1 | order-goal-third | TestPageReadsShortToLong | `8a0ybb_test.go:85: headings` |
| 2 | order-principle-before-purpose | TestPageReadsShortToLong | `8a0ybb_test.go:85: headings` |
| 3 | threshold-5 | TestDetailsOpenAtSixLinesFoldedAtSeven | `hya1st_test.go:126: headings GOAL\|PURPOSE\|PRINCIPLE\|DETAILS (3)\|ACCEPTANCE, want GOAL\|PURPOSE\|PRINCIPLE\|DETAILS\|ACCEPTAN` |
| 4 | threshold-7 | TestDetailsOpenAtSixLinesFoldedAtSeven | `hya1st_test.go:139: headings GOAL\|PURPOSE\|PRINCIPLE\|DETAILS\|ACCEPTANCE, want GOAL\|PURPOSE\|PRINCIPLE\|DETAILS (4)\|ACCEPTAN` |
| 5 | threshold-not-wrapped | TestDetailsOpenAtSixLinesFoldedAtSeven | `hya1st_test.go:139: headings GOAL\|PURPOSE\|PRINCIPLE\|DETAILS\|ACCEPTANCE, want GOAL\|PURPOSE\|PRINCIPLE\|DETAILS (3)\|ACCEPTAN` |
| 6 | details-always-folded | TestPageReadsGoalPurposePrincipleDetails | `hya1st_test.go:98: headings` |
| 7 | details-always-open | TestPageReadsShortToLong | `8a0ybb_test.go:85: headings` |
| 8 | details-fold-at-bottom | TestPageReadsShortToLong | `8a0ybb_test.go:85: headings` |
| 9 | ring-details-at-end | TestTabFollowsGoalPurposePrincipleDetails (after c8b22551) | `hya1st_test.go:190: ring 0 7 1 7, want 0 7 1 (page, DETAILS, checks)` |
| 10 | stops-goal-third | TestTabFollowsGoalPurposePrincipleDetails | `hya1st_test.go:163: stops` |
| 11 | np-keeps-unfolded | TestNextPreviousKeepTheDetailsRule | `hya1st_test.go:225: p: long Details not folded again:` |
| 12 | rule-constant-word | TestDetailsWordingForAgents | `details_test.go:58: agentSection:` |
| 13 | rule-agents-md | TestSectionsRuleWording | `sections_rule_test.go:42: repository AGENTS.md does not say "Describe a task with Goal (what: where we aim), Purpose (wh` |
| 14 | rule-setup-template | TestDetailsWordingForAgents | `details_test.go:58: agentSection:` |
| 15 | rule-help-new | TestSectionsRuleWording (mutant re-made to change the sentence) | `sections_rule_test.go:42: help new does not say "Describe a task with Goal (what: where we aim), ..."` |
| 16 | rule-lint-principle | TestSectionsRuleWording | `sections_rule_test.go:51: lint texts` |
| 17 | rule-lint-purpose | TestSectionsRuleWording | `sections_rule_test.go:51: lint texts` |
| 18 | setup-no-sentence-edit | TestSetupFlagsOldSectionsSentence | `sections_rule_test.go:100: old sentence: step map[detail:AGENTS.md (and CLAUDE.md if present) tell agents to track work ` |
| 19 | setup-no-flatten | TestSetupFlagsOldSectionsSentence | `sections_rule_test.go:100: old sentence: step map[detail:AGENTS.md (and CLAUDE.md if present) tell agents to track work ` |
| 20 | rule-mcp-task-new | TestTaskNewSaysSectionsRule | `sections_rule_test.go:36: task_new description does not say "Describe a task with Goal (what: where we aim), Purpose (wh` |
