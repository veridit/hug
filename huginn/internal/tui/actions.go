package tui

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/huginn/internal/editor"
	"github.com/veridit/hug/huginn/internal/task"
)

var actionNames = map[string]string{"t": "take", "d": "done", "x": "drop", " ": "toggle acceptance", "e": "edit", "n": "new"}

// dropPrompt asks why a task is dropped: the reason is required (6ephqk).
const dropPrompt = "drop: why never?"

// waitPrompt asks what a doing task waits on (e3sevr: w on the task).
const waitPrompt = "wait on what?"

// waitKey opens the wait prompt on a doing task; another state says why
// not, with the key that gets it there.
func (m *model) waitKey() tea.Cmd {
	t, ok := m.selected()
	if !ok {
		return nil
	}
	if t.State != "doing" {
		m.message = t.ID + " is " + t.State + ": only a doing task waits; t takes it first"
		return nil
	}
	m.confirmPath = t.Path
	return m.openInput(waitPrompt, "")
}

// wait parks the task at confirmPath on what the reader typed.
func (m *model) wait(on string) {
	t, ok := m.taskAt(m.confirmPath)
	m.confirmPath = ""
	if !ok {
		m.message = "task disappeared; refresh and try again"
		return
	}
	defer m.hold()()
	r, err := m.store.Wait(t.ID, on, m.me())
	m.finish(r, err, "wait")
}

// startAction begins the action for key k on the selected task: x first
// asks the reason, d on a todo task is refused (take before done, Jørgen
// 11:37), the rest go to requestAction's confirm.
func (m *model) startAction(k string) tea.Cmd {
	t, ok := m.selected()
	switch {
	case ok && k == "d" && t.State == "todo":
		m.message = t.ID + " is todo: t takes it, then d marks it done"
		return nil
	case ok && k == "x" && t.State != "dropped":
		return m.openInput(dropPrompt, "")
	case ok && k == "t" && t.State == "doing" && m.me() != "" && contains(t.Assignee, m.me()):
		// t on your own task gives it back to todo, keeping its stage
		// (6ephqk check 6): confirmed as "Give back".
		m.requestAction("t", "give back")
		return nil
	}
	m.requestAction(k, "")
	return nil
}

func (m *model) requestAction(k, value string) {
	t, ok := m.selected()
	if k != "n" && !ok {
		return
	}
	m.confirm = k
	m.bulk = nil // one task, not the marked (rxr4z8)
	m.confirmPath = t.Path
	m.pendingValue = value
	if k == "e" {
		// Name the editor in the prompt; resolved once here, not on every
		// frame (2026-11-16, user: the prompt did not say which editor).
		m.pendingValue = strings.Join(editor.Command(), " ")
	}
}
func (m *model) executeAction() tea.Cmd {
	k, value, path := m.confirm, m.pendingValue, m.confirmPath
	m.confirm = ""
	m.pendingValue = ""
	m.editNotice = ""
	// Resolve again by path so confirmation never applies to another selected task.
	var t task.Task
	if k != "n" {
		found := false
		for _, v := range m.all {
			if v.Path == path {
				t = v
				found = true
				break
			}
		}
		if !found {
			m.message = "task disappeared; refresh and try again"
			return nil
		}
	}
	var r task.Result
	var err error
	action := actionNames[k]
	// The write and its commit (finish) hold the board turn (95ahtk). Not the
	// editor, which can stay open for minutes (its commit holds it), and not
	// a take through opts.Take, which takes the turn itself.
	if k != "e" && !(k == "t" && value != "give back" && m.opts.Take != nil) {
		defer m.hold()()
	}
	switch k {
	case "n":
		r, err = m.store.New(task.NewOptions{Title: value, Goal: value, Acceptance: []string{"Verify " + value}})
	case "t":
		if value == "give back" {
			r, err = m.store.Set(t.ID, "state", "todo")
			action = "give back"
			break
		}
		if m.opts.Take != nil {
			r, err = m.opts.Take(t.ID)
		} else {
			r, err = m.store.Take(t.ID, m.me())
		}
	case "d":
		r, err = m.store.Done(t.ID)
	case "x":
		r, err = m.store.Drop(t.ID, value)
	case " ":
		if len(t.Acceptance) == 0 {
			m.message = "no acceptance items"
			return nil
		}
		n := min(m.acceptance, len(t.Acceptance)-1) + 1
		if !t.Acceptance[n-1].Checked {
			r, err = m.store.Check(t.ID, n)
			action = "check"
			// The person asked ticking the asked check answered it
			// (qsqhpe): it goes to the coordinator as a typed answer.
			if err == nil && r.Answered {
				did, _ := m.store.AfterTick(t.ID, n, m.me(), true)
				m.finish(r, err, action)
				extra := ""
				if did != "" {
					extra = "after the release: " + did
				}
				m.message = m.movedOr("answered "+r.Task.ID+" by ticking check "+strconv.Itoa(n), extra)
				m.deliver(r, err, "answer", m.me())
				return nil
			}
			// The owner's tick of the check a release handed on (yf39ya
			// check 8) asks the next person check or closes the task.
			if err == nil && r.Changed {
				if did, _ := m.store.AfterTick(t.ID, n, m.me(), false); did != "" {
					m.finish(r, err, action)
					m.message = m.movedOr("checked "+r.Task.ID+" "+strconv.Itoa(n), "after the release: "+did)
					return nil
				}
			}
		} else {
			d, e := m.store.Get(t.ID)
			err = e
			if err == nil {
				lines := strings.Split(string(d.Bytes()), "\n")
				line := d.Task.Acceptance[n-1].Line - 1
				if line >= 0 && line < len(lines) {
					lines[line] = strings.Replace(strings.Replace(lines[line], "[x]", "[ ]", 1), "[X]", "[ ]", 1)
					err = task.AtomicWrite(filepath.Join(m.store.Root, d.Task.Path), []byte(strings.Join(lines, "\n")))
				}
			}
			r = task.Result{Task: t, Changed: err == nil}
			action = "uncheck"
		}
	case "e":
		return m.runEditor(t)
	case "-":
		n := min(m.acceptance, len(t.Acceptance)-1) + 1
		r, err = m.store.Strike(t.ID, n, "", m.me(), true)
		action = "restore"
	}
	m.finish(r, err, action)
	return nil
}

