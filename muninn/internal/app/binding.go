package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/holds"
	"github.com/veridit/hug/muninn/internal/ingest"
	"github.com/veridit/hug/muninn/internal/project"
	"github.com/veridit/hug/muninn/internal/search"
	"github.com/veridit/hug/muninn/internal/store"
	"github.com/veridit/hug/muninn/internal/trigger"
)

// Binding is the single visibility and write policy shared by every surface.
// Scopes supports exact names, trailing * prefixes and leading ! exclusions.
// Owner view overrides change reads only, never the underlying write policy.
type Binding struct {
	Scopes        []string
	DefaultScope  string
	Name          string
	Client        string
	Err           error
	writeScopes   []string
	sourceClients map[int64]string
	sourceScopes  map[int64]string
	legacyOwn     map[string]string
	legacyAll     bool
	context       *bindingContext
}

type bindingContext struct {
	home, cwd string
	all       bool
	scopes    []string
}

// Bound resolves explicit home, then JCODE_HOME, then the global instance.
// Client instances are allow-listed. Global excludes every known client,
// including indexed scopes whose instance is temporarily absent on disk.
func (a *App) Bound(jcodeHome, cwd string) Binding {
	if jcodeHome == "" {
		jcodeHome = os.Getenv("JCODE_HOME")
	}
	if jcodeHome == "" {
		home, _ := os.UserHomeDir()
		jcodeHome = filepath.Join(home, ".jcode")
	}
	jcodeHome, _ = filepath.Abs(jcodeHome)
	if cwd == "" {
		cwd, _ = os.Getwd()
	}
	b := a.bound(jcodeHome, cwd)
	b.context = &bindingContext{home: jcodeHome, cwd: cwd}
	// Already resolved above. Refresh would repeat project identity discovery
	// (including git subprocesses) before anything could have changed.
	return b.withSources(a)
}

// Refresh discovers newly indexed sources while keeping an owner's explicit
// view and the original write policy. Filesystem ancestry stays process-local.
func (b Binding) Refresh(a *App) Binding {
	if c := b.context; c != nil {
		fresh := a.bound(c.home, c.cwd).View(c.all, c.scopes)
		fresh.context = c
		b = fresh
	}
	return b.withSources(a)
}

func (b Binding) withSources(a *App) Binding {
	if b.Err == nil && a.DB != nil {
		b.sourceClients, b.sourceScopes, b.Err = a.sourceClients(b.Client, b.legacyAll, b.legacyOwn)
	}
	return b
}

