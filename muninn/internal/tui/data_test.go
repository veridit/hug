package tui

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/scopetest"
	"github.com/veridit/muninn/internal/tidy"
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
		if _, err := f.App.DB.Exec(`INSERT INTO injections(session,unit_id,content_hash,harness,at_ns) SELECT 'session_crab_12345_abcdef',id,content_hash,'jcode',? FROM units WHERE sid=?`, time.Now().UnixNano(), u); err != nil {
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