// runEditor opens the task file in the user's editor; editorMsg comes back
// with the path, so the result is checked for that task (rx74bp).
func (m *model) runEditor(t task.Task) tea.Cmd {
	filename, e := m.editorPath(t)
	if e != nil {
		m.message = e.Error()
		return nil
	}
	args := editor.Command()
	cmd := exec.Command(args[0], append(args[1:], filename)...)
	path := t.Path
	// Remember what was wrong before, so the notice names what this edit
	// introduced, not an old finding.
	m.editBefore = map[string]bool{}
	for _, f := range m.editFindings(path) {
		m.editBefore[f.Problem] = true
	}
	m.editing = true // a change on disk meanwhile waits for the editor to close
	return tea.ExecProcess(cmd, func(e error) tea.Msg { return editorMsg{path: path, err: e} })
}

// editFindings are the lint findings of the task file at path (an open
// task: a closed one is not judged, t01kqm).
func (m model) editFindings(path string) []task.Finding {
	var out []task.Finding
	for _, f := range m.findings {
		if f.Path == path {
			out = append(out, f)
		}
	}
	return out
}

// noticeTask is the task the edit notice is about, with the findings the
// edit introduced, while it still exists and still has one: only then does
// e reopen the editor on it.
func (m model) noticeTask() (task.Task, []task.Finding, bool) {
	if m.editNotice == "" {
		return task.Task{}, nil, false
	}
	var fs []task.Finding
	for _, f := range m.editFindings(m.editNotice) {
		if !m.editBefore[f.Problem] {
			fs = append(fs, f)
		}
	}
	if len(fs) == 0 {
		return task.Task{}, nil, false
	}
	for _, t := range m.all {
		if t.Path == m.editNotice {
			return t, fs, true
		}
	}
	return task.Task{}, nil, false
}
func (m *model) finish(r task.Result, err error, action string) {
	// Its place before the write, while the board shown is the old one
	// (24403z check 4).
	before := m.placeBefore(r.Task.ID)
	m.movedTo = ""
	if err != nil {
		path := ""
		if t, ok := m.selected(); ok {
			path = t.Path
		}
		m.refresh(path)
		m.message = err.Error()
		return
	}
	m.message = pastTense(action) + " " + r.Task.ID
	if len(r.Warnings) > 0 {
		m.message += "\n" + strings.Join(r.Warnings, "\n")
	}
	if r.Changed && m.opts.Commit != nil && !(action == "take" && m.opts.Take != nil) {
		if e := m.opts.Commit("hug task " + action + " " + r.Task.ID); e != nil {
			m.message = e.Error()
		}
	}
	if m.detail {
		m.detailPath = r.Task.Path
	}
	m.refresh(r.Task.Path)
	if m.movedTo = m.movedNote(r.Task.ID, before); m.movedTo != "" {
		// Where it went, in place of what was done (crab 20:07Z): the
		// warnings and a failed commit stay on the lines after it.
		_, rest, _ := strings.Cut(m.message, "\n")
		if rest != "" {
			rest = "\n" + rest
		}
		if strings.HasPrefix(m.message, pastTense(action)+" "+r.Task.ID) {
			m.message = m.movedTo + rest
		} else {
			m.movedTo = "" // a failed commit says so instead
		}
	}
}

