package task

import (
	"strings"
	"testing"
)

// jr55s8: the [models] roster is read and written as hug.toml lines, and a
// Model: value is checked against it, or against EffortWords without one.
func TestModelRoster(t *testing.T) {
	text := "[people.jhf]\nname = \"J\"\n\n[models]\n\"claude-opus-5-5\" = [\"low\", \"high\"]\n# a comment\n"
	ms := ParseModels(text)
	if len(ms) != 1 || ms[0].ID != "claude-opus-5-5" || strings.Join(ms[0].Levels, ",") != "low,high" {
		t.Fatalf("parse: %+v", ms)
	}
	// Replace in place, add after the last line, create the table.
	got := WithModel(text, Model{ID: "claude-opus-5-5", Levels: []string{"max"}})
	if !strings.Contains(got, "\"claude-opus-5-5\" = [\"max\"]\n") || strings.Contains(got, "\"low\"") || !strings.HasPrefix(got, "[people.jhf]\nname = \"J\"\n") {
		t.Fatalf("replace:\n%s", got)
	}
	got = WithModel(text, Model{ID: "gpt-6.1-sol", Levels: []string{"xhigh"}})
	if ms := ParseModels(got); len(ms) != 2 || ms[1].ID != "gpt-6.1-sol" {
		t.Fatalf("add:\n%s", got)
	}
	if got = WithModel("[people.jhf]\nname = \"J\"\n", Model{ID: "m", Levels: []string{"low"}}); got != "[people.jhf]\nname = \"J\"\n\n[models]\n\"m\" = [\"low\"]\n" {
		t.Fatalf("create:\n%q", got)
	}
	for _, c := range []struct {
		v      string
		roster []Model
		bad    string
	}{
		{"human", nil, ""},
		{"claude-opus-5-5/high", nil, ""},
		{"claude-opus-5-5/hihg", nil, "not an effort word"},
		{"claude-opus-5-5", nil, "names no effort"},
		{"claude-opus-5-5/high", ms, ""},
		{"claude-opus-5-5/max", ms, "not one of claude-opus-5-5's levels"},
		{"gpt-6.1-sol/high", ms, "not in hug.toml [models]"},
		{"/high", nil, "names no model"},
	} {
		p, _ := ModelProblem(c.v, c.roster)
		if (c.bad == "") != (p == "") || !strings.Contains(p, c.bad) {
			t.Errorf("%q: %q, want %q", c.v, p, c.bad)
		}
	}
}
