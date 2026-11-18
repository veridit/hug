package task

import (
	"errors"
	"strings"
	"testing"
)

// siblingFields are the stages of state that own fields (asked, waiting,
// review in doing; deferred, design, review in todo), with the fields of
// each that t carries.
func siblingFields(t Task) map[string][]string {
	out := map[string][]string{}
	for _, g := range StageNames(t.State) {
		for _, f := range stageFields(t.State, g) {
			if strings.TrimSpace(t.Fields[f]) != "" {
				out[g] = append(out[g], f)
			}
		}
	}
	return out
}

// stageLint is d's stage findings on board ts: what lint says about its
// stage and node fields.
func stageLint(d *Document, ts []Task) []Finding {
	var out []Finding
	for _, f := range Lint([]*Document{d}) {
		if c := FindingCode(f.Problem); c == "stage-disagrees" || c == "stray-node-field" || c == "missing-node-field" {
			out = append(out, f)
		}
	}
	return out
}

// 3nhp67 check 1 (crab 02:26): s6ff5k held asked_* and waiting_* at once
// after a wait on an asked task (8366bc00 on hug/tasks), and lint called it
// broken. Every stage command from every stage leaves the task in one
// stage: the fields of the stage it replaces go to History, or the command
// refuses with a fix and writes nothing.
func TestStageChangeClearsTheOldStage(t *testing.T) {
	type cmd struct {
		name string
		run  func(s *Store, id string) (Result, error)
	}
	doingCmds := []cmd{
		{"wait", func(s *Store, id string) (Result, error) { return s.Wait(id, "CI run 9", "x") }},
		{"ask", func(s *Store, id string) (Result, error) {
			return s.Ask(id, Question{What: "which one?", For: "jhf"}, "x")
		}},
		{"review", func(s *Store, id string) (Result, error) { return s.Review(id, "rev", "") }},
		{"resume", func(s *Store, id string) (Result, error) { return s.Resume(id) }},
	}
	doingFrom := map[string]func(s *Store, id string){
		"active":  func(s *Store, id string) {},
		"asked":   func(s *Store, id string) { must(t)(s.Ask(id, Question{What: "this or that?", For: "crab"}, "x")) },
		"waiting": func(s *Store, id string) { must(t)(s.Wait(id, "a release", "x")) },
		"review":  func(s *Store, id string) { must(t)(s.Review(id, "rev", "")) },
	}
	for from, into := range doingFrom {
		for _, c := range doingCmds {
			t.Run("doing/"+from+" "+c.name, func(t *testing.T) {
				s := &Store{Root: t.TempDir()}
				id := newTest(t, s, "Stage pair").ID
				must(t)(s.Take(id, "x"))
				into(s, id)
				before, _ := s.Get(id)
				_, e := c.run(s, id)
				after, _ := s.Get(id)
				if e != nil {
					var te *Error
					if !errors.As(e, &te) || te.Fix == "" {
						t.Fatalf("refused without a fix: %v", e)
					}
					if string(before.Bytes()) != string(after.Bytes()) {
						t.Fatalf("refused but wrote:\n%s", after.Bytes())
					}
					return
				}
				checkOneStage(t, s, after, before)
			})
		}
	}
	todoFrom := map[string]func(s *Store, id string){
		"ready":  func(s *Store, id string) {},
		"design": func(s *Store, id string) { must(t)(s.Design(id, "which way?", "x", "jhf")) },
		"review": func(s *Store, id string) {
			must(t)(s.Take(id, "x"))
			must(t)(s.Review(id, "rev", ""))
			must(t)(s.Set(id, "state", "todo"))
		},
	}
	for from, into := range todoFrom {
		t.Run("todo/"+from+" design", func(t *testing.T) {
			s := &Store{Root: t.TempDir()}
			id := newTest(t, s, "Stage pair").ID
			into(s, id)
			before, _ := s.Get(id)
			if _, e := s.Design(id, "a new question?", "x", "jhf"); e != nil {
				t.Fatal(e)
			}
			after, _ := s.Get(id)
			checkOneStage(t, s, after, before)
		})
	}
}

// checkOneStage: after carries one stage's fields at most, its stage line
// is the stage those fields make, lint has nothing to say about its stage,
// and every field before had that is gone is named in History.
func checkOneStage(t *testing.T, s *Store, after, before *Document) {
	t.Helper()
	ts, _ := s.Tasks()
	if held := siblingFields(after.Task); len(held) > 1 {
		t.Fatalf("two stages' fields %v:\n%s", held, after.Bytes())
	}
	if fs := stageLint(after, ts); len(fs) > 0 {
		t.Fatalf("lint: %s (fix: %s)\n%s", fs[0].Problem, fs[0].FixLine(), after.Bytes())
	}
	for k, v := range before.Task.Fields {
		if _, kept := after.Task.Fields[k]; !kept && !strings.Contains(string(after.Bytes()), k+": "+v) {
			t.Fatalf("%s: %s went, History does not keep it:\n%s", k, v, after.Bytes())
		}
	}
}

