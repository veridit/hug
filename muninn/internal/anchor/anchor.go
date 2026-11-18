// Package anchor finds the exact things a tool call and a memory can share
// (n7jbmx): a path of two or more segments, a file name, a command head.
// Tool-call recall matches on these, not on text similarity: message recall
// reused for tool calls was 38% precise (muninn/docs/RECALL-TOOL.md).
//
// Anchors are strings "kind:value": "path:muninn/run/main.go",
// "file:main.go", "cmd:go test". A memory states each anchor once; a call
// states every contiguous run of two or more segments of its paths, so a
// memory naming a folder (muninn/run) or a shorter path matches a file in it.
package anchor

import (
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// Recipe names the extraction rules. Changing them bumps it, and the next
// index pass recomputes every memory's anchors.
const Recipe = "1"

var (
	backticked = regexp.MustCompile("`([^`\n]{1,200})`")
	url        = regexp.MustCompile(`[a-z]+://\S+`)
	// A path: segments of word characters, dots and dashes joined by /.
	pathLike = regexp.MustCompile(`(?:~/|\.{0,2}/)?[\w.@-]+(?:/[\w.@-]+)+/?`)
	// A file name: a word with an extension of 1-5 letters.
	fileLike = regexp.MustCompile(`(?:^|[\s(\["'=:,])([\w@-][\w.@-]*\.[A-Za-z][A-Za-z0-9]{0,4})\b`)
)

// tools are the programs whose first words name what a call does. A
// command head is the program and its first one or two subcommand words.
var tools = map[string]int{
	"git": 2, "go": 2, "hug": 3, "huginn": 3, "muninn": 3, "jcode": 2, "gh": 3, "npm": 2, "pnpm": 2,
	"yarn": 2, "cargo": 2, "docker": 2, "kubectl": 2, "make": 2, "brew": 2, "pip": 2, "uv": 2, "terraform": 2, "systemctl": 2, "launchctl": 2, "tmux": 2,
	"backlog": 3, "bun": 2, "deno": 2,
}

// common file names say nothing about a project on their own.
var common = map[string]bool{"README.md": true, "go.mod": true, "go.sum": true, "package.json": true, "main.go": false}

// Memory is the anchors a memory's text names: backticked commands, paths
// and file names. URLs are not paths.
func Memory(text string) []string {
	set := map[string]bool{}
	for _, m := range backticked.FindAllStringSubmatch(text, -1) {
		if h := head(m[1]); h != "" {
			set["cmd:"+h] = true
		}
	}
	for _, p := range paths(text) {
		segs := segments(p)
		set["path:"+strings.Join(segs, "/")] = true
		if f := segs[len(segs)-1]; strings.Contains(f, ".") && !common[f] {
			set["file:"+f] = true
		}
	}
	for _, m := range fileLike.FindAllStringSubmatch(text, -1) {
		if f := m[1]; !common[f] && !strings.Contains(f, "..") {
			set["file:"+f] = true
		}
	}
	return sorted(set)
}

// Call is the anchors of one tool call: its command heads and the paths it
// names or touches, made relative to root (the project) or home.
func Call(tool, command string, touched []string, root, home string) []string {
	return CallNear(tool, command, touched, root, home, false)
}

// CallNear is Call; with near, a path run must end at the path itself or at
// its folder, so a memory about a far ancestor ("Projects/Brovs/.jcode")
// does not match every file under it.
func CallNear(tool, command string, touched []string, root, home string, near bool) []string {
	set := map[string]bool{}
	var named []string
	if command != "" {
		for _, step := range steps(command) {
			if h := head(step); h != "" {
				set["cmd:"+h] = true
				if f := strings.Fields(h); len(f) == 3 {
					set["cmd:"+f[0]+" "+f[1]] = true
				}
			}
		}
		named = paths(command)
	}
	for _, p := range append(named, touched...) {
		if p = relative(p, root, home); p == "" {
			continue
		}
		segs := segments(p)
		if len(segs) == 0 {
			continue
		}
		if f := segs[len(segs)-1]; strings.Contains(f, ".") && !common[f] {
			set["file:"+f] = true
		}
		for i := 0; i < len(segs); i++ {
			for j := i + 2; j <= len(segs); j++ {
				if near && j < len(segs)-1 {
					continue
				}
				set["path:"+strings.Join(segs[i:j], "/")] = true
			}
		}
	}
	return sorted(set)
}

// paths are the paths a text names: two or more segments, not a URL, not
// after a $ (a variable), and either rooted (/, ./, ~/) or ending in /, three or more
// segments long, or with a dot in a segment, so "origin/main" and
// "and/or" are not paths.
func paths(text string) []string {
	text = url.ReplaceAllString(text, " ")
	var out []string
	for _, loc := range pathLike.FindAllStringIndex(text, -1) {
		if loc[0] > 0 && strings.ContainsRune("${", rune(text[loc[0]-1])) {
			continue
		}
		p := strings.TrimRight(text[loc[0]:loc[1]], ".")
		segs := segments(p)
		rooted := strings.HasPrefix(p, "/") || strings.HasPrefix(p, "./") || strings.HasPrefix(p, "~/") || strings.HasSuffix(p, "/")
		if len(segs) >= 2 && (rooted || len(segs) >= 3 || strings.Contains(p, ".")) {
			out = append(out, p)
		}
	}
	return out
}

// relative is p inside root or home, "" for root or home itself.
func relative(p, root, home string) string {
	p = strings.TrimRight(filepath.ToSlash(p), "/")
	for _, base := range []string{root, home} {
		if base == "" {
			continue
		}
		base = strings.TrimRight(filepath.ToSlash(filepath.Clean(base)), "/")
		if p == base {
			return ""
		}
		if strings.HasPrefix(p, base+"/") {
			return strings.TrimPrefix(p, base+"/")
		}
	}
	return p
}

// segments splits a path into its parts, without ".", "..", "~" or empty.
func segments(p string) []string {
	var out []string
	for _, s := range strings.Split(strings.Trim(p, "/"), "/") {
		if s == "" || s == "." || s == ".." || s == "~" || strings.HasPrefix(s, "$") {
			continue
		}
		out = append(out, s)
	}
	return out
}

// steps splits a shell command into its simple commands.
func steps(cmd string) []string {
	r := strings.NewReplacer("&&", "\n", "||", "\n", ";", "\n", "|", "\n", "$(", "\n", "`", "\n")
	return strings.Split(r.Replace(cmd), "\n")
}

// head is a simple command's program and subcommands ("go test",
// "hug task done"), after env assignments and wrappers; "" when the program
// is not one whose subcommands say something.
func head(step string) string {
	f := strings.Fields(step)
	for len(f) > 0 {
		w := f[0]
		switch {
		case strings.Contains(w, "=") && !strings.HasPrefix(w, "-"):
			f = f[1:]
		case w == "nice" || w == "time" || w == "env" || w == "sudo" || w == "exec" || w == "command":
			f = f[1:]
			for len(f) > 0 && (strings.HasPrefix(f[0], "-") || isNumber(f[0])) {
				f = f[1:]
			}
		default:
			goto done
		}
	}
done:
	if len(f) == 0 {
		return ""
	}
	prog := filepath.Base(f[0])
	n, ok := tools[prog]
	if !ok {
		return ""
	}
	out := []string{prog}
	for _, w := range f[1:] {
		if len(out) >= n || strings.HasPrefix(w, "-") || !isWord(w) {
			break
		}
		out = append(out, w)
	}
	if n > 1 && len(out) == 1 {
		return "" // "git" alone says nothing
	}
	return strings.Join(out, " ")
}

func isWord(w string) bool {
	if w == "" {
		return false
	}
	for _, r := range w {
		if !(r >= 'a' && r <= 'z' || r == '-' || r == '_') {
			return false
		}
	}
	return true
}

func isNumber(w string) bool {
	for _, r := range w {
		if r < '0' || r > '9' {
			return false
		}
	}
	return w != ""
}

func sorted(set map[string]bool) []string {
	if len(set) == 0 {
		return nil
	}
	out := make([]string, 0, len(set))
	for k := range set {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
