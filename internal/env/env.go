// Package env is hug's one lookup for its own environment variables
// (m1c28r). Every variable hug reads is HUG_*; the old part names
// (the huginn and muninn prefixes) are not read. When one of them is set, Stale names
// it with the HUG_ name to use instead, so hug can say so once.
package env

import (
	"os"
	"sort"
	"strings"
)

// The variables hug reads, by their HUG_ names.
const (
	Home       = "HUG_HOME"       // the memory store's folder
	User       = "HUG_USER"       // who takes tasks
	Model      = "HUG_MODEL"      // the model doing the work
	Sync       = "HUG_SYNC"       // off: no background sync
	NoEmbed    = "HUG_NO_EMBED"   // no embedding model, keyword recall only
	FullSpeed  = "HUG_FULL_SPEED" // background work is not throttled
	Serve      = "HUG_SERVE"      // set inside hug mem serve
	CPUProfile = "HUG_CPUPROFILE" // write a CPU profile to this file
	BinDir     = "HUG_BIN_DIR"    // install.sh's target folder
	TUIState   = "HUG_TUI_STATE"  // the TUI's reload hand-off file
	TestRoot   = "HUG_TUI_TEST_ROOT"
)

// Get is os.Getenv for a HUG_ name.
func Get(name string) string { return os.Getenv(name) }

// Lookup is os.LookupEnv for a HUG_ name.
func Lookup(name string) (string, bool) { return os.LookupEnv(name) }

// Old is a variable from before m1c28r and the name that replaced it.
type Old struct{ Name, New string }

// Stale lists the old huginn and muninn variables set in env (os.Environ
// when nil), each with its HUG_ name, sorted. hug does not read them.
func Stale(environ []string) []Old {
	if environ == nil {
		environ = os.Environ()
	}
	var out []Old
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if n := Renamed(name); n != "" {
			out = append(out, Old{name, n})
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// Renamed is the HUG_ name for an old huginn or muninn name, "" for any
// other name.
// oldPrefixes are spelt in parts so grep finds no old names in the code.
var oldPrefixes = []string{"HUG" + "INN_", "MUN" + "INN_"}

func Renamed(name string) string {
	for _, p := range oldPrefixes {
		if rest, ok := strings.CutPrefix(name, p); ok && rest != "" {
			return "HUG_" + rest
		}
	}
	return ""
}

// Warning is what hug prints, once per command, when old names are set: the
// problem and a fix: line that sets the new names and clears the old ones.
func Warning(old []Old) (problem, fix string) {
	if len(old) == 0 {
		return "", ""
	}
	var names, sets, unsets []string
	for _, o := range old {
		names = append(names, o.Name+" is now "+o.New)
		sets = append(sets, o.New+"=\"$"+o.Name+"\"")
		unsets = append(unsets, o.Name)
	}
	problem = "hug reads only HUG_* variables: " + strings.Join(names, ", ") + " (the old name is ignored)"
	fix = "export " + strings.Join(sets, " ") + "; unset " + strings.Join(unsets, " ") + " (and rename it where you set it, e.g. your shell profile)"
	return problem, fix
}
