# Noise examples, dev: 5 per arm, verbatim, seeded

## default (406 noise injections)

- call 300242: `bash cd /Users/jhf/Veridit/src/seid-lang; hug task show 9r5pcf 2>&1 | grep -E "^[0-9]\. \[|status|9r5pcf  " | head -6; hug task show 9r5pcf --json 2>/dev/null | python3 -c "import json,sys;t=json.load(sys.stdin);t=t.get('task',t);n=t.get('notes','');print(n[-700:] if isinstance(n,str) else n)"; hug `
  memory: .mcp.json should define a "hug" stdio server (command hug, args ["mcp"]) and drop the "backlog" server; in this repo's jcode the task tools come from the combined `hug mcp` server.
  judge: Memory is about .mcp.json hug server config; this call uses the hug CLI directly for task show/lint, unaffected.

- call 300097: `bash cat .jcode/scratch/4569-run-5.log`
  memory: For Brovs-4626 drill runs: work from a pristine `git archive` extract of origin/master in $JCODE_SCRATCH_DIR (delete afterwards), treat the DevOps repo as read-only, and report on YouTrack only via the ytpost.py helper.
  judge: Memory is about Brovs-4626 drill runs, unrelated to this 4569 log inspection.

- call 300282: `read /Users/jhf/Projects/hug/.hug/tasks/e3sevr/header-spec.md`
  memory: 2026-11-13: All muninn and huginn work lives in ONE place: ~/Projects/hug (github.com/veridit/hug). The old local checkouts ~/Projects/muninn, ~/Projects/huginn and their worktrees were deleted 2026-11-13 after every unique branch was pushed to hug (muninn/irbench, muninn/recall-passages, muninn/capture2, muninn/recall3), memories were moved to hug's project scope, and tasks live on hug's huginn branch. Handover files are in ~/Projects/hug/tmp (git-excluded).
  judge: Historical note about consolidating muninn/huginn into ~/Projects/hug; the read path already shows the location and no concrete rule changes this read.

- call 300140: `bash hug task show n8zmjw`
  memory: hug task attachments spec (option A): text evidence files stored on the huginn branch under .huginn/evidence/<task-id>/<name>, copied in not linked, max 256 KiB, with attach/show --file/detach commands, MCP attach/detach actions, and lint for missing or banned-path citations.
  judge: Memory describes the hug attachments spec generally, not task n8zmjw or its shown content; no concrete change to this call.

- call 300262: `write /Users/jhf/Projects/Brovs/.jcode/scratch/wt-4870f/bapi/Tests/BusinessPlaceUnknownTimezoneTests.cs namespace Tests;`
  memory: For Brovs-4569-style provisioning tests, clone DevOps into .jcode/scratch and never touch /Users/jhf/Projects/Brovs/DevOps directly.
  judge: Rule is about not touching the DevOps directory; this write is a bapi test in a scratch worktree, so it does not apply.

## anchor (88 noise injections)

- call 300254: `read /Users/jhf/Projects/Brovs/.jcode/scratch/wt-4870f/polish-browser.log`
  memory: Brovs DevOps work should be done in a fresh clone under ~/Projects/Brovs/.jcode/scratch, never editing ~/Projects/Brovs/DevOps or ~/Projects/Brovs/bapi directly.
  judge: General rule about not editing DevOps/bapi directly; reading an existing log in scratch already complies and no action changes.

