package gitbranch

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// objects reads one repository's objects for one replay or one pending
// list (0n4rnx): through one `git cat-file --batch` child instead of a
// git process per tree, file and parent (a replayed commit touching one
// file cost ~13 processes, ~2 now). It belongs to the call that made it:
// the caller closes it with defer, on every return and on a panic, and it
// is never stored on a Repository. Trees and blobs are kept by object id,
// which names their content, so nothing it keeps can go stale.
type objects struct {
	dir   string
	cmd   *exec.Cmd
	in    io.WriteCloser
	out   *bufio.Reader
	err   error // a broken stream: every later read fails with it
	blobs map[string][]byte
	trees map[string][]treeEntry // by root tree id: its files, as ls-tree -r lists them
}

// treeEntry is one file of a tree, as git ls-tree -r reports it.
type treeEntry struct{ path, mode, kind, id string }

// readObjects reads objects of the repository at dir. It starts git on
// the first read; close ends it.
func readObjects(dir string) *objects {
	return &objects{dir: dir, blobs: map[string][]byte{}, trees: map[string][]treeEntry{}}
}

func (o *objects) start() error {
	c := exec.Command("git", "-C", o.dir, "cat-file", "--batch")
	c.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	in, e := c.StdinPipe()
	if e != nil {
		return e
	}
	out, e := c.StdoutPipe()
	if e != nil {
		return e
	}
	c.Stderr = io.Discard
	if e := c.Start(); e != nil {
		return &Error{Op: "cat-file --batch", Detail: e.Error(), Fix: "git -C " + quote(o.dir) + " cat-file -t HEAD"}
	}
	o.cmd, o.in, o.out = c, in, bufio.NewReaderSize(out, 64<<10)
	return nil
}

// close ends the git child: its input closes, and it is killed if it has
// not exited 5 s later. It is safe to call twice and on a reader that
// never started.
func (o *objects) close() {
	if o == nil || o.cmd == nil {
		return
	}
	c := o.cmd
	o.cmd = nil
	_ = o.in.Close()
	done := make(chan struct{})
	go func() { _ = c.Wait(); close(done) }()
	select {
	case <-done:
	case <-time.After(5 * time.Second):
		_ = c.Process.Kill()
		<-done
	}
	if o.err == nil {
		o.err = errors.New("object reader closed")
	}
}

// read returns rev's object: its id, type and bytes. ok is false when
// git has no such object.
func (o *objects) read(rev string) (id, kind string, body []byte, ok bool, err error) {
	if o.err != nil {
		return "", "", nil, false, o.err
	}
	if strings.ContainsAny(rev, "\n\r") {
		return "", "", nil, false, fmt.Errorf("object name %q has a line break", rev)
	}
	if o.cmd == nil {
		if e := o.start(); e != nil {
			o.err = e
			return "", "", nil, false, e
		}
	}
	broken := func(e error) (string, string, []byte, bool, error) {
		o.err = &Error{Op: "cat-file --batch " + rev, Detail: e.Error(), Fix: "git -C " + quote(o.dir) + " cat-file -t " + quote(rev)}
		return "", "", nil, false, o.err
	}
	if _, e := io.WriteString(o.in, rev+"\n"); e != nil {
		return broken(e)
	}
	header, e := o.out.ReadString('\n')
	if e != nil {
		return broken(e)
	}
	f := strings.Fields(header)
	if len(f) == 2 && (f[1] == "missing" || f[1] == "ambiguous") {
		return "", "", nil, false, nil
	}
	if len(f) != 3 {
		return broken(fmt.Errorf("unexpected reply %q", strings.TrimSpace(header)))
	}
	size, e := strconv.Atoi(f[2])
	if e != nil || size < 0 {
		return broken(fmt.Errorf("unexpected reply %q", strings.TrimSpace(header)))
	}
	body = make([]byte, size+1) // the object and its closing newline
	if _, e := io.ReadFull(o.out, body); e != nil {
		return broken(e)
	}
	return f[0], f[1], body[:size], true, nil
}

