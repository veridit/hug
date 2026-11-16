# Tool-call recall (66 calls, 7 sessions), strict judge

| arm | useful /100 calls | noise /100 calls | calls with any injection | useful vs default [CI] | noise vs default [CI] |
|---|---:|---:|---:|---:|---:|
| default | 30.3 | 48.5 | 44% | - | - |
| norecords1 | 18.2 | 27.3 | 30% | -12.1 [-18.4, -4.4] | -21.2 [-37.8, -5.0] |

## Reading (2026-11-16)

66 real tool calls from 7 jcode sessions, replayed against the frozen typed
snapshot, every injected memory judged blind with a strict rubric (would the
agent act differently on THIS call, and is it not already shown).

- Message recall reused as-is (default) injects on 44% of calls; 38% of
  those memories are useful (30 useful vs 49 noise per 100 calls).
- Leaving records out (norecords1) cuts noise by 21 per 100 [CI -38, -5]
  but also useful by 12 [-18, -4]; precision stays near 40%.
- In the wide candidate pool, rule memories are the most precise (28%),
  records the least (8%).

Decision: no tool-call hook ships yet. A tool-call trigger needs a signal
message recall does not have (the memory names this file, command or
service), not a stricter version of the same gate.
