package tui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/search"
)

func TestScopeMatrixTUI(t *testing.T) {
	for _, binding := range []string{"alpha", "global", "client"} {
		t.Run(binding, func(t *testing.T) {
			f := scopetest.New(t)
			b := f.Bind(t, binding)
			cwd := f.Cwd
			if binding == "client" {
				cwd = f.ClientCwd
			}
			t.Chdir(cwd)
			m := startModel(t, f.App, 400, 40)
			f.Clean(t, binding, m.View())
			scopetest.Pass(t, "tui.overview", binding)
			m.command("search " + scopetest.Query)
			var tm tea.Model = drive(t, m, m.run()())
			f.Read(t, binding, tm.View())
			if !strings.Contains(tm.View(), b.Name) {
				t.Fatal("binding missing")
			}
			scopetest.Pass(t, "tui.search", binding)
			preview := tm.(model)
			preview.command("recall " + scopetest.Query)
			f.Read(t, binding, preview.View())
			scopetest.Pass(t, "tui.recall", binding)
			tm = keypress(t, tm, tea.KeyCtrlR)
			f.Clean(t, binding, tm.View())
			if !strings.Contains(tm.View(), "quartz deployment") {
				t.Fatal("empty related")
			}
			scopetest.Pass(t, "tui.related", binding)
			for _, ref := range f.HiddenRefs(binding) {
				u, err := f.App.DB.Resolve(ref)
				if err != nil {
					t.Fatal(err)
				}
				for _, key := range []string{"related", "x", "e", "m"} {
					forged := tm.(model)
					forged.screen = listing
					forged.focus = noInput
					forged.results = []search.Result{{Unit: u}}
					forged.cursor = 0
					if key == "related" {
						tm = keypress(t, forged, tea.KeyCtrlR)
					} else {
						tm = typeText(t, forged, key)
					}
					f.Clean(t, binding, tm.View())
					if tm.(model).focus != noInput {
						t.Fatalf("hidden %s reached owner action", key)
					}
				}
				forged := tm.(model)
				forged.command("goto " + ref)
				f.Clean(t, binding, forged.View())
				if forged.screen == detail {
					t.Fatal("hidden goto opened")
				}
				if got, err := f.App.DB.Resolve(u.SID); err != nil || !got.Active {
					t.Fatal("hidden memory changed")
				}
			}
			m = startModel(t, f.App, 400, 40)
			cmd := m.command("search " + scopetest.Query)
			tm = drive(t, m, cmd())
			selected := tm.(model).results[0].Unit
			tm = typeText(t, tm, "xy")
			f.Clean(t, binding, tm.View())
			if !strings.Contains(tm.View(), "forgot ") {
				t.Fatal(tm.View())
			}
			if u, err := f.App.DB.Resolve(selected.SID); err != nil || u.Active {
				t.Fatal("visible row not forgotten")
			}
			scopetest.Pass(t, "tui.forget", binding)
		})
	}
}
func TestScopeMatrixInvalidClientTUI(t *testing.T) {
	f := scopetest.New(t)
	f.Invalidate(t)
	m := newModel(f.App)
	m.input.SetValue(scopetest.Query)
	m.filter = scopetest.Query
	var tm tea.Model = drive(t, m, m.run()())
	if !strings.Contains(tm.View(), "invalid client name") {
		t.Fatal(tm.View())
	}
	f.Clean(t, "client", tm.View())
	scopetest.Pass(t, "tui.fail-closed", "invalid-client")
}
