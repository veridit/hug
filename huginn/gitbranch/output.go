package gitbranch

import (
	"bytes"
	"io"
	"os"
	"os/exec"
)

// output runs c with its stdout and stderr in temporary files, not pipes,
// and returns both once git has exited (ad62aa).
//
// With pipes, os/exec copies git's output in goroutines and, after git
// exits, gives them WaitDelay to finish; at load 300-700 a niced process's
// goroutines did not run within those 2 s, and a git that exited 0 came back
// as "exec: WaitDelay expired before I/O complete" with its output dropped:
// a commit that landed was reported failed, a lost rev-parse read as "not a
// Git checkout". A file is written by git itself, so nothing in this process
// has to run before git's answer is complete, and no process that outlives
// git (ssh, a hook's background job) can hold the answer open.
func output(c *exec.Cmd) (stdout, stderr []byte, err error) {
	out, e := os.CreateTemp("", "hug-git-out-")
	if e != nil {
		return piped(c)
	}
	defer os.Remove(out.Name())
	defer out.Close()
	errf, e := os.CreateTemp("", "hug-git-err-")
	if e != nil {
		return piped(c)
	}
	defer os.Remove(errf.Name())
	defer errf.Close()
	c.Stdout, c.Stderr = out, errf
	err = c.Run()
	stdout, e = readAll(out)
	if e != nil && err == nil {
		err = e
	}
	stderr, _ = readAll(errf)
	return stdout, stderr, err
}

// readAll reads f from its start: git shared the descriptor, so its offset
// is where git stopped writing.
func readAll(f *os.File) ([]byte, error) {
	if _, e := f.Seek(0, io.SeekStart); e != nil {
		return nil, e
	}
	return io.ReadAll(f)
}

// piped is the old way, through pipes, for a machine where no temporary
// file can be made (TMPDIR missing or read-only): git still runs, and only
// the load case above can fail.
func piped(c *exec.Cmd) (stdout, stderr []byte, err error) {
	var b bytes.Buffer
	c.Stderr = &b
	stdout, err = c.Output()
	return stdout, b.Bytes(), err
}
