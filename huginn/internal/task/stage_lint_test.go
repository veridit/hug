package task

import (
	"strings"
	"testing"
)

// stageDoc is a complete task file with the given extra frontmatter lines
// (status, stage, node fields) and dependencies.
func stageDoc(id, front string, deps ...string) *Document {
	body := "---\nid: \"" + id + "\"\ntitle: \"T\"\n" + front + "\ntype: \"feature\"\npriority: \"medium\"\nlabels: []\nassignee: []\ncreated: \"2026-11-01\"\nupdated: \"2026-11-01\"\n---\n\n## Goal\ng\n"
	if len(deps) > 0 {
		body += "\n## Depends on\n"
		for _, d := range deps {
			body += "- " + d + "\n"
		}
	}
	body += "\n## Acceptance\n- [ ] a\n"
	return Parse(id+".t.md", []byte(body))
}

// codes are the finding codes for docs linted together, one "code: problem"
// per finding.
func codes(docs ...*Document) []string {
	var out []string
	for _, f := range Lint(docs) {
		out = append(out, FindingCode(f.Problem)+": "+f.Problem)
	}
	return out
}

func wantOnly(t *testing.T, got []string, code, part string) {
	t.Helper()
	if len(got) != 1 || !strings.HasPrefix(got[0], code+": ") || !strings.Contains(got[0], part) {
		t.Fatalf("want one %s finding containing %q, got %q", code, part, got)
	}
}

// 6ephqk check 2, rule 1: the stage is a stage of the task's state.
func TestLintStageOutsideItsState(t *testing.T) {
	if got := codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"ready\"")); len(got) != 0 {
		t.Fatalf("todo/ready is clean: %q", got)
	}
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"waiting\"")), "unknown-stage", "a todo task's stage is one of deferred, depends, design, draft, review, ready")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"ready\"")), "unknown-stage", "a doing task's stage is one of active, waiting, depends, review")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"dropped\"\nstage: \"active\"\ndropped_reason: \"dup\"")), "unknown-stage", "a dropped task has no stage")
	// review is a stage of both todo and doing: no finding under either.
	if got := codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"review\"\nreview_since: \"2026-11-17T10:00:00Z\"")); len(got) != 0 {
		t.Fatalf("todo/review: %q", got)
	}
	// No stage yet (a board before hug upgrade) is not flagged.
	if got := codes(stageDoc("aaaaaa", "status: \"doing\"")); len(got) != 0 {
		t.Fatalf("no stage: %q", got)
	}
}

// Rule 2: a node field is present only while the task is in its node.
func TestLintNodeFieldOutsideItsNode(t *testing.T) {
	ok := stageDoc("aaaaaa", "status: \"doing\"\nstage: \"waiting\"\ndoing_since: \"2026-11-17T08:46:00Z\"\nwaiting_on: \"CI run 44057678727\"\nwaiting_since: \"2026-11-17T09:41:00Z\"")
	if got := codes(ok); len(got) != 0 {
		t.Fatalf("doing/waiting with its fields is clean: %q", got)
	}
	if ok.Task.Fields["waiting_on"] != "CI run 44057678727" || ok.Task.Fields["doing_since"] != "2026-11-17T08:46:00Z" || ok.Task.Stage != "waiting" {
		t.Fatalf("parsed %q %q", ok.Task.Stage, ok.Task.Fields)
	}
	// A field of another state is never allowed.
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"ready\"\nwaiting_on: \"CI\"")), "stray-node-field", "node field waiting_on belongs to doing/waiting, and this task is todo/ready")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"active\"\ndone_at: \"2026-11-17T10:00:00Z\"")), "stray-node-field", "node field done_at belongs to done, and this task is doing/active")
	// Proposal A (crab 11:37): an outranked stage keeps its fields while
	// its condition holds: design_question on a todo task that depends.
	if got := codes(stageDoc("bbbbbb", "status: \"doing\"\nstage: \"active\""), stageDoc("aaaaaa", "status: \"todo\"\nstage: \"depends\"\ndesign_question: \"q\"\ndesign_for: \"jhf\"", "bbbbbb")); len(got) != 0 {
		t.Fatalf("design_question under depends: %q", got)
	}
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"ready\"\ndoing_since: \"2026-11-17T08:46:00Z\"")), "stray-node-field", "node field doing_since belongs to doing, and this task is todo/ready")
	// review_by belongs to todo/review and doing/review alike.
	if got := codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"review\"\nreview_by: \"Jørgen\"")); len(got) != 0 {
		t.Fatalf("todo/review review_by: %q", got)
	}
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"dropped\"\ndropped_reason: \"dup\"\nreview_by: \"Jørgen\"")), "stray-node-field", "belongs to todo/review and doing/review")
	// Unknown keys stay unknown keys, not node fields.
	if d := stageDoc("aaaaaa", "status: \"todo\"\nstage: \"ready\"\nshipped: \"v1\""); len(d.Task.Fields) != 0 {
		t.Fatal(d.Task.Fields)
	}
}

