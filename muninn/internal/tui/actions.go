package tui

import (
	"context"
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/tidy"
)

type action struct {
	kind         string
	unit         store.Unit
	target, text string
	places       []placeCount
	choice       int
}

var commands = []string{"overview", "recent", "place", "told", "search", "category", "date", "today", "yesterday", "dupes", "moves", "goto", "clear", "recall"}
var categories = []string{"fact", "preference", "correction", "decision"}

func (m *model) complete() {
	value := m.input.Value()
	var candidates []string
	cmd, arg, has := strings.Cut(value, " ")
	if !has {
		for _, c := range commands {
			if strings.HasPrefix(c, value) {
				candidates = append(candidates, c)
			}
		}
	} else {
		switch cmd {
		case "place":
			if strings.HasPrefix("all", arg) {
				candidates = append(candidates, "place all")
			}
			for _, p := range m.snapshot.places {
				if strings.HasPrefix(p.Name, arg) {
					candidates = append(candidates, "place "+p.Name)
				}
				if p.Scope != p.Name && strings.HasPrefix(p.Scope, arg) {
					candidates = append(candidates, "place "+p.Scope)
				}
			}
		case "category":
			for _, c := range categories {
				if strings.HasPrefix(c, arg) {
					candidates = append(candidates, "category "+c)
				}
			}
		case "date":
			values := []string{"today", "yesterday"}
			seen := map[string]bool{}
			for _, u := range m.snapshot.units {
				v := u.Updated.Local().Format("2006-01-02")
				if !seen[v] {
					seen[v] = true
					values = append(values, v)
				}
			}
			for _, c := range values {
				if strings.HasPrefix(c, arg) {
					candidates = append(candidates, "date "+c)
				}
			}
		case "goto":
			for _, u := range m.snapshot.units {
				if strings.HasPrefix(u.SID, arg) {
					candidates = append(candidates, "goto "+u.SID)
				}
			}
		}
	}
	m.completions = candidates
	m.completion = 0
}
func (m model) updateInput(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	k := msg.String()
	if m.focus == jumpInput || m.focus == filterInput {
		return m.updatePicker(msg)
	}
	switch m.focus {
	case confirmInput:
		if k == "y" {
			m.busy = true
			m.status = "Applying confirmed action…"
			return m, m.executeAction()
		}
		if k == "n" {
			m.focus = noInput
			m.status = "Cancelled"
		}
		return m, nil
	case moveInput:
		switch k {
		case "j", "down", "ctrl+n":
			m.pending.choice = min(len(m.pending.places)-1, m.pending.choice+1)
		case "k", "up", "ctrl+p":
			m.pending.choice = max(0, m.pending.choice-1)
		case "enter":
			if len(m.pending.places) > 0 {
				m.pending.target = m.pending.places[m.pending.choice].Scope
				m.focus = confirmInput
			}
		}
		return m, nil
	case correctionInput:
		if k == "ctrl+s" {
			if strings.TrimSpace(m.editor.Value()) == "" {
				m.status = "Correction cannot be empty"
				return m, nil
			}
			m.pending.text = strings.TrimSpace(m.editor.Value())
			m.editor.Blur()
			m.focus = confirmInput
			return m, nil
		}
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	case findInput:
		if k == "enter" {
			m.focus = noInput
			m.input.Blur()
			return m, nil
		}
		if k == "up" || k == "down" {
			if k == "up" {
				m.move(-1)
			} else {
				m.move(1)
			}
			return m, nil
		}
		before := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if before != m.input.Value() {
			m.filter = m.input.Value()
			m.searchSeq++
			if strings.TrimSpace(m.filter) == "" {
				m.results = m.allResults()
				m.cursor = 0
				return m, cmd
			}
			return m, tea.Batch(cmd, m.run())
		}
		return m, cmd
	case paletteInput:
		switch k {
		case "ctrl+n", "down":
			if len(m.completions) > 0 {
				m.completion = (m.completion + 1) % len(m.completions)
			}
			return m, nil
		case "ctrl+p", "up":
			if len(m.completions) > 0 {
				m.completion = (m.completion + len(m.completions) - 1) % len(m.completions)
			}
			return m, nil
		case "tab", "enter":
			if len(m.completions) > 0 {
				c := m.completions[m.completion]
				if m.input.Value() != c || k == "tab" {
					m.input.SetValue(c)
					m.input.CursorEnd()
					m.complete()
					return m, nil
				}
			}
			if k == "enter" {
				value := m.input.Value()
				m.focus = noInput
				m.input.Blur()
				return m, m.command(value)
			}
			return m, nil
		}
		before := m.input.Value()
		var cmd tea.Cmd
		m.input, cmd = m.input.Update(msg)
		if m.input.Value() != before {
			m.complete()
		}
		return m, cmd
	}
	return m, nil
}
func (m *model) command(value string) tea.Cmd {
	cmd, arg, _ := strings.Cut(strings.TrimSpace(value), " ")
	arg = strings.TrimSpace(arg)
	switch cmd {
	case "clear":
		m.clearFilters()
	case "overview":
		m.screen = overview
		m.title = "Overview"
		m.history = nil
		m.filter = ""
		m.searchSeq++
		m.cursor = 0
		m.pane = 0
	case "recent":
		m.showList("Recent", m.allResults())
	case "place":
		if arg == "all" {
			m.addChip("place", "all")
			break
		}
		var scope string
		for _, p := range m.snapshot.places {
			if p.Name == arg || p.Scope == arg {
				scope = p.Scope
				break
			}
		}
		if scope == "" {
			m.status = "Choose a visible place with :place <name>"
			break
		}
		m.addChip("place", scope)
	case "category":
		valid := false
		for _, c := range categories {
			if arg == c {
				valid = true
			}
		}
		if !valid {
			m.status = "Choose fact, preference, correction or decision"
			break
		}
		m.addChip("category", arg)
	case "date":
		if arg != "today" && arg != "yesterday" {
			if _, err := time.Parse("2006-01-02", arg); err != nil {
				m.status = "Usage: date today|yesterday|YYYY-MM-DD"
				break
			}
		}
		m.addChip("date", arg)
	case "today", "yesterday":
		m.addChip("date", cmd)
	case "search":
		if arg == "" {
			m.status = "Usage: search <query>"
			break
		}
		m.showList("Find", nil)
		m.filter = arg
		m.input.SetValue(arg)
		m.searchSeq++
		return m.run()
	case "goto":
		if arg == "" {
			m.status = "Usage: goto <visible id>"
			break
		}
		m.openUnit(arg)
	case "told":
		var rows []search.Result
		for _, t := range m.snapshot.told {
			rows = append(rows, search.Result{Unit: t.unit})
		}
		m.showList("Told agents lately", rows)
	case "dupes", "moves":
		m.startReview(cmd)
	case "recall":
		if arg == "" {
			m.status = "Usage: recall <message> (read-only preview)"
			break
		}
		m.push()
		m.screen = preview
		m.title = "Recall preview"
		b := m.binding.Refresh(m.a)
		if b.Err != nil {
			m.status = b.Err.Error()
			break
		}
		r := m.recallPreview(arg)().(resultsMsg)
		if r.err != nil {
			m.status = r.err.Error()
			break
		}
		m.block = r.block
		m.sizeViewport()
		m.viewport.SetContent(wrapText(r.block, m.width-4))
		m.viewport.GotoTop()
	default:
		m.status = "Unknown command. Press : for completion"
	}
	return nil
}
func (m model) beginAction(kind string) (tea.Model, tea.Cmd) {
	u, ok := m.selected()
	if !ok {
		m.status = "Open or select a memory first"
		return m, nil
	}
	b := m.binding.Refresh(m.a)
	u, err := b.Resolve(m.a, u.SID)
	if err == nil {
		_, err = b.WriteScope(u.Scope)
	}
	if err != nil {
		m.status = err.Error()
		return m, nil
	}
	if !u.Active {
		m.status = "Memory is already retired"
		return m, nil
	}
	m.binding = b
	m.pending = action{kind: kind, unit: u}
	switch kind {
	case "e":
		m.focus = correctionInput
		m.editor.SetValue("")
		return m, m.editor.Focus()
	case "m":
		for _, p := range m.snapshot.places {
			if p.Scope != u.Scope {
				if _, err := b.WriteScope(p.Scope); err == nil {
					m.pending.places = append(m.pending.places, p)
				}
			}
		}
		if len(m.pending.places) == 0 {
			m.status = "No other writable place is available"
			return m, nil
		}
		m.focus = moveInput
	case "x":
		m.focus = confirmInput
	}
	return m, nil
}
func (m model) executeAction() tea.Cmd {
	p, a, b := m.pending, m.a, m.binding
	return func() tea.Msg {
		// Re-resolve and recheck the original write policy at confirmation time.
		u, err := b.Resolve(a, p.unit.SID)
		if err != nil {
			return actionMsg{err: err}
		}
		if _, err = b.Refresh(a).WriteScope(u.Scope); err != nil {
			return actionMsg{err: err}
		}
		if tidy.Fingerprint(u) != tidy.Fingerprint(p.unit) {
			return actionMsg{err: fmt.Errorf("memory changed while confirming; reopen it")}
		}
		switch p.kind {
		case "x":
			r, err := b.Forget(context.Background(), a, u.SID)
			return actionMsg{note: r.Message, err: err}
		case "m":
			id, err := b.Rescope(context.Background(), a, u.SID, p.target, false)
			return actionMsg{note: "Moved " + id + " to " + p.target, err: err}
		case "e":
			next, backup, err := b.Correct(context.Background(), a, u.SID, p.text)
			return actionMsg{note: "Correction stored · backup " + backup, unit: next, err: err}
		}
		return actionMsg{err: fmt.Errorf("unknown action")}
	}
}
func (m *model) startReview(kind string) {
	path := m.snapshot.movePath
	if kind == "dupes" {
		path = m.snapshot.dupePath
	}
	if path == "" {
		m.status = "No plan yet. Run muninn tidy " + map[string]string{"dupes": "dupes", "moves": "scan"}[kind]
		return
	}
	m.push()
	m.screen = review
	m.title = "Needs a look · " + kind
	m.reviewKind = kind
	m.reviewIndex = 0
	m.reviewOffset = 0
	m.clampReview()
}
func (m model) reviewCount() int {
	if m.reviewKind == "dupes" {
		return len(m.snapshot.dupes.Duplicates)
	}
	return len(m.snapshot.moves.Entries)
}
func (m model) reviewKey() string {
	if m.reviewIndex >= m.reviewCount() {
		return ""
	}
	if m.reviewKind == "dupes" {
		return "dupes:" + m.snapshot.dupes.Duplicates[m.reviewIndex].Keeper.SID
	}
	return "moves:" + m.snapshot.moves.Entries[m.reviewIndex].SID
}
func (m *model) clampReview() {
	for m.reviewIndex < m.reviewCount() && m.reviewDone[m.reviewKey()] {
		m.reviewIndex++
	}
	if m.reviewIndex >= m.reviewCount() {
		m.status = "Review complete. Esc returns without applying skipped entries"
	}
}
func (m *model) applyReview() tea.Cmd {
	if m.reviewIndex >= m.reviewCount() {
		return nil
	}
	a, b := m.a, m.binding
	var p tidy.Plan
	if m.reviewKind == "dupes" {
		p = m.snapshot.dupes
		c := p.Duplicates[m.reviewIndex]
		c.Decision = "merge"
		p.Duplicates = []tidy.DuplicateCluster{c}
		p.DuplicatePairs = nil
	} else {
		p = m.snapshot.moves
		e := p.Entries[m.reviewIndex]
		e.Decision = "move"
		p.Entries = []tidy.Entry{e}
		p.General = nil
	}
	m.busy = true
	m.status = "Applying reviewed entry with backup…"
	return func() tea.Msg {
		b = b.Refresh(a)
		if b.Err != nil {
			return actionMsg{err: b.Err}
		}
		refs := []string{}
		if p.Kind == "dupes" {
			for _, c := range p.Duplicates {
				refs = append(refs, c.Keeper.SID)
				for _, r := range c.Retire {
					refs = append(refs, r.SID)
				}
			}
		} else {
			for _, e := range p.Entries {
				refs = append(refs, e.SID)
				if _, err := b.WriteScope(e.Target); err != nil {
					return actionMsg{err: err}
				}
			}
		}
		for _, ref := range refs {
			u, err := b.Resolve(a, ref)
			if err != nil {
				return actionMsg{err: err}
			}
			if _, err = b.WriteScope(u.Scope); err != nil {
				return actionMsg{err: err}
			}
		}
		r, err := a.ApplyTidy(context.Background(), p, "", false)
		return actionMsg{note: fmt.Sprintf("Applied · %d moves · %d retirements · backup %s", len(r.Moves), len(r.Retirements), r.Backup), err: err, next: err == nil}
	}
}