// messagePlace names the screen: the view, the open task and the file
// shown in it. A message belongs to the screen that made it (z82qsy for
// action results, tz9z14 for every message).
func (m model) messagePlace() string {
	p := m.view
	if m.detail {
		p += "\x00" + m.detailPath
		if m.glance {
			p += "\x00" + m.glanceHead
		}
	}
	return p
}

// clearStaleMessage clears the message once the reader has moved on to
// another task, file or view (Jørgen 11:37Z: "answer vzghj8" stayed under
// a file of another task; 15:19: "reloaded hug v0.7.63" stayed under a
// task's detail, "noise because it's from the main screen"). A message
// set in a step belongs to the screen the step ends on; one that was
// already showing goes when the screen changes.
func (m *model) clearStaleMessage(before string) {
	place := m.messagePlace()
	switch {
	case m.message == "":
		m.messageAt = ""
	case m.message != before:
		m.messageAt = place
	case m.messageAt != place:
		m.message, m.messageAt = "", ""
	}
}

// Malformed files stay repairable, but neither absolute paths nor symlinks may escape the store.
func (m model) editorPath(t task.Task) (string, error) {
	if t.Path == "" || filepath.IsAbs(t.Path) {
		return "", fmt.Errorf("invalid task path")
	}
	root, err := filepath.EvalSymlinks(m.store.Root)
	if err != nil {
		return "", err
	}
	root, err = filepath.Abs(root)
	if err != nil {
		return "", err
	}
	path := filepath.Join(root, t.Path)
	rel, err := filepath.Rel(root, path)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("task path escapes store")
	}
	actual, err := filepath.EvalSymlinks(path)
	if err != nil {
		return "", err
	}
	if actual != path {
		return "", fmt.Errorf("symlinked task paths cannot be edited")
	}
	info, err := os.Stat(path)
	if err != nil {
		return "", err
	}
	if !info.Mode().IsRegular() {
		return "", fmt.Errorf("task is not a regular file")
	}
	return path, nil
}

// strikePrompt is the input asking why a check no longer applies.
const strikePrompt = "strike out, reason:"

// strikeKey is - on the open task: with a check marked >, an open one asks
// for the reason (Enter strikes, Esc keeps it), a struck one asks to be
// restored; a ticked one was done and stays (sfa8w4, Jørgen 2026-11-16:
// striking out must be his in the TUI too).
func (m *model) strikeKey() tea.Cmd {
	t, ok := m.selected()
	switch {
	case !ok || len(t.Acceptance) == 0:
		m.message = "no acceptance checks to strike out"
		return nil
	case m.detailPane != focusAcceptance:
		m.message = "- strikes out the check marked >: l focuses the checks"
		return nil
	}
	n := min(m.acceptance, len(t.Acceptance)-1)
	a := t.Acceptance[n]
	switch {
	case a.Struck:
		m.requestAction("-", "")
		return nil
	case a.Checked:
		m.message = fmt.Sprintf("check %d is ticked: it was done, so it is not struck out (Space unticks it)", n+1)
		return nil
	}
	m.confirmPath = t.Path
	return m.openInput(strikePrompt, "")
}

