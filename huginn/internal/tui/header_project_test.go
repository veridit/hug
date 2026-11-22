package tui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/hug/huginn/internal/task"
)

// 0sw0dd check 2 (Jørgen 2026-11-16): the header reads 'HUG · <project>',
// the project being the checkout's folder name, so two checkouts of one
// repository read differently (hug, hug-testing).
func TestHeaderNamesProjectFromCheckout(t *testing.T) {
	m := fixture(t)
	m.opts.Checkout = filepath.Join(t.TempDir(), "hug-testing")
	m = newModel(m.store, m.opts)
	m.setView("list")
	first := strings.Split(ansi.Strip(m.View()), "\n")[0]
	if !strings.HasPrefix(first, "HUG · hug-testing · list ") {
		t.Fatalf("list header: %q", first)
	}
	if strings.Contains(first, "HUG TASKS") {
		t.Fatalf("old header still drawn: %q", first)
	}
	m = press(m, "enter")
	if first = strings.Split(ansi.Strip(m.View()), "\n")[0]; !strings.HasPrefix(first, "HUG · hug-testing · list › ") {
		t.Fatalf("detail breadcrumb: %q", first)
	}
}

// The override is there from the start (Jørgen 2026-11-16): hug.toml
// [board] project names the project on every checkout, and the TUI reads
// it again when hug.toml changes.
func TestHeaderProjectOverrideFromHugToml(t *testing.T) {
	m := fixture(t)
	m.opts.Checkout = filepath.Join(t.TempDir(), "hug-testing")
	if err := os.WriteFile(filepath.Join(m.store.Root, task.ConfigName), []byte("[board]\ncoordinator = \"crab\"\nproject = \"Seid\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	m = newModel(m.store, m.opts)
	m.setView("list")
	if first := strings.Split(ansi.Strip(m.View()), "\n")[0]; !strings.HasPrefix(first, "HUG · Seid · list ") {
		t.Fatalf("override header: %q", first)
	}
	if got, err := task.ProjectName(m.store.Root); err != nil || got != "Seid" {
		t.Fatalf("ProjectName = %q, %v", got, err)
	}
}

// Without a checkout (a store in a test, no git) and no override, the
// header is HUG and the view: never an empty project name.
func TestHeaderWithoutProjectIsHUG(t *testing.T) {
	m := fixture(t)
	if first := strings.Split(ansi.Strip(m.View()), "\n")[0]; !strings.HasPrefix(first, "HUG · list ") {
		t.Fatalf("header: %q", first)
	}
}

// 0sw0dd check 3: state:done (and the older status:done) in / search
// returns the done tasks; field terms combine with words and with each
// other, and the words alone still search.
func TestSearchFieldStateDone(t *testing.T) {
	m := fixture(t)
	ts, _ := m.store.Tasks()
	ids := map[string]string{}
	for _, tk := range ts {
		ids[tk.Title] = tk.ID
	}
	if _, err := m.store.New(task.NewOptions{Title: "Gamma task", Goal: "G", Acceptance: []string{"C"}, Type: "bug"}); err != nil {
		t.Fatal(err)
	}
	done := func(id string, checks int) {
		t.Helper()
		for n := 1; n <= checks; n++ {
			if _, err := m.store.Check(id, n); err != nil {
				t.Fatal(err)
			}
		}
		if _, err := m.store.Take(id, "tester"); err != nil {
			t.Fatal(err)
		}
		if _, err := m.store.Done(id); err != nil {
			t.Fatal(err)
		}
	}
	done(ids["Alpha task"], 2)
	done(ids["Beta task"], 1)
	for _, c := range []struct {
		query string
		want  []string
	}{
		{"state:done", []string{"Alpha task", "Beta task"}},
		{"status:done", []string{"Alpha task", "Beta task"}},
		{"State:Done", []string{"Alpha task", "Beta task"}},
		{"state:done alpha", []string{"Alpha task"}},
		{"state:done type:bug", []string{"Alpha task"}},
		{"state:todo", []string{"Gamma task"}},
		{"type:bug", []string{"Alpha task", "Gamma task"}},
		// tag is label's new name (txjkrm): both, until the CLI says tag.
		{"tag:urgent", []string{"Alpha task"}},
		{"label:urgent", []string{"Alpha task"}},
		{"gamma", []string{"Gamma task"}},
	} {
		got := press(newModel(m.store, m.opts), "/", c.query)
		got.retireDefault() // every task: the list's default aside (7zg9mf)
		got.setView("list")
		got.applyFilter()
		if strings.Join(sortedTitles(got.visible), ",") != strings.Join(c.want, ",") {
			t.Errorf("/ %s: %v, want %v", c.query, titles(got.visible), c.want)
		}
	}
}

func sortedTitles(ts []task.Task) []string {
	out := titles(ts)
	for i := range out {
		for j := i + 1; j < len(out); j++ {
			if out[j] < out[i] {
				out[i], out[j] = out[j], out[i]
			}
		}
	}
	return out
}
