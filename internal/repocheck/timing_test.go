package repocheck

import (
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// khq1fq: release.yml never fails on a wall-clock budget once CI passed
// the same SHA. v0.7.113's release (run 45627554531) failed on
// TestBigTaskKeys, 'tab, Notes open' p50 53.8 ms over its 50 ms budget,
// after CI 45625116019 passed the same code 8/8.
//
// A wall-clock budget is a test that fails when something took longer
// (or less long) than a fixed time: an ordered comparison (<, >, <=, >=)
// with time.Since(...), a time unit (time.Second ...) or a package
// duration constant on one side, outside a for loop's condition (a poll
// until a deadline waits for an event; it does not judge speed). Every
// function of a _test.go file holding one calls timing.Budget(t), which
// skips it under HUG_SKIP_TIMING=1, or is on notBudgets with the reason.

// notBudgets are functions the scan flags that judge no speed, by
// path:function, never by pattern.
var notBudgets = map[string]string{
	"huginn/internal/cache/versions_test.go:TestOpenMarksCacheUsed":                      "an mtime freshness check: open touched the cache within the hour",
	"muninn/internal/app/fixes_test.go:TestForgetLeavesTrail":                            "a recorded age (shown 90 s ago) checked for sense, not a measured duration",
	"huginn/internal/cli/sync_timeout_output_test.go:TestSettleWaitsOnProgressNotAClock": "synctest's fake clock: no wall clock is read",
	"huginn/internal/cli/sync_test.go:settleWait":                                        "the sync settle deadline (30 s stall, 3 min cap): waits for an event, says why it gave up",
	"cmd/hug/sync_test.go:busySyncers":                                                   "a pending file's age, not a measured duration",
}

// budgetCall is the helper a wall-clock budget test calls.
const budgetCall = "timing.Budget"

// durationUnits are the time package's units.
var durationUnits = map[string]bool{"Nanosecond": true, "Microsecond": true, "Millisecond": true, "Second": true, "Minute": true, "Hour": true}

// repoRoot is the checkout's top folder.
func repoRoot(t *testing.T) string {
	t.Helper()
	top, err := exec.Command("git", "rev-parse", "--show-toplevel").Output()
	if err != nil {
		t.Skipf("not in a git checkout: %v", err)
	}
	return strings.TrimSpace(string(top))
}

// trackedTestFiles lists the checkout's _test.go files by package folder.
func trackedTestFiles(t *testing.T, root string) map[string][]string {
	t.Helper()
	list := exec.Command("git", "ls-files", "-z", "--", "*_test.go")
	list.Dir = root
	out, err := list.Output()
	if err != nil {
		t.Fatalf("git ls-files: %v", err)
	}
	byDir := map[string][]string{}
	for _, rel := range strings.Split(strings.TrimRight(string(out), "\x00"), "\x00") {
		if rel == "" || strings.Contains("/"+rel, "/testdata/") {
			continue
		}
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(rel))); err != nil {
			continue // deleted in the worktree
		}
		byDir[path.Dir(rel)] = append(byDir[path.Dir(rel)], rel)
	}
	return byDir
}

// mentionsDuration reports whether e holds time.Since(...), a time unit,
// or one of the package's duration constants.
func mentionsDuration(e ast.Expr, consts map[string]bool) bool {
	found := false
	ast.Inspect(e, func(n ast.Node) bool {
		switch x := n.(type) {
		case *ast.SelectorExpr:
			if id, ok := x.X.(*ast.Ident); ok && id.Name == "time" && (x.Sel.Name == "Since" || durationUnits[x.Sel.Name]) {
				found = true
			}
		case *ast.Ident:
			if consts[x.Name] {
				found = true
			}
		}
		return !found
	})
	return found
}

// budgetGates lists, per function of the files, the lines of its
// wall-clock comparisons, and whether it calls timing.Budget.
type gatedFunc struct {
	lines  []int
	budget bool
}

func budgetGates(t *testing.T, root string, files []string) map[string]*gatedFunc {
	t.Helper()
	fset := token.NewFileSet()
	parsed := map[string]*ast.File{}
	consts := map[string]bool{} // package-level names whose value holds a time unit
	for _, rel := range files {
		f, err := parser.ParseFile(fset, filepath.Join(root, filepath.FromSlash(rel)), nil, parser.SkipObjectResolution)
		if err != nil {
			t.Fatalf("parse %s: %v", rel, err)
		}
		parsed[rel] = f
		for _, d := range f.Decls {
			g, ok := d.(*ast.GenDecl)
			if !ok || (g.Tok != token.CONST && g.Tok != token.VAR) {
				continue
			}
			for _, s := range g.Specs {
				v := s.(*ast.ValueSpec)
				for i, name := range v.Names {
					if i < len(v.Values) && mentionsDuration(v.Values[i], nil) {
						consts[name.Name] = true
					}
				}
			}
		}
	}
	out := map[string]*gatedFunc{}
	for rel, f := range parsed {
		for _, d := range f.Decls {
			fn, ok := d.(*ast.FuncDecl)
			if !ok || fn.Body == nil {
				continue
			}
			g := &gatedFunc{}
			loopConds := map[ast.Expr]bool{}
			ast.Inspect(fn.Body, func(n ast.Node) bool {
				switch x := n.(type) {
				case *ast.ForStmt:
					if x.Cond != nil {
						loopConds[x.Cond] = true
					}
				case *ast.CallExpr:
					if s, ok := x.Fun.(*ast.SelectorExpr); ok {
						if id, ok := s.X.(*ast.Ident); ok && id.Name+"."+s.Sel.Name == budgetCall {
							g.budget = true
						}
					}
				case *ast.BinaryExpr:
					if loopConds[x] {
						return false // a poll's deadline, and what it holds
					}
					switch x.Op {
					case token.LSS, token.GTR, token.LEQ, token.GEQ:
						if mentionsDuration(x.X, consts) || mentionsDuration(x.Y, consts) {
							g.lines = append(g.lines, fset.Position(x.Pos()).Line)
						}
					}
				}
				return true
			})
			if len(g.lines) > 0 || g.budget {
				out[rel+":"+fn.Name.Name] = g
			}
		}
	}
	return out
}

