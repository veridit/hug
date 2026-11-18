package app

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/veridit/hug/muninn/internal/extract"
	"github.com/veridit/hug/muninn/internal/harness"
	"github.com/veridit/hug/muninn/internal/project"
)

// Place is a session-relative destination. Scope never comes from model output.
type Place struct {
	Name    string `json:"name"`
	Scope   string `json:"scope"`
	Meaning string `json:"meaning"`
}

// placementDir resolves the longest existing ancestor, including a future
// session directory beneath a symlink into a client root.
func placementDir(dir string) (string, error) {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return "", err
	}
	var tail []string
	for d := abs; ; d = filepath.Dir(d) {
		physical, err := filepath.EvalSymlinks(d)
		if err == nil {
			for i := len(tail) - 1; i >= 0; i-- {
				physical = filepath.Join(physical, tail[i])
			}
			return physical, nil
		}
		if !errors.Is(err, os.ErrNotExist) || filepath.Dir(d) == d {
			return "", err
		}
		tail = append(tail, filepath.Base(d))
	}
}

// placementScopes derives the three aliases before duplicate destinations collapse.
// It never creates a project marker or modifies a harness store.
func (b Binding) placementScopes() (here, client, global string) {
	global = b.DefaultScope
	if global == "" {
		global = "global"
	}
	here = global
	if b.Client != "" {
		client = "client:" + b.Client
		global = client
		here = client
	}
	if b.context == nil || b.context.cwd == "" {
		return
	}
	cwd, err := placementDir(b.context.cwd)
	if err != nil {
		return
	}
	if b.Client != "" {
		_, root, err := project.ClientRoot(cwd)
		if err != nil || cwd == root {
			return
		}
		if p, err := project.Find(cwd, false); err == nil && p.Root != root && FolderOwns(p.Root, cwd) {
			here = p.Scope()
		} else {
			here = client + ":folder:" + cwd
		}
		return
	}
	if p, err := project.Find(cwd, false); err == nil && FolderOwns(p.Root, cwd) {
		here = p.Scope()
	} else {
		here = "folder:" + cwd
		if _, err := b.WriteScope(here); err != nil {
			here = global + ":" + harness.JcodeProjectHash(cwd)
		}
	}
	return
}

// Places is the shared inventory for capture, MCP, CLI and web. Aliases with the
// same scope collapse to the narrowest name. Client global is never user global.
func (b Binding) Places() []Place {
	if b.Err != nil {
		return nil
	}
	return b.places(b.placementScopes())
}

// places is Places for aliases already derived: a caller that needs both
// the aliases and the inventory looks the project up once (ce3dmq).
func (b Binding) places(here, client, global string) []Place {
	candidates := []Place{{"here", here, "this project or folder; code, repo, paths, tickets and deployment"}}
	if client != "" {
		candidates = append(candidates, Place{"client", client, "facts shared across this client's projects, servers, people and processes"})
	}
	meaning := "the user, preferences and tools across all their work"
	if client != "" {
		meaning = "this client only, never the user's personal global"
	}
	candidates = append(candidates, Place{"global", global, meaning})
	seen := map[string]bool{}
	var out []Place
	for _, p := range candidates {
		if seen[p.Scope] {
			continue
		}
		if _, err := b.WriteScope(p.Scope); err != nil {
			continue
		}
		seen[p.Scope] = true
		out = append(out, p)
	}
	return out
}

func (b Binding) extractionPlaces() []extract.Place {
	return b.extractionPlacesFor(false)
}

// extractionPlacesFor adds the wide places (rnpdnm A+) when wide is on and
// this binding may write the personal scope: never in a client session,
// where personal is out of reach and the list stays as before.
func (b Binding) extractionPlacesFor(wide bool) []extract.Place {
	var out []extract.Place
	for _, p := range b.Places() {
		out = append(out, extract.Place{Name: p.Name, Meaning: p.Meaning})
	}
	if wide && b.personalWritable() {
		out = append(out, extract.WidePlaces...)
	}
	return out
}

// PersonalScope is the person's own scope: local, never synced to a
// repository (project.Sync writes only the project's scope).
const PersonalScope = "personal"

