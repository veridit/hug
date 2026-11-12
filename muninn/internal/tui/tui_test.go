package tui

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/cursor"
	tea "github.com/charmbracelet/bubbletea"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/index"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/recall"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/testutil"
)

func testApp(t *testing.T) *app.App {
	t.Helper()
	// Standalone owner-workflow fixtures use an isolated global instance, not
	// whichever client/instance JCODE_HOME the outer test process inherited.
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home) // os.UserHomeDir on Windows
	t.Setenv("JCODE_HOME", filepath.Join(home, ".jcode"))
	t.Setenv("MUNINN_HOME", t.TempDir())
	db, err := store.Open(config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { db.Close() })
	for _, c := range []string{"Bake banana cake at 175 degrees.", "Deploys go through staging first.", "Use ripgrep for searches."} {
		if _, _, err := ingest.Remember(app.NativePath(), c, "fact", "global", nil); err != nil {
			t.Fatal(err)
		}
	}
	fe := &testutil.FakeEmbedder{}
	cfg := config.Defaults()
	cfg.JcodeMemoryDir = ""
	a := &app.App{Cfg: cfg, DB: db, Embedder: fe, Indexer: &index.Indexer{DB: db, Embedder: fe}}
	a.Searcher = &search.Searcher{DB: db, Embedder: fe}
	a.Recall = &recall.Engine{DB: db, Searcher: a.Searcher, Options: a.RecallOptions()}
	if _, err := a.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	return a
}

// drive feeds a message and synchronously resolves the commands it returns
// (search runs in a tea.Cmd), the way the bubbletea runtime would.
func drive(t *testing.T, m tea.Model, msg tea.Msg) tea.Model {
	t.Helper()
	queue := []tea.Msg{msg}
	for i := 0; len(queue) > 0 && i < 50; i++ {
		next := queue[0]
		queue = queue[1:]
		var cmd tea.Cmd
		m, cmd = m.Update(next)
		queue = append(queue, resolve(cmd)...)
	}
	return m
}

func resolve(cmd tea.Cmd) []tea.Msg {
	if cmd == nil {
		return nil
	}
	switch msg := cmd().(type) {
	case tea.BatchMsg:
		var out []tea.Msg
		for _, c := range msg {
			out = append(out, resolve(c)...)
		}
		return out
	case resultsMsg, snapshot, actionMsg:
		return []tea.Msg{msg}
	default:
		return nil // blink ticks etc.
	}
}

func typeText(t *testing.T, m tea.Model, s string) tea.Model {
	for _, r := range s {
		m = drive(t, m, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	return m
}

func keypress(t *testing.T, m tea.Model, key tea.KeyType) tea.Model {
	return drive(t, m, tea.KeyMsg{Type: key})
}
func startModel(t *testing.T, a *app.App, w, h int) model {
	m := newModel(a)
	m.input.Cursor.SetMode(cursor.CursorStatic)
	m.editor.Cursor.SetMode(cursor.CursorStatic)
	var tm tea.Model = drive(t, m, m.Init()())
	tm = drive(t, tm, tea.WindowSizeMsg{Width: w, Height: h})
	return tm.(model)
}
func TestTUIOverviewFindDetailExplainAndRetire(t *testing.T) {
	a := testApp(t)
	var tm tea.Model = startModel(t, a, 140, 40)
	for _, label := range []string{"Places", "Recent", "Told agents lately", "Needs a look", "Bake banana cake", "3 memories"} {
		if !strings.Contains(tm.View(), label) {
			t.Fatalf("missing %q on first screen:\n%s", label, tm.View())
		}
	}
	tm = typeText(t, tm, "/")
	tm = typeText(t, tm, "banana")
	v := tm.View()
	if !strings.Contains(v, "Bake banana cake") || !strings.Contains(v, "results in") {
		t.Fatalf("search:\n%s", v)
	}
	before := tm.(model).input.Value()
	tm = typeText(t, tm, "?")
	if !tm.(model).help || tm.(model).input.Value() != before {
		t.Fatal("? went into find input")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = keypress(t, tm, tea.KeyEnter)
	kept := tm.(model)
	if kept.focus != noInput || kept.filter != "banana" {
		t.Fatal("Enter did not keep filter")
	}
	tm = keypress(t, tm, tea.KeyEnter)
	if tm.(model).screen != detail || !strings.Contains(tm.View(), "Origin/session:") {
		t.Fatalf("Enter did not open detail:\n%s", tm.View())
	}
	if tm.(model).results[0].Total != kept.results[0].Total {
		t.Fatal("Enter reran search")
	}
	tm = keypress(t, tm, tea.KeyCtrlE)
	if !strings.Contains(tm.View(), "keyword=") {
		t.Fatalf("explain missing:\n%s", tm.View())
	}
	tm = keypress(t, tm, tea.KeyEsc)
	if tm.(model).filter != "banana" {
		t.Fatal("back lost filter")
	}
	tm = typeText(t, tm, "x")
	if tm.(model).focus != confirmInput {
		t.Fatal("retire was not confirmed")
	}
	selected := tm.(model).pending.unit
	if u, _ := a.DB.Resolve(selected.SID); !u.Active {
		t.Fatal("retired before confirmation")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = typeText(t, tm, "xy")
	if !strings.Contains(tm.View(), "forgot ") {
		t.Fatalf("retire receipt:\n%s", tm.View())
	}
	if u, _ := a.DB.Resolve(selected.SID); u.Active {
		t.Fatal("confirmed retirement did not apply")
	}
	if strings.Contains(tm.View(), "Bake banana cake") {
		t.Fatal("retired memory still in overview")
	}
	_, cmd := tm.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if cmd != nil && cmd() == tea.Quit() {
		t.Fatal("Esc quit")
	}
	tm = typeText(t, tm, "q")
	_ = tm
}
