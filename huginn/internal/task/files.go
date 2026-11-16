package task

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/veridit/hug/huginn/internal/id"
)

// Task files (2026-11-15). A task owns a folder named by its id, next to its
// Markdown file: <group>/<id>/<path>. Anything inside is just files (test
// logs, tables, probes); the tool does not care what they prove. The folder
// moves with the task and is deleted only with the task.

// MaxFileBytes is the largest task file; files must be UTF-8 text.
const MaxFileBytes = 256 << 10

// File is one file in a task's folder.
type File struct {
	Name   string `json:"name"`
	Bytes  int    `json:"bytes"`
	SHA256 string `json:"sha256"`
}

// IsTaskFolder reports whether dir (a path below the store root, slash
// separated) is the files folder of a task: its base name is a task id and a
// task file with that id sits in the same parent folder.
func IsTaskFolder(root, dir string) bool {
	base := filepath.Base(dir)
	if !id.Valid(base) {
		return false
	}
	matches, _ := filepath.Glob(filepath.Join(root, filepath.Dir(filepath.FromSlash(dir)), "*."+base+".*.md"))
	more, _ := filepath.Glob(filepath.Join(root, filepath.Dir(filepath.FromSlash(dir)), base+".*.md"))
	return len(matches)+len(more) > 0
}

// FolderOf is the files folder of a task (relative, slash separated).
func FolderOf(t Task) string {
	return filepath.ToSlash(filepath.Join(filepath.Dir(t.Path), t.ID))
}

var safeName = regexp.MustCompile(`^[A-Za-z0-9_.+,=@-][A-Za-z0-9_.+,=@ -]*$`)

// cleanName validates a file name inside a task folder: relative, no '..',
// no hidden or control characters. Subfolders are allowed (bench/README.md).
func cleanName(name string) (string, error) {
	name = filepath.ToSlash(strings.TrimSpace(name))
	if name == "" || strings.HasPrefix(name, "/") || strings.Contains(name, "\\") {
		return "", &Error{Kind: "invalid-file-name", Value: name, Fix: "use a relative name such as bench/README.md"}
	}
	for _, part := range strings.Split(name, "/") {
		if part == "" || part == "." || part == ".." || strings.HasPrefix(part, ".") || !safeName.MatchString(part) {
			return "", &Error{Kind: "invalid-file-name", Value: name, Fix: "use letters, digits and . _ - + , = @ in each part, no leading dot"}
		}
	}
	return name, nil
}

// CheckText refuses content that is not small UTF-8 text.
func CheckText(name string, b []byte) error {
	if len(b) > MaxFileBytes {
		return &Error{Kind: "file-too-large", Value: fmt.Sprintf("%s is %d bytes; the limit is %d (256 KiB)", name, len(b), MaxFileBytes), Fix: "attach a smaller excerpt, or the script that regenerates it"}
	}
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return &Error{Kind: "file-not-text", Value: name + " contains NUL bytes or is not valid UTF-8", Fix: "attach text only: logs, tables, scripts, notes"}
	}
	return nil
}

func digest(b []byte) string { h := sha256.Sum256(b); return hex.EncodeToString(h[:]) }

// Files lists a task's files, sorted by name.
func (s *Store) Files(ref string) (Task, []File, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Task{}, nil, e
	}
	dir, e := s.diskPath(FolderOf(d.Task))
	if e != nil {
		return d.Task, nil, e
	}
	var out []File
	filepath.WalkDir(dir, func(p string, de os.DirEntry, err error) error {
		if err != nil || de.IsDir() || de.Type()&os.ModeSymlink != 0 {
			return nil
		}
		b, err := os.ReadFile(p)
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(dir, p)
		out = append(out, File{Name: filepath.ToSlash(rel), Bytes: len(b), SHA256: digest(b)})
		return nil
	})
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return d.Task, out, nil
}

// ReadFile returns one task file.
func (s *Store) ReadFile(ref, name string) ([]byte, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return nil, e
	}
	name, e = cleanName(name)
	if e != nil {
		return nil, e
	}
	p, e := s.diskPath(FolderOf(d.Task) + "/" + name)
	if e != nil {
		return nil, e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return nil, &Error{Kind: "file-not-found", ID: d.Task.ID, Value: name, Fix: "hug task show " + d.Task.ID}
	}
	return b, e
}

// AttachResult reports one attach.
type AttachResult struct {
	File    File   `json:"file"`
	Changed bool   `json:"changed"`
	Old     string `json:"old_sha256,omitempty"`
}

// Attach copies content into the task's folder as name and records a note
//
//	file:NAME (BYTES bytes, sha256:PREFIX) WHY
//
// Same name and content: nothing changes. Same name, other content: refused
// unless replace, which notes the old and new hash.
func (s *Store) Attach(ref, name string, content []byte, why string, replace bool) (Result, AttachResult, error) {
	var ar AttachResult
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, ar, e
	}
	name, e = cleanName(name)
	if e != nil {
		return Result{}, ar, e
	}
	if e = CheckText(name, content); e != nil {
		return Result{}, ar, e
	}
	p, e := s.diskPath(FolderOf(d.Task) + "/" + name)
	if e != nil {
		return Result{}, ar, e
	}
	sum := digest(content)
	ar.File = File{Name: name, Bytes: len(content), SHA256: sum}
	if old, e := os.ReadFile(p); e == nil {
		if digest(old) == sum {
			r, e := s.finish(d, false)
			return r, ar, e
		}
		if !replace {
			return Result{}, ar, &Error{Kind: "file-exists", ID: d.Task.ID, Value: name + " has other content (sha256:" + digest(old)[:8] + ")",
				Fix: "hug task attach " + d.Task.ID + " FILE --name " + name + " --replace"}
		}
		ar.Old = digest(old)
	}
	if e = os.MkdirAll(filepath.Dir(p), 0o755); e != nil {
		return Result{}, ar, e
	}
	if e = AtomicWrite(p, content); e != nil {
		return Result{}, ar, e
	}
	note := fmt.Sprintf("file:%s (%d bytes, sha256:%s)", name, len(content), sum[:8])
	if ar.Old != "" {
		note += " replaces sha256:" + ar.Old[:8]
	}
	if why = strings.TrimSpace(why); why != "" {
		note += " " + why
	}
	d.Note(note)
	ar.Changed = true
	r, e := s.finish(d, true)
	return r, ar, e
}

