package cache

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
	"time"

	"github.com/veridit/hug/huginn/internal/task"
)

// schemaFingerprints is the fingerprint of each SchemaVersion: one line
// per bump, never edited, so a reader sees when what a cache holds
// changed (6q1d1t). 30's was taken afterwards, on 4b1a5781 with this test
// copied in: every commit from there to 0201dca6 has it, 99e99eb6 (8a0ybb)
// is the first that does not.
var schemaFingerprints = map[string]string{
	"30": "f4e73995e99e72daf32a7a484c157ddaff7cc3059ff95f6480c78c7825c0a5b9",
	"31": "d99149ebcf924c95d74b898fc67b6cbef660ff0b5c198a29eb0ee89d96c2db34",
	"32": "5830158daac53a2126430b36c2691c945f0763ce3753494a2b6c83af215919f4",
}

// schemaCorpus holds the task files whose parse the fingerprint covers.
const schemaCorpus = "testdata/schema"

// schemaFingerprint hashes what a cache written by this hug holds and
// another hug would serve as fresh: the rule tables Parse reads
// (task.ParseRules), the shape of the cached record (a field an older hug
// never filled decodes as zero), and for each corpus file the Task, the
// Findings and the texts the sync stores.
func schemaFingerprint(t *testing.T) string {
	t.Helper()
	old := task.Now // untilFindings' fix names tomorrow
	task.Now = func() time.Time { return time.Date(2026, 11, 22, 12, 0, 0, 0, time.Local) }
	defer func() { task.Now = old }()
	h := sha256.New()
	add := func(label string, v any) {
		b, e := json.Marshal(v)
		if e != nil {
			t.Fatal(label, e)
		}
		fmt.Fprintf(h, "%s %d\n", label, len(b))
		h.Write(b)
	}
	add("rules", task.ParseRules())
	add("record", typeShape(reflect.TypeOf(record{}), map[reflect.Type]bool{}))
	var paths []string
	e := filepath.WalkDir(schemaCorpus, func(p string, d fs.DirEntry, e error) error {
		if e == nil && !d.IsDir() && strings.HasSuffix(p, ".md") && filepath.Base(p) != "README.md" {
			rel, e := filepath.Rel(schemaCorpus, p)
			if e != nil {
				return e
			}
			paths = append(paths, filepath.ToSlash(rel)) // one order and one label on every OS
		}
		return e
	})
	if e != nil {
		t.Fatal(e)
	}
	sort.Strings(paths)
	if len(paths) < 10 {
		t.Fatalf("the schema corpus has %d task files, want 10", len(paths))
	}
	for _, rel := range paths {
		b, e := os.ReadFile(filepath.Join(schemaCorpus, filepath.FromSlash(rel)))
		if e != nil {
			t.Fatal(e)
		}
		// A Windows checkout may turn LF into CRLF; the corpus is LF.
		b = bytes.ReplaceAll(b, []byte("\r\n"), []byte("\n"))
		d := task.Parse(rel, b)
		// As a cache row holds them: Findings written with their Code and
		// Fix (MarshalJSON fills both) and read back, so a change to the
		// code table or a default fix counts and severity, looked up on
		// every read, does not. Texts as table texts and the lean Refs.
		rows, e := json.Marshal(d.Findings)
		if e != nil {
			t.Fatal(e)
		}
		var fs []task.Finding
		if e := json.Unmarshal(rows, &fs); e != nil {
			t.Fatal(e)
		}
		type stored struct {
			Path, Problem, Code, Fix string
			Line                     int
		}
		read := make([]stored, len(fs))
		for i, f := range fs {
			read[i] = stored{f.Path, f.Problem, f.Code, f.Fix, f.Line}
		}
		add("task "+rel, d.Task)
		add("findings "+rel, read)
		add("bodies "+rel, string(encodeBodies(d.Task.Texts)))
		add("refs "+rel, lean(d.Task.Texts))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// typeShape names t's fields and their types, recursively: a field added
// to Task, Finding or Texts changes it.
func typeShape(t reflect.Type, seen map[reflect.Type]bool) string {
	switch t.Kind() {
	case reflect.Pointer:
		return "*" + typeShape(t.Elem(), seen)
	case reflect.Slice:
		return "[]" + typeShape(t.Elem(), seen)
	case reflect.Map:
		return "map[" + typeShape(t.Key(), seen) + "]" + typeShape(t.Elem(), seen)
	case reflect.Struct:
		if seen[t] {
			return t.String()
		}
		seen[t] = true
		var b strings.Builder
		b.WriteString(t.String() + "{")
		for i := 0; i < t.NumField(); i++ {
			f := t.Field(i)
			fmt.Fprintf(&b, "%s %s %q;", f.Name, typeShape(f.Type, seen), f.Tag)
		}
		return b.String() + "}"
	}
	return t.String()
}

// 6q1d1t: a change to what Parse makes of a task file, or to the record
// the cache keeps, bumps SchemaVersion, else an older hug's cache (same
// file, cache-SchemaVersion.db) is served as this one's. 8a0ybb changed
// SectionOrder under schema 30 and v0.7.103 served v0.7.101's
// section-order finding.
func TestSchemaFingerprint(t *testing.T) {
	fp := schemaFingerprint(t)
	want, ok := schemaFingerprints[SchemaVersion]
	if !ok {
		t.Fatalf("SchemaVersion %s has no fingerprint\nfix: add %q: %q to schemaFingerprints in huginn/internal/cache/fingerprint_test.go", SchemaVersion, SchemaVersion, fp)
	}
	if fp != want {
		t.Fatalf("what a cache holds changed (parse rules, the parse of %s, or the cached record) and SchemaVersion is still %s: an older hug's cache-%s.db would be served as this one's\nfix: bump SchemaVersion in huginn/internal/cache/cache.go with a comment for the change, then add \"NEW\": %q to schemaFingerprints in huginn/internal/cache/fingerprint_test.go", schemaCorpus, SchemaVersion, SchemaVersion, fp)
	}
}
