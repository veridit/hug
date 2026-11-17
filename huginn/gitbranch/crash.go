package gitbranch

import (
	"errors"
	"os/exec"
	"runtime"
)

// Crashed reports a git process that did not exit on its own: killed by a
// signal (SIGSEGV, SIGKILL...) or, on Windows, ended by an exception
// (an NTSTATUS exit code such as 0xC0000005, access violation). Its output
// says nothing about the repository, so callers report the crash itself
// rather than read it as an answer (xv39b9, 3z65k1).
func Crashed(err error) bool {
	var e *exec.ExitError
	if !errors.As(err, &e) || e.ProcessState == nil {
		return false
	}
	if runtime.GOOS == "windows" {
		return uint32(e.ExitCode()) >= 0xC0000000
	}
	return !e.Exited()
}
