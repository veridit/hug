package tui

import (
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// / search is one syntax for free text and field:value (0sw0dd check 3,
// principle "search is one syntax"): 'state:done memory' is the done tasks
// with the word memory. Field terms are cut off the query; the words left
// search as before (the index's ranking, else substring). A term whose
// field is not one of these stays a word, so 'http://x' or 'a:b' still
// search as typed.

// searchFields are the fields / understands, and the older names hug used
// for them: status is state since 6ephqk. tag is label's new name (txjkrm);
// both find a task's labels until the CLI field is renamed too.
var searchFields = map[string]string{
	"state": "state", "status": "state",
	"type": "type", "priority": "priority", "group": "group",
	"label": "label", "tag": "label", "assignee": "assignee", "epic": "epic",
}

// fieldTerm is one field:value of a search, both lower case.
type fieldTerm struct{ field, value string }

// parseSearch splits query into its field terms and the words left, in
// their order.
func parseSearch(query string) (terms []fieldTerm, words string) {
	var rest []string
	for _, w := range strings.Fields(query) {
		name, value, ok := strings.Cut(w, ":")
		field, known := searchFields[strings.ToLower(name)]
		if base, _ := task.Not(value); !ok || !known || base == "" {
			rest = append(rest, w)
			continue
		}
		terms = append(terms, fieldTerm{field, strings.ToLower(value)})
	}
	return terms, strings.Join(rest, " ")
}

// fieldsMatch says whether t has every field term. state matches the
// task's own state or its place on the board, so state:todo, state:done
// and state:depends all find what the board shows under that name.
func (m model) fieldsMatch(t task.Task, terms []fieldTerm) bool {
	for _, f := range terms {
		// state:!done is every task but the done ones (7zg9mf): one mark,
		// !, as in the f picker and ':'.
		value, not := task.Not(f.value)
		f := fieldTerm{f.field, value}
		ok := false
		switch f.field {
		case "state":
			ok = strings.EqualFold(t.State, f.value) || m.place(t) == f.value
		case "type":
			ok = strings.EqualFold(t.Type, f.value) || t.Type == "" && f.value == "(none)"
		case "priority":
			ok = strings.EqualFold(t.Priority, f.value) || t.Priority == "" && f.value == "(none)"
		case "group":
			ok = strings.EqualFold(t.Group, f.value) || t.Group == "" && f.value == "(root)"
		case "label":
			ok = containsFold(t.Labels, f.value)
		case "assignee":
			ok = containsFold(t.Assignee, f.value) || len(t.Assignee) == 0 && f.value == "(unassigned)"
		case "epic":
			ok = task.InEpic(t, m.all, f.value) || epicPrefix(t, m.all, f.value)
		}
		if ok == not {
			return false
		}
	}
	return true
}

func containsFold(list []string, v string) bool {
	for _, s := range list {
		if strings.EqualFold(s, v) {
			return true
		}
	}
	return false
}

// epicPrefix lets epic: take the id in any case: InEpic compares exactly.
func epicPrefix(t task.Task, all []task.Task, value string) bool {
	for _, e := range all {
		if strings.EqualFold(e.ID, value) && e.ID != value {
			return task.InEpic(t, all, e.ID)
		}
	}
	return false
}
