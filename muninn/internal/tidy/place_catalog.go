package tidy

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/veridit/hug/muninn/internal/project"
	"github.com/veridit/hug/muninn/internal/store"
)

// PlaceCandidate is an observed, concrete destination, never model-created.
// Several checkouts may share a scope but retain their names in the catalog.
type PlaceCandidate struct {
	Scope       string `json:"scope"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type PlaceCatalog struct {
	Candidates []PlaceCandidate `json:"candidates"`
	Clients    []string         `json:"clients"`
}

// DiscoverPlaces only reads identities and the first README line. It never
// creates project markers, follows symlinks or descends into repository trees.
func DiscoverPlaces(ctx context.Context, home string, bases []string) (PlaceCatalog, error) {
	catalog := PlaceCatalog{Candidates: []PlaceCandidate{
		{Scope: "global", Name: "global", Description: "About the user, their tools, machine and preferences across work"},
		{Scope: "folder:" + filepath.Clean(home), Name: "home folder", Description: "Specific to the home folder " + filepath.Clean(home)},
	}}
	clients := map[string]bool{}
	seen := map[string]bool{}
	var walk func(string) error
	walk = func(dir string) error {
		if err := ctx.Err(); err != nil {
			return err
		}
		if seen[dir] {
			return nil
		}
		seen[dir] = true
		info, err := os.Lstat(dir)
		if os.IsNotExist(err) {
			return nil
		}
		if err != nil {
			return err
		}
		if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
			return nil
		}
		client, _, err := project.ClientRoot(dir)
		if err != nil {
			return err
		}
		if client != "" {
			clients[client] = true
		}
		if _, err := os.Lstat(filepath.Join(dir, ".git")); err == nil {
			p, err := project.Find(dir, false)
			if err == project.ErrNoProject { // unborn repository has no stable scope
				return nil
			}
			if err != nil {
				return err
			}
			if !writableScope(p.Scope()) {
				return fmt.Errorf("invalid project scope at %s", dir)
			}
			catalog.Candidates = append(catalog.Candidates, PlaceCandidate{Scope: p.Scope(), Name: redacted(p.Label), Description: readmeLine(dir)})
			return nil
		} else if !os.IsNotExist(err) {
			return err
		}
		children, err := os.ReadDir(dir)
		if err != nil {
			return err
		}
		for _, c := range children {
			if !c.IsDir() || strings.HasPrefix(c.Name(), ".") {
				continue
			}
			switch c.Name() {
			case "node_modules", "vendor", "target", "build", "dist":
				continue
			}
			if err := walk(filepath.Join(dir, c.Name())); err != nil {
				return err
			}
		}
		return nil
	}
	for _, base := range bases {
		if err := walk(filepath.Clean(base)); err != nil {
			return catalog, err
		}
	}
	for c := range clients {
		catalog.Clients = append(catalog.Clients, redacted(c))
	}
	sort.Strings(catalog.Clients)
	sort.Slice(catalog.Candidates, func(i, j int) bool {
		a, b := catalog.Candidates[i], catalog.Candidates[j]
		if a.Scope != b.Scope {
			return a.Scope < b.Scope
		}
		return a.Name < b.Name
	})
	return catalog, nil
}

func readmeLine(dir string) string {
	files, _ := os.ReadDir(dir)
	for _, f := range files {
		n := strings.ToLower(f.Name())
		if f.Type().IsRegular() && (n == "readme" || strings.HasPrefix(n, "readme.")) {
			r, err := os.Open(filepath.Join(dir, f.Name()))
			if err != nil {
				continue
			}
			s := bufio.NewScanner(io.LimitReader(r, 4096))
			line := ""
			if s.Scan() {
				line = strings.TrimSpace(s.Text())
			}
			r.Close()
			return redacted(string([]rune(line)[:min(240, len([]rune(line)))]))
		}
	}
	return "No README description available"
}

// SelectPlace combines selector families with AND, repeated values with OR.
// Max=0 explicitly selects every match. Unknown requested SIDs fail closed.
type PlaceSelection struct {
	FromScopes  []string
	CwdOrigins  []string
	SIDs        []string
	SessionDirs []string
	Max         int
}

func SelectPlace(ctx context.Context, g Graph, o PlaceSelection) ([]store.Unit, error) {
	if o.Max < 0 {
		return nil, fmt.Errorf("--max must be nonnegative")
	}
	wanted := map[string]bool{}
	for _, sid := range o.SIDs {
		wanted[sid] = true
	}
	for _, scope := range o.FromScopes {
		if !literalScope(scope) {
			return nil, fmt.Errorf("--from-scope requires a literal scope")
		}
	}
	for i, cwd := range o.CwdOrigins {
		if !filepath.IsAbs(cwd) {
			return nil, fmt.Errorf("--cwd-origin requires an absolute directory")
		}
		o.CwdOrigins[i] = filepath.Clean(cwd)
	}
	found := map[string]bool{}
	cwdCache := map[string]string{}
	var out []store.Unit
	for _, u := range g.Units {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		found[u.SID] = true
		if !u.Active || (len(wanted) > 0 && !wanted[u.SID]) || (len(o.FromScopes) > 0 && !containsPlace(o.FromScopes, u.Scope)) {
			continue
		}
		if len(o.CwdOrigins) > 0 {
			if !strings.HasPrefix(u.Origin, "capture:jcode:") {
				continue
			}
			session, _, _ := strings.Cut(strings.TrimPrefix(u.Origin, "capture:jcode:"), ":")
			cwd, ok := cwdCache[session]
			if !ok {
				cwd = originCwd(ctx, session, o.SessionDirs)
				cwdCache[session] = cwd
			}
			if !containsPlace(o.CwdOrigins, cwd) {
				continue
			}
		}
		out = append(out, u)
	}
	for sid := range wanted {
		if !found[sid] {
			return nil, fmt.Errorf("SID %q is not an active canonical memory", sid)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].SID < out[j].SID })
	if o.Max > 0 && len(out) > o.Max {
		out = out[:o.Max]
	}
	return out, nil
}

func containsPlace(ss []string, s string) bool {
	for _, x := range ss {
		if x == s {
			return true
		}
	}
	return false
}

var workingDirRE = regexp.MustCompile(`"working_dir"\s*:\s*("(?:[^"\\]|\\.)*")`)

// Read only metadata, not a transcript in memory. Journals precede snapshots,
// matching capture provenance. Conflicting instance session IDs are excluded.
func originCwd(ctx context.Context, session string, dirs []string) string {
	if session == "" || session == "." || session == ".." || strings.ContainsAny(session, "/\\\x00") {
		return ""
	}
	cwd := ""
	for _, dir := range dirs {
		for _, suffix := range []string{".journal.jsonl", ".json"} {
			value := readOriginCwd(ctx, filepath.Join(dir, session+suffix))
			if value == "" {
				continue
			}
			if cwd != "" && cwd != value {
				return ""
			}
			cwd = value
			break
		}
	}
	return cwd
}

func readOriginCwd(ctx context.Context, path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	r := io.LimitReader(f, 512<<20)
	buf := make([]byte, 64<<10)
	var tail []byte
	for ctx.Err() == nil {
		n, err := r.Read(buf)
		data := append(tail, buf[:n]...)
		if m := workingDirRE.FindSubmatch(data); len(m) > 0 {
			var cwd string
			if json.Unmarshal(m[1], &cwd) == nil && filepath.IsAbs(cwd) {
				return filepath.Clean(cwd)
			}
			return ""
		}
		if err != nil {
			return ""
		}
		tail = append([]byte(nil), data[max(0, len(data)-(16<<10)):]...)
	}
	return ""
}