func (a *App) bound(jcodeHome, cwd string) Binding {
	client, clientRoot, markerErr := project.ClientRoot(cwd)
	if markerErr != nil {
		return Binding{Name: "client (invalid marker)", Err: markerErr, Scopes: []string{"!*"}, writeScopes: []string{"!*"}}
	}
	if client != "" {
		scope := "client:" + client
		rules := []string{scope}
		here, projectRoot := scope, ""
		physical, err := placementDir(cwd)
		if err != nil {
			return Binding{Name: scope, Client: client, Err: err, Scopes: []string{"!*"}, writeScopes: []string{"!*"}}
		}
		// Root reads include the whole client. Writes retain the original
		// top-level-only policy. Nested sessions never gain sibling scopes.
		if physical != clientRoot {
			here = scope + ":folder:" + physical
			if p, err := project.Find(physical, false); err == nil && p.Root != clientRoot && FolderOwns(p.Root, physical) {
				rules = append(rules, p.Scope())
				here, projectRoot = p.Scope(), p.Root
			}
			for dir := physical; dir != clientRoot && filepath.Dir(dir) != dir; dir = filepath.Dir(dir) {
				if FolderOwns(dir, physical) {
					rules = append(rules, scope+":folder:"+dir)
				}
			}
		}
		writes := append([]string(nil), rules...)
		if physical == clientRoot {
			rules = append(rules, scope+":*")
		}
		return Binding{Name: scope, Client: client, DefaultScope: scope, Scopes: rules, writeScopes: writes, legacyAll: physical == clientRoot, legacyOwn: clientFolderStores(cwd, physical, clientRoot, projectRoot, here, scope, writes)}
	}
	if jcodeHome == "" {
		jcodeHome = os.Getenv("JCODE_HOME")
	}
	home, _ := os.UserHomeDir()
	global := filepath.Join(home, ".jcode")
	if jcodeHome == "" {
		jcodeHome = global
	}
	jcodeHome, _ = filepath.Abs(jcodeHome)
	inst := harness.InstanceForHome(a.JcodeInstances(), jcodeHome)
	if filepath.Clean(jcodeHome) == global {
		inst.Name = "global"
	}
	b := Binding{Name: inst.Scope(), DefaultScope: inst.Scope()}
	if inst.Name != "global" {
		b.Scopes = []string{inst.Scope(), inst.Scope() + ":*", "global"}
		if cwd != "" {
			if p, err := project.Find(cwd, false); err == nil {
				b.Scopes = append(b.Scopes, p.Scope())
			}
		}
	} else {
		names := map[string]bool{}
		for _, i := range a.JcodeInstances() {
			if i.Name != "global" {
				names[i.Scope()] = true
			}
		}
		// Without a database (Preview, vrx1ja) only the exclusions of other
		// stores are missing; this binding's name and places are the same.
		if a.DB == nil {
			for s := range names {
				b.Scopes = append(b.Scopes, "!"+s, "!"+s+":*")
			}
			sort.Strings(b.Scopes)
			b.Scopes = append(b.Scopes, "!client:*")
			b.writeScopes = append([]string{}, b.Scopes...)
			return b
		}
		rows, err := a.DB.Query(`SELECT DISTINCT scope FROM units WHERE scope LIKE 'jcode@%'`)
		if err != nil {
			b.Scopes = []string{"!*"}
			b.Err = err
		}
		if err == nil {
			for rows.Next() {
				var s string
				if err := rows.Scan(&s); err != nil {
					b.Err = err
					break
				} else {
					s, _, _ = strings.Cut(s, ":")
					if s != "jcode@global" {
						names[s] = true
					}
				}
			}
			if err := rows.Err(); err != nil {
				b.Err = err
			}
			rows.Close()
		}
		for s := range names {
			b.Scopes = append(b.Scopes, "!"+s, "!"+s+":*")
		}
		// Project memories belong to their project: global sees only the
		// project of its working directory, like an instance binding does.
		current := ""
		if cwd != "" {
			if p, err := project.Find(cwd, false); err == nil {
				current = p.Scope()
			}
		}
		// jcode's own per-folder stores (jcode@global:<hash of the session's
		// folder>) belong to that folder too (user 2026-11-06): visible from the
		// folder itself, or from below it within the same git project. The home
		// and other parent folders are not projects, so their stores show only there.
		own := globalFolderStores(cwd)
		if grow, err := a.DB.Query(`SELECT DISTINCT scope FROM units WHERE scope LIKE 'jcode@global:%'`); err != nil {
			b.Scopes, b.Err = []string{"!*"}, err
		} else {
			for grow.Next() {
				var s string
				if grow.Scan(&s) == nil && !own[strings.TrimPrefix(s, "jcode@global:")] {
					b.Scopes = append(b.Scopes, "!"+s)
				}
			}
			if err := grow.Err(); err != nil {
				b.Err = err
			}
			grow.Close()
		}
		// Imports from folders that are not projects keep their folder as the
		// scope, folder:<path> (user 2026-11-07). Same rule as the stores above.
		if frow, err := a.DB.Query(`SELECT DISTINCT scope FROM units WHERE scope LIKE 'folder:%'`); err != nil {
			b.Scopes, b.Err = []string{"!*"}, err
		} else {
			for frow.Next() {
				var s string
				if frow.Scan(&s) == nil && !FolderOwns(strings.TrimPrefix(s, "folder:"), cwd) {
					b.Scopes = append(b.Scopes, "!"+s)
				}
			}
			if err := frow.Err(); err != nil {
				b.Err = err
			}
			frow.Close()
		}
		if prow, err := a.DB.Query(`SELECT DISTINCT scope FROM units WHERE scope LIKE 'project:%'`); err != nil {
			b.Scopes, b.Err = []string{"!*"}, err
		} else {
			for prow.Next() {
				var s string
				if prow.Scan(&s) == nil && s != current {
					b.Scopes = append(b.Scopes, "!"+s)
				}
			}
			if err := prow.Err(); err != nil {
				b.Err = err
			}
			prow.Close()
		}
		sort.Strings(b.Scopes)
	}
	b.Scopes = append(b.Scopes, "!client:*")
	b.writeScopes = append([]string{}, b.Scopes...)
	return b
}

