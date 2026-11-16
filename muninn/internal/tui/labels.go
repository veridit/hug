package tui

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/project"
)

// projectLabels maps project scopes (project:rc-..., client:x:rc-...) to the
// folder names they came from, so the places pane reads "seid-lang" rather
// than "project:rc-0ca4bf6be5d989c3". The map is built from the working
// folders of jcode sessions and cached for a day in the data dir.
func projectLabels(a *app.App) map[string]string {
	cache := filepath.Join(config.DataDir(), "project-labels.json")
	out := map[string]string{}
	info, statErr := os.Stat(cache)
	if statErr == nil {
		if b, e := os.ReadFile(cache); e == nil {
			_ = json.Unmarshal(b, &out)
		}
	}
	// The scan reads every session file (seconds): never block the TUI on
	// it. A stale or missing cache is rebuilt in the background and the
	// names appear on the next refresh.
	if statErr != nil || time.Since(info.ModTime()) > 24*time.Hour {
		labelsOnce.Do(func() { go buildProjectLabels(a, cache) })
	}
	return out
}

var labelsOnce sync.Once

func buildProjectLabels(a *app.App, cache string) map[string]string {
	out := map[string]string{}
	dirs := map[string]bool{}
	if a != nil {
		for _, inst := range a.JcodeInstances() {
			files, _ := filepath.Glob(filepath.Join(inst.Home, "sessions", "*.json"))
			for _, f := range files {
				b, e := os.ReadFile(f)
				if e != nil {
					continue
				}
				// Byte search, not a full decode: sessions run to 50 MB and
				// the field can sit megabytes in.
				key := []byte(`"working_dir":"`)
				for rest := b; ; {
					i := bytes.Index(rest, key)
					if i < 0 {
						break
					}
					rest = rest[i+len(key):]
					if j := bytes.IndexByte(rest, '"'); j > 0 {
						var d string
						if json.Unmarshal(append(append([]byte{'"'}, rest[:j]...), '"'), &d) == nil && d != "" {
							dirs[d] = true
						}
					}
				}
			}
		}
	}
	for d := range dirs {
		if fi, e := os.Stat(d); e != nil || !fi.IsDir() {
			continue
		}
		p, e := project.Find(d, false)
		if e != nil || p.Label == "" {
			continue
		}
		out[p.Scope()] = p.Label
	}
	if b, e := json.Marshal(out); e == nil {
		_ = os.WriteFile(cache, b, 0o600)
	}
	return out
}