func (b Binding) personalWritable() bool {
	if b.Client != "" {
		return false
	}
	_, err := b.WriteScope(PersonalScope)
	return err == nil
}

// capturePlacement is Placement for a capture candidate. With wide places
// on, personal and other land in the personal scope (crab, rnpdnm: never
// written into another project's scope automatically; hug mem tidy place
// moves them deliberately, and other's about:NAME tag makes that cheap).
// Off, or where personal is out of reach, it is Placement as before.
func (b Binding) capturePlacement(name string, wide bool) (Place, error) {
	if wide && (name == extract.PersonalPlace || name == extract.OtherPlace) && b.personalWritable() {
		return Place{Name: name, Scope: PersonalScope, Meaning: "the person's own memory, never shared with a repository"}, nil
	}
	return b.Placement("", name)
}

// AboutTag is the tag an "other" capture carries: the project, client or
// product the extractor said it is about.
func AboutTag(name string) string { return "about:" + name }

// Placement maps missing, unavailable and unknown names to here. Literal scopes
// retain the binding's existing owner policy and take precedence over place.
func (b Binding) Placement(scope, name string) (Place, error) {
	if scope == "" {
		here, client, global := b.placementScopes()
		scope = here
		switch name {
		case "client":
			if client != "" {
				scope = client
			}
		case "global":
			scope = global
		}
		scope, err := b.WriteScope(scope)
		if err != nil {
			return Place{}, err
		}
		return placeIn(b.places(here, client, global), scope), nil
	}
	scope, err := b.WriteScope(scope)
	if err != nil {
		return Place{}, err
	}
	return b.PlaceForScope(scope), nil
}

func (b Binding) PlaceForScope(scope string) Place {
	return placeIn(b.Places(), scope)
}

func placeIn(places []Place, scope string) Place {
	for _, p := range places {
		if p.Scope == scope {
			return p
		}
	}
	return Place{Name: "scope", Scope: scope, Meaning: "explicit literal scope"}
}

// PlaceCheckError is a question, not a classifier. A writer's reason overrides it.
type PlaceCheckError struct{ Hint string }

func (e *PlaceCheckError) Error() string {
	return fmt.Sprintf(NotStored+" this mentions %s from this project. Are you sure it applies beyond it? Store again with a reason (CLI --place-reason, MCP/web place_reason), or use place here.", e.Hint)
}

var placementPath = regexp.MustCompile(`[[:alnum:]_.-]+(/[[:alnum:]_.-]+)+`)

// PlacementHint intentionally uses cheap, local anchors only. No model, corpus
// scan, ticket-prefix guess, recursive walk or project marker creation is needed.
func (b Binding) PlacementHint(content, scope string) string {
	here, client, global := b.placementScopes()
	if scope == here || (scope != client && scope != global) || b.context == nil {
		return ""
	}
	root := b.context.cwd
	if p, err := project.Find(root, false); err == nil && FolderOwns(p.Root, root) {
		root = p.Root
	}
	root = filepath.Clean(root)
	if root == "." || root == string(filepath.Separator) {
		return ""
	}
	paths := []string{root}
	if phys, err := filepath.EvalSymlinks(root); err == nil && phys != root {
		paths = append(paths, phys)
	}
	if home, err := os.UserHomeDir(); err == nil && strings.HasPrefix(root, home+string(filepath.Separator)) {
		paths = append(paths, "~/"+strings.TrimPrefix(root, home+string(filepath.Separator)))
	}
	for _, p := range paths {
		if strings.Contains(content, p) {
			return p
		}
	}
	// Recognize a bounded number of existing relative paths, without scanning
	// the repository or guessing ticket ownership from the text itself.
	for _, path := range placementPath.FindAllString(content, 16) {
		clean := filepath.Clean(path)
		if clean == ".." || strings.HasPrefix(clean, "../") {
			continue
		}
		if info, err := os.Stat(filepath.Join(root, clean)); err == nil && !info.IsDir() {
			return path
		}
	}
	name := filepath.Base(root)
	if len(name) >= 3 {
		re := regexp.MustCompile(`(?i)(^|[^\pL\pN_-])` + regexp.QuoteMeta(name) + `($|[^\pL\pN_-])`)
		if re.MatchString(content) {
			return name
		}
	}
	return ""
}
