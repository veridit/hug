package main

import (
	"fmt"
	"io"

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
