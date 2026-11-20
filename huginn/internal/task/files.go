package task

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"github.com/veridit/hug/internal/sharefile"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/veridit/hug/huginn/internal/id"
)

// Task files (2026-11-15). A task owns a folder named by its id, next to its
// Markdown file: <group>/<id>/<path>. Anything inside is just files (test
// logs, tables, probes); the tool does not care what they prove. The folder
// moves with the task and is deleted only with the task.

// MaxFileBytes is the largest text task file; files are UTF-8 text, or an
// image (5z4zt4) up to MaxImageBytes.
const MaxFileBytes = 256 << 10

// MaxImageBytes is the largest image a task carries: a screenshot, not a
// photo album (5z4zt4).
const MaxImageBytes = 2 << 20

// ImageKind names the image b is, by its first bytes, never its name:
// "png", "jpeg" or "gif", else "". A task may carry these as pictures of
// what it is about (5z4zt4, Jørgen 14:51); any other binary stays out.
func ImageKind(b []byte) string {
	switch {
	case bytes.HasPrefix(b, []byte("\x89PNG\r\n\x1a\n")):
		return "png"
	case bytes.HasPrefix(b, []byte{0xff, 0xd8, 0xff}):
		return "jpeg"
	case bytes.HasPrefix(b, []byte("GIF87a")), bytes.HasPrefix(b, []byte("GIF89a")):
		return "gif"
	}
	return ""
}

// imageExt is the names an image may carry, by kind.
var imageExt = map[string][]string{"png": {".png"}, "jpeg": {".jpg", ".jpeg"}, "gif": {".gif"}}

// IsImageName says whether name ends like an image file.
func IsImageName(name string) bool {
	ext := strings.ToLower(filepath.Ext(name))
	for _, exts := range imageExt {
		for _, x := range exts {
			if ext == x {
				return true
			}
		}
	}
	return false
}

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
	if len(matches)+len(more) > 0 {
		return true
	}
	// Layout 3 (w40cmk): C/ID.md owns C/ID/.
	st, e := os.Lstat(filepath.Join(root, filepath.FromSlash(dir)+".md"))
	return e == nil && st.Mode().IsRegular()
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

