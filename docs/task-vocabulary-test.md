| arm | zoom | grounding | relevance | fidelity | separation | mean | irrelevant share | n |
|---|---:|---:|---:|---:|---:|---:|---:|---:|
| V0 (none) | 2.47 | 3.93 | 3.27 | 3.60 | 1.93 | 3.04 | 21% | 15 |
| V1 Goal/Purpose/Principle/Relevant details/Acceptance | 3.27 | 3.93 | 3.20 | 3.53 | 3.73 | 3.53 | 26% | 15 |
| V2 Goal/Purpose/Principle/Details/Acceptance | 3.40 | 3.93 | 3.40 | 3.33 | 3.73 | 3.56 | 22% | 15 |
| V3 Goal/Purpose/Principle/Decision details/Acceptance | 3.27 | 3.93 | 3.13 | 3.20 | 3.73 | 3.45 | 26% | 15 |
| V4 Goal/Purpose/Principle/Constraints/Acceptance | 2.87 | 3.80 | 3.00 | 3.13 | 3.47 | 3.25 | 24% | 15 |
| V5 Goal/Why/Approach/What matters/Done when | 2.93 | 3.87 | 3.20 | 3.13 | 3.73 | 3.37 | 22% | 15 |

Per model (mean of the five scores):

- flash: V0 2.88, V1 3.52, V2 3.52, V3 3.36, V4 3.28, V5 3.16
- sol: V0 3.20, V1 3.68, V2 3.72, V3 3.68, V4 3.36, V5 3.76
- haiku: V0 3.04, V1 3.40, V2 3.44, V3 3.32, V4 3.12, V5 3.20

Typical worst flaw per arm (3 samples):

- V0: invents rebase need for parity lane and an em-dash constraint not in source | no zoomed-out goal/purpose opening; Context section jumps straight into numbers and ticket history | goal lacks a distinct purpose, and invented scope (source backfill decision, regression mitigation, recall4-format report) is added
- V1: Principle section packs file-level fields and AUC numbers into the zoomed-out part, and Acceptance adds invented criteria (records excluded, CI thresholds) beyond the source | invented constraints like 'regression is a blocker for further rollout' and 'do not implement user-weighting until source exists', which are not in the source | historical landing range and verification numbers padding the detail section
- V2: principle section mixes process trivia (gatelock flags, commit rewording) with direction | goal and purpose packed with ticket IDs and numbers, plus invented acceptance targets (cl_100 baseline, 'no regression' claims) | detail section carries completion history and evidence narration that is irrelevant to a doer on a finished task
- V3: Goal and Purpose are crammed with numbers. The rewrite also folds TASK-2057's blockers into this task's scope, which conflicts with its own 'do not flip until TASK-2057 closes'. | Goal and Purpose are packed with numbers and file paths, and the rewrite invents constraints the source never states (e.g. 'test with a live Raise grant' read as frame gate, task1966 pins added, NULL-root target) | goal and purpose invent a mandate to fix the statbus long-prompt regression and add new acceptance gates the source never set
- V4: Purpose and Principle mix in numbers and steps, and they attribute the kind-gate result to when/then triggers | goal sentence overloaded with mechanism details (files, ceiling, alarm), not zoomed out | Purpose section is mostly numbers, and the Constraints section carries landed AC#4 history
- V5: Approach adds rollout and evaluation directives that go beyond the source's acceptance criteria | Why section packed with numbers and Approach mixes history with invented don'ts (em dashes, sed, DMs) | resets completed acceptance items to unchecked and invents new done-when conditions (long-query regression gate, 'do not enable user-weighting on existing memories')
## Paired differences (same model and source task, bootstrap 95% CI, n=15)

- V1 (Goal/Purpose/Principle/Relevant details/Acceptance) vs no sections: +0.49 [+0.28, +0.71]
- V2 (… Details …) vs no sections: +0.52 [+0.36, +0.68]
- V3 (… Decision details …): +0.41 [+0.21, +0.59]
- V5 (Goal/Why/Approach/What matters/Done when): +0.33 [+0.07, +0.59]
- V4 (… Constraints …): +0.21 [-0.01, +0.45]
- V2 vs V1: +0.03 [-0.15, +0.23] (no difference); V4 vs V1: -0.28 [-0.43, -0.12] (worse)

Conclusion (2026-11-16): named sections clearly help; the separation score
(goal vs purpose distinct) carries most of it (1.93 -> 3.73). "Relevant
details" and "Details" are equivalent, "Constraints" is worse. Keep V1, the
user's naming. Six Flash cells first came back as tool-call markup instead
of a rewrite; they were rerun with a guard, so every arm has n=15.
Method: 3 writers (deepseek-flash, gpt-6.1-sol, claude-haiku-5-5) x 5 real
tasks x 6 arms; blind judge claude-opus-5-5 on five 1-5 criteria.