// TestEveryWallClockBudgetUsesTimingBudget: every test function that
// compares a measured time with a fixed one calls timing.Budget(t), so
// HUG_SKIP_TIMING=1 skips it in release.yml.
func TestEveryWallClockBudgetUsesTimingBudget(t *testing.T) {
	root := repoRoot(t)
	gates, budgets := 0, 0
	used := map[string]bool{}
	var missing []string
	for _, files := range trackedTestFiles(t, root) {
		for key, g := range budgetGates(t, root, files) {
			if g.budget {
				budgets++
			}
			if len(g.lines) == 0 {
				continue
			}
			gates++
			if _, ok := notBudgets[key]; ok {
				used[key] = true
				continue
			}
			if !g.budget {
				rel, fn, _ := strings.Cut(key, ":")
				missing = append(missing, fmt.Sprintf("%s:%d: %s compares a measured time with a fixed one (lines %v) without calling timing.Budget(t)\nfix: add timing.Budget(t) (github.com/veridit/hug/internal/timing) at the top of %s, or list %q in notBudgets (internal/repocheck/timing_test.go) with why it judges no speed", rel, g.lines[0], fn, g.lines, fn, key))
			}
		}
	}
	sort.Strings(missing)
	for _, m := range missing {
		t.Error(m)
	}
	for key := range notBudgets {
		if !used[key] {
			t.Errorf("notBudgets lists %s, which the scan no longer flags\nfix: delete its line in internal/repocheck/timing_test.go", key)
		}
	}
	if gates < 10 {
		t.Fatalf("the scan found only %d functions with a wall-clock comparison: is the root right (%s)?", gates, root)
	}
	t.Logf("%d functions compare a measured time, %d call timing.Budget, %d listed as no budget", gates, budgets, len(notBudgets))
}

// workflow is the part of a GitHub Actions file this check reads.
type workflow struct {
	Env  map[string]string `yaml:"env"`
	Jobs map[string]struct {
		Env   map[string]string `yaml:"env"`
		Steps []struct {
			Name string            `yaml:"name"`
			Run  string            `yaml:"run"`
			Env  map[string]string `yaml:"env"`
		} `yaml:"steps"`
	} `yaml:"jobs"`
}

func readWorkflow(t *testing.T, root, name string) workflow {
	t.Helper()
	b, err := os.ReadFile(filepath.Join(root, ".github", "workflows", name))
	if err != nil {
		t.Fatal(err)
	}
	var w workflow
	if err := yaml.Unmarshal(b, &w); err != nil {
		t.Fatalf("%s: %v", name, err)
	}
	return w
}

// TestReleaseSkipsWallClockBudgets: every go test step of release.yml
// runs with HUG_SKIP_TIMING=1, and ci.yml never sets it, so CI still
// gates the budgets (on macOS: ubuntu runs -race, where they skip).
func TestReleaseSkipsWallClockBudgets(t *testing.T) {
	root := repoRoot(t)
	const skip = "HUG_SKIP_TIMING"
	rel := readWorkflow(t, root, "release.yml")
	tests := 0
	for job, j := range rel.Jobs {
		for _, s := range j.Steps {
			if !strings.Contains(s.Run, "go test") {
				continue
			}
			tests++
			v, ok := s.Env[skip]
			if !ok {
				v, ok = j.Env[skip]
			}
			if !ok {
				v = rel.Env[skip]
			}
			if v != "1" {
				t.Errorf("release.yml job %s step %q runs go test with %s=%q, want 1: a wall-clock budget would re-roll a release CI already passed\nfix: add 'env: {%s: \"1\"}' to that step", job, s.Name, skip, v, skip)
			}
		}
	}
	if tests == 0 {
		t.Error("release.yml has no go test step: the check found nothing to judge\nfix: restore the Test step of .github/workflows/release.yml")
	}
	ci := readWorkflow(t, root, "ci.yml")
	if _, ok := ci.Env[skip]; ok {
		t.Errorf("ci.yml sets %s for the whole workflow: CI must gate the wall-clock budgets\nfix: delete it from ci.yml", skip)
	}
	for job, j := range ci.Jobs {
		if _, ok := j.Env[skip]; ok {
			t.Errorf("ci.yml job %s sets %s: CI must gate the wall-clock budgets\nfix: delete it from that job", job, skip)
		}
		for _, s := range j.Steps {
			if _, ok := s.Env[skip]; ok {
				t.Errorf("ci.yml job %s step %q sets %s: CI must gate the wall-clock budgets\nfix: delete it from that step", job, s.Name, skip)
			}
		}
	}
}
