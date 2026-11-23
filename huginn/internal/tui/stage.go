package tui

import (
	"fmt"
	"strings"
	"sync/atomic"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/veridit/hug/huginn/internal/task"
)

// Rows show the stage, not the state (6ephqk check 5, Jørgen 10:28 and
// 10:56): the section header is the state, so a row reads pie, id, type,
// priority, stage, then the text. A stage with extra data shows
// "stage: reason" in place of the title (waiting: "CI run N · 12 min");
// a bare stage (active, ready) shows the title. depends keeps the title
// behind a small bar of dependencies done (Jørgen 11:44; the look is
// still open, so it is one function: dependsCell).

// stageCell is the stage column's width with the space after it: the
// longest stage word and its colon, "deferred:".
const stageCell = 10

// now is the clock "· 12 min" counts from; tests set it.
var now = time.Now

// rowStage is t's stage as line 1 of its row shows it: the stage word,
// and the title beside it. What a stage waits on goes on line 2 (89y0b7,
// waitrow.go), so the title always shows; text is always "" now and
// stays for the callers that pass a reason in place of the title.
// Derived from the fields and the board (task.Stage), so a board hug
// upgrade has not migrated yet shows the same stages.
//
// A wait for a date says the date in the cell, "til 11-24" (atex3d).
func (m model) rowStage(t task.Task) (word, text string) {
	word = m.stageOf(t)
	if c := untilCell(t.Fields["waiting_until"]); word == "waiting" && c != "" {
		word = c
	}
	return word, ""
}

// dependsBar draws done of total as filled and empty cells, at most 5
// cells, then the count.
func dependsBar(done, total int) string {
	if total <= 0 {
		return ""
	}
	cells := min(total, 5)
	filled := done * cells / total
	return strings.Repeat("▰", filled) + strings.Repeat("▱", cells-filled) + fmt.Sprintf(" %d/%d", done, total)
}

// since is " · 12 min" from a stamp ("2006-01-02T15:04"), or "" when it
// does not read.
func since(stamp string) string {
	at, e := time.ParseInLocation("2006-01-02T15:04", strings.TrimSpace(stamp), time.Local)
	if e != nil {
		return ""
	}
	d := now().Sub(at)
	switch {
	case d < 0:
		return ""
	case d < time.Hour:
		return fmt.Sprintf(" · %d min", int(d.Minutes()))
	case d < 48*time.Hour:
		return fmt.Sprintf(" · %d h", int(d.Hours()))
	}
	return fmt.Sprintf(" · %d d", int(d.Hours()/24))
}

// stageWord colours t's stage word: depends in the colour BLOCKED had; a
// question to you (asked, design, review) amber, as everything that waits
// on you; a wait on a thing or on someone else dim, the rest dim
// (zhvkmj, crab 02:52).
func (m model) stageWord(t task.Task, word string, width int) string {
	return m.stageStyle(t, word).Render(fit(word, width))
}

// stageStyle is the colour stageWord gives word; a closed task's state
// word is in its state colour, as its row draws it.
func (m model) stageStyle(t task.Task, word string) lipgloss.Style {
	style := m.styles.dim
	switch strings.TrimSuffix(word, ":") {
	case "done", "dropped":
		style = m.styles.status[word]
	case "depends":
		style = m.styles.status["depends"]
	case "waiting", "asked", "design", "review":
		if m.forYou(t) {
			style = m.styles.forYou
		}
	case "active":
		style = m.styles.status["doing"]
	case "ready":
		style = m.styles.status["todo"]
	}
	return style
}

// me is the TUI user as tasks name them: Options.User (an agent's
// session name, or HUG_USER while it is still read), else the person whose
// git email hug.toml [people] lists, by short name, else the email before
// the @ (89y0b7, Jørgen 15:31).
func (m model) me() string {
	if u := strings.TrimSpace(m.opts.User); u != "" {
		return u
	}
	if p, ok := task.PersonByEmail(m.people, m.opts.Email); ok {
		return p.Short
	}
	return task.EmailShort(m.opts.Email)
}