// 3nhp67 check 2: resume from asked or waiting clears that stage's fields,
// and only those (doing_since, the owner, the rest stay).
func TestResumeClearsOnlyItsStage(t *testing.T) {
	for _, from := range []string{"asked", "waiting"} {
		t.Run(from, func(t *testing.T) {
			s := &Store{Root: t.TempDir()}
			id := newTest(t, s, "Resume").ID
			must(t)(s.Take(id, "x"))
			if from == "asked" {
				must(t)(s.Ask(id, Question{What: "which?", For: "jhf", Check: 1}, "x"))
			} else {
				must(t)(s.Wait(id, "CI", "x"))
			}
			before, _ := s.Get(id)
			must(t)(s.Resume(id))
			after, _ := s.Get(id)
			if after.Task.Stage != "active" || len(siblingFields(after.Task)) > 0 {
				t.Fatalf("resume left %v, stage %s", siblingFields(after.Task), after.Task.Stage)
			}
			for k, v := range before.Task.Fields {
				if !strings.HasPrefix(k, from+"_") && after.Task.Fields[k] != v {
					t.Fatalf("resume changed %s (%q -> %q)", k, v, after.Task.Fields[k])
				}
			}
			if a := after.Task.Assignee; len(a) != 1 || a[0] != "x" {
				t.Fatalf("assignee %v", a)
			}
		})
	}
}

// 3nhp67 check 3 and 4: a file as s6ff5k is on hug's board (5b8733b9),
// stage waiting with asked_* and waiting_* fields, lints as
// stage-disagrees, and running the finding's own fix settles it: stage
// waiting with waiting_* only, asked_* in History, lint quiet.
func TestStageDisagreesFixSettles(t *testing.T) {
	for _, c := range []struct {
		name, extra, stage, want string
	}{
		// s6ff5k: the stage line names waiting; asked_* are left over.
		{"s6ff5k", "asked_by: \"hatchling\"\nasked_for: \"crab\"\nasked_since: \"2026-11-17T20:20\"\nasked_what: \"H3 failed Part 2; parked per crab ruling\"\nwaiting_on: \"parked: no fourth arm\"\nwaiting_since: \"2026-11-18T02:10\"\nwaiting_asked_by: \"hatchling\"\n", "waiting", "waiting"},
		// Agreeing line, two stages' fields: still one stage after the fix.
		{"asked line", "asked_for: \"crab\"\nasked_what: \"which?\"\nwaiting_on: \"CI\"\n", "asked", "asked"},
		// A line naming a stage with no fields: the fields decide.
		{"active line", "waiting_on: \"CI\"\n", "active", "waiting"},
	} {
		t.Run(c.name, func(t *testing.T) {
			s := &Store{Root: t.TempDir()}
			id := newTest(t, s, "Settle").ID
			must(t)(s.Take(id, "x"))
			d, _ := s.Get(id)
			raw := strings.Replace(string(d.Bytes()), "\nstage: \"active\"\n", "\nstage: \""+c.stage+"\"\n"+c.extra, 1)
			if raw == string(d.Bytes()) {
				t.Fatalf("no stage line to replace:\n%s", raw)
			}
			path, _ := s.diskPath(d.Task.Path)
			if e := AtomicWrite(path, []byte(raw)); e != nil {
				t.Fatal(e)
			}
			s.Invalidate()
			d, _ = s.Get(id)
			ts, _ := s.Tasks()
			fs := stageLint(d, ts)
			if len(fs) == 0 {
				t.Fatalf("lint is quiet on two stages' fields:\n%s", raw)
			}
			fix := fs[0].FixLine()
			want := "hug task set " + id + " --state doing"
			if fix != want {
				t.Fatalf("fix %q, want %q", fix, want)
			}
			// The fix, as the CLI runs it.
			if _, e := s.Set(id, "state", "doing"); e != nil {
				t.Fatal(e)
			}
			d, _ = s.Get(id)
			ts, _ = s.Tasks()
			if fs := stageLint(d, ts); len(fs) > 0 {
				t.Fatalf("after the fix, lint: %s\n%s", fs[0].Problem, d.Bytes())
			}
			if d.Task.Stage != c.want || len(siblingFields(d.Task)) > 1 {
				t.Fatalf("stage %s, fields %v, want %s:\n%s", d.Task.Stage, siblingFields(d.Task), c.want, d.Bytes())
			}
		})
	}
}

func must(t *testing.T) func(Result, error) {
	return func(_ Result, e error) {
		t.Helper()
		if e != nil {
			t.Fatal(e)
		}
	}
}