// CheckText refuses content that is not small UTF-8 text or an image whose
// name says what it is (5z4zt4: shot.png holds a PNG, never a renamed
// binary).
func CheckText(name string, b []byte) error {
	if kind := ImageKind(b); kind != "" {
		ext := strings.ToLower(filepath.Ext(name))
		ok := false
		for _, x := range imageExt[kind] {
			ok = ok || ext == x
		}
		if !ok {
			return &Error{Kind: "file-not-text", Value: name + " holds a " + strings.ToUpper(kind) + " image but is not named ." + imageExt[kind][0][1:], Fix: "hug task attach ID FILE --name " + strings.TrimSuffix(name, filepath.Ext(name)) + imageExt[kind][0]}
		}
		if len(b) > MaxImageBytes {
			return &Error{Kind: "file-too-large", Value: fmt.Sprintf("%s is %d bytes; the limit for an image is %d (2 MiB)", name, len(b), MaxImageBytes), Fix: "attach a smaller screenshot (crop it), or an ANSI capture: S in hug tui"}
		}
		return nil
	}
	if len(b) > MaxFileBytes {
		return &Error{Kind: "file-too-large", Value: fmt.Sprintf("%s is %d bytes; the limit is %d (256 KiB)", name, len(b), MaxFileBytes), Fix: "attach a smaller excerpt, or the script that regenerates it"}
	}
	if bytes.IndexByte(b, 0) >= 0 || !utf8.Valid(b) {
		return &Error{Kind: "file-not-text", Value: name + " contains NUL bytes or is not valid UTF-8", Fix: "hug task attach ID FILE (text such as a log, table or .ans screen capture, or a PNG, JPEG or GIF image)"}
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
		b, err := sharefile.ReadFile(p)
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
	b, e := sharefile.ReadFile(p)
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
	if old, e := sharefile.ReadFile(p); e == nil {
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
	b, e := sharefile.ReadFile(p)
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

// Reference forms in notes and sections, the links lint checks:
//
//	file:NAME            a file of this task
//	task:REF/NAME        a file of another task (REF = id, prefix or alias)
//
// A path such as .hug/tasks/ID/NAME is text, not a link (7j706s): prose
// quotes it, often a name proposed and never made. Only live text links:
// a reference in record text (## History, an answered or withdrawn
// question, a resolved request; see recordSpans) is not checked. A
// reference followed by " (lost)" names a file known to be gone and is
// not checked (cvbpdy). The last part of NAME may be a shell glob (* ? and
// [...]), as in "file:compare-*.png": the pattern is matched against the
// task's files (ydwamb).
var (
	fileRef = regexp.MustCompile(`(?:^|[\s(\[])file:([A-Za-z0-9_.+,=@/*?\[\]-]*[A-Za-z0-9_+=@*?\]-])`)
	taskRef = regexp.MustCompile(`(?:^|[\s(\[])task:([A-Za-z0-9.@_-]+)/([A-Za-z0-9_.+,=@/*?\[\]-]*[A-Za-z0-9_+=@*?\]-])`)
)

// globChars are the shell glob characters a citation's last part may hold.
const globChars = "*?["

// citedName is name as a citation means it: a plain file name, or a glob
// in its last path part (ydwamb). Prose around a plain name is not part of
// it: a closing "]" with no "[" ("[file:a.md]"), and what follows a
// whole name with an extension ("is it file:a.md?", "**file:a.md**",
// "file:a.md?raw=1"). A glob character before the last "/" is no citation
// hug can check. ok is false when no name is left.
func citedName(name string) (string, bool) {
	if i := strings.IndexAny(name, "*?"); i > 0 && wholeName(name[:i]) {
		name = name[:i]
	}
	for strings.HasSuffix(name, "]") && !strings.Contains(name, "[") {
		name = name[:len(name)-1]
	}
	if name == "" {
		return "", false
	}
	if i := strings.LastIndexByte(name, '/'); i >= 0 && strings.ContainsAny(name[:i], globChars+"]") {
		return "", false
	}
	if isGlob(name) {
		if _, e := path.Match(name, ""); e != nil {
			return "", false // a malformed pattern ("a[b") is prose
		}
	}
	return name, true
}

// wholeName says whether s is a plain file name as a citation ends one: no
// glob, an extension in its last part, a letter or digit at its end.
func wholeName(s string) bool {
	if s == "" || isGlob(s) || strings.Contains(s, "]") {
		return false
	}
	c := s[len(s)-1]
	base := s[strings.LastIndexByte(s, '/')+1:]
	return strings.Contains(base, ".") && (c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9')
}

// isGlob says whether a cited name is a pattern.
func isGlob(name string) bool { return strings.ContainsAny(name, globChars) }

// Lost is what follows a citation of a file known to be gone.
const Lost = " (lost)"

// gapMarks follow a citation of a file known not to be kept: gone, or not
// kept because too large or run output (" (not kept: too large)").
var gapMarks = [][]byte{[]byte(Lost), []byte(" (not kept")}

// lostAfter reports whether doc[end:] starts with a gap mark, allowing
// a closing quote or backtick between the citation and the mark.
func lostAfter(doc []byte, end int) bool {
	for end < len(doc) && strings.IndexByte("`'\"", doc[end]) >= 0 {
		end++
	}
	for _, m := range gapMarks {
		if bytes.HasPrefix(doc[end:], m) {
			return true
		}
	}
	return false
}

// Ref is one file reference found in a task's text.
type Ref struct {
	Task, Name, Raw string
}

// FileRefs returns the file references in a task document's text.
func FileRefs(doc []byte, self string) []Ref {
	type at struct {
		ref Ref
		pos int
	}
	var all []at
	// gone maps a file of this task to where hug task detach noted it
	// ("detached file:NAME", bqfrda): a citation of it before that is of a
	// file removed on purpose, not a missing one. A later one (attached
	// again) is checked.
	var gone map[string]int
	// lost are the citations the text marks gone somewhere (" (lost)" or
	// "(not kept ...)" after one of them): every citation of the same ref
	// is then known gone, so a note marking it settles an earlier one
	// (1phmre: hug task note ID "REF (lost)" is the fix an agent runs
	// without an editor).
	lost := map[string]bool{}
	// The literals first (3mavpt): most task files cite nothing, and the
	// regexps cost 17 s over Seid's 2,677 imported files when run on each.
	if bytes.Contains(doc, []byte("file:")) {
		for _, m := range fileRef.FindAllSubmatchIndex(doc, -1) {
			name, ok := citedName(string(doc[m[2]:m[3]]))
			// Prose such as "file:line" and URLs (file:///...) are not
			// references: a reference names a file (has an extension) or a path.
			if !ok || strings.HasPrefix(name, "/") || (!strings.Contains(name, ".") && !strings.Contains(name, "/")) {
				continue
			}
			if lostAfter(doc, m[2]+len(name)) {
				lost["file:"+name] = true
				continue
			}
			if bytes.HasSuffix(doc[:m[2]], []byte("detached file:")) {
				if gone == nil {
					gone = map[string]int{}
				}
				gone[name] = m[2]
				continue
			}
			all = append(all, at{Ref{Task: self, Name: name, Raw: "file:" + name}, m[2]})
		}
	}
	if bytes.Contains(doc, []byte("task:")) {
		for _, m := range taskRef.FindAllSubmatchIndex(doc, -1) {
			// Syntax in prose ("task:REF/NAME", "task:ID/PATH") is not a
			// reference: a real ref is a lowercase id, alias or slug.
			ref, name := string(doc[m[2]:m[3]]), string(doc[m[4]:m[5]])
			if ref == strings.ToUpper(ref) && strings.ToLower(ref) != ref && !strings.ContainsAny(ref, "0123456789") {
				continue
			}
			name, ok := citedName(name)
			if !ok {
				continue
			}
			if lostAfter(doc, m[4]+len(name)) {
				lost["task:"+ref+"/"+name] = true
				continue
			}
			all = append(all, at{Ref{Task: ref, Name: name, Raw: "task:" + ref + "/" + name}, m[2]})
		}
	}
	var out []Ref
	var record [][2]int
	if len(all) > 0 {
		record = recordSpans(doc)
	}
	for _, a := range all {
		if p, ok := gone[a.ref.Name]; (ok && a.ref.Task == self && a.pos < p) || lost[a.ref.Raw] || inSpans(record, a.pos) {
			continue
		}
		out = append(out, a.ref)
	}
	return out
}

// forbiddenCite is one citation of a path under a forbidden prefix.
type forbiddenCite struct {
	path, prefix string
	line         int
}

// forbiddenCites finds each citation of a path under one of the prefixes,
// with its 1-based line. A prefix is a path relative to the repository: it
// counts where a path starts (not after a letter, digit or "/", so
// /home/x/.jcode/scratch/ is another folder), and the cited path runs over
// path characters, braces and commas included ({a,b}.log), ending in a
// letter, digit or one of _+}-, as tools/evidence.py in Seid reads them. One
// followed by a gap mark (" (lost)", " (not kept: ...)") is not reported.
func forbiddenCites(doc []byte, forbid []string) []forbiddenCite {
	var out []forbiddenCite
	var res []*regexp.Regexp
	for _, prefix := range forbid {
		res = append(res, regexp.MustCompile(`(?:^|[^\w/])(`+regexp.QuoteMeta(prefix)+`[A-Za-z0-9_.+,{}/-]*[A-Za-z0-9_+}-]/?)`))
	}
	for n, line := range bytes.Split(doc, []byte("\n")) {
		for i, re := range res {
			for _, m := range re.FindAllSubmatchIndex(line, -1) {
				if lostAfter(line, m[3]) {
					continue
				}
				out = append(out, forbiddenCite{path: string(line[m[2]:m[3]]), prefix: forbid[i], line: n + 1})
			}
		}
	}
	return out
}

// globMatches says whether pattern, the last part of a citation, matches a
// file in t's folder (ydwamb). A pattern with a folder part matches in that
// folder. Reading only: lint never writes.
func (s *Store) globMatches(t Task, pattern string) bool {
	dir, base := path.Split(pattern)
	folder, e := s.diskPath(FolderOf(t) + "/" + strings.TrimSuffix(dir, "/"))
	if e != nil {
		return false
	}
	entries, e := os.ReadDir(folder)
	if e != nil {
		return false
	}
	for _, en := range entries {
		if en.IsDir() {
			continue
		}
		if ok, _ := path.Match(base, en.Name()); ok {
			return true
		}
	}
	return false
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
	// Which task folders exist: one ReadDir per group folder instead of an
	// Lstat chain per task (44w1ev: diskPath was 56% of lint at 10,000
	// tasks, and almost no task has files).
	folders := map[string]bool{}
	listed := map[string]bool{}
	for _, t := range ts {
		group := filepath.Dir(filepath.FromSlash(t.Path))
		if listed[group] {
			continue
		}
		listed[group] = true
		entries, _ := os.ReadDir(filepath.Join(s.Root, group))
		for _, en := range entries {
			if en.IsDir() || en.Type()&os.ModeSymlink != 0 {
				folders[filepath.ToSlash(filepath.Join(group, en.Name()))] = true
			}
		}
	}
	// Each group folder is checked for traversal and symlinks once; the
	// task file itself is opened only when it is not a symlink.
	safe := map[string]bool{}
	for _, t := range ts {
		group := filepath.Dir(filepath.FromSlash(t.Path))
		ok, seen := safe[group]
		if !seen {
			_, e := s.diskPath(filepath.ToSlash(group))
			ok = e == nil
			safe[group] = ok
		}
		if !ok {
			continue
		}
		// The references come from the parse the cache kept (44w1ev): the
		// file is read only to check forbidden prefixes, or when the Task
		// was made without its texts.
		var doc []byte
		refs := []Ref(nil)
		if t.Texts != nil {
			refs = t.Texts.Refs
		}
		if len(forbid) > 0 || t.Texts == nil {
			p := filepath.Join(s.Root, filepath.FromSlash(t.Path))
			if st, e := os.Lstat(p); e != nil || st.Mode()&os.ModeSymlink != 0 {
				continue
			}
			if doc, e = sharefile.ReadFile(p); e != nil {
				continue
			}
			if t.Texts == nil {
				refs = FileRefs(doc, t.ID)
			}
		}
		// One finding per missing file (bqfrda): a file cited three times
		// is one fact.
		reported := map[string]bool{}
		for _, ref := range refs {
			owner := t
			if ref.Task != t.ID {
				o, e := Resolve(ts, ref.Task)
				if e != nil {
					out = append(out, Finding{Path: t.Path, Line: 0, Problem: ref.Raw + ": no task " + ref.Task + " (fix the id, or mark the link" + Lost + ")", Fix: "hug task note " + t.ID + " " + strconv.Quote(ref.Raw+Lost)})
					continue
				}
				owner = o
			}
			if isGlob(ref.Name) {
				// A pattern (ydwamb): fine when it matches a file of the
				// task; else one finding naming the whole pattern, whose fix
				// attaches a file under its own name.
				key := FolderOf(owner) + "/" + ref.Name
				if s.globMatches(owner, ref.Name) || reported[key] {
					continue
				}
				reported[key] = true
				out = append(out, Finding{Path: t.Path, Line: 0, Problem: ref.Raw + ": no attached file matches (attach one, or mark the citation" + Lost + ")", Fix: "hug task attach " + owner.ID + " FILE"})
				continue
			}
			fp, e := s.diskPath(FolderOf(owner) + "/" + ref.Name)
			if e != nil {
				continue
			}
			if _, e := os.Stat(fp); e != nil && !reported[fp] {
				reported[fp] = true
				out = append(out, Finding{Path: t.Path, Line: 0, Problem: ref.Raw + ": file not found (attach it, or mark the citation" + Lost + ")", Fix: "hug task attach " + owner.ID + " FILE --name " + ref.Name})
			}
		}
		for _, f := range forbiddenCites(doc, forbid) {
			out = append(out, Finding{Path: t.Path, Line: f.line, Problem: "cites " + f.path + " (" + f.prefix + " is a folder that gets deleted); attach the file instead, or mark it" + Lost, Fix: "hug task attach " + t.ID + " FILE, then hug task edit " + t.ID})
		}
		if !folders[FolderOf(t)] {
			continue
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
			name, _ := filepath.Rel(dir, fp)
			if de.Type()&os.ModeSymlink != 0 {
				out = append(out, Finding{Path: filepath.ToSlash(rel), Line: 0, Problem: "task file is a symlink", Fix: "hug task detach " + t.ID + " " + filepath.ToSlash(name)})
				return nil
			}
			b, err := sharefile.ReadFile(fp)
			if err == nil {
				if e := CheckText(filepath.ToSlash(rel), b); e != nil {
					msg := e.Error()
					if i := strings.Index(msg, "\n"); i >= 0 {
						msg = msg[:i]
					}
					out = append(out, Finding{Path: filepath.ToSlash(rel), Line: 0, Problem: msg, Fix: "hug task detach " + t.ID + " " + filepath.ToSlash(name)})
				}
			}
			return nil
		})
	}
	return out, nil
}

// FolderOwners are the names in a folder listing that are a task's files
// folder: a valid id that names a Markdown file in the same listing, as
// IsTaskFolder decides it ("*.<id>.*.md" or "<id>.*.md"), from the one
// listing instead of two globs per folder (np2kk2).
func FolderOwners(entries []os.DirEntry) map[string]bool {
	ids := map[string]bool{}
	for _, d := range entries {
		n := d.Name()
		if d.IsDir() || !strings.HasSuffix(n, ".md") {
			continue
		}
		p := strings.Split(n, ".")
		// "<id>.md" (layout 3, w40cmk).
		if len(p) == 2 && id.Valid(p[0]) {
			ids[p[0]] = true
		}
		// "<id>.*.md" and "*.<id>.*.md": any part but the last two.
		for i := 0; i+2 < len(p); i++ {
			if id.Valid(p[i]) {
				ids[p[i]] = true
			}
		}
	}
	return ids
}
