---
id: n7jbmx
title: "Recall on the agent's tool calls (post_tool_feedback)"
state: "doing"
type: spike
priority: ""
assignee: ["hatchling"]
labels: ["recall"]
created: 2026-11-15
updated: 2026-11-15
parent: "d634kr"
doing_since: "2026-11-18T05:32"
stage: "active"
model: "claude-opus-5-5"
branch: "main"
worktree: "/Users/jhf/Projects/hug"
---

## Goal
Memories reach the agent mid-turn when what it is doing (files touched, commands run) matches them, with a stricter bar than message recall

## Depends on

## Acceptance
- [x] offline set: tool calls from past jcode sessions (name, input, touched paths) with the memories recall would inject for them, judged blind with a stricter rubric: would the agent act differently, and is it not already in front of it
- [ ] ~~trigger uses CASE paths/tools from stored memory types and when-then fields, not only text similarity~~ (not relevant: crab 04:33Z: local anchor extraction counts until jrj1vq lands; replaced by check 5, hatchling, 2026-11-18)
- [ ] hook: hug mem hook jcode-tool reads JCODE_HOOK_TOOL_NAME/TOUCHED_PATHS, prints at most 1-2 memories, under 200 ms p95, silent when nothing passes
- [ ] ship only if judged-useful injections per 100 tool calls rises with irrelevant ones under a stated cap
- [x] trigger uses anchors from the memory text (paths, commands) and, once jrj1vq lands, its CASE fields; each memory at most once per session

## Notes

Item 1 done (2026-11-16): offline set real/tool (66 tool calls, 7 sessions) replayed through recall4 on the typed snapshot, judged blind with the strict rubric (judge_tool.py, DeepSeek Flash). default: 30.3 useful and 48.5 noise per 100 calls, 44% of calls get an injection. union (all candidates): 259 useful vs 1741 noise per 100. Conclusion: message recall reused for tool calls is net noise (precision ~38%); the hook must not ship on the default gate. Next (item 2): a trigger keyed on CASE paths/commands from rule-type memories and when-then fields, then rejudge; ship cap proposal: noise <= 10 per 100 calls.

Gate sweep 2026-11-16: norecords1 vs default on 66 calls: useful -12.1 [-18.4,-4.4], noise -21.2 [-37.8,-5.0] per 100; precision ~40% either way. Wide pool precision by type: rule 28%, fact 15%, record 8%, preference 4%. Decision recorded in muninn/docs/RECALL-TOOL.md: do not ship a hook on message-recall gates. Item 2 (CASE paths/commands trigger) is the next experiment; it needs memories that name a path/command, which the when-then extraction (jrj1vq) would supply.


file:n7jbmx-plan.txt (3047 bytes, sha256:506d2708)
Check 2 plan sent to crab (file:n7jbmx-plan.txt). The trigger matches exact anchors: touched paths, basenames and bash command heads against paths and commands named in memories, which a local extractor writes into an anchors table. jrj1vq's CASE fields feed the same table later. Types rule, fact and preference only, at most 2 per call. Dev set: 300 calls from 11-16..18. Ship decision only on a fresh held-out set of 150 calls from 11-19 on, judged blind with judge_tool.py unchanged. Bar, fixed before any run: noise <=10/100 (CI upper <=15), useful >=10/100 (CI lower >0), precision >=50%, noise below default with the CI excluding 0, hook p95 <200 ms. No code until crab rules.
crab 04:33Z approved the plan with these rulings. (1) Local extraction counts; check 2 is struck and replaced by check 5. (2) The bar stands, plus: hook p95 is measured on the live store under real load, and a memory is shown at most once per session. (3) Fresh set: journals from 11-19 and 11-20, so the ship decision waits for them and dev work starts now. (4) The report quotes 5 noise examples per arm verbatim.
file:n7jbmx-dev-report.md (3777 bytes, sha256:1a8b2e66)
file:n7jbmx-dev-noise-examples.md (8351 bytes, sha256:71a75445)
Dev replay done (file:n7jbmx-dev-report.md, file:n7jbmx-dev-noise-examples.md with 5 verbatim noise examples each for default, anchor and the chosen arm). 300 calls, 77 sessions, 11-16..18T04:34:59Z, store frozen at 04:34:59Z, each call sees only earlier memories. Blind, judge_tool.py rubric unchanged, 0 unjudged. default: 57.3 useful [47.0, 68.4] and 135.3 noise [121.0, 150.0] per 100 calls, 30% precise. Plain anchor: 28.0 / 29.3, 49%. I tried 15 arms on dev. The chosen arm is recall.ToolRequest.Shipped(), recall4 -variant shipped (near paths + no command-only match + cos>=0.65): 10.7 useful [6.4, 15.7] and 9.0 noise [5.2, 13.3], 54% precise, noise vs default -126.3 [-140.9, -112.1]. It is now FROZEN, before any 11-19/20 call exists. The fresh set judges only shipped and default, with no tuning. It sits on the bar edge, so it may fail; the hook then stays off (tool_recall defaults to off). Hook p95 on the real path, one 'hug mem hook jcode' process per call on a 05:11Z copy of the live store, not niced, load 14.9: p50 59.1 ms, p95 150.3 ms, max 339.6 ms, 0 nonzero exits. The anchor backfill on an existing store runs once and took 2.2 s for 29,770 units.
Landed 94ef2140 (CI 44243962498). tool_recall is off by default. Check 5: anchor.Memory and anchor.Call pull paths, file names and command heads into unit_anchors at index time, and backfill once (anchor_recipe). TestMemory, TestCall, TestAnchorsIndexed (index, change, forget, backfill), and TestToolRecallAnchors cover types rule/fact/preference only, scope, once per session and the 2-cap; TestToolRecallShipped covers near, needpath, the cos gate and no embedder means nothing shown. Check 3: post_tool_feedback runs deliverParked, then toolRecallHook. That reads JCODE_HOOK_TOOL_NAME, TOUCHED_PATHS and TOOL_INPUT.command, opens the store with no embedder and reaches Ollama only after an anchor matched. It prints at most 2 and nothing when none pass (TestJcodeToolRecall: off prints and logs nothing; on, it shows the in-scope rule, not the other project's, once). p95 150.3 ms on the live-store copy at load 14.9. Note: there is no separate 'hug mem hook jcode-tool'; it is the same 'hug mem hook jcode' on post_tool_feedback, which upgrade already installs. -race on anchor, recall, index, store, search, app, run and repocheck: all ok.
## History
- 2026-11-18T05:32 todo -> doing/active (take)
