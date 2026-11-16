package cli

import (
	"fmt"
	"os"
	"strings"
	"testing"
)

// The test binary doubles as $VISUAL for hug task edit: a shell script with
// a #!/bin/sh line cannot be started on Windows ("%1 is not a valid Win32
// application", 4emme8), and perl and sh are not on every PATH there.
const (
	editFrom = "HUG_TEST_EDIT_FROM"
	editTo   = "HUG_TEST_EDIT_TO"
)

func TestMain(m *testing.M) {
	if len(os.Args) == 4 && os.Args[3] == SyncChildArg {
		runSyncChild() // a background pass spawned by sync_test.go
	}
	if from, ok := os.LookupEnv(editFrom); ok && len(os.Args) > 1 {
		os.Exit(fakeEdit(os.Args[len(os.Args)-1], from, os.Getenv(editTo)))
	}
	os.Exit(m.Run())
}

// fakeEdit replaces the first from with to in path, as the editor would.
func fakeEdit(path, from, to string) int {
	b, err := os.ReadFile(path)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if from != "" && !strings.Contains(string(b), from) {
		fmt.Fprintf(os.Stderr, "fake editor: %q not in %s\n", from, path)
		return 1
	}
	if err := os.WriteFile(path, []byte(strings.Replace(string(b), from, to, 1)), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

// useEditor makes hug task edit replace the first from with to in the task
// file; from "" leaves the file as it is.
func useEditor(t *testing.T, from, to string) {
	t.Helper()
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	if strings.ContainsAny(exe, " \t") {
		t.Fatalf("test binary path %q has a space; editor.Command splits $VISUAL on spaces", exe)
	}
	t.Setenv("VISUAL", exe)
	t.Setenv(editFrom, from)
	t.Setenv(editTo, to)
}
