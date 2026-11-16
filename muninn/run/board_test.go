package run

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
)

// nudge_board = false in config.json turns board nudges off: the hook gets
// no board source, so BoardCue returns no line (qawmnb).
func TestNudgeBoardOff(t *testing.T) {
	for _, tc := range []struct {
		json string
		on   bool
	}{
		{`{}`, true},
		{`{"nudge_board": true}`, true},
		{`{"nudge_board": false}`, false},
	} {
		home := t.TempDir()
		t.Setenv("HUG_HOME", home)
		if err := os.MkdirAll(config.ConfigDir(), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(config.ConfigDir(), "config.json"), []byte(tc.json), 0o600); err != nil {
			t.Fatal(err)
		}
		cfg, err := config.Load()
		if err != nil {
			t.Fatal(err)
		}
		if cfg.BoardNudges() != tc.on {
			t.Fatalf("%s: BoardNudges() = %v", tc.json, cfg.BoardNudges())
		}
		bs := boardSource(context.Background(), cfg.BoardNudges(), t.TempDir(), "guppy", func(time.Time) harness.Activity { return harness.Activity{} })
		if (bs != nil) != tc.on {
			t.Fatalf("%s: board source present = %v", tc.json, bs != nil)
		}
	}
}
