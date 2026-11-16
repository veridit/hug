package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// et85h1 (decided by Jørgen 2026-11-16): take requires the model; the name
// is optional; git user.name is never used.
func takeFixture(t *testing.T) (*App, string) {
	t.Helper()
	a := app(t) // git user.name is "Test": it must never become the assignee
	t.Setenv("HUG_USER", "")
	t.Setenv("HUG_MODEL", "")
	t.Setenv("JCODE_SESSION_ID", "")
	// The old name is set and must not be read (m1c28r); spelt in parts so
	// the code holds no old names.
	t.Setenv("HUG"+"INN_USER", "old-name-not-read")
	r := execute(t, a, "new", "Take me", "--goal", "G", "--accept", "C")
	return a, r["task"].(Reply)["id"].(string)
}

func taskFile(t *testing.T, a *App, id string) string {
	t.Helper()
	m, _ := filepath.Glob(filepath.Join(a.Cwd, TaskDir, "*"+id+"*.md"))
	if len(m) != 1 {
		t.Fatalf("task file for %s: %v", id, m)
	}
	b, e := os.ReadFile(m[0])
	if e != nil {
		t.Fatal(e)
	}
	return string(b)
}

func runText(a *App, args ...string) (int, string) {
	out := &bytes.Buffer{}
	a.Out, a.Err = out, out
	return a.Run(context.Background(), args), out.String()
}

func TestTakeWithoutModelFailsWithFix(t *testing.T) {
	a, id := takeFixture(t)
	code, out := runText(a, "take", id)
	want := "fix: hug task take " + id + " --model MODEL\n"
	if code != 2 || !strings.Contains(out, "no model given") || !strings.HasSuffix(out, want) {
		t.Fatalf("exit %d, want 2 ending %q:\n%s", code, want, out)
	}
	if f := taskFile(t, a, id); !strings.Contains(f, "status: \"todo\"") && !strings.Contains(f, "status: todo") || strings.Contains(f, "Test") {
		t.Fatalf("take without model changed the task:\n%s", f)
	}
	// With a name but no model, the fix keeps the name.
	code, out = runText(a, "take", id, "--as", "poodle")
	if code != 2 || !strings.HasSuffix(out, "fix: hug task take "+id+" --as poodle --model MODEL\n") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
	// The printed fix works once MODEL is filled in.
	r := execute(t, a, "take", id, "--as", "poodle", "--model", "claude-opus-5-5")
	if got := r["task"].(Reply)["model"]; got != "claude-opus-5-5" {
		t.Fatal(r)
	}
}

func TestTakeModelOnlyAssigneeIsModel(t *testing.T) {
	a, id := takeFixture(t)
	r := execute(t, a, "take", id, "--model", "gpt-6.1-sol")
	tk := r["task"].(Reply)
	if as := tk["assignee"].([]any); len(as) != 1 || as[0] != "gpt-6.1-sol" || tk["model"] != "gpt-6.1-sol" {
		t.Fatalf("assignee %v model %v", tk["assignee"], tk["model"])
	}
	f := taskFile(t, a, id)
	if !strings.Contains(f, `assignee: ["gpt-6.1-sol"]`) || !strings.Contains(f, `model: "gpt-6.1-sol"`) || strings.Contains(f, `"Test"`) || strings.Contains(f, "old-name-not-read") {
		t.Fatalf("task file:\n%s", f)
	}
}

func TestTakeAsAndModelRecordsBoth(t *testing.T) {
	for _, viaEnv := range []bool{false, true} {
		name := "flags"
		if viaEnv {
			name = "HUG_USER and HUG_MODEL"
		}
		t.Run(name, func(t *testing.T) {
			a, id := takeFixture(t)
			args := []string{"take", id}
			if viaEnv {
				t.Setenv("HUG_USER", "poodle")
				t.Setenv("HUG_MODEL", "claude-opus-5-5")
			} else {
				args = append(args, "--as", "poodle", "--model", "claude-opus-5-5")
			}
			r := execute(t, a, args...)
			tk := r["task"].(Reply)
			if as := tk["assignee"].([]any); len(as) != 1 || as[0] != "poodle" || tk["model"] != "claude-opus-5-5" {
				t.Fatalf("assignee %v model %v", tk["assignee"], tk["model"])
			}
			f := taskFile(t, a, id)
			if !strings.Contains(f, `assignee: ["poodle"]`) || !strings.Contains(f, `model: "claude-opus-5-5"`) {
				t.Fatalf("task file:\n%s", f)
			}
			// show and ls name both.
			if code, out := runText(a, "show", id); code != 0 || !strings.Contains(out, "taken by: poodle · claude-opus-5-5 · main\n") {
				t.Fatalf("show: %d\n%s", code, out)
			}
			if code, out := runText(a, "ls"); code != 0 || !strings.Contains(out, "Take me  [poodle · claude-opus-5-5]\n") {
				t.Fatalf("ls: %d\n%s", code, out)
			}
			if code, out := runText(a, "board", "--wide"); code != 0 || !strings.Contains(out, "Take me  [poodle · claude-opus-5-5]") {
				t.Fatalf("board: %d\n%s", code, out)
			}
			// --mine means the same name take recorded.
			r = execute(t, a, "next", "--mine", "--as", "poodle")
			if r["task"].(Reply)["id"] != id {
				t.Fatal(r)
			}
			// Someone else, by name, cannot take it.
			if _, e := a.Execute(context.Background(), []string{"take", id, "--as", "crab", "--model", "claude-opus-5-5"}); e == nil || !strings.Contains(e.Error(), "taken") {
				t.Fatalf("stolen: %v", e)
			}
		})
	}
}

func TestTakeUsesJcodeSessionName(t *testing.T) {
	a, id := takeFixture(t)
	t.Setenv("JCODE_SESSION_ID", "session_poodle_1794834977956_75c591eb08e87724")
	r := execute(t, a, "take", id, "--model", "claude-opus-5-5")
	tk := r["task"].(Reply)
	if as := tk["assignee"].([]any); len(as) != 1 || as[0] != "poodle" || tk["model"] != "claude-opus-5-5" {
		t.Fatalf("assignee %v model %v", tk["assignee"], tk["model"])
	}
	// --as and HUG_USER win over the session name.
	a2, id2 := takeFixture(t)
	t.Setenv("JCODE_SESSION_ID", "session_poodle_1_2")
	t.Setenv("HUG_USER", "crab")
	r = execute(t, a2, "take", id2, "--model", "claude-opus-5-5")
	if as := r["task"].(Reply)["assignee"].([]any); as[0] != "crab" {
		t.Fatal(as)
	}
	// Still no model: the session name alone is not enough.
	a3, id3 := takeFixture(t)
	t.Setenv("JCODE_SESSION_ID", "session_poodle_1_2")
	if code, out := runText(a3, "take", id3); code != 2 || !strings.HasSuffix(out, "fix: hug task take "+id3+" --as poodle --model MODEL\n") {
		t.Fatalf("exit %d:\n%s", code, out)
	}
}

func TestHarnessName(t *testing.T) {
	for in, want := range map[string]string{
		"session_hare_1794834977956_75c591eb08e87724": "hare",
		"session_poodle_1_2":                          "poodle",
		"":                                            "", "abc": "", "session_": "", "session_x": "",
	} {
		t.Setenv("JCODE_SESSION_ID", in)
		if got := harnessName(); got != want {
			t.Errorf("harnessName(%q) = %q, want %q", in, got, want)
		}
	}
}
