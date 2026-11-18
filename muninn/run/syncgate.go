package run

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/veridit/hug/huginn/gitbranch"
	"github.com/veridit/hug/internal/memshare"
	"github.com/veridit/hug/muninn/internal/agents"
	"github.com/veridit/hug/muninn/internal/app"
	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/project"
)

// syncGate loads the placement gate for p from the local config (rnpdnm
// B). With no sync_gate in config.json it seeds the term list once, saves
// it, and says so on w, naming the file, never the terms.
func syncGate(a *app.App, p project.Project, w io.Writer, dry bool) (*project.Gate, error) {
	cfg, err := config.Load()
	if err != nil {
		return nil, err
	}
	if cfg.SyncGate == nil {
		terms := seedTerms(a, p)
		cfg.SyncGate = &config.SyncGate{Terms: terms}
		if !dry {
			if err := config.Save(cfg); err != nil {
				return nil, err
			}
			fmt.Fprintf(w, "placement gate: seeded %d names of other projects, clients and people into %s (sync_gate.terms; local, never pushed)\n", len(terms), config.ConfigPath())
		}
	}
	own := []string{p.Label}
	if p.Client != "" {
		own = append(own, p.Client)
	}
	g := &project.Gate{Terms: cfg.SyncGate.Terms, Release: cfg.SyncGate.Release, Own: own}
	if cfg.Extract.WidePlaces {
		g.Team = teamTerms(a, p)
	}
	return g, nil
}

// teamTerms is the team set (rnpdnm attempt 2, PREREG2 (a)): the board
// coordinator from hug.toml, the agents whose jcode sessions worked in this
// repository or one of its worktrees, and the fixed phrases. It is read at
// each sync and never stored or printed: the names come and go, and a list
// of them is not for a repository.
func teamTerms(a *app.App, p project.Project) []string {
	terms := append([]string(nil), project.TeamPhrases...)
	if toml, ok := memshare.Config(p.Root); ok {
		if c := project.CoordinatorTerm(toml); c != "" {
			terms = append(terms, c)
		}
	}
	roots := []string{p.Root}
	if l, err := gitbranch.Locate(p.Root); err == nil {
		roots = append(roots, l.Main)
		es, _ := os.ReadDir(filepath.Join(l.Common, "worktrees"))
		for _, e := range es {
			if b, err := os.ReadFile(filepath.Join(l.Common, "worktrees", e.Name(), "gitdir")); err == nil {
				roots = append(roots, filepath.Dir(strings.TrimSpace(string(b))))
			}
		}
	}
	var homes []string
	if a != nil {
		for _, i := range a.JcodeInstances() {
			homes = append(homes, i.Home)
		}
	}
	return append(terms, agents.NamesIn(homes, roots)...)
}

// seedTerms: the names of the places this store holds memories for, other
// than p (project and client folder labels, client names, jcode instance
// names), and the people hug.toml [people] names. Measured on hug's 868
// public memories against the folder catalog: 13 leaks against 17, and no
// common words ("search", "v") from folders nobody writes memories for.
func seedTerms(a *app.App, p project.Project) []string {
	set := map[string]bool{}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if len([]rune(s)) >= 2 && !strings.EqualFold(s, p.Label) && !strings.EqualFold(s, p.Client) {
			set[s] = true
		}
	}
	scopes := map[string]bool{}
	if a != nil && a.DB != nil {
		if rows, err := a.DB.Query(`SELECT DISTINCT scope FROM units`); err == nil {
			for rows.Next() {
				var s string
				if rows.Scan(&s) == nil {
					scopes[s] = true
				}
			}
			rows.Close()
		}
	}
	if mems, err := ingest.LoadNative(app.NativePath()); err == nil {
		for _, m := range mems {
			scopes[m.Scope] = true
		}
	}
	labels := placeLabels(a)
	for s := range scopes {
		if s == p.Scope() {
			continue
		}
		switch {
		case strings.HasPrefix(s, "client:"):
			client, _, _ := strings.Cut(strings.TrimPrefix(s, "client:"), ":")
			add(client)
		case strings.HasPrefix(s, "jcode@"):
			name, _, _ := strings.Cut(strings.TrimPrefix(s, "jcode@"), ":")
			if name != "global" {
				add(name)
			}
		}
		if l := labels[s]; l != "" {
			add(l)
		}
	}
	if a != nil {
		for _, i := range a.JcodeInstances() {
			if i.Name != "global" {
				add(i.Name)
			}
		}
	}
	if toml, ok := memshare.Config(p.Root); ok {
		for _, t := range project.PeopleTerms(toml) {
			add(t)
		}
	}
	add("the user")
	add("the owner")
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

