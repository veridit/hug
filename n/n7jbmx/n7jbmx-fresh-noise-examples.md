# Noise examples, fresh: 5 per arm, verbatim, seeded

## shipped (8 noise injections)

- call 400123: `read /Users/jhf/Projects/Brovs/AGENTS.md`
  memory: The standing rules were approved to move into Brovs/AGENTS.md and memory as proposed.
  judge: Meta-history that rules were approved into AGENTS.md; no actionable content for reading the file.

- call 400130: `bash python3 - <<'PY'
import json,urllib.request
s=json.load(open('/Users/jhf/Projects/Brovs/.jcode/secrets/brovs-youtrack.json'))
d=json.load(urllib.request.urlopen(urllib.request.Request('https://brovs.youtrack.cloud/api/issues/Brovs-4897?fields=description,comments(id,text)',headers=s['headers'])`
  memory: Brovs YouTrack API access: load /Users/jhf/Projects/Brovs/.jcode/secrets/brovs-youtrack.json and use its headers key; post comments with python3 .jcode/scratch/ytpost.py <issue> from /Users/jhf/Projects/Brovs.
  judge: Call already loads the same YouTrack secrets and fetches the issue; memory adds no needed rule for this read.

- call 400109: `read /Users/jhf/Projects/Brovs/AGENTS.md`
  memory: The standing rules were approved to move into Brovs/AGENTS.md and memory as proposed.
  judge: Only records that rules were moved into AGENTS.md; no actionable rule content, and the call already reads that file.

- call 400149: `read /Users/jhf/Veridit/src/seid-lang/.jcode/scratch/TASK-1960/storekey/compiler/test/dune`
  memory: The compiler unit executable is `compiler/test/test_compiler.exe`, defined in `compiler/test/dune`.
  judge: The memory just states what compiler/test/dune contains, which the agent is about to read directly; no action-changing rule.

- call 400104: `read /Users/jhf/Projects/Brovs/AGENTS.md`
  memory: The standing rules were approved to move into Brovs/AGENTS.md and memory as proposed.
  judge: Background that approved standing rules were moved into AGENTS.md, but reading the file itself already reveals its contents and the memory adds no actionable constraint.

## default (178 noise injections)

- call 400131: `bash pwd; git status --short; git show --format=fuller --stat 0311d6a4; git diff 242b6b2e 0311d6a4; rg -n --hidden -g '!.git/**' 'One-way|full synchronization|user-images-mirror|daemon_reload|daemon-reload' .`
  memory: Manifest validation command: `python3 bin/cluster-config-impact.py --validate-manifest` run against a `git archive` extraction of the commit SHA.
  judge: Manifest validation rule is unrelated to this call's git history inspection and code search.

- call 400070: `edit /Users/jhf/.jcode/scratch/rnpdnm-step2/REPORT-STEP2.md`
  memory: hug worker reports are appended to /Users/jhf/.jcode/scratch/lint-gaps2/REPORT.md with test output beside it in the same folder.
  judge: Describes a hug lint-gaps2 report location, unrelated to REPORT-STEP2.md in rnpdnm-step2.

- call 400087: `edit /Users/jhf/.jcode/scratch/4gj7z6/huginn/internal/cli/sync_test.go`
  memory: hug worker reports go to /Users/jhf/.jcode/scratch/<task>/REPORT.md and the final response is that report.
  judge: Report-location process note does not affect editing this sync_test.go file.

- call 400147: `bash cd /Users/jhf/.jcode/scratch/24403z/huginn/internal/tui && gofmt -l . ; GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -run 'TestActionKeepsTheSlot|TestActionInPlace|TestActionSays|TestActionMovedOutside|TestBreadcrumbNamesThePane' -count=1 . 2>&1 | grep -E "^\s*(---|slot_test|FAIL|ok|PASS|\./)" `
  memory: hug's full local suite is run as `GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./...`.
  judge: Memory restates the exact GOMAXPROCS/GOFLAGS/nice invocation the call already uses; adds nothing for this targeted test run.

- call 400139: `edit /Users/jhf/.jcode/scratch/ec6z6j-r/huginn/internal/tui/idstops.go`
  memory: hug worker reports go to /Users/jhf/.jcode/scratch/<task>/REPORT.md and the final response is that report.
  judge: Report-location convention for hug workers does not affect editing idstops.go and is not a trap for this call.