// Detach removes a task file and notes it.
func (s *Store) Detach(ref, name string) (Result, error) {
	d, _, e := s.load(ref)
	if e != nil {
		return Result{}, e
	}
	name, e = cleanName(name)
	if e != nil {
		return Result{}, e
	}
	p, e := s.diskPath(FolderOf(d.Task) + "/" + name)
	if e != nil {
		return Result{}, e
	}
	b, e := os.ReadFile(p)
	if os.IsNotExist(e) {
		return Result{}, &Error{Kind: "file-not-found", ID: d.Task.ID, Value: name, Fix: "hug task show " + d.Task.ID}
	}
	if e != nil {
		return Result{}, e
	}
	if e = os.Remove(p); e != nil {
		return Result{}, e
	}
	// Drop now-empty parent folders up to the task folder itself.
	top, _ := s.diskPath(FolderOf(d.Task))
	for dir := filepath.Dir(p); strings.HasPrefix(dir, top); dir = filepath.Dir(dir) {
		if os.Remove(dir) != nil {
			break
		}
	}
	d.Note(fmt.Sprintf("detached file:%s (sha256:%s)", name, digest(b)[:8]))
	return s.finish(d, true)
}

// Reference forms in notes and sections:
//
//	file:NAME            a file of this task
//	task:REF/NAME        a file of another task (REF = id, prefix or alias)
var (
	fileRef = regexp.MustCompile(`(?:^|[\s(\[])file:([A-Za-z0-9_.+,=@/-]*[A-Za-z0-9_+=@-])`)
	taskRef = regexp.MustCompile(`(?:^|[\s(\[])task:([A-Za-z0-9.@_-]+)/([A-Za-z0-9_.+,=@/-]*[A-Za-z0-9_+=@-])`)
)

// Ref is one file reference found in a task's text.
type Ref struct {
	Task, Name, Raw string
}

// FileRefs returns the file references in a task document's text.
func FileRefs(doc []byte, self string) []Ref {
	var out []Ref
	for _, m := range fileRef.FindAllSubmatch(doc, -1) {
		name := string(m[1])
		// Prose such as "file:line" and URLs (file:///...) are not
		// references: a reference names a file (has an extension) or a path.
		if strings.HasPrefix(name, "/") || (!strings.Contains(name, ".") && !strings.Contains(name, "/")) {
			continue
		}
		out = append(out, Ref{Task: self, Name: name, Raw: "file:" + name})
	}
	for _, m := range taskRef.FindAllSubmatch(doc, -1) {
		out = append(out, Ref{Task: string(m[1]), Name: string(m[2]), Raw: "task:" + string(m[1]) + "/" + string(m[2])})
	}
	return out
}

// LintFiles checks task files and the references to them:
//   - a file:NAME or task:REF/NAME that does not resolve
//   - a file that is binary or over the size limit (edited by hand)
//   - text citing a path under one of the forbidden prefixes
func (s *Store) LintFiles(forbid []string) ([]Finding, error) {
	ts, e := s.Tasks()
	if e != nil {
		return nil, e
	}
	var out []Finding
	for _, t := range ts {
		p, e := s.diskPath(t.Path)
		if e != nil {
			continue
		}
		doc, e := os.ReadFile(p)
		if e != nil {
			continue
		}
		for _, ref := range FileRefs(doc, t.ID) {
			owner := t
			if ref.Task != t.ID {
				o, e := Resolve(ts, ref.Task)
				if e != nil {
					out = append(out, Finding{Path: t.Path, Line: 0, Problem: ref.Raw + ": no task " + ref.Task})
					continue
				}
				owner = o
			}
			fp, e := s.diskPath(FolderOf(owner) + "/" + ref.Name)
			if e != nil {
				continue
			}
			if _, e := os.Stat(fp); e != nil {
				out = append(out, Finding{Path: t.Path, Line: 0, Problem: ref.Raw + ": file not found"})
			}
		}
		for _, prefix := range forbid {
			if strings.Contains(string(doc), prefix) {
				out = append(out, Finding{Path: t.Path, Line: 0, Problem: "cites " + prefix + " (a folder that gets deleted); attach the file instead"})
			}
		}
		dir, e := s.diskPath(FolderOf(t))
		if e != nil {
			continue
		}
		filepath.WalkDir(dir, func(fp string, de os.DirEntry, err error) error {
			if err != nil || de.IsDir() {
				return nil
			}
			rel, _ := filepath.Rel(s.Root, fp)
			if de.Type()&os.ModeSymlink != 0 {
				out = append(out, Finding{Path: filepath.ToSlash(rel), Line: 0, Problem: "task file is a symlink"})
				return nil
			}
			b, err := os.ReadFile(fp)
			if err == nil {
				if e := CheckText(filepath.ToSlash(rel), b); e != nil {
					msg := e.Error()
					if i := strings.Index(msg, "\n"); i >= 0 {
						msg = msg[:i]
					}
					out = append(out, Finding{Path: filepath.ToSlash(rel), Line: 0, Problem: msg})
				}
			}
			return nil
		})
	}
	return out, nil
}