// blob is a blob's exact bytes.
func (o *objects) blob(id string) ([]byte, error) {
	if b, ok := o.blobs[id]; ok {
		return b, nil
	}
	_, kind, body, ok, e := o.read(id)
	if e == nil && (!ok || kind != "blob") {
		e = fmt.Errorf("not a blob in this repository")
	}
	if e != nil {
		return nil, &Error{Op: "cat-file blob " + id, Detail: e.Error(), Fix: "git -C " + quote(o.dir) + " cat-file -t " + id}
	}
	o.blobs[id] = body
	return body, nil
}

// files lists rev's tree recursively, in git ls-tree -r order, with
// modes as ls-tree prints them.
func (o *objects) files(rev string) ([]treeEntry, error) {
	id, kind, body, ok, e := o.read(rev + "^{tree}")
	if e == nil && (!ok || kind != "tree") {
		e = fmt.Errorf("%s has no tree in this repository", rev)
	}
	if e != nil {
		return nil, &Error{Op: "ls-tree " + rev, Detail: e.Error(), Fix: "git -C " + quote(o.dir) + " ls-tree -r " + quote(rev)}
	}
	if list, ok := o.trees[id]; ok {
		return list, nil
	}
	var list []treeEntry
	if e := o.walk(body, len(id)/2, "", &list); e != nil {
		return nil, &Error{Op: "ls-tree " + rev, Detail: e.Error(), Fix: "git -C " + quote(o.dir) + " ls-tree -r " + quote(rev)}
	}
	o.trees[id] = list
	return list, nil
}

// walk appends a raw tree's files under prefix, depth first in stored
// order (ls-tree -r's order).
func (o *objects) walk(raw []byte, idLen int, prefix string, list *[]treeEntry) error {
	for len(raw) > 0 {
		sp := bytes.IndexByte(raw, ' ')
		nul := bytes.IndexByte(raw, 0)
		if sp < 0 || nul < sp || len(raw) < nul+1+idLen {
			return fmt.Errorf("cannot read a tree entry under %q", prefix)
		}
		mode, e := strconv.ParseUint(string(raw[:sp]), 8, 32)
		if e != nil {
			return fmt.Errorf("cannot read a tree entry's mode under %q", prefix)
		}
		name := string(raw[sp+1 : nul])
		id := fmt.Sprintf("%x", raw[nul+1:nul+1+idLen])
		raw = raw[nul+1+idLen:]
		path := prefix + name
		switch mode & 0o170000 {
		case 0o040000:
			_, kind, body, ok, e := o.read(id)
			if e == nil && (!ok || kind != "tree") {
				e = fmt.Errorf("tree %s of %q is missing", id, path)
			}
			if e != nil {
				return e
			}
			if e := o.walk(body, idLen, path+"/", list); e != nil {
				return e
			}
		case 0o100000:
			// Canonical, as ls-tree prints it (git's canon_mode): any
			// executable bit is 100755, else 100644.
			m := "100644"
			if mode&0o100 != 0 {
				m = "100755"
			}
			*list = append(*list, treeEntry{path, m, "blob", id})
		case 0o120000:
			*list = append(*list, treeEntry{path, "120000", "blob", id})
		default:
			*list = append(*list, treeEntry{path, "160000", "commit", id})
		}
	}
	return nil
}

// parent is commit's first parent; ok is false for a root commit.
func (o *objects) parent(commit string) (string, bool, error) {
	id, kind, _, ok, e := o.read(commit + "^")
	if e != nil {
		return "", false, e
	}
	if !ok || kind != "commit" {
		return "", false, nil
	}
	return id, true, nil
}

// layoutVia is Repository.layoutAt through the reader: the board layout
// hug.toml at rev names, or the base layout when rev has no hug.toml.
func (r *Repository) layoutVia(o *objects, rev string) (int, error) {
	_, kind, body, ok, e := o.read(rev + ":hug.toml")
	if e != nil {
		return 0, e
	}
	text := ""
	if ok && kind == "blob" {
		text = strings.TrimSuffix(string(body), "\n")
	} else {
		// No hug.toml at rev: the base layout, as on disk. A rev that
		// cannot be read is not proof that the file is missing.
		if _, kind, _, ok, e := o.read(rev + "^{tree}"); e != nil {
			return 0, e
		} else if !ok || kind != "tree" {
			return 0, &Error{Op: "cat-file -p " + rev + ":hug.toml", Detail: rev + " has no tree in this repository", Fix: r.fix("ls-tree " + quote(rev) + " -- hug.toml")}
		}
	}
	return r.Layout(text)
}