// View explicitly widens an owner view. It never widens writes.
func (b Binding) View(all bool, scopes []string) Binding {
	if b.Client != "" || b.Err != nil {
		return b
	}
	if b.context != nil {
		c := *b.context
		c.all = all
		c.scopes = append([]string(nil), scopes...)
		b.context = &c
	}
	if all {
		b.Scopes = nil
		b.Name += " (all)"
	}
	if len(scopes) > 0 {
		b.Scopes = append([]string(nil), scopes...)
		b.Name += " (scope: " + strings.Join(scopes, ", ") + ")"
	}
	// Owner view flags may cross instance views, never a client boundary.
	b.Scopes = append(b.Scopes, "!client:*")
	return b
}
func (b Binding) Allows(scope string) bool { return b.Err == nil && search.Allowed(scope, b.Scopes) }
func (b Binding) WriteScope(scope string) (string, error) {
	if b.Err != nil {
		return "", b.Err
	}
	if scope == "" {
		scope = b.DefaultScope
	}
	if scope == "" {
		scope = "global"
	}
	if strings.HasPrefix(scope, "jcode@") && b.DefaultScope != "" {
		instance, _, _ := strings.Cut(scope, ":")
		own, _, _ := strings.Cut(b.DefaultScope, ":")
		if instance != own {
			return "", fmt.Errorf("scope %q is not available in this binding", scope)
		}
	}
	rules := b.writeScopes
	if rules == nil {
		rules = b.Scopes
	}
	if !search.Allowed(scope, rules) {
		return "", fmt.Errorf("scope %q is not available in this binding", scope)
	}
	return scope, nil
}

func (b Binding) ResolveAll(a *App, ref string) ([]store.Unit, error) {
	b = b.Refresh(a)
	if b.Err != nil {
		return nil, b.Err
	}
	units, err := a.DB.ResolveAll(ref)
	if err != nil {
		return nil, fmt.Errorf("no visible memory")
	}
	if _, ok := b.Present(units[0]); !ok {
		return nil, fmt.Errorf("no visible memory")
	}
	var visible []store.Unit
	for _, u := range units {
		if shown, ok := b.Present(u); ok {
			visible = append(visible, shown)
		}
	}
	if len(visible) == 0 {
		return nil, fmt.Errorf("no visible memory")
	}
	return visible, nil
}
func (b Binding) Resolve(a *App, ref string) (store.Unit, error) {
	units, err := b.ResolveAll(a, ref)
	if err != nil {
		return store.Unit{}, err
	}
	return units[0], nil
}
func (b Binding) Forget(ctx context.Context, a *App, ref string, sessions ...string) (ForgetResult, error) {
	out, err := b.ForgetMany(ctx, a, []string{ref}, sessions...)
	if err != nil {
		return ForgetResult{}, err
	}
	return out[0], nil
}

// ForgetMany checks every ref against this binding, then forgets them all in
// one write turn (App.ForgetMany). One ref that is not visible here fails
// the whole call before anything changes.
func (b Binding) ForgetMany(ctx context.Context, a *App, refs []string, sessions ...string) ([]ForgetResult, error) {
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return nil, err
	}
	defer release()
	b = b.Refresh(a)
	if b.Err != nil {
		return nil, b.Err
	}
	var hidden []string
	for _, ref := range refs {
		if err := b.mayForget(a, ref, sessions); err != nil {
			if len(refs) == 1 {
				return nil, err
			}
			hidden = append(hidden, ref)
		}
	}
	if len(hidden) > 0 {
		return nil, fmt.Errorf("nothing forgotten: no visible memory %s", strings.Join(hidden, ", "))
	}
	return forgetHeld(a.forgetMany(withSessions(ctx, sessions), refs))
}

// withSessions adds the caller's sessions to the fix context on ctx, so the
// trail sees whether recall showed the memory to them (rjk44w).
func withSessions(ctx context.Context, sessions []string) context.Context {
	if len(sessions) == 0 {
		return ctx
	}
	fc := fixFrom(ctx)
	fc.Sessions = append(append([]string(nil), fc.Sessions...), sessions...)
	return WithFix(ctx, fc)
}

// forgetHeld reads a forget back: every memory it reports forgotten is
// inactive in the store now. A writer that put one back (tn8bmn: a project
// sync read a stale line over it) makes the forget fail, never report
// success it does not have. A store file whose stat, under its lock, is
// still the one this forget saved has had no other writer since, so it holds
// what the forget wrote and is not read again (4vd1mz: the reload was
// 120 ms of a forget on a 30,000-memory store).
func forgetHeld(out []ForgetResult, saved map[string]ingest.FileStat, err error) ([]ForgetResult, error) {
	if err != nil {
		return out, err
	}
	WriteLanded()
	if unchangedSince(saved) {
		return out, nil
	}
	mems, err := loadLocked()
	if err != nil {
		return out, err
	}
	var back []string
	for _, r := range out {
		for _, m := range mems {
			if m.SID == r.ID && m.Active {
				back = append(back, r.ID)
				break
			}
		}
	}
	if len(back) > 0 {
		ids := strings.Join(back, " ")
		return out, fmt.Errorf("forget did not hold: %s is active again in the store (another writer restored it)\nfix: hug mem forget %s", ids, ids)
	}
	return out, nil
}