// Rule 3: the required fields of the state and stage are present and not
// empty.
func TestLintRequiredNodeFieldMissing(t *testing.T) {
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"waiting\"")), "missing-node-field", "missing node field waiting_on: a doing/waiting task needs it (what it waits on)")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"waiting\"\nwaiting_on: \"  \"")), "missing-node-field", "waiting_on")
	got := codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"design\""))
	if len(got) != 2 || !strings.Contains(strings.Join(got, "\n"), "missing node field design_question") || !strings.Contains(strings.Join(got, "\n"), "missing node field design_for") {
		t.Fatalf("design without question and person: %q", got)
	}
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"design\"\ndesign_question: \"q\"")), "missing-node-field", "missing node field design_for: a todo/design task needs it")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"review\"")), "missing-node-field", "review_by")
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"deferred\"")), "missing-node-field", "deferred_reason")
	// todo/review's review_by is optional (review_since marks it).
	if got := codes(stageDoc("aaaaaa", "status: \"todo\"\nstage: \"review\"\nreview_since: \"2026-11-17T10:00:00Z\"")); len(got) != 0 {
		t.Fatalf("todo/review: %q", got)
	}
	// A state's required field: dropped needs dropped_reason, once the
	// file has stages (a stage line or a node field marks a migrated file).
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"dropped\"\ndropped_at: \"2026-11-17T10:00:00Z\"")), "missing-node-field", "missing node field dropped_reason: a dropped task needs it (why never)")
	// A file hug upgrade has not migrated (neither) is not flagged.
	if got := codes(stageDoc("aaaaaa", "status: \"dropped\"")); len(got) != 0 {
		t.Fatalf("unmigrated dropped: %q", got)
	}
	if got := codes(stageDoc("aaaaaa", "status: \"dropped\"\ndropped_reason: \"duplicate of 805p6y\"")); len(got) != 0 {
		t.Fatalf("dropped with reason: %q", got)
	}
}

// Rule 4: the stage is the first in order whose condition holds
// (Proposal A, crab 11:37).
func TestLintDerivedStageDisagrees(t *testing.T) {
	dep := stageDoc("bbbbbb", "status: \"doing\"\nstage: \"active\"")
	done := stageDoc("cccccc", "status: \"dropped\"")
	wantOnly(t, codes(dep, stageDoc("aaaaaa", "status: \"todo\"\nstage: \"ready\"", "bbbbbb")), "stage-disagrees", "stage ready disagrees with the task: it depends on bbbbbb (doing), so its stage is depends")
	if got := codes(dep, stageDoc("aaaaaa", "status: \"todo\"\nstage: \"depends\"", "bbbbbb")); len(got) != 0 {
		t.Fatalf("todo/depends on a doing task: %q", got)
	}
	wantOnly(t, codes(done, stageDoc("aaaaaa", "status: \"todo\"\nstage: \"depends\"", "cccccc")), "stage-disagrees", "stage depends disagrees with the task: nothing holds it: no open dependency, and none of its stage's fields is set, so its stage is ready")
	wantOnly(t, codes(done, stageDoc("aaaaaa", "status: \"doing\"\nstage: \"depends\"", "cccccc")), "stage-disagrees", "stage depends disagrees with the task: nothing holds it: no open dependency, and none of its stage's fields is set, so its stage is active")
	// Precedence (Jørgen 10:18): depends beats design, deferred beats depends.
	wantOnly(t, codes(dep, stageDoc("aaaaaa", "status: \"todo\"\nstage: \"design\"\ndesign_question: \"q\"\ndesign_for: \"jhf\"", "bbbbbb")), "stage-disagrees", "it depends on bbbbbb (doing), so its stage is depends")
	if got := codes(dep, stageDoc("aaaaaa", "status: \"todo\"\nstage: \"deferred\"\ndeferred_reason: \"after v1\"", "bbbbbb")); len(got) != 0 {
		t.Fatalf("deferred over depends: %q", got)
	}
	// A doing task waiting outside the board: waiting beats depends.
	if got := codes(dep, stageDoc("aaaaaa", "status: \"doing\"\nstage: \"waiting\"\nwaiting_on: \"CI\"", "bbbbbb")); len(got) != 0 {
		t.Fatalf("waiting over depends: %q", got)
	}
	// Setting a stage's field without its stage: the stage disagrees.
	wantOnly(t, codes(stageDoc("aaaaaa", "status: \"doing\"\nstage: \"active\"\nwaiting_on: \"CI\"")), "stage-disagrees", "its waiting fields are set, so its stage is waiting")
	// draft: no acceptance item.
	draft := Parse("aaaaaa.t.md", []byte("---\nid: \"aaaaaa\"\ntitle: \"T\"\nstatus: \"todo\"\nstage: \"ready\"\n---\n\n## Goal\ng\n"))
	got := codes(draft)
	if len(got) != 2 || !strings.Contains(strings.Join(got, "\n"), "stage-disagrees: stage ready disagrees with the task: it is not complete (no Goal or no acceptance item), so its stage is draft") {
		t.Fatalf("draft: %q", got)
	}
	if DerivedStage(done.Task, nil) != "" || DerivedStage(stageDoc("aaaaaa", "status: \"todo\"").Task, nil) != "ready" {
		t.Fatal("DerivedStage")
	}
}