// placeLabels maps project scopes to their folder names from the local
// label cache hug mem tui keeps (project-labels.json in the data dir).
func placeLabels(*app.App) map[string]string {
	out := map[string]string{}
	b, err := os.ReadFile(filepath.Join(config.DataDir(), "project-labels.json"))
	if err == nil {
		_ = json.Unmarshal(b, &out)
	}
	return out
}

// printHeld writes what the gate kept local: one line per reason, the note
// that dropping a pushed line from the tip does not unpublish it, and the
// read-only list command. Never a fix: line that releases: an agent runs a
// fix: line as given, and a release publishes (crab, rnpdnm).
func printHeld(w io.Writer, st project.SyncStats) {
	if st.HeldTotal() == 0 {
		return
	}
	for _, r := range project.HeldReasons {
		if n := st.Held[r]; n > 0 {
			fmt.Fprintf(w, "held %d: %s\n", n, r)
		}
	}
	if st.HeldPushed > 0 {
		fmt.Fprintf(w, "held %d lines that are already on the remote's history: dropping them from the tip does not unpublish them\n", st.HeldPushed)
	}
	fmt.Fprintln(w, "fix: hug mem sync --held")
}

// listHeld is hug mem sync --held: read-only, nothing is written, fetched
// or pushed. The release is named as a sentence for a person.
func listHeld(w io.Writer, a *app.App, p project.Project) error {
	g, err := syncGate(a, p, w, true)
	if err != nil {
		return err
	}
	if main, err := gitbranch.MainRoot(p.Root); err == nil {
		p.Notes = filepath.Join(main, project.NotesDir)
	}
	held, err := project.HeldLines(p, app.NativePath(), g)
	if err != nil {
		return err
	}
	if len(held) == 0 {
		fmt.Fprintf(w, "%s (%s): no project memory is held by the placement gate\n", p.Label, p.ID)
		return nil
	}
	fmt.Fprintf(w, "%s (%s): %d project memories stay on this machine (placement gate, terms in %s)\n", p.Label, p.ID, len(held), config.ConfigPath())
	for _, h := range held {
		id := h.SID
		if id == "" {
			id = h.ID
		}
		why := h.Reason
		if h.Term != "" {
			why += " (" + h.Term + ")"
		}
		pushed := ""
		if h.Pushed {
			pushed = " [already in the remote's history]"
		}
		fmt.Fprintf(w, "%s  %s%s\n    %s\n", id, why, pushed, oneLine(h.Text, 160))
	}
	fmt.Fprintln(w, "A person who has read a held memory and finds it belongs to this project can let it push with hug mem sync --release ID; one that belongs elsewhere moves with hug mem rescope ID --to personal (or its project).")
	return nil
}

// releaseHeld is hug mem sync --release ID: one local exception, written
// to config.json. It pushes on the next sync.
func releaseHeld(w io.Writer, a *app.App, p project.Project, ref string) error {
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		return err
	}
	var hit *ingest.NativeMemory
	for i := range mems {
		m := &mems[i]
		if m.Scope == p.Scope() && (m.ID == ref || (m.SID != "" && m.SID == ref)) {
			hit = m
			break
		}
	}
	if hit == nil {
		return fmt.Errorf("no memory %s in this project's scope %s\nfix: hug mem sync --held", ref, p.Scope())
	}
	cfg, err := config.Load()
	if err != nil {
		return err
	}
	if cfg.SyncGate == nil {
		cfg.SyncGate = &config.SyncGate{Terms: seedTerms(a, p)}
	}
	for _, r := range cfg.SyncGate.Release {
		if r == hit.ID {
			fmt.Fprintf(w, "%s is released already; it pushes on the next sync\n", ref)
			return nil
		}
	}
	cfg.SyncGate.Release = append(cfg.SyncGate.Release, hit.ID)
	if err := config.Save(cfg); err != nil {
		return err
	}
	fmt.Fprintf(w, "released %s (sync_gate.release in %s): it pushes on the next sync\n", ref, config.ConfigPath())
	return nil
}

func oneLine(s string, n int) string {
	s = strings.Join(strings.Fields(s), " ")
	if r := []rune(s); len(r) > n {
		return string(r[:n-1]) + "…"
	}
	return s
}