// strike strikes the check marked > out with reason, as the TUI user.
func (m *model) strike(reason string) {
	var t task.Task
	found := false
	for _, v := range m.all {
		if v.Path == m.confirmPath {
			t, found = v, true
			break
		}
	}
	m.confirmPath = ""
	if !found || len(t.Acceptance) == 0 {
		m.message = "task disappeared; refresh and try again"
		return
	}
	n := min(m.acceptance, len(t.Acceptance)-1) + 1
	who := m.me()
	if who == "" {
		who = "human"
	}
	defer m.hold()()
	r, err := m.store.Strike(t.ID, n, reason, who, false)
	m.finish(r, err, "strike")
}

// answerPrompt is the input under a question the task asks you (3jtfw9).
const answerPrompt = "answer:"

// answerKey opens the answer input when the task's stage waits on the TUI
// user; otherwise it says why not.
func (m *model) answerKey() tea.Cmd {
	t, ok := m.selected()
	if !ok {
		return nil
	}
	if !m.forYou(t) {
		g := m.taskStage(t)
		m.message = t.ID + " asks you nothing (stage " + g + "): a answers a design question, a wait or a review that names you"
		return nil
	}
	m.confirmPath = t.Path
	// Which question a answers (q2adr8): the one marked > (its check or
	// its block), else the oldest asked of you.
	q, _ := m.askedOf(t)
	switch {
	case m.detailPane == focusAcceptance:
		if c, ok := m.checkQuestion(t, m.acceptance+1); ok {
			q = c
		}
	case m.detailPane == focusQuestion:
		if bs := m.blockQuestions(t); m.question >= 0 && m.question < len(bs) {
			q = bs[m.question]
		}
	}
	m.answerN = q.n
	// The box opens under the question it answers, which may be far
	// down the page (qcq3tm check 4, p87ngv: a opened it out of view and
	// seemed to do nothing): the focus goes to that question and the page
	// scrolls so the box shows.
	m.focusOn(t, q)
	cmd := m.openAnswer()
	m.revealAnswerBox()
	return cmd
}

// answer writes text as the TUI user's answer to the task opened with a.
func (m *model) answer(text string) {
	var t task.Task
	found := false
	for _, v := range m.all {
		if v.Path == m.confirmPath {
			t, found = v, true
			break
		}
	}
	m.confirmPath = ""
	if !found {
		m.message = "task disappeared; refresh and try again"
		return
	}
	defer m.hold()()
	n := m.answerN
	m.answerN = 0
	r, err := m.store.AnswerQuestion(t.ID, n, text, m.me())
	m.finish(r, err, "answer")
	m.deliver(r, err, "answer", m.me())
	// Questions left (q2adr8): the > goes to the oldest still open, so
	// the next a answers it.
	if tk, ok := m.selected(); ok && err == nil && len(m.askedAll(tk)) > 0 {
		m.startFocus()
		m.syncDetail()
	}
}

// deliver hands what a write just stored to the coordinator (gmzd87) and
// says the receipt in the message line.
func (m *model) deliver(r task.Result, err error, kind, by string) {
	if err != nil || !r.Changed || m.opts.Deliver == nil {
		return
	}
	receipt, e := m.opts.Deliver(r.Task.ID, kind, by)
	switch {
	case e != nil:
		m.message = e.Error()
	case receipt != "" && m.movedTo != "" && strings.HasPrefix(m.message, m.movedTo):
		// The receipt after the moved note (crab 20:07Z).
		m.message = m.movedTo + " · " + receipt + strings.TrimPrefix(m.message, m.movedTo)
	case receipt != "":
		m.message += ": " + receipt
	}
	m.refresh(r.Task.Path)
}

// hold takes the board's write turn when the TUI was given one (95ahtk):
// a write and its commit run inside it, as every CLI write does.
func (m *model) hold() func() {
	if m.opts.Turn == nil {
		return func() {}
	}
	return m.opts.Turn()
}

// pastTense is an action as its result reads (z82qsy): what was done,
// not the command that did it.
func pastTense(action string) string {
	if p, ok := map[string]string{
		"take": "took", "give back": "gave back", "done": "marked done", "drop": "dropped",
		"wait": "parked", "answer": "answered", "request": "requested", "strike": "struck",
		"restore": "restored", "check": "checked", "uncheck": "unchecked", "new": "created",
		"move": "moved",
	}[action]; ok {
		return p
	}
	return action
}
