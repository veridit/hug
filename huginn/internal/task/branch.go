package task

import (
	"os/exec"
	"strings"
)

// BranchFindings flags doing tasks whose branch is merged into the main
// branch or no longer exists (x1t1t1): the work has landed or was dropped,
// and the task most likely should be closed. It reads git state, not task
// files, so it is never cached with Lint's findings: call it on each lint.
// It runs two git commands in total, however many tasks there are. Tasks
// on main itself, and tasks without a branch, are skipped. Nothing is
// changed: the finding only says so.
func BranchFindings(repoDir, main string, ts []Task) []Finding {
	var doing []Task
	for _, t := range ts {
		if t.Status == "doing" && t.Branch != "" && t.Branch != main {
			doing = append(doing, t)
		}
	}
	if len(doing) == 0 || main == "" {
		return nil
	}
	refs, err := gitLines(repoDir, "for-each-ref", "--format=%(refname:short)", "refs/heads")
	if err != nil {
		return nil // not a git checkout here: nothing to say
	}
	exists := map[string]bool{}
	for _, r := range refs {
		exists[r] = true
	}
	if !exists[main] {
		return nil
	}
	merged := map[string]bool{}
	if lines, err := gitLines(repoDir, "branch", "--format=%(refname:short)", "--merged", main); err == nil {
		for _, b := range lines {
			merged[b] = true
		}
	}
	var out []Finding
	for _, t := range doing {
		var problem string
		switch {
		case !exists[t.Branch]:
			problem = "branch " + t.Branch + " is deleted, but " + t.ID + " is still doing"
		case merged[t.Branch]:
			problem = "branch " + t.Branch + " is merged into " + main + ", but " + t.ID + " is still doing"
		default:
			continue
		}
		out = append(out, Finding{Path: t.Path, Line: 1, Problem: problem, Code: "branch-gone", Fix: "hug task done " + t.ID})
	}
	return out
}

func gitLines(dir string, args ...string) ([]string, error) {
	c := exec.Command("git", append([]string{"-C", dir}, args...)...)
	c.Env = append(c.Environ(), "GIT_OPTIONAL_LOCKS=0")
	b, err := c.Output()
	if err != nil {
		return nil, err
	}
	var out []string
	for _, l := range strings.Split(string(b), "\n") {
		if l = strings.TrimSpace(l); l != "" {
			out = append(out, l)
		}
	}
	return out, nil
}
