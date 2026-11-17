package sharefile

import (
	"errors"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"
)

// Callers tell a missing file by os.IsNotExist, on every OS (kbc99j).
func TestReadMissingIsNotExist(t *testing.T) {
	path := filepath.Join(t.TempDir(), "missing.md")
	_, err := ReadFile(path)
	if !os.IsNotExist(err) || !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("ReadFile of a missing file: %v, want not-exist", err)
	}
	var pe *fs.PathError
	if !errors.As(err, &pe) || pe.Path != path {
		t.Fatalf("error %v does not name %s", err, path)
	}
}

// A rename replaces a file that is open for reading, and the reader keeps
// the bytes it opened (POSIX semantics, which Windows needs asked for).
func TestRenameOverOpenFile(t *testing.T) {
	dir := t.TempDir()
	path, tmp := filepath.Join(dir, "task.md"), filepath.Join(dir, ".tmp")
	if err := os.WriteFile(path, []byte("old"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(tmp, []byte("new"), 0o644); err != nil {
		t.Fatal(err)
	}
	f, err := Open(path)
	if err != nil {
		t.Fatal(err)
	}
	defer f.Close()
	if err = Rename(tmp, path); err != nil {
		t.Fatalf("rename over a file open for reading: %v", err)
	}
	kept, err := io.ReadAll(f)
	if err != nil || string(kept) != "old" {
		t.Fatalf("the open reader got %q, %v; want the old bytes", kept, err)
	}
	if b, err := ReadFile(path); err != nil || string(b) != "new" {
		t.Fatalf("after rename read %q, %v", b, err)
	}
	if _, err := os.Stat(tmp); !os.IsNotExist(err) {
		t.Fatalf("temp file still there: %v", err)
	}
}