// isMe says whether name (a short name, an agent's name, or an email)
// is the TUI user. A leading @ and case do not count.
func (m model) isMe(name string) bool {
	me := m.me()
	name = strings.TrimPrefix(strings.TrimSpace(name), "@")
	if me == "" || name == "" {
		return false
	}
	if strings.EqualFold(name, me) {
		return true
	}
	// A wait written with the person's email or long name still finds them.
	if p, ok := task.PersonByName(m.people, me); ok {
		if strings.EqualFold(name, p.Name) {
			return true
		}
		for _, e := range p.Email {
			if strings.EqualFold(name, e) {
				return true
			}
		}
	}
	return false
}

// waitingFor is who t's question is for and what it asks (stage asked,
// 41hhgn), or, for a wait, "" and what it waits on. A wait written before
// 41hhgn as waiting_for, or before 89y0b7 as waiting_on "NAME: what" with
// NAME a listed person, reads as a question until hug upgrade moves it.
func (m model) waitingFor(t task.Task) (who, what string) {
	if q := strings.TrimSpace(t.Fields["asked_what"]); q != "" {
		return strings.TrimSpace(t.Fields["asked_for"]), q
	}
	what = strings.TrimSpace(t.Fields["waiting_on"])
	if w := strings.TrimSpace(t.Fields["waiting_for"]); w != "" {
		return w, what
	}
	if name, rest, ok := strings.Cut(what, ":"); ok {
		if _, listed := task.PersonByName(m.people, name); listed || m.isMe(name) {
			return strings.TrimSpace(name), strings.TrimSpace(rest)
		}
	}
	return "", what
}

// forYou says whether t waits on the TUI user (6ephqk check 8, Jørgen
// 11:42, crab 12:38): its stage names them in review_by, design_for, or
// waiting_for. Only the stage that holds counts, so an outranked design
// question waits until its dependency is done.
func (m model) forYou(t task.Task) bool {
	// A question asked with hug task ask on a todo task (yphbra, crab
	// 21:59) is for the person now, whatever stage outranks it: the asker
	// chose to ask now. A hug task design question keeps the 11-17 rule
	// below and waits for its dependency.
	if t.State == "todo" {
		for _, q := range task.OpenQuestions(t) {
			if m.isMe(q.To) {
				return true
			}
		}
	}
	switch m.taskStage(t) {
	case "review":
		return m.isMe(t.Fields["review_by"])
	case "design":
		return m.isMe(t.Fields["design_for"])
	case "waiting", "asked":
		who, _ := m.waitingFor(t)
		return m.isMe(who)
	}
	return false
}

// askedSince is when t's question to the viewer was asked, as written
// (stamps sort as text), for ASKED's oldest-first order.
func askedSince(t task.Task) string {
	for _, k := range []string{"asked_since", "design_since", "review_since", "waiting_since"} {
		if v := strings.TrimSpace(t.Fields[k]); v != "" {
			return v
		}
	}
	return "~" // no stamp: after every dated question
}

// forYouCount is how many shown tasks wait on the TUI user: counted once
// per set of shown rows, as the sections are (46ek1f).
func (m model) forYouCount() int {
	if m.panes == nil {
		return m.countForYou()
	}
	return m.panes.count(m.visibleGen, m.countForYou)
}

func (m model) countForYou() int {
	n := 0
	taskScans.Add(int64(len(m.visible)))
	for _, t := range m.visible {
		if m.forYou(t) {
			n++
		}
	}
	return n
}

// stageComputes counts task.Stage computations (46ek1f: a test checks that
// a frame computes none once the snapshot is loaded).
var stageComputes atomic.Int64

// taskScans counts tasks looked at to find a section's rows (46ek1f: a
// key and its frame must not look at more on a bigger board).
var taskScans atomic.Int64

// taskStage is task.Stage(t, m.all): the snapshot's, computed once per
// load in apply (46ek1f), for a task of the board as loaded. A task built
// on the spot (a give-back preview) is computed, and counted.
func (m model) taskStage(t task.Task) string {
	if g, ok := m.stages[t.Path]; ok && t.Path != "" {
		return g
	}
	stageComputes.Add(1)
	return task.Stage(t, m.all)
}

// computeStages fills m.stages for the board just loaded.
func (m *model) computeStages() {
	m.stages = make(map[string]string, len(m.all))
	for _, t := range m.all {
		stageComputes.Add(1)
		m.stages[t.Path] = task.Stage(t, m.all)
	}
}
