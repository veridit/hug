//go:build !(darwin || linux || freebsd || netbsd || openbsd || dragonfly)

package main

import (
	"io"
	"os"
)

// idleStdin: here (Windows) stdin cannot be waited on without reading it, and
// there is no exec, so hug mcp reloads at the next request (mbdfzs).
func idleStdin(f *os.File) io.Reader { return f }
