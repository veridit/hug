package importer

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/veridit/hug/huginn/internal/task"
)

// parent_task_id becomes parent: <hug id>; rerunning changes nothing.
func TestImportParent(t *testing.T) {
	src := t.TempDir()
	dst := &task.Store{Root: t.TempDir()}
	epic := strings.Replace(strings.ReplaceAll(fixture, "DEMO-001", "DEMO-010"), "dependencies: [DEMO-002]", "dependencies: []", 1)
	child := strings.Replace(strings.ReplaceAll(fixture, "DEMO-001", "DEMO-011"), "dependencies: [DEMO-002]", "dependencies: []\nparent_task_id: demo-010", 1)
	write(t, filepath.Join(src, "epic.md"), []byte(epic))
	write(t, filepath.Join(src, "child.md"), []byte(child))
	run(t, src, dst, Options{})
	before := hashes(t, dst.Root)
	if r := run(t, src, dst, Options{}); r.Unchanged != 2 || !reflect.DeepEqual(before, hashes(t, dst.Root)) {
		t.Fatal("not idempotent", r)
	}
	e, err := dst.Get("DEMO-010")
	if err != nil {
		t.Fatal(err)
	}
	c, err := dst.Get("DEMO-011")
	if err != nil {
		t.Fatal(err)
	}
	if c.Task.Parent != e.Task.ID {
		t.Fatalf("parent = %q, want epic id %q", c.Task.Parent, e.Task.ID)
	}
	ts, _ := dst.Tasks()
	if !task.IsEpic(e.Task, ts) {
		t.Fatal("epic not recognised")
	}
}