// WriteLanded runs between a remember or forget and its readback; tests put
// another writer there.
var WriteLanded = func() {}

// loadLocked reads the native store under its write lock: a write's
// readback sees every writer that finished before it, none half done.
func loadLocked() ([]ingest.NativeMemory, error) {
	unlock, err := ingest.LockNative(NativePath())
	if err != nil {
		return nil, err
	}
	defer unlock()
	return ingest.LoadNative(NativePath())
}

// unchangedSince reports every saved store file still at the stat its write
// left it with, checked under the file's lock; false when nothing was saved
// (a suppression only) or any file moved on.
func unchangedSince(saved map[string]ingest.FileStat) bool {
	if len(saved) == 0 {
		return false
	}
	for path, want := range saved {
		unlock, err := ingest.LockNative(path)
		if err != nil {
			return false
		}
		got, err := ingest.StatFile(path)
		unlock()
		if err != nil || got != want || want.Mtime <= 0 {
			return false
		}
	}
	return true
}

func (b Binding) mayForget(a *App, ref string, sessions []string) error {
	// Check the underlying write policy even for an explicitly widened view.
	units, err := a.DB.ResolveAll(ref)
	if err != nil {
		return fmt.Errorf("no visible memory")
	}
	// The recall block tells the agent "Fix: memory_forget <id>". A memory
	// recalled into this session must be forgettable from it, even when the
	// command runs in another folder than the session started in (2026-11-11:
	// d8jb1b, scope folder:~/Projects, recalled into a session started in
	// ~/Projects, refused from ~/Projects/muninn). Client sealing still holds:
	// a memory is only ever recalled into a session that may see it.
	if a.shownInSession(units[0].ID, sessions) {
		return nil
	}
	if _, ok := b.Present(units[0]); !ok {
		return fmt.Errorf("no visible memory")
	}
	for _, u := range units {
		var kind string
		if err := a.DB.QueryRow(`SELECT kind FROM sources WHERE id=?`, u.SourceID).Scan(&kind); err != nil {
			return err
		}
		// The canonical memory owns the identity. Suppressing an upstream
		// mirror is safe after an explicit rescope, even in its old scope.
		if kind != "native" && u.ID != units[0].ID {
			continue
		}
		shown, ok := b.Present(u)
		if !ok {
			return fmt.Errorf("no visible memory")
		}
		if _, err := b.WriteScope(shown.Scope); err != nil {
			return fmt.Errorf("no visible memory")
		}
	}
	return nil
}

type sourceBoundary struct {
	client string
	root   string
	hash   string
	err    error
}

// Cache filesystem ancestry only in this process, never in the index. Native
// storage pools many scopes, so only non-native paths carry source provenance.
func (a *App) sourceClients(client string, all bool, own map[string]string) (map[int64]string, map[int64]string, error) {
	rows, err := a.DB.Query(`SELECT id,path,kind FROM sources`)
	if err != nil {
		return nil, nil, err
	}
	type source struct {
		id         int64
		path, kind string
	}
	var sources []source
	for rows.Next() {
		var s source
		if err := rows.Scan(&s.id, &s.path, &s.kind); err != nil {
			rows.Close()
			return nil, nil, err
		}
		sources = append(sources, s)
	}
	err = rows.Err()
	rows.Close()
	if err != nil {
		return nil, nil, err
	}
	a.sourceMu.Lock()
	defer a.sourceMu.Unlock()
	if a.sourceRoots == nil {
		a.sourceRoots = map[string]sourceBoundary{}
	}
	out := map[int64]string{}
	scopes := map[int64]string{}
	var cache *project.LegacyCache
	if client != "" {
		cache = project.LoadLegacyCache(config.DataDir())
	}
	for _, s := range sources {
		if s.kind == "native" {
			out[s.id] = "!native"
			continue
		}
		// Hash recognition depends on source kind, which can change between
		// refreshes without changing the unique source path.
		key := s.kind + "\x00" + s.path
		root, ok := a.sourceRoots[key]
		if !ok {
			root.client, root.root, root.err = project.SourceClientRoot(s.path)
			if root.client != "" {
				root.hash = project.LegacyStoreHash(s.path, s.kind)
			}
			a.sourceRoots[key] = root
		}
		if root.err != nil {
			out[s.id] = "!invalid"
		} else {
			out[s.id] = root.client
			if root.client != "" && root.client == client {
				scopes[s.id], _ = project.ClientSourceScope(cache, s.path, s.kind, root.client, root.root, root.hash, all, own)
			}
		}
	}
	return out, scopes, nil
}

