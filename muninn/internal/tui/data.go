package tui

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/veridit/muninn/internal/app"
	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/ingest"
	"github.com/veridit/muninn/internal/project"
	"github.com/veridit/muninn/internal/store"
	"github.com/veridit/muninn/internal/tidy"
)

type placeCount struct {
	app.Place
	count int
}
type injection struct {
	unit             store.Unit
	session, harness string
	at               time.Time
}
type snapshot struct {
	binding            app.Binding
	units              []store.Unit
	places             []placeCount
	told               []injection
	dupes, moves       tidy.Plan
	dupePath, movePath string
	hereLabel          string
	err                error
}

func loadSnapshot(a *app.App, b app.Binding) snapshot {
	b = b.Refresh(a)
	s := snapshot{binding: b, err: b.Err}
	if b.Err != nil {
		return s
	}
	where, args := b.SQL("u.scope", "u.source_id")
	rows, err := a.DB.Query(`SELECT u.id FROM units u WHERE (`+where+`) AND u.active=1 AND NOT EXISTS(SELECT 1 FROM suppressed x WHERE x.sid=u.sid)
 AND u.id=(SELECT c.id FROM units c JOIN sources cs ON cs.id=c.source_id WHERE c.sid=u.sid ORDER BY (cs.kind='native') DESC,c.updated_ns DESC,c.id LIMIT 1)
 ORDER BY u.updated_ns DESC,u.sid`, args...)
	if err != nil {
		s.err = err
		return s
	}
	var ids []int64
	for rows.Next() {
		var id int64
		if err = rows.Scan(&id); err != nil {
			break
		}
		ids = append(ids, id)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		s.err = err
		return s
	}
	counts := map[string]int{}
	for _, id := range ids {
		u, e := a.DB.GetUnit(id)
		if e != nil {
			s.err = e
			return s
		}
		if u, ok := b.Present(u); ok {
			s.units = append(s.units, u)
			counts[u.Scope]++
		}
	}
	seen := map[string]bool{}
	for _, p := range b.Places() {
		if !b.Allows(p.Scope) {
			continue
		}
		s.places = append(s.places, placeCount{p, counts[p.Scope]})
		seen[p.Scope] = true
		if p.Name == "here" {
			s.hereLabel = scopeLabel(p.Scope)
			cwd, _ := os.Getwd()
			if pr, err := project.Find(cwd, false); err == nil && pr.Scope() == p.Scope {
				s.hereLabel = pr.Label
			}
		}
	}
	var scopes []string
	for scope := range counts {
		if !seen[scope] {
			scopes = append(scopes, scope)
		}
	}
	sort.Strings(scopes)
	for _, scope := range scopes {
		name := scopeLabel(scope)
		for _, p := range s.places {
			if p.Name == name {
				name = "scope:" + scope
				break
			}
		}
		s.places = append(s.places, placeCount{app.Place{Name: name, Scope: scope, Meaning: "visible scope"}, counts[scope]})
	}
	// Follow injection mirrors to the canonical identity, using the exact same
	// provenance-aware Binding SQL as list/search. Retired rows remain useful history.
	rows, err = a.DB.Query(`SELECT u.id,i.session,i.harness,i.at_ns FROM injections i
 JOIN units mirror ON mirror.id=i.unit_id JOIN units u ON u.sid=mirror.sid
 WHERE (`+where+`) AND u.id=(SELECT c.id FROM units c JOIN sources cs ON cs.id=c.source_id WHERE c.sid=u.sid ORDER BY (cs.kind='native') DESC,c.updated_ns DESC,c.id LIMIT 1)
 ORDER BY i.at_ns DESC,i.session,u.sid LIMIT 200`, args...)
	if err != nil {
		s.err = err
		return s
	}
	type row struct {
		id               int64
		session, harness string
		ns               int64
	}
	var log []row
	for rows.Next() {
		var r row
		if err = rows.Scan(&r.id, &r.session, &r.harness, &r.ns); err != nil {
			break
		}
		log = append(log, r)
	}
	if err == nil {
		err = rows.Err()
	}
	rows.Close()
	if err != nil {
		s.err = err
		return s
	}
	names := map[string]string{}
	for _, r := range log {
		u, e := a.DB.GetUnit(r.id)
		if e != nil {
			continue
		}
		u, ok := b.Present(u)
		if !ok {
			continue
		}
		name := names[r.session]
		if name == "" {
			// Hooks namespace the session as harness + ":" + id. Metadata
			// files and the friendly session_<animal> prefix use the bare id.
			name = sessionName(a, strings.TrimPrefix(r.session, r.harness+":"))
			names[r.session] = name
		}
		s.told = append(s.told, injection{u, name, r.harness, time.Unix(0, r.ns)})
	}
	s.dupes, s.dupePath = latestPlan("dupes", b, a)
	s.moves, s.movePath = latestPlan("moves", b, a)
	return s
}
func sessionName(a *app.App, id string) string {
	// Session metadata is optional. Never let a stored id become a filesystem path.
	if id == "" {
		return "(preview)"
	}
	if strings.HasPrefix(id, "session_") {
		if parts := strings.Split(id, "_"); len(parts) >= 3 && parts[1] != "" {
			return parts[1]
		}
	}
	if strings.ContainsAny(id, "/\\") || id == ".." {
		return id
	}
	for _, inst := range a.JcodeInstances() {
		raw, err := os.ReadFile(filepath.Join(inst.Home, "sessions", id+".json"))
		if err != nil {
			continue
		}
		var meta struct{ Name, Title, FriendlyName string }
		if json.Unmarshal(raw, &meta) != nil {
			continue
		}
		for _, v := range []string{meta.Name, meta.FriendlyName, meta.Title} {
			if v != "" {
				return v
			}
		}
	}
	return id
}
func scopeLabel(scope string) string {
	switch {
	case strings.HasPrefix(scope, "folder:"):
		return filepath.Base(strings.TrimPrefix(scope, "folder:"))
	case strings.HasPrefix(scope, "project:"):
		p := strings.Split(scope, ":")
		if len(p) > 1 {
			return p[1]
		}
	case strings.HasPrefix(scope, "jcode@"):
		return strings.TrimPrefix(scope, "jcode@")
	}
	return scope
}
func (m model) placeName(scope string) string {
	for _, p := range m.snapshot.places {
		if p.Scope == scope {
			if p.Name == "here" {
				return "here"
			}
			return p.Name
		}
	}
	return scopeLabel(scope)
}
func latestPlan(kind string, b app.Binding, a *app.App) (tidy.Plan, string) {
	type candidate struct {
		path string
		at   time.Time
	}
	var files []candidate
	for _, dir := range []string{config.DataDir(), filepath.Join(config.DataDir(), "tidy")} {
		entries, _ := os.ReadDir(dir)
		for _, e := range entries {
			if e.IsDir() || !strings.HasSuffix(e.Name(), ".json") || e.Name() == "memories.json" || e.Name() == "config.json" || e.Name() == "legacy-folders.json" {
				continue
			}
			info, err := e.Info()
			if err == nil {
				files = append(files, candidate{filepath.Join(dir, e.Name()), info.ModTime()})
			}
		}
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].at.Equal(files[j].at) {
			return files[i].path > files[j].path
		}
		return files[i].at.After(files[j].at)
	})
	for _, f := range files {
		p, err := tidy.Load(f.path)
		if err != nil {
			continue
		}
		if p.Created.IsZero() {
			continue
		}
		if (kind == "dupes") != (p.Kind == "dupes") {
			continue
		}
		if p.Kind != "" && p.Kind != "dupes" {
			continue
		}
		if p.Database != "" && filepath.Clean(p.Database) != filepath.Clean(a.DB.Path) {
			continue
		}
		// Copy only entirely visible clusters. No plan text, neighbour, hidden id or
		// target scope is displayed merely because a plan file mentions it.
		if kind == "dupes" {
			var ds []tidy.DuplicateCluster
			for _, c := range p.Duplicates {
				ok := true
				for _, mem := range append([]tidy.DuplicateMember{c.Keeper}, c.Retire...) {
					if _, e := resolveVisible(a, b, mem.SID); e != nil {
						ok = false
						break
					}
				}
				if ok {
					ds = append(ds, c)
				}
			}
			p.Duplicates = ds
		} else {
			var es []tidy.Entry
			for _, e := range p.Entries {
				if _, err := resolveVisible(a, b, e.SID); err == nil && b.Allows(e.Target) {
					e.Neighbors = nil
					e.AnchorScopes = nil
					e.TargetClientGroups = nil
					e.OwnClientGroups = nil
					es = append(es, e)
				}
			}
			p.Entries = es
			p.General = nil
		}
		return p, f.path
	}
	return tidy.Plan{}, ""
}

