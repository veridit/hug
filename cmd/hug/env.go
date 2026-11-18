package main

import (
	"fmt"
	"io"
	"slices"
	"strings"

	"github.com/veridit/hug/internal/env"
)

// warnStale prints which old variable names are set and the HUG_ names to
// use, with a fix: line (m1c28r). environ nil is os.Environ.
func warnStale(w io.Writer, environ []string) {
	problem, fix := env.Warning(env.Stale(environ))
	if problem == "" {
		return
	}
	fmt.Fprintln(w, "hug: "+problem)
	fmt.Fprintln(w, "fix: "+fix)
}

// refuseUnknown stops a command when a HUG_ variable hug does not know is
// set (a0p5kt): a misspelt switch, such as NO_SYNC=1 for HUG_SYNC=off,
// must not fail open. It prints the problem and a fix: line on w and
// reports whether to stop. environ nil is os.Environ.
func refuseUnknown(w io.Writer, environ []string, command string) bool {
	problem, fix := env.Refusal(env.Unknowns(environ))
	if problem == "" {
		return false
	}
	fmt.Fprintln(w, command+": "+problem)
	fmt.Fprintln(w, "fix: "+fix)
	return true
}

// commandName is how a refusal names the command: "hug mem forget".
func commandName(name string, args []string) string {
	words := []string{"hug"}
	switch strings.TrimSuffix(name, ".exe") {
	case "huginn":
		words = append(words, "task")
	case "muninn":
		words = append(words, "mem")
	}
	for _, a := range args {
		if strings.HasPrefix(a, "-") || len(words) == 3 {
			break
		}
		words = append(words, a)
	}
	return strings.Join(words, " ")
}

// isHook reports whether args run a harness hook (hug mem hook ..., hug mem
// capture --hook): a hook must never break the harness, so a refusal there
// does nothing and exits 0.
func isHook(name string, args []string) bool {
	if n := strings.TrimSuffix(name, ".exe"); n == "muninn" {
		args = append([]string{"mem"}, args...)
	} else if n == "huginn" {
		return false
	}
	if len(args) < 2 || (args[0] != "mem" && args[0] != "memory" && args[0] != "muninn") {
		return false
	}
	return args[1] == "hook" || args[1] == "capture" && slices.Contains(args[2:], "--hook")
}

// helpOrVersion reports whether args only ask for help or the version,
// which still answer with an unknown variable set: the refusal points to
// hug help.
func helpOrVersion(args []string) bool {
	if len(args) == 0 {
		return true
	}
	switch args[0] {
	case "help", "-h", "--help", "version", "--version":
		return true
	}
	return false
}
