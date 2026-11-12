package app_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/scopetest"
	"github.com/veridit/hug/muninn/internal/tidy"
)

func duplicatePlanFixture(t *testing.T, f *scopetest.Fixture, upstream bool) tidy.Plan {
	t.Helper()
	if upstream {
		path := filepath.Join(f.Root, "alpha", ".jcode", "memory", "global.json")
		if err := os.MkdirAll(filepath.Dir(path), 0700); err != nil {
			t.Fatal(err)
		}
		raw := `{"memories":{"dupe_old":{"id":"dupe_old","content":"  2025-01-01: synthetic upstream quartz policy remains useful.  ","category":"fact"},"dupe_new":{"id":"dupe_new","content":"2025-02-01: synthetic quartz policy is useful upstream.","category":"fact"}}}`
		if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
			t.Fatal(err)
		}
	} else {
		ms, err := ingest.LoadNative(app.NativePath())
		if err != nil {
			t.Fatal(err)
		}
		ms = append(ms, ingest.NativeMemory{ID: "dupe_old", Content: "  2025-01-01: synthetic quartz policy remains useful.  \n", Category: "fact", Scope: "global", Active: true, Strength: 2, Created: time.Unix(100, 0), Updated: time.Unix(100, 0)}, ingest.NativeMemory{ID: "dupe_new", Content: "2025-02-01: synthetic quartz policy is useful permanently.", Category: "fact", Scope: "global", Active: true, Strength: 5, Created: time.Unix(200, 0), Updated: time.Unix(200, 0)})
		if err = ingest.SaveNative(app.NativePath(), ms); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := f.App.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	g, err := tidy.ReadGraph(context.Background(), config.DBPath())
	if err != nil {
		t.Fatal(err)
	}
	var a, b int64
	for _, u := range g.Units {
		if u.Key == "dupe_old" {
			a = u.ID
		}
		if u.Key == "dupe_new" {
			b = u.ID
		}
	}
	if a == 0 || b == 0 {
		t.Fatalf("fixture not indexed: %d %d", a, b)
	}
	g.Edges = []tidy.Edge{{From: a, To: b, Sim: .96}}
	p, err := tidy.ScanDuplicates(g, .95, .88)
	if err != nil || len(p.Duplicates) != 1 {
		t.Fatalf("%+v %v", p, err)
	}
	p.Database = config.DBPath()
	return p
}
func TestTidyDuplicateApplyUndoTextStrengthAndUpstream(t *testing.T) {
	for _, upstream := range []bool{false, true} {
		name := "native"
		if upstream {
			name = "upstream"
		}
		t.Run(name, func(t *testing.T) {
			f := scopetest.New(t)
			p := duplicatePlanFixture(t, f, upstream)
			ctx := context.Background()
			before, _ := os.ReadFile(app.NativePath())
			var upBefore []byte
			upPath := filepath.Join(f.Root, "alpha", ".jcode", "memory", "global.json")
			if upstream {
				upBefore, _ = os.ReadFile(upPath)
			}
			dry, err := f.App.ApplyTidy(ctx, p, "", true)
			if err != nil || len(dry.Retirements) != 1 || dry.Backup != "" {
				t.Fatalf("dry=%+v %v", dry, err)
			}
			after, _ := os.ReadFile(app.NativePath())
			if !bytes.Equal(before, after) {
				t.Fatal("dry run wrote native")
			}
			result, err := f.App.ApplyTidy(ctx, p, "", false)
			if err != nil || len(result.Retirements) != 1 {
				t.Fatalf("result=%+v err=%v", result, err)
			}
			keeper := p.Duplicates[0].Keeper
			retired := p.Duplicates[0].Retire[0]
			ms, _ := ingest.LoadNative(app.NativePath())
			foundKeeper, foundRetired := false, false
			for _, m := range ms {
				if m.SID == keeper.SID {
					foundKeeper = m.Active && m.Strength == keeper.Strength+retired.Strength && m.Content == keeper.Text && m.Updated.Equal(keeper.Updated)
				}
				if m.RetiredInto == keeper.SID {
					foundRetired = !m.Active && m.RetiredOrigin == retired.Origin && m.Content == retired.Text && m.Updated.Equal(retired.Updated)
				}
			}
			if !foundKeeper || !foundRetired || !f.App.DB.IsSuppressed(retired.SID) {
				t.Fatalf("keeper=%v retired=%v cluster=%+v memories=%+v", foundKeeper, foundRetired, p.Duplicates[0], ms)
			}
			if upstream {
				raw, _ := os.ReadFile(upPath)
				if !bytes.Equal(raw, upBefore) {
					t.Fatal("upstream rewritten")
				}
			}
			if err = f.App.UndoTidy(ctx, result.Backup); err != nil {
				t.Fatal(err)
			}
			after, _ = os.ReadFile(app.NativePath())
			if !bytes.Equal(before, after) || f.App.DB.IsSuppressed(retired.SID) {
				t.Fatal("undo not byte-exact or suppression survived")
			}
			restored, err := f.App.DB.Resolve(retired.SID)
			if err != nil || !restored.Active {
				t.Fatalf("undo lost original: %+v %v", restored, err)
			}
		})
	}
}
func TestTidyDuplicateFailuresAndUndoConflict(t *testing.T) {
	t.Run("failure-rollback", func(t *testing.T) {
		f := scopetest.New(t)
		p := duplicatePlanFixture(t, f, false)
		before, _ := os.ReadFile(app.NativePath())
		if _, err := f.App.DB.Exec(`CREATE TRIGGER duplicate_failure BEFORE INSERT ON suppressed BEGIN SELECT RAISE(FAIL,'synthetic duplicate suppression failure'); END`); err != nil {
			t.Fatal(err)
		}
		if _, err := f.App.ApplyTidy(context.Background(), p, "", false); err == nil {
			t.Fatal("injected failure ignored")
		}
		after, _ := os.ReadFile(app.NativePath())
		if !bytes.Equal(before, after) || f.App.DB.IsSuppressed(p.Duplicates[0].Retire[0].SID) {
			t.Fatal("failed batch mutated memory")
		}
	})
	t.Run("undo-conflict", func(t *testing.T) {
		f := scopetest.New(t)
		p := duplicatePlanFixture(t, f, false)
		result, err := f.App.ApplyTidy(context.Background(), p, "", false)
		if err != nil {
			t.Fatal(err)
		}
		if _, _, err = ingest.Remember(app.NativePath(), "Synthetic later durable note must survive undo.", "fact", "global", nil); err != nil {
			t.Fatal(err)
		}
		before, _ := os.ReadFile(app.NativePath())
		if err = f.App.UndoTidy(context.Background(), result.Backup); err == nil {
			t.Fatal("undo overwrote later memory")
		}
		after, _ := os.ReadFile(app.NativePath())
		if !bytes.Equal(before, after) {
			t.Fatal("conflict changed file")
		}
	})
	t.Run("unindexed-edit", func(t *testing.T) {
		f := scopetest.New(t)
		p := duplicatePlanFixture(t, f, false)
		ms, _ := ingest.LoadNative(app.NativePath())
		for i := range ms {
			if ms[i].ID == "dupe_old" {
				ms[i].Content = "Synthetic unindexed edit must never be retired."
			}
		}
		if err := ingest.SaveNative(app.NativePath(), ms); err != nil {
			t.Fatal(err)
		}
		if _, err := f.App.ApplyTidy(context.Background(), p, "", false); err == nil {
			t.Fatal("unindexed edit accepted")
		}
	})
}
