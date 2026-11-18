package task

import (
	"bufio"
	"os"
	"sort"
	"strconv"
	"strings"
)

// Tags (txjkrm): each board lists its tags in hug.toml, top level,
//
//	tags = ["compiler", "parser", "performance"]
//
// optionally with the merges it has decided, a variant and the listed tag
// it means:
//
//	[tag_merges]
//	perf = "performance"
//
// Tags carry no meaning to hug: display, filter, group only. The list is
// the truth; a task's tag not on it is an unlisted-tag finding. A board
// with no list is not judged. The task field is still labels; check 5's
// hug upgrade renames it.

// TagList is a board's tags and its stated merges (variant -> listed tag).
type TagList struct {
	List   []string
	Merges map[string]string
}

// Tags reads hug.toml's top-level tags = [...] (it may span lines) and
// [tag_merges]. No file or no key: an empty TagList.
func Tags(root string) (TagList, error) {
	var out TagList
	f, e := os.Open(configPath(root))
	if os.IsNotExist(e) {
		return out, nil
	}
	if e != nil {
		return out, e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	section, value, open := "", "", false
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if open {
			value += " " + line
			if strings.Contains(line, "]") {
				out.List, open = tomlStrings(value), false
			}
			continue
		}
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			section = strings.TrimSpace(strings.Trim(line, "[]"))
			continue
		}
		key, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		switch {
		case section == "" && key == "tags":
			v = strings.TrimSpace(v)
			if strings.HasPrefix(v, "[") && !strings.Contains(v, "]") {
				value, open = v, true
				continue
			}
			out.List = tomlStrings(v)
		case section == "tag_merges":
			if out.Merges == nil {
				out.Merges = map[string]string{}
			}
			out.Merges[scalar(key)] = scalar(v)
		}
	}
	return out, scan.Err()
}

// tagKey is a tag as variants compare: case, - and _ do not count.
func tagKey(s string) string {
	return strings.Map(func(r rune) rune {
		if r == '-' || r == '_' {
			return -1
		}
		return r
	}, strings.ToLower(strings.TrimSpace(s)))
}

// TagFindings flags each open task's tag that is not on the list. A
// variant of a listed tag (the same word ignoring case, - and _, or a
// merge [tag_merges] states) has one fix: replace it with the listed tag.
// Any other has two: add it to the list, or remove it from the task.
func TagFindings(list TagList, ts []Task) []Finding {
	if len(list.List) == 0 {
		return nil
	}
	listed := map[string]bool{}
	byKey := map[string]string{}
	for _, t := range list.List {
		listed[t] = true
		if _, ok := byKey[tagKey(t)]; !ok {
			byKey[tagKey(t)] = t
		}
	}
	var out []Finding
	for _, t := range ts {
		if t.State == "done" || t.State == "dropped" {
			continue
		}
		for _, l := range t.Labels {
			if listed[l] {
				continue
			}
			want, ok := list.Merges[l]
			if !ok || !listed[want] {
				want, ok = byKey[tagKey(l)]
			}
			if ok {
				fix := "hug task set " + t.ID + " --tag " + l + "=" + want
				out = append(out, Finding{Path: t.Path, Line: 1, Code: "unlisted-tag",
					Problem: "unlisted tag: " + l + " is a variant of " + want + ", which hug.toml tags lists",
					Fix:     fix, Fixes: []string{fix}})
				continue
			}
			add, drop := "hug task tags add "+l, "hug task set "+t.ID+" --tag "+l+"="
			out = append(out, Finding{Path: t.Path, Line: 1, Code: "unlisted-tag",
				Problem: "unlisted tag: " + l + " is not in hug.toml tags",
				Fix:     add, Fixes: []string{add, drop}})
		}
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].Path < out[j].Path })
	return out
}

// WithTag returns hug.toml text with name added to the top-level tags
// list, once; the list is written on one line, every other line kept.
func WithTag(text, name string) string {
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if text == "" {
		lines = nil
	}
	quote := func(ts []string) string {
		q := make([]string, len(ts))
		for i, t := range ts {
			q[i] = strconv.Quote(t)
		}
		return "tags = [" + strings.Join(q, ", ") + "]"
	}
	for i := 0; i < len(lines); i++ {
		t := strings.TrimSpace(lines[i])
		if strings.HasPrefix(t, "[") {
			break
		}
		key, v, ok := strings.Cut(t, "=")
		if !ok || strings.TrimSpace(key) != "tags" {
			continue
		}
		end := i
		v = strings.TrimSpace(v)
		for strings.HasPrefix(v, "[") && !strings.Contains(v, "]") && end+1 < len(lines) {
			end++
			v += " " + strings.TrimSpace(lines[end])
		}
		ts := tomlStrings(v)
		for _, x := range ts {
			if x == name {
				return text
			}
		}
		lines = append(lines[:i], append([]string{quote(append(ts, name))}, lines[end+1:]...)...)
		return strings.Join(lines, "\n") + "\n"
	}
	// No list yet: after the top-level keys, before the first table.
	at := len(lines)
	for i, l := range lines {
		if strings.HasPrefix(strings.TrimSpace(l), "[") {
			at = i
			break
		}
	}
	for at > 0 && strings.TrimSpace(lines[at-1]) == "" {
		at--
	}
	entry := []string{quote([]string{name})}
	if at < len(lines) && at == 0 {
		entry = append(entry, "")
	}
	lines = append(lines[:at], append(entry, lines[at:]...)...)
	return strings.Join(lines, "\n") + "\n"
}

// Tag changes one tag of ref (txjkrm, crab 14:52): NAME adds it, OLD=NEW
// replaces OLD with NEW, OLD= removes OLD. A second run changes nothing.
// The field is labels until hug upgrade renames it (txjkrm check 5).
func (s *Store) Tag(ref, spec string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	old, repl, isReplace := strings.Cut(spec, "=")
	old, repl = strings.TrimSpace(old), strings.TrimSpace(repl)
	if old == "" {
		fix := "hug task set " + d.Task.ID + " --tag " + repl
		if repl == "" {
			fix = "hug task set " + d.Task.ID + " --tag NAME"
		}
		return Result{}, usageErr(d.Task.ID, "--tag needs a tag: NAME adds it, OLD=NEW replaces OLD, OLD= removes it", fix)
	}
	if strings.ContainsAny(old+repl, " \t,[]\"") {
		return Result{}, usageErr(d.Task.ID, "a tag is one word: "+strconv.Quote(spec), "hug task set "+d.Task.ID+" --tag "+strings.Join(strings.Fields(old), "-"))
	}
	labels := append([]string(nil), d.Task.Labels...)
	has := func(x string) bool {
		for _, l := range labels {
			if l == x {
				return true
			}
		}
		return false
	}
	var next []string
	switch {
	case !isReplace:
		if has(old) {
			return s.finish(d, false)
		}
		next = append(labels, old)
	default:
		if !has(old) {
			return s.finish(d, false)
		}
		for _, l := range labels {
			switch {
			case l == old && repl != "" && !has(repl):
				next = append(next, repl)
			case l == old:
			default:
				next = append(next, l)
			}
		}
	}
	if next == nil {
		next = []string{}
	}
	if e = d.SetField("labels", next); e != nil {
		return Result{}, e
	}
	return s.finish(d, true)
}