// The caller already refreshed the binding for this snapshot. Avoid rediscovering
// project ancestry once for every entry in a large tidy plan.
func resolveVisible(a *app.App, b app.Binding, ref string) (store.Unit, error) {
	u, err := a.DB.Resolve(ref)
	if err != nil {
		return store.Unit{}, fmt.Errorf("no visible memory")
	}
	if shown, ok := b.Present(u); ok {
		return shown, nil
	}
	return store.Unit{}, fmt.Errorf("no visible memory")
}
func nativeMeta(a *app.App, u store.Unit) (ingest.NativeMemory, error) {
	var path, kind string
	if err := a.DB.QueryRow(`SELECT path,kind FROM sources WHERE id=?`, u.SourceID).Scan(&path, &kind); err != nil {
		return ingest.NativeMemory{}, err
	}
	if kind != "native" {
		return ingest.NativeMemory{}, fmt.Errorf("upstream")
	}
	// Read raw JSON, never LoadNative: rendering must not trigger legacy repairs.
	raw, err := os.ReadFile(path)
	if err != nil {
		return ingest.NativeMemory{}, err
	}
	var f struct{ Memories []ingest.NativeMemory }
	if err = json.Unmarshal(raw, &f); err != nil {
		return ingest.NativeMemory{}, err
	}
	for _, mem := range f.Memories {
		if mem.ID == u.Key {
			return mem, nil
		}
	}
	return ingest.NativeMemory{}, fmt.Errorf("memory missing from native file")
}