// ScopeFor overlays a physical client's scope without changing stored ids,
// scope or origin. Unmarked non-native sources have no client provenance.
func (b Binding) ScopeFor(u store.Unit) (string, bool) {
	if b.Err != nil {
		return "", false
	}
	if b.sourceClients != nil {
		client, known := b.sourceClients[u.SourceID]
		if !known {
			return "", false
		}
		if client == "!native" {
			return u.Scope, true
		}
		if client == "!invalid" || client != b.Client {
			return "", false
		}
		if client != "" {
			scope := b.sourceScopes[u.SourceID]
			return scope, scope != ""
		}
	}
	return u.Scope, true
}
func (b Binding) Present(u store.Unit) (store.Unit, bool) {
	scope, ok := b.ScopeFor(u)
	u.Scope = scope
	return u, ok && b.Allows(scope)
}

// ScopeExpr and SQL are the SQL equivalent of ScopeFor + Allows, used by
// list, stats and injection history so no surface maintains a second policy.
func (b Binding) ScopeExpr(scopeCol, sourceCol string) string {
	var ids []int64
	for id, c := range b.sourceClients {
		if c != "" && c != "!invalid" && c != "!native" {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if len(ids) == 0 {
		return scopeCol
	}
	var out strings.Builder
	out.WriteString("CASE " + sourceCol)
	for _, id := range ids {
		scope := b.sourceScopes[id]
		if scope == "" {
			scope = "!unresolved-legacy"
		}
		fmt.Fprintf(&out, " WHEN %d THEN '%s'", id, strings.ReplaceAll(scope, "'", "''"))
	}
	out.WriteString(" ELSE " + scopeCol + " END")
	return out.String()
}
func (b Binding) SQL(scopeCol, sourceCol string) (string, []any) {
	if b.Err != nil {
		return "0", nil
	}
	where, args := search.ScopeSQL(b.ScopeExpr(scopeCol, sourceCol), b.Scopes)
	// A concurrent indexer may register a source after this snapshot. Unknown
	// provenance is invisible until the next refresh, just like ScopeFor.
	if b.sourceClients != nil {
		var known []string
		for id := range b.sourceClients {
			known = append(known, strconv.FormatInt(id, 10))
		}
		if len(known) == 0 {
			return "0", nil
		}
		sort.Strings(known)
		where = "(" + where + ") AND " + sourceCol + " IN (" + strings.Join(known, ",") + ")"
	}
	var hidden []int64
	for id, c := range b.sourceClients {
		if c != "!native" && (c == "!invalid" || c != b.Client) {
			hidden = append(hidden, id)
		}
	}
	sort.Slice(hidden, func(i, j int) bool { return hidden[i] < hidden[j] })
	if len(hidden) > 0 {
		var ids []string
		for _, id := range hidden {
			ids = append(ids, strconv.FormatInt(id, 10))
		}
		where = "(" + where + ") AND " + sourceCol + " NOT IN (" + strings.Join(ids, ",") + ")"
	}
	return where, args
}
func (b Binding) Remember(ctx context.Context, a *App, content, category, scope string, tags []string) (store.Unit, bool, error) {
	return b.RememberSafe(ctx, a, content, category, scope, tags, "")
}

// SecretCheckError asks the writer to judge text that looked secret. It is a
// question, not a verdict: storing again with a safe reason proceeds.
type SecretCheckError struct{ Hint string }

func (e *SecretCheckError) Error() string {
	return NotStored + " this looks like it may contain " + e.Hint + ". Are you sure there are no secrets here? " +
		"If it is safe (for example a browser key that ships in public pages, a secrets.NAME reference, a placeholder, or a URL meant to be shared), " +
		"store it again with a reason saying why (CLI --safe-reason, MCP safe_reason). Otherwise remove the secret and store the rest."
}

// RememberSafe stores a memory. When the cheap secret check fires, the write
// needs safeReason: the writer's judgment that the text is safe to keep.
func (b Binding) RememberSafe(ctx context.Context, a *App, content, category, scope string, tags []string, safeReason string) (store.Unit, bool, error) {
	return b.RememberPlaced(ctx, a, content, category, scope, tags, safeReason, "", "")
}

// RememberPlaced accepts a session-relative place alongside an optional literal
// scope. Both advisory checks ask the writer, never silently reclassify text.
func (b Binding) RememberPlaced(ctx context.Context, a *App, content, category, scope string, tags []string, safeReason, place, placeReason string, decisions ...RememberDecision) (store.Unit, bool, error) {
	// The text is embedded before the write turn (t0qap9): the duplicate
	// check searches with that vector, so no embed runs under the turn.
	ctx = a.embedForDedupe(ctx, strings.TrimSpace(content))
	// One write turn for the whole flow: duplicate check, save, index.
	ctx, release, err := a.WriteTurn(ctx)
	if err != nil {
		return store.Unit{}, false, err
	}
	defer release()
	b = b.Refresh(a)
	safeReason = strings.TrimSpace(safeReason)
	if hint := project.SecretHint(content); hint != "" && safeReason == "" {
		return store.Unit{}, false, &SecretCheckError{Hint: hint}
	} else if hint == "" {
		safeReason = ""
	}
	destination, err := b.Placement(scope, place)
	if err != nil {
		return store.Unit{}, false, err
	}
	scope = destination.Scope
	if hint := b.PlacementHint(content, scope); hint != "" && strings.TrimSpace(placeReason) == "" {
		return store.Unit{}, false, &PlaceCheckError{Hint: hint}
	}
	if category == "" {
		category = "fact"
	}
	var decision RememberDecision
	if len(decisions) > 0 {
		decision = decisions[0]
	}
	// The condition is checked before anything is written; a malformed one
	// fails with its fix: line (ez50rb).
	cond, err := holds.Parse(decision.Holds)
	if err != nil {
		return store.Unit{}, false, err
	}
	// So is the trigger (jrj1vq): canonical, within its limits.
	tr, err := trigger.Canonical(decision.Trigger)
	if err != nil {
		return store.Unit{}, false, err
	}
	if decision.ClearTrigger && tr != nil {
		return store.Unit{}, false, fmt.Errorf("a trigger cannot be stated and cleared at once\nfix: leave out --no-trigger, or leave out --when, --when-not and --case-*")
	}
	if decision.ClearTrigger && strings.TrimSpace(decision.Supersedes) == "" {
		return store.Unit{}, false, fmt.Errorf("clearing a trigger corrects a memory: name it\nfix: hug mem remember --supersedes ID --no-trigger \"TEXT\" (hug mem search WORDS finds the id)")
	}
	hits, err := b.rememberMatches(ctx, a, strings.TrimSpace(content))
	if err != nil {
		return store.Unit{}, false, err
	}
	if decision.Reinforce != "" {
		u, err := b.Resolve(a, decision.Reinforce)
		if err != nil || !u.Active {
			return store.Unit{}, false, fmt.Errorf("no visible active memory")
		}
		if _, err = b.WriteScope(u.Scope); err != nil {
			return store.Unit{}, false, err
		}
		if err = fillTrigger(u.SID, tr); err != nil {
			return store.Unit{}, false, err
		}
		score := 0.0
		for _, h := range hits {
			if h.Unit.SID == u.SID {
				score = h.Semantic
			}
		}
		got, re, err := b.reinforceMemory(ctx, a, u, score, "writer confirmed same fact")
		if err == nil {
			recordFixes(ctx, a, "reinforce", []fixTarget{a.fixTargetFor(u, "")})
		}
		return got, re, err
	}
	// supersedes names the memory this one corrects (rjk44w): checked before
	// anything is written, recorded after; it retires nothing. When that
	// memory is the near-duplicate, naming it answers the duplicate question.
	var superseded store.Unit
	if ref := strings.TrimSpace(decision.Supersedes); ref != "" {
		if superseded, err = b.Resolve(a, ref); err != nil {
			return store.Unit{}, false, fmt.Errorf("supersedes %s: no visible memory\nfix: hug mem search WORDS to find its id, or leave out --supersedes", ref)
		}
		// The same text with a new trigger, or none (jrj1vq): the note is
		// unchanged, so the memory keeps its id and only its trigger is
		// replaced; the trail records the supersede on itself.
		if superseded.Body == strings.TrimSpace(content) && (tr != nil || decision.ClearTrigger) {
			return b.replaceTrigger(ctx, a, superseded, tr)
		}
	}
	automatic, review := a.Cfg.Extract.DedupeThresholds()
	if len(hits) > 0 {
		h := hits[0]
		if h.Semantic >= automatic {
			if err := fillTrigger(h.Unit.SID, tr); err != nil {
				return store.Unit{}, false, err
			}
			return b.reinforceMemory(ctx, a, h.Unit, h.Semantic, "cosine meets dedupe threshold")
		}
		if h.Semantic >= review && strings.TrimSpace(decision.NewReason) == "" && (superseded.SID == "" || h.Unit.SID != superseded.SID) {
			return store.Unit{}, false, &DuplicateCheckError{SID: h.Unit.SID, Text: h.Unit.Body, Score: h.Semantic}
		}
	}
	branch, repo := b.workBranch(), ""
	if branch != "" || cond.Text != "" {
		repo = b.workRepo() // only then: a git lookup per remember is not free
	}
	m, reinforced, err := ingest.RememberWith(NativePath(), ingest.Remembering{Content: content, Category: category, Type: NormalType(decision.Type), Scope: scope, Tags: tags,
		SafeReason: safeReason, NewReason: decision.NewReason, Branch: branch, Holds: cond.Text, Repo: repo, Trigger: tr,
		OriginFor: project.OriginForScope(scope)})
	if err != nil {
		return store.Unit{}, false, err
	}
	if _, err := a.Reindex(ctx); err != nil {
		return store.Unit{}, false, err
	}
	u, err := b.Resolve(a, m.ID)
	if err != nil {
		return u, reinforced, err
	}
	if err := rememberHeld(m, u); err != nil {
		return u, reinforced, err
	}
	if superseded.SID != "" {
		recordFixes(ctx, a, "supersede", []fixTarget{a.fixTargetFor(superseded, u.SID)})
	}
	return u, reinforced, nil
}

// fillTrigger is the trigger side of a reinforce (jrj1vq): no trigger stated
// keeps the memory's; one stated fills a memory that has none; a different
// one is refused (crab: a changed trigger is a correction, --supersedes).
func fillTrigger(sid string, tr *trigger.Trigger) error {
	if tr == nil {
		return nil
	}
	mems, err := ingest.LoadNative(NativePath())
	if err != nil {
		return err
	}
	for _, m := range mems {
		if m.SID != sid || !m.Active {
			continue
		}
		if trigger.Equal(m.Trigger, tr) {
			return nil
		}
		if !trigger.Empty(m.Trigger) {
			return &ingest.TriggerChangeError{ID: m.ID, SID: sid, Old: m.Trigger, New: tr}
		}
		_, _, err := ingest.ReplaceTrigger(NativePath(), m.ID, tr)
		return err
	}
	return notOwnedTrigger(sid)
}

func notOwnedTrigger(sid string) error {
	return fmt.Errorf("NOT STORED: memory %s comes from another memory system, so hug cannot keep a trigger on it\nfix: hug mem remember --supersedes %s --new-reason \"with a trigger\" --when ... \"TEXT\" (a hug memory that corrects it)", sid, sid)
}

// replaceTrigger is a supersede whose text is the memory's own: the trigger
// is replaced in place (tr nil stores None, a clear sync carries), the id
// stays, and the fix trail records the supersede on itself (jrj1vq).
func (b Binding) replaceTrigger(ctx context.Context, a *App, old store.Unit, tr *trigger.Trigger) (store.Unit, bool, error) {
	if _, err := b.WriteScope(old.Scope); err != nil {
		return store.Unit{}, false, err
	}
	_, found, err := ingest.ReplaceTrigger(NativePath(), old.SID, tr)
	if err != nil {
		return store.Unit{}, false, err
	}
	if !found {
		return store.Unit{}, false, notOwnedTrigger(old.SID)
	}
	if _, err := a.Reindex(ctx); err != nil {
		return store.Unit{}, false, err
	}
	u, err := b.Resolve(a, old.SID)
	if err != nil {
		return u, false, err
	}
	recordFixes(ctx, a, "supersede", []fixTarget{a.fixTargetFor(old, old.SID)})
	return u, false, nil
}

// fixTargetFor is the trail's view of u: its sid, hash and every unit row
// that carries its sid (twins).
func (a *App) fixTargetFor(u store.Unit, newSID string) fixTarget {
	t := fixTarget{sid: u.SID, newSID: newSID, hash: u.ContentHash, ids: []int64{u.ID}}
	if units, err := a.DB.ResolveAll(u.SID); err == nil {
		t.ids = t.ids[:0]
		for _, x := range units {
			t.ids = append(t.ids, x.ID)
		}
	}
	return t
}

// rememberHeld reads a remember back: the memory the store holds now is
// active and has the short id remember prints (tn8bmn).
func rememberHeld(m ingest.NativeMemory, u store.Unit) error {
	WriteLanded()
	mems, err := loadLocked()
	if err != nil {
		return err
	}
	for _, got := range mems {
		if got.ID != m.ID {
			continue
		}
		if !got.Active || got.SID != u.SID {
			return fmt.Errorf("remember did not hold: the store has %s (active=%v), not %s as written\nfix: hug mem show %s", got.SID, got.Active, u.SID, got.SID)
		}
		return nil
	}
	return fmt.Errorf("remember did not hold: memory %s is not in the store after the write\nfix: hug mem remember again, then hug mem show ID", u.SID)
}

// globalFolderStores names the jcode@global:<hash> stores that belong to a
// session in cwd: the folder itself, and inside a git project every folder up
// to its top level, so a subfolder session still sees the project root's
// store. (Hashes cannot be inverted, so a subfolder's store shows only from
// that subfolder down; tidy can move it to the project.) Outside git only the
// exact folder counts, so a home or parent-folder store never leaks into
// child folders.
// Paths are tried as given, with symlinks resolved, and in the literal "~/"
// form some jcode sessions recorded.
func globalFolderStores(cwd string) map[string]bool {
	own := map[string]bool{}
	if cwd == "" {
		return own
	}
	home, _ := os.UserHomeDir()
	add := func(dir string) {
		own[harness.JcodeProjectHash(dir)] = true
		if home != "" {
			if rel, err := filepath.Rel(home, dir); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
				own[harness.JcodeProjectHash("~/"+rel)] = true
			}
		}
	}
	dirs := []string{filepath.Clean(cwd)}
	if phys, err := filepath.EvalSymlinks(cwd); err == nil && phys != dirs[0] {
		dirs = append(dirs, phys)
	}
	top := ""
	if p, err := project.Find(cwd, false); err == nil && p.Root != "" {
		top = filepath.Clean(p.Root)
	}
	for _, d := range dirs {
		add(d)
		if top == "" {
			continue
		}
		for anc := d; ; anc = filepath.Dir(anc) {
			add(anc)
			if anc == top || filepath.Dir(anc) == anc {
				break
			}
			if rel, err := filepath.Rel(top, anc); err != nil || strings.HasPrefix(rel, "..") {
				break
			}
		}
	}
	return own
}

// FolderOwns reports whether a session in cwd sees a folder:<dir> memory: the
// session is in dir itself, or below it. The home folder and workspace folders
// (two or more git repositories directly inside, such as ~/Projects) are parents,
// not projects, so their memories show only in exactly that folder.
func FolderOwns(dir, cwd string) bool {
	if dir == "" || cwd == "" {
		return false
	}
	dir = filepath.Clean(dir)
	cwds := []string{filepath.Clean(cwd)}
	if phys, err := filepath.EvalSymlinks(cwd); err == nil && phys != cwds[0] {
		cwds = append(cwds, phys)
	}
	for _, c := range cwds {
		if c == dir {
			return true
		}
	}
	if isParentFolder(dir) {
		return false
	}
	for _, c := range cwds {
		if strings.HasPrefix(c, dir+string(filepath.Separator)) {
			return true
		}
	}
	return false
}

func isParentFolder(dir string) bool {
	if home, err := os.UserHomeDir(); err == nil && filepath.Clean(home) == dir {
		return true
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return false
	}
	repos := 0
	for _, e := range entries {
		if e.IsDir() {
			if _, err := os.Stat(filepath.Join(dir, e.Name(), ".git")); err == nil {
				if repos++; repos >= 2 {
					return true
				}
			}
		}
	}
	return false
}

// shownInSession reports whether recall showed this unit to any of the given
// harness sessions ("jcode:<id>", "mcp:<id>", ...), per the injections log.
func (a *App) shownInSession(unitID int64, sessions []string) bool {
	for _, s := range sessions {
		if s == "" || strings.HasSuffix(s, ":") {
			continue
		}
		var one int
		if a.DB.QueryRow(`SELECT 1 FROM injections WHERE session=? AND unit_id=? LIMIT 1`, s, unitID).Scan(&one) == nil {
			return true
		}
	}
	return false
}

// HarnessSessions lists the session keys this process may belong to, from the
// environment each harness sets (jcode: JCODE_SESSION_ID; Claude Code:
// CLAUDE_SESSION_ID; Codex: CODEX_SESSION_ID). Used to honour forget for
// memories recalled into the calling session.
func HarnessSessions() []string {
	var out []string
	for env, harness := range map[string]string{"JCODE_SESSION_ID": "jcode", "CLAUDE_SESSION_ID": "claude", "CODEX_SESSION_ID": "codex"} {
		if v := strings.TrimSpace(os.Getenv(env)); v != "" {
			out = append(out, harness+":"+v)
		}
	}
	return out
}
