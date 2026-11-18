package task

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

// DepsPolicy reads the project's dependency policy, defaulting to block.
// Unrelated TOML keys are left to their owning components.
func DepsPolicy(root string) (string, error) {
	return readPolicy(root, "deps")
}

// AcceptancePolicy defaults to block, mirroring DepsPolicy.
func AcceptancePolicy(root string) (string, error) { return readPolicy(root, "acceptance") }

// ConfigName is the task settings file at the root of the task branch.
// The old name huginn.toml is still read; hug upgrade renames it.
const ConfigName = "hug.toml"

func configPath(root string) string {
	p := filepath.Join(root, ConfigName)
	if _, e := os.Stat(p); e != nil {
		if old := filepath.Join(root, "huginn.toml"); fileExists(old) {
			return old
		}
	}
	return p
}

func fileExists(p string) bool { _, e := os.Stat(p); return e == nil }

func readPolicy(root, field string) (string, error) {
	f, e := os.Open(configPath(root))
	if os.IsNotExist(e) {
		return "block", nil
	}
	if e != nil {
		return "", e
	}
	defer f.Close()
	scan := bufio.NewScanner(f)
	section := ""
	policy := "block"
	for scan.Scan() {
		line := strings.TrimSpace(scan.Text())
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		if section != "" {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok || strings.TrimSpace(key) != field {
			continue
		}
		value = scalar(value)
		if !contains([]string{"block", "warn"}, value) {
			e := Invalid(field, value, []string{"block", "warn"})
			e.Fix = "set " + field + " = \"block\" in " + ConfigName
			return "", e
		}
		policy = value
	}
	return policy, scan.Err()
}

// ForbidCite lists path prefixes that task text must not cite, because those
// folders get deleted (hug.toml: forbid_cite = [".jcode/scratch/"]).
func ForbidCite(root string) ([]string, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return nil, nil
	}
	if e != nil {
		return nil, e
	}
	section := ""
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if section != "" || !ok || strings.TrimSpace(key) != "forbid_cite" {
			continue
		}
		value = strings.TrimSpace(value)
		value = strings.TrimSuffix(strings.TrimPrefix(value, "["), "]")
		var out []string
		for _, v := range strings.Split(value, ",") {
			if v = scalar(v); v != "" {
				out = append(out, v)
			}
		}
		return out, nil
	}
	return nil, nil
}

// Board layout (dx9vw2). A board says which layout its task files use with a
// top-level layout = N in hug.toml. A board without the key is BaseLayout:
// from before the key, its task files may still say status:. Layout 2 is
// the state: board (6ephqk step 4): hug upgrade moves every task to state:,
// then writes layout = 2, and hug task init writes it on a new board.
// Layout is the newest this hug reads and writes. A hug that meets a newer
// board refuses to write it (it would write what it can read, and lose the
// rest) and says so once on a read. Raise Layout when task files change in
// a way an older hug misreads, and have hug upgrade move the board to it.
const (
	BaseLayout = 1
	// Layout 3 (w40cmk): every task at C/ID.md, rank: and group: in the
	// front matter.
	Layout = 3
)

// BoardLayout reads the board's layout from hug.toml at root.
func BoardLayout(root string) (int, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return BaseLayout, nil
	}
	if e != nil {
		return 0, e
	}
	return LayoutOf(string(b))
}

// LayoutOf reads the layout from hug.toml's text ("" is BaseLayout), so
// a hug.toml at a git ref reads the same as the one on disk (w40cmk check
// 9: sync compares the layout at HEAD and at the remote).
func LayoutOf(text string) (int, error) {
	section := ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "[") {
			section = line
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if section != "" || !ok || strings.TrimSpace(key) != "layout" {
			continue
		}
		n, e := strconv.Atoi(scalar(value))
		if e != nil || n < 1 {
			return 0, &Error{Kind: "invalid", Field: "layout", Value: strings.TrimSpace(value),
				Fix: fmt.Sprintf("set layout = %d in %s on the task branch, or brew upgrade veridit/tap/hug if a newer hug wrote it", Layout, ConfigName)}
		}
		return n, nil
	}
	return BaseLayout, nil
}

// WithLayout returns hug.toml text with the top-level layout set to n,
// everything else unchanged. A new key goes first: top-level keys must come
// before any [table].
func WithLayout(text string, n int) string {
	entry := fmt.Sprintf("layout = %d", n)
	lines := strings.Split(strings.TrimRight(text, "\n"), "\n")
	if text == "" {
		return entry + "\n"
	}
	for i, l := range lines {
		t := strings.TrimSpace(l)
		if strings.HasPrefix(t, "[") {
			break
		}
		if key, _, ok := strings.Cut(t, "="); ok && strings.TrimSpace(key) == "layout" {
			lines[i] = entry
			return strings.Join(lines, "\n") + "\n"
		}
	}
	if strings.HasPrefix(strings.TrimSpace(lines[0]), "[") {
		entry += "\n"
	}
	return entry + "\n" + strings.Join(lines, "\n") + "\n"
}
