package gitbranch

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Refs and HEAD read from the files, as git rev-parse resolves them, so a
// command spends no git process on them (0sbvnq: a git process costs ~45 ms
// on the Windows runner, every hug command asked for HEAD and for the old
// layout's refs). Whatever this does not read with certainty (reftable, a
// symbolic ref other than HEAD, a file it cannot parse) answers ok=false and
// the caller asks git.

// RefAt resolves the full ref name (refs/heads/x, refs/remotes/origin/x) in
// the common git dir: the loose file, then packed-refs. found is false when
// neither holds it. ok is false when the answer needs git.
func RefAt(common, ref string) (hash string, found, ok bool) {
	if _, e := os.Stat(filepath.Join(common, "reftable")); e == nil {
		return "", false, false
	}
	b, e := os.ReadFile(filepath.Join(common, filepath.FromSlash(ref)))
	switch {
	case e == nil:
		h := strings.TrimSpace(string(b))
		if !isHash(h) {
			return "", false, false // a symbolic ref, or a half-written file
		}
		return h, true, true
	case !os.IsNotExist(e) && !isDirError(common, ref):
		return "", false, false
	}
	f, e := os.Open(filepath.Join(common, "packed-refs"))
	if os.IsNotExist(e) {
		return "", false, true
	}
	if e != nil {
		return "", false, false
	}
	defer f.Close()
	sc := bufio.NewScanner(f)
	sc.Buffer(make([]byte, 64*1024), 1<<20)
	for sc.Scan() {
		h, name, cut := strings.Cut(sc.Text(), " ")
		if cut && name == ref && isHash(h) {
			return h, true, true
		}
	}
	if sc.Err() != nil {
		return "", false, false
	}
	return "", false, true
}

// isDirError: ref is a folder (refs/heads/huginn holding huginn/x), so no
// loose ref of that name exists.
func isDirError(common, ref string) bool {
	st, e := os.Stat(filepath.Join(common, filepath.FromSlash(ref)))
	return e == nil && st.IsDir()
}

// isHash is a full object id: 40 (SHA-1) or 64 (SHA-256) hex digits.
func isHash(s string) bool {
	if len(s) != 40 && len(s) != 64 {
		return false
	}
	for _, c := range s {
		if !('0' <= c && c <= '9' || 'a' <= c && c <= 'f') {
			return false
		}
	}
	return true
}

// fastHead is rev-parse --verify HEAD in the worktree at dir, from the
// files: a detached HEAD's id, or the branch it names resolved in the
// common dir. ok is false when git must answer (an unborn branch included:
// Head tells unborn from broken with git).
func fastHead(dir string) (string, bool) {
	gitdir, linked, e := gitDir(dir)
	if e != nil {
		return "", false
	}
	common := gitdir
	if linked {
		if common, e = commonDir(dir); e != nil {
			return "", false
		}
	}
	b, e := os.ReadFile(filepath.Join(gitdir, "HEAD"))
	if e != nil {
		return "", false
	}
	head := strings.TrimSpace(string(b))
	ref, symbolic := strings.CutPrefix(head, "ref: ")
	if !symbolic {
		return head, isHash(head)
	}
	if !strings.HasPrefix(ref, "refs/heads/") {
		return "", false
	}
	h, found, ok := RefAt(common, ref)
	return h, ok && found
}
