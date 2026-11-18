package cli

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// hug upgrade --tags (txjkrm check 5) cleans a board's labels against its
// hug.toml tags list: it previews, in counts and one line per task, and
// writes nothing; --apply writes it in one commit. A board with no list
// gets one proposed: the tags on --min (default 3) or more tasks. It is its
// own step, apart from hug upgrade's mechanical migrations, because it
// changes what tasks say, which a person approves by reading the diff.
func (a *App) runUpgradeTags(ctx context.Context, f flags) (Reply, error) {
	if e := f.count(0); e != nil {
		return nil, e
	}
	if f.has("machine") {
		return nil, usage("--tags changes this board only, never this machine", "hug upgrade --tags")
	}
	min := 3
	if v := f.get("min"); v != "" {
		n, e := strconv.Atoi(v)
		if e != nil || n < 1 {
			return nil, usage("--min takes a number of tasks, 1 or more: "+v, "hug upgrade --tags --min 3")
		}
		min = n
	}
	dry := !f.has("apply")
	s, e := a.open(ctx)
	if e != nil {
		return nil, e
	}
	defer s.c.Close()
	locks, e := s.r.Background()
	if e != nil {
		return nil, e
	}
	release, e := locks.Exclusive()
	if e != nil {
		return nil, e
	}
	defer release()
	turn, e := locks.Turn()
	if e != nil {
		return nil, e
	}
	defer turn()
	s.s.Invalidate()
	list, e := task.Tags(s.r.Dir)
	if e != nil {
		return nil, e
	}
	ts, e := s.s.Tasks()
	if e != nil {
		return nil, e
	}
	plan := task.PlanTags(list, ts, min)
	if plan.Empty() {
		return Reply{"upgrade": []string{"tags: nothing to do"}, "next": "hug task lint --code unlisted-tag"}, nil
	}
	lines := plan.Lines(min)
	prefix := "tags: "
	if dry {
		prefix = "tags: would: "
	}
	for i := range lines {
		lines[i] = prefix + lines[i]
	}
	reply := Reply{"upgrade": lines, "diff": plan.Diff(), "next": "hug task lint --code unlisted-tag"}
	if dry {
		reply["pending"] = true
		reply["fix"] = "hug upgrade --tags --apply"
		reply["next"] = "hug upgrade --tags --apply"
		reply["exit_code"] = UpgradePending
		return reply, nil
	}
	paths, e := s.s.ApplyTags(plan)
	if e == nil && len(plan.List) > 0 {
		e = writeTags(s.r.Dir, plan.List)
		paths = append(paths, task.ConfigName)
	}
	if e == nil {
		e = s.r.Commit(paths, "hug upgrade --tags: "+tagsCommitSummary(plan))
	}
	if e != nil {
		return nil, e
	}
	return reply, nil
}

func tagsCommitSummary(p task.TagPlan) string {
	var parts []string
	if len(p.List) > 0 {
		parts = append(parts, fmt.Sprintf("hug.toml lists %d tags", len(p.List)))
	}
	n := len(p.Changes)
	noun := "tasks"
	if n == 1 {
		noun = "task"
	}
	parts = append(parts, fmt.Sprintf("labels cleaned in %d %s", n, noun))
	return strings.Join(parts, ", ")
}

// writeTags puts a proposed tags list in the board's hug.toml.
func writeTags(dir string, names []string) error {
	path := filepath.Join(dir, task.ConfigName)
	b, e := os.ReadFile(path)
	if e != nil && !os.IsNotExist(e) {
		return e
	}
	return os.WriteFile(path, []byte(task.WithTags(string(b), names)), 0o644)
}
