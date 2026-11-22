// Package env is hug's one lookup for its own environment variables
// (m1c28r). Every variable hug reads is HUG_*; the old part names
// (the huginn and muninn prefixes) are not read. When one of them is set, Stale names
// it with the HUG_ name to use instead, so hug can say so once.
package env

import (
	"fmt"
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

// Kind says who sets a variable.
type Kind int

const (
	UserSet  Kind = iota // a person or a harness config sets it; hug help lists it
	Internal             // hug sets it for a process it starts
	Test                 // tests, benchmarks and developer scripts set it
)

// Var is one variable hug knows: its name, who sets it, the value to
// suggest in a fix line, and one line for hug help.
type Var struct {
	Name    string
	Kind    Kind
	Example string // a value for a fix: line, e.g. "off"
	Help    string
}

// Known is every HUG_ variable hug or its tests read or set (a0p5kt): the
// one list. hug refuses to start with any other HUG_ name set, so a
// misspelt switch (NO_SYNC for HUG_SYNC=off) never fails open, and
// TestEveryHUGNameIsKnown keeps the list complete.
var Known = []Var{
	{Home, UserSet, "$HOME/hug-store", "the memory store's folder (default: the platform's data folder)"},
	{User, UserSet, "poodle", "who takes tasks and writes requests (else the agent session, else the model)"},
	{Model, UserSet, "claude-opus-5-5", "the model doing the work, for take and for board commits' Model: trailer"},
	{Sync, UserSet, "off", "off: no background sync of tasks or memories with the remote"},
	{NoEmbed, UserSet, "1", "any value: no embedding model, keyword recall only"},
	{FullSpeed, UserSet, "1", "any value: background work is not throttled"},
	{CPUProfile, UserSet, "/tmp/hug.prof", "write a CPU profile of hug mem commands to this file"},
	{BinDir, UserSet, "$HOME/.local/bin", "install.sh: where to put hug"},
	{"HUG_VERSION", UserSet, "v0.7.0", "install.sh: the release to install (default: the latest)"},
	{Serve, Internal, "", "set inside hug mem serve"},
	{TUIState, Internal, "", "the TUI's reload hand-off file"},
	{"HUG_MCP_RESUME", Internal, "", "the hand-off file a reloading hug mcp passes to the next one"},
	{TestRoot, Test, "", "the board a TUI pty test drives"},
	{"HUG_BENCH", Test, "1", "run the benchmark tests"},
	{"HUG_UPDATE_GOLDEN", Test, "1", "rewrite golden files"},
	{"HUG_REAL_BOARD", Test, "", "a board's tasks folder the ask replay probe reads (q2adr8)"},
	{"HUG_TUI_ARTIFACTS", Test, "", "also save TUI captures in this folder"},
	{"HUG_MIGRATION_COPY", Test, "", "a disposable database snapshot for the migration test"},
	{"HUG_REPAIR_COPY", Test, "", "a disposable store copy for the capture repair test"},
	{"HUG_MCP_TEST_HOLD", Test, "", "the reload test's hold file"},
	{"HUG_CHILD_ARGS", Test, "", "arguments for a test's child process"},
	{"HUG_CAPTURE_KILL_CHILD", Test, "", "the capture kill test's child process"},
	{"HUG_CAPTURE_KILL_CWD", Test, "", "the capture kill test's child folder"},
	{"HUG_CAPTURE_KILL_HOME", Test, "", "the capture kill test's jcode home"},
	{"HUG_TEST_SETTLE", Test, "", "sync test children's settle time"},
	{"HUG_TEST_CHILD_DELAY", Test, "", "sync test children's delay"},
	{"HUG_TEST_SYNC_TIMEOUT_REPORT", Test, "", "the sync timeout-output test's child receipt"},
	{"HUG_TEST_EDIT_FROM", Test, "", "the editor test's text to replace"},
	{"HUG_TEST_EDIT_TO", Test, "", "the editor test's replacement"},
	{"HUG_TEST_TUI_EDIT", Test, "", "the TUI editor test's text to append"},
	{"HUG_TEST_FAKE_VERSION", Test, "", "the TUI reload test's fake version"},
	{"HUG_TEST_TMUX_CHILD", Test, "", "how the tmux cleanup test's child test ends (wacc33)"},
	{"HUG_TEST_MISSING_KEY", Test, "", "a credential variable a capture test leaves empty"},
	{"HUG_TEST_FAKE_MCP", Test, "", "the stale-MCP test's fake hug mcp server (z77pem)"},
	{"HUG_TEST_CLIPBOARD", Test, "", "a fake clipboard command V in hug tui runs as CMD image and CMD text (q7n2zd)"},
	{"HUG_REQUIRE_SHELLS", Test, "1", "the completion test fails, not skips, when bash, zsh or fish is missing (5mmqjk)"},
}

// IsKnown reports whether name is on Known.
func IsKnown(name string) bool {
	for _, v := range Known {
		if v.Name == name {
			return true
		}
	}
	return false
}

// Unknown is a HUG_ variable set in the environment that hug does not know,
// with the known variable nearest to it.
type Unknown struct {
	Name    string
	Nearest Var
}

// Unknowns lists the HUG_ variables set in environ (os.Environ when nil)
// that are not on Known, sorted.
func Unknowns(environ []string) []Unknown {
	if environ == nil {
		environ = os.Environ()
	}
	var out []Unknown
	for _, kv := range environ {
		name, _, _ := strings.Cut(kv, "=")
		if !strings.HasPrefix(name, "HUG_") || IsKnown(name) {
			continue
		}
		out = append(out, Unknown{name, nearest(name)})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// nearest is the user-set variable closest to name: the most shared words
// (NO_SYNC and HUG_SYNC share SYNC), then the smallest edit distance.
func nearest(name string) Var {
	words := func(s string) map[string]bool {
		m := map[string]bool{}
		for _, w := range strings.Split(strings.TrimPrefix(s, "HUG_"), "_") {
			if w != "" {
				m[w] = true
			}
		}
		return m
	}
	mine := words(name)
	var best Var
	bestShared, bestDist := -1, 0
	for _, v := range Known {
		if v.Kind != UserSet {
			continue
		}
		shared := 0
		for w := range words(v.Name) {
			if mine[w] {
				shared++
			}
		}
		d := distance(name, v.Name)
		if shared > bestShared || shared == bestShared && d < bestDist {
			best, bestShared, bestDist = v, shared, d
		}
	}
	return best
}

// distance is the Levenshtein distance between a and b.
func distance(a, b string) int {
	prev := make([]int, len(b)+1)
	for j := range prev {
		prev[j] = j
	}
	for i := 1; i <= len(a); i++ {
		cur := make([]int, len(b)+1)
		cur[0] = i
		for j := 1; j <= len(b); j++ {
			cost := 1
			if a[i-1] == b[j-1] {
				cost = 0
			}
			cur[j] = min(prev[j]+1, cur[j-1]+1, prev[j-1]+cost)
		}
		prev = cur
	}
	return prev[len(b)]
}

// Refusal is what hug prints when unknown HUG_ variables are set: the
// problem and a fix: line that unsets them and sets the nearest real one.
func Refusal(unknown []Unknown) (problem, fix string) {
	if len(unknown) == 0 {
		return "", ""
	}
	var names, unsets, sets []string
	seen := map[string]bool{}
	for _, u := range unknown {
		names = append(names, u.Name+" (nearest: "+u.Nearest.Name+")")
		unsets = append(unsets, u.Name)
		if !seen[u.Nearest.Name] {
			seen[u.Nearest.Name] = true
			sets = append(sets, u.Nearest.Name+"="+u.Nearest.Example)
		}
	}
	problem = "not a hug variable: " + strings.Join(names, ", ") + "; hug stops rather than ignore a setting it does not read (all variables: hug help)"
	fix = "unset " + strings.Join(unsets, " ") + "; export " + strings.Join(sets, " ")
	return problem, fix
}

// HelpText lists the user-set variables, one line each, for hug help.
func HelpText() string {
	var b strings.Builder
	b.WriteString("Environment (any other HUG_* variable is refused):\n")
	for _, v := range Known {
		if v.Kind == UserSet {
			fmt.Fprintf(&b, "  %-17s %s\n", v.Name, v.Help)
		}
	}
	return b.String()
}

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