- call 300132: `read /Users/jhf/Veridit/src/seid-lang/.jcode/scratch/TASK-1946.05/lane/compiler/lib/backend_js.ml`
  memory: King rule (2026-09-16 19:07): use DIFFERENT agents for OCaml compiler code (compiler/lib/*.ml) vs native self-hosted Seid code (compiler/stdlib/std/compiler/*.sd, the declarative parser, stdlib .sd). They have different styles and one agent editing both gets tainted (conventions bleed across). Parser-pair work (a fix that must land in both the OCaml parser and expr_parser.sd) should be split into two lanes: an OCaml lane and a Seid lane, coordinated by the foreman, with the pair fixpoint (regen-port.sh + gate-pair-reproducible.sh) as the shared gate.
  judge: General agent-lane allocation rule; a read of backend_js.ml by this agent does not hinge on it.

- call 300172: `read /Users/jhf/Veridit/src/seid-lang/.jcode/scratch/TASK-1960/storekey/compiler/lib/content_hash.ml`
  memory: Seid repo /Users/jhf/Veridit/src/seid-lang: tools/shape_ratchet.py counts bug shapes in compiler/lib/*.ml after lexer-based stripping of nested comments and strings; ~0.4 s full scan.
  judge: Memory describes shape_ratchet.py tooling, not content_hash.ml behavior; won't change this read

- call 300270: `read /Users/jhf/Veridit/src/seid-lang/compiler/stdlib/std/compiler/ast.sd`
  memory: **What**: Plain Seid Int now emits and remains JavaScript BigInt through literals, arithmetic, comparisons, storage, display, typed FFI results, and core Bytes operations. Number conversion occurs only at explicit host API islands. Int64 and UInt64 emission remains unchanged.
**Why**: JS Number lost precision before runtime at the SOFA maximum 2^62-1, breaking identity, ordering, display, and round-trips relative to C.
**Where**: compiler/lib/backend_js.ml; compiler/lib/typecheck.ml; compiler/stdlib/std/*.sd and compiler/stdlib/std/num/*.sd changed by TASK-891; compiler/stdlib/JS_INT_FFI_BOUND
  judge: General TASK-891 Int/BigInt background for stdlib .sd files; not specific to ast.sd or a rule/trap that changes reading it.

- call 300216: `bash export TMPDIR=/tmp; printf 'MODEL ENV\n'; env | grep -iE '(^|_)(model|provider|session)' ; hug task lint --all --json > /tmp/hug-lint3-all-start.json; hug task lint --json > /tmp/hug-lint3-open-start.json; hug task show whtv5x --json; hug task show ntczrb --json; hug task show snwdmj --json; hu`
  memory: Daily huginn workflow: `hug task next` or `hug task take ID`, `hug task show ID`, `hug task check ID N` with evidence, `hug task note ID "evidence"`, `hug task done ID` once all checks are ticked, `hug task sync` to share.
  judge: Generic daily hug task workflow; the agent already uses hug task show/lint, and the memory adds no call-specific rule or trap.

## anchor+near+needpath+cos65 (27 noise injections)

- call 300171: `read /Users/jhf/Veridit/src/seid-lang/.jcode/scratch/TASK-1960/storekey/compiler/lib/typecheck.ml`
  memory: Seid repo /Users/jhf/Veridit/src/seid-lang: tools/shape_ratchet.py counts bug shapes in compiler/lib/*.ml after lexer-based stripping of nested comments and strings; ~0.4 s full scan.
  judge: Memory is a general description of tools/shape_ratchet.py; no stated rule or trap tied to reading typecheck.ml.

- call 300099: `read /Users/jhf/Veridit/src/seid-lang/.jcode/scratch/TASK-2057/hug-lint/done-evidence/followup-existing-ac.md`
  memory: Hug-lint done-evidence artifacts live under .jcode/scratch/TASK-2057/hug-lint/done-evidence/ (followup-001 through 014, supplements, exact proofs).
  judge: Path already shows the done-evidence location; memory only restates the directory naming convention.

- call 300203: `bash cd /Users/jhf/Projects/Brovs/.jcode/scratch/wt-p4/BackofficeWeb && ls ui/scripts; cat ui/rsbuild.config.ts; cat ui/src/components/ui/ConfirmModal.tsx; rg -n 'Button|underline|text-body-copy-link' ui/src/lotteries/{EditLotteryPage.tsx,components/*}; ls /Users/jhf/Projects/Brovs/.jcode/scratch | `
  memory: Brovs DevOps work should be done in a fresh clone under ~/Projects/Brovs/.jcode/scratch, never editing ~/Projects/Brovs/DevOps or ~/Projects/Brovs/bapi directly.
  judge: Rule is about Brovs DevOps/bapi edits; this call is BackofficeWeb UI work already inside a scratch clone.

- call 300113: `bash cd .jcode/scratch/4505-devops && cat docker-swarm/crowdsec/config.yaml && cat cluster/tests/crowdsec_m6_compose.sh | head -45 && rg -n 'crowdsec|seq' docker-swarm/devops-stack.yml | head -35 && tail -3 ../4505-baseline-tests.log && cat cluster/pyinfra/tests/test_crowdsec_shared.py | head -65 &&`
  memory: Brovs-3498 S2: a common stack-name guard must run before any deploy action in docker-swarm/deploy.sh, bin/renew-images.sh, bin/converge-devops-stack.sh and Jenkinsfile.deploy, refusing jenkins-agent and devops_jenkins* unless BROVS_ATTENDED_AGENT_RESPEC is set; default deny.
  judge: Stack-name guard rule from a different ticket; this call inspects crowdsec configs and would not change based on it.

- call 300246: `read /Users/jhf/Projects/Brovs/AGENTS.md`
  memory: The standing rules were approved to move into Brovs/AGENTS.md and memory as proposed.
  judge: Process note that rules were approved to go into this file; reading it will already expose the contents, no action changes.
