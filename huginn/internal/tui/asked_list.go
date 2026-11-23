package tui

import (
	"sort"
	"strings"

	"github.com/veridit/hug/huginn/internal/task"
)

// Question is one open question a board asks someone, as hug task ls
// --asked lists it (fthher): the task, the question's number on it (0 for
// a stage question without one), the check it is about (0 for none), its
// text verbatim, who asked and when (the stamp as written).
type Question struct {
	ID, Title     string
	N, Check      int
	Text, By, At  string
	Stage, Status string
}

// Asked is every open question tasks ask the user, oldest first, by the
// rule ASKED and '? N for you' use (forYou, askedAll): the CLI and the TUI
// count the same questions on the same board. user is a name tasks write
// (--as, HUG_USER, an agent's session); with none, email finds the person
// in people as the TUI does, else the email before the @.
func Asked(tasks []task.Task, people []task.Person, user, email string) []Question {
	m := model{opts: Options{User: user, Email: email}, people: people, all: tasks}
	m.computeStages()
	var out []Question
	for _, t := range tasks {
		for _, a := range m.askedAll(t) {
			out = append(out, Question{ID: t.ID, Title: t.Title, N: a.n, Check: a.check, Text: a.what, By: a.by, At: a.at, Stage: a.stage, Status: t.State})
		}
	}
	// Oldest first, as ASKED (askedSince): a question with no stamp after
	// every dated one. Stamps compare as times, so a UTC one (Z) and a
	// local one order right.
	sort.SliceStable(out, func(i, j int) bool { return askedBefore(out[i].At, out[j].At) })
	return out
}

// askedBefore says whether stamp a is older than b; a stamp that does not
// read sorts after every one that does, as askedSince's "~".
func askedBefore(a, b string) bool {
	ta, oka := stampAt(a)
	tb, okb := stampAt(b)
	switch {
	case oka && okb:
		return ta.Before(tb)
	case oka != okb:
		return oka
	}
	return sortStamp(a) < sortStamp(b)
}

func sortStamp(at string) string {
	if at = strings.TrimSpace(at); at != "" {
		return at
	}
	return "~"
}

// Me is the name Asked finds the user by: user, else the person whose
// email people lists, else the email before the @ (the TUI's me).
func Me(people []task.Person, user, email string) string {
	return model{opts: Options{User: user, Email: email}, people: people}.me()
}

// SinceText is "since 14:42" for a stamp of today, "since 15 Nov" for an
// older one, as the TUI's ASKED rows say it; "" when it does not read.
func SinceText(stamp string) string { return sinceText(stamp) }
