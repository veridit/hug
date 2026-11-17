package importer

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

func TestGitHubIssuesImport(t *testing.T) {
	var issues []Issue
	if err := json.Unmarshal([]byte(`[
	 {"number":7,"title":"Fix login","body":"Users cannot log in.\n- [x] reproduce\n- [ ] fix","state":"OPEN","stateReason":"","url":"https://github.com/o/r/issues/7","createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-03T03:04:05Z","labels":[{"name":"bug"}],"assignees":[{"login":"ann"}],"milestone":{"title":"v1"}},
	 {"number":8,"title":"Old idea","body":"","state":"CLOSED","stateReason":"NOT_PLANNED","url":"https://github.com/o/r/issues/8","createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-04T03:04:05Z","labels":[],"assignees":[],"milestone":null},
	 {"number":9,"title":"Done thing","body":"Shipped.","state":"CLOSED","stateReason":"COMPLETED","url":"https://github.com/o/r/issues/9","createdAt":"2026-01-02T03:04:05Z","updatedAt":"2026-01-05T03:04:05Z","labels":[],"assignees":[],"milestone":null}
	]`), &issues); err != nil {
		t.Fatal(err)
	}
	stage := t.TempDir()
	if err := WriteIssues(stage, issues); err != nil {
		t.Fatal(err)
	}
	dst := &task.Store{Root: t.TempDir()}
	run := func() Report {
		r, err := Run(context.Background(), stage, dst, Options{Origin: "github"})
		if err != nil {
			t.Fatal(err)
		}
		return r
	}
	if r := run(); r.Added != 3 {
		t.Fatalf("first import: %+v", r)
	}
	if r := run(); r.Unchanged != 3 || r.Added+r.Updated != 0 {
		t.Fatalf("re-import not idempotent: %+v", r)
	}
	tasks, err := dst.Tasks()
	if err != nil {
		t.Fatal(err)
	}
	by := map[string]task.Task{}
	for _, tk := range tasks {
		for _, a := range tk.Aliases {
			by[a] = tk
		}
	}
	login := by["GH-7"]
	if login.State != "todo" || len(login.Acceptance) != 2 || !login.Acceptance[0].Checked || login.Acceptance[1].Checked {
		t.Errorf("GH-7: status %q acceptance %+v", login.State, login.Acceptance)
	}
	if len(login.Labels) != 1 || login.Labels[0] != "bug" || len(login.Assignee) != 1 || login.Assignee[0] != "ann" {
		t.Errorf("GH-7 labels/assignee: %+v %+v", login.Labels, login.Assignee)
	}
	if !strings.Contains(login.Goal, "Users cannot log in.") || strings.Contains(login.Goal, "reproduce") {
		t.Errorf("GH-7 goal should hold the body without the checklist: %q", login.Goal)
	}
	if by["GH-8"].State != "dropped" || by["GH-9"].State != "done" {
		t.Errorf("status: GH-8 %q, GH-9 %q", by["GH-8"].State, by["GH-9"].State)
	}
	if a := by["GH-9"].Acceptance; len(a) != 1 || !a[0].Checked {
		t.Errorf("GH-9 default acceptance: %+v", a)
	}
	files, _ := filepath.Glob(filepath.Join(dst.Root, "*.md"))
	for _, f := range files {
		b, _ := os.ReadFile(f)
		if !strings.Contains(string(b), `origin: "github:GH-`) {
			t.Errorf("%s: origin not github: %s", filepath.Base(f), b)
		}
	}
}
