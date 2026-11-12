package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/tidy"
)

func TestScopeMatrixOverviewHistoryAndPlans(t *testing.T) {
	f := scopetest.New(t)
	b := f.Bind(t, "alpha")
	t.Chdir(f.Cwd)
	visible, hidden := f.Units["Alpha"], f.Units["Beta"]
	if _, err := f.App.DB.Exec(`DELETE FROM injections`); err != nil {
		t.Fatal(err)
	}
	for _, u := range []string{visible.SID, hidden.SID} {
		if _, err := f.App.DB.Exec(`INSERT INTO injections(session,unit_id,content_hash,harness,at_ns) SELECT 'jcode:session_crab_12345_abcdef',id,content_hash,'jcode',? FROM units WHERE sid=?`, time.Now().UnixNano(), u); err != nil {
			t.Fatal(err)
		}
	}
	save := func(path string, p tidy.Plan) {
		t.Helper()
		raw, err := json.Marshal(p)
		if err != nil {
			t.Fatal(err)
		}
		if err = os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		if err = os.WriteFile(path, raw, 0600); err != nil {
			t.Fatal(err)
		}
	}
	// Arbitrary plan filenames at the data root are accepted, not just tidy/*.
	moves := tidy.Plan{Version: 1, Created: time.Now(), Database: f.App.DB.Path, Entries: []tidy.Entry{
		{SID: visible.SID, Source: visible.Scope, Target: "global", Text: "stale plan text must not be displayed"},
		{SID: hidden.SID, Source: hidden.Scope, Target: "global", Text: hidden.Body},
	}, General: []tidy.Entry{{Text: hidden.Body}}}
	save(filepath.Join(config.DataDir(), "review-me.json"), moves)
	dupes := tidy.Plan{Version: 1, Kind: "dupes", Created: time.Now(), Database: f.App.DB.Path, Duplicates: []tidy.DuplicateCluster{{Keeper: tidy.DuplicateMember{SID: visible.SID}, Retire: []tidy.DuplicateMember{{SID: hidden.SID, Text: hidden.Body}}}}}
	save(filepath.Join(config.DataDir(), "tidy", "duplicates.json"), dupes)
	s := loadSnapshot(f.App, b)
	if s.err != nil {
		t.Fatal(s.err)
	}
	if len(s.told) != 1 || s.told[0].unit.SID != visible.SID || s.told[0].session != "crab" {
		t.Fatalf("history escaped its binding: %+v", s.told)
	}
	if len(s.moves.Entries) != 1 || len(s.moves.General) != 0 || len(s.dupes.Duplicates) != 0 || s.movePath == "" || s.dupePath == "" {
		t.Fatal("plan visibility or root discovery failed")
	}
	m := startModel(t, f.App, 140, 40)
	m.startReview("moves")
	f.Clean(t, "alpha", m.View())
	if strings.Contains(m.View(), "stale plan text") || !strings.Contains(m.View(), "Matrix(Alpha)") {
		t.Fatal("review rendered untrusted plan text instead of current visible memory")
	}
	scopetest.Pass(t, "tui.history-and-plans", "alpha")
}

func TestTUIHereUsesProjectLabelNotIdentityHash(t *testing.T) {
	cwd := filepath.Join(t.TempDir(), "friendly-repo")
	if err := os.MkdirAll(filepath.Join(cwd, ".muninn"), 0700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cwd, ".muninn", "project"), []byte("rc-0123456789abcdef\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Chdir(cwd)
	m := startModel(t, testApp(t), 140, 40)
	if m.snapshot.hereLabel != "friendly-repo" || !strings.Contains(m.View(), "here  friendly-repo") || strings.Contains(m.View(), "rc-0123456789abcdef") {
		t.Fatal("overview must label here with its project name, not an opaque identity")
	}
}

func TestScopeMatrixQuickPlaceShortcuts(t *testing.T) {
	f := scopetest.New(t)
	for _, name := range []string{"alpha", "global", "client"} {
		t.Run(name, func(t *testing.T) {
			b := f.Bind(t, name)
			cwd := f.Cwd
			if name == "client" {
				cwd = f.ClientCwd
			}
			t.Chdir(cwd)
			m := startModel(t, f.App, 140, 40)
			for i, alias := range []string{"here", "client", "global"} {
				got := typeText(t, m, string(rune('1'+i))).(model)
				p, err := b.Placement("", alias)
				if alias == "client" && b.Client == "" {
					if len(got.chips) != 0 || !strings.Contains(got.status, "No client") {
						t.Fatal("unavailable client shortcut must explain without silently selecting here")
					}
				} else if err != nil {
					if len(got.chips) != 0 {
						t.Fatalf("unavailable %s created a chip", alias)
					}
				} else if len(got.chips) != 1 || got.chips[0].value != p.Scope {
					t.Fatalf("%s shortcut did not select its Binding place", alias)
				}
				f.Clean(t, name, got.View())
				all := typeText(t, got, "4").(model)
				if len(all.chips) != 0 {
					t.Fatal("all did not remove the place filter")
				}
				f.Clean(t, name, all.View())
			}
			scopetest.Pass(t, "tui.quick-places", name)
		})
	}
}
