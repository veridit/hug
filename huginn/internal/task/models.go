package task

import (
	"os"
	"strings"
)

// The model roster (jr55s8, Jørgen 2026-11-17 in Seid): every commit names
// the model that made it and the effort it ran at, "Model: <id>/<effort>",
// since effort can change a model's behaviour entirely. hug.toml [models]
// lists each model's own levels, one line per model:
//
//	[models]
//	"claude-opus-5-5" = ["low", "medium", "high", "max"]
//	"gpt-6.1-sol" = ["none", "low", "medium", "high", "xhigh"]
//
// No model list lives in code. Without a [models] table an effort is
// checked against EffortWords only: the shape of the word, not a roster.

// EffortWords are the effort levels hug knows the words of (crab 20:01):
// a commit without a [models] roster may name any of them, so a typo
// ("hihg") is still refused.
var EffortWords = []string{"none", "minimal", "low", "medium", "high", "xhigh", "max"}

// Model is one [models] line: a model id and the efforts it runs at.
type Model struct {
	ID     string   `json:"id"`
	Levels []string `json:"levels"`
}

// Models reads hug.toml [models], in the order written. A missing file or
// table is no roster (nil, nil).
func Models(root string) ([]Model, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	return ParseModels(string(b)), nil
}

// ParseModels reads the [models] table of hug.toml text.
func ParseModels(text string) []Model {
	var out []Model
	in := false
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = strings.TrimSpace(strings.Trim(line, "[]")) == "models"
			continue
		}
		if !in {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		if id := scalar(key); id != "" {
			out = append(out, Model{ID: id, Levels: tomlStrings(value)})
		}
	}
	return out
}

// SplitModel splits a Model: value "id/effort" at its last slash; effort
// is "" when there is none. Model ids never hold a slash (jr55s8).
func SplitModel(v string) (id, effort string) {
	v = strings.TrimSpace(v)
	if i := strings.LastIndex(v, "/"); i >= 0 {
		return strings.TrimSpace(v[:i]), strings.TrimSpace(v[i+1:])
	}
	return v, ""
}

// ModelProblem is what is wrong with the Model: value v against roster
// (nil: no [models] table): "" when it is "human" or "id/effort" with an
// effort the roster lists for id, or one of EffortWords when there is no
// roster. levels are the efforts a fix may offer.
func ModelProblem(v string, roster []Model) (problem string, levels []string) {
	v = strings.TrimSpace(v)
	if v == "human" {
		return "", nil
	}
	id, effort := SplitModel(v)
	levels = EffortWords
	known := roster == nil
	for _, m := range roster {
		if m.ID == id {
			levels, known = m.Levels, true
		}
	}
	switch {
	case id == "":
		return "Model: " + v + " names no model", levels
	case !known:
		return "model " + id + " is not in hug.toml [models]", levels
	case effort == "":
		return "Model: " + v + " names no effort (" + id + "/<effort>)", levels
	}
	for _, l := range levels {
		if l == effort {
			return "", levels
		}
	}
	if roster == nil {
		return "effort " + effort + " is not an effort word (" + strings.Join(levels, ", ") + ")", levels
	}
	return "effort " + effort + " is not one of " + id + "'s levels in hug.toml [models] (" + strings.Join(levels, ", ") + ")", levels
}

// ModelEntry is the hug.toml [models] line for m.
func ModelEntry(m Model) string {
	q := func(s string) string { return `"` + strings.ReplaceAll(s, `"`, `\"`) + `"` }
	ls := make([]string, len(m.Levels))
	for i, l := range m.Levels {
		ls[i] = q(l)
	}
	return q(m.ID) + " = [" + strings.Join(ls, ", ") + "]"
}

// WithModel is hug.toml text with m's [models] line added or replaced,
// the table created at the end when there is none. Every other byte stays.
func WithModel(text string, m Model) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}
	entry := ModelEntry(m)
	in, table, last := false, -1, -1
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			in = strings.TrimSpace(strings.Trim(t, "[]")) == "models"
			if in {
				table, last = i, i
			}
			continue
		}
		if !in || t == "" || strings.HasPrefix(t, "#") {
			continue
		}
		last = i
		if key, _, ok := strings.Cut(t, "="); ok && scalar(key) == m.ID {
			lines[i] = entry
			return strings.Join(lines, "\n") + "\n"
		}
	}
	if table < 0 {
		out := strings.Join(lines, "\n")
		if out != "" {
			out += "\n\n"
		}
		return out + "[models]\n" + entry + "\n"
	}
	lines = append(lines[:last+1], append([]string{entry}, lines[last+1:]...)...)
	return strings.Join(lines, "\n") + "\n"
}
