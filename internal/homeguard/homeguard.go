// Package homeguard keeps tests away from the real machine's harness
// configs (ynh84t incident, 2026-11-19 00:32: an upgrade test rewrote seven
// live jcode and Claude configs). Run wraps a package's tests: it points
// HOME, USERPROFILE, JCODE_HOME, XDG_CONFIG_HOME and CODEX_HOME at a temp
// folder before any test runs, hashes the real configs hug installs into
// before and after, and fails the run, naming the files and never their
// contents, if one changed.
package homeguard

import (
	"bufio"
	"crypto/sha256"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// Files are the real configs a hug test must never change: the ones hug's
// installers and hug upgrade write, and each jcode instance's config.toml
// and mcp.json that ~/.zshrc registers (JCODE_INSTANCES).
func Files(home string) []string {
	files := []string{
		filepath.Join(home, ".jcode", "config.toml"),
		filepath.Join(home, ".jcode", "mcp.json"),
		filepath.Join(home, ".claude", "settings.json"),
		filepath.Join(home, ".claude", "settings.local.json"),
		filepath.Join(home, ".claude.json"),
		filepath.Join(home, ".codex", "config.toml"),
		filepath.Join(home, ".codex", "hooks.json"),
		filepath.Join(home, "Library", "Application Support", "Claude", "claude_desktop_config.json"),
		filepath.Join(home, ".config", "Claude", "claude_desktop_config.json"),
	}
	if h := os.Getenv("JCODE_HOME"); h != "" {
		files = append(files, filepath.Join(h, "config.toml"), filepath.Join(h, "mcp.json"))
	}
	for _, root := range instances(filepath.Join(home, ".zshrc"), home) {
		files = append(files, filepath.Join(root, ".jcode", "config.toml"), filepath.Join(root, ".jcode", "mcp.json"))
	}
	return files
}

var instanceLine = regexp.MustCompile(`^\s*\[?([A-Za-z0-9_.-]+)\]?=?\s+"?([^"\s)]+)"?`)

// guarded is set while a guarded run's tests run. Not HUG_*: hug refuses
// a HUG_ variable it does not read.
const guarded = "HOMEGUARD_TMP"

// instances reads the roots in a zsh JCODE_INSTANCES=( name root ... ) block.
func instances(zshrc, home string) []string {
	f, err := os.Open(zshrc)
	if err != nil {
		return nil
	}
	defer f.Close()
	var out []string
	in := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		l := sc.Text()
		if strings.Contains(l, "JCODE_INSTANCES=(") {
			in = true
			continue
		}
		if !in {
			continue
		}
		if strings.HasPrefix(strings.TrimSpace(l), ")") {
			break
		}
		if m := instanceLine.FindStringSubmatch(l); m != nil {
			root := m[2]
			for _, pre := range []string{"~", "$HOME", "${HOME}"} {
				if root == pre || strings.HasPrefix(root, pre+"/") {
					root = home + root[len(pre):]
					break
				}
			}
			out = append(out, filepath.Clean(root))
		}
	}
	return out
}

// hashes is each file's sha256, or "absent".
func hashes(files []string) map[string]string {
	out := map[string]string{}
	for _, p := range files {
		f, err := os.Open(p)
		if err != nil {
			out[p] = "absent"
			continue
		}
		h := sha256.New()
		io.Copy(h, f)
		f.Close()
		out[p] = fmt.Sprintf("%x", h.Sum(nil))
	}
	return out
}

// goDirs pins GOPATH, GOMODCACHE and GOCACHE to the values go derives
// from the real home, unless already set.
func goDirs(home string) {
	gopath := os.Getenv("GOPATH")
	if gopath == "" {
		gopath = filepath.Join(home, "go")
		os.Setenv("GOPATH", gopath)
	}
	if os.Getenv("GOMODCACHE") == "" {
		os.Setenv("GOMODCACHE", filepath.Join(filepath.SplitList(gopath)[0], "pkg", "mod"))
	}
	if os.Getenv("GOCACHE") == "" {
		if c, err := os.UserCacheDir(); err == nil {
			os.Setenv("GOCACHE", filepath.Join(c, "go-build"))
		}
	}
}

// Run is a package's TestMain body: os.Exit(homeguard.Run(m)).
func Run(m *testing.M) int { return Guard(m.Run) }

// Guard runs tests with HOME and the harness homes in a temp folder and
// fails, naming the files, if a real config changed meanwhile.
func Guard(tests func() int) int {
	// A test binary re-run as a child of a guarded test keeps the env its
	// parent's test set: the parent's guard already covers its writes.
	if os.Getenv(guarded) != "" {
		return tests()
	}
	real, err := os.UserHomeDir()
	if err != nil {
		fmt.Fprintln(os.Stderr, "homeguard:", err, "\nfix: set HOME before go test")
		return 1
	}
	files := Files(real)
	before := hashes(files)
	tmp, err := os.MkdirTemp("", "hug-test-home-")
	if err != nil {
		fmt.Fprintln(os.Stderr, "homeguard:", err)
		return 1
	}
	defer os.RemoveAll(tmp)
	for _, k := range []string{"HOME", "USERPROFILE", "JCODE_HOME", "XDG_CONFIG_HOME", "CODEX_HOME", guarded, "GOCACHE", "GOMODCACHE", "GOPATH"} {
		if v, ok := os.LookupEnv(k); ok {
			defer os.Setenv(k, v)
		} else {
			defer os.Unsetenv(k)
		}
	}
	// Tests that build hug keep the real Go caches: under a fresh HOME a
	// go build starts cold (minutes on a loaded machine).
	goDirs(real)
	for _, k := range []string{"HOME", "USERPROFILE"} {
		os.Setenv(k, tmp)
	}
	os.Setenv("JCODE_HOME", filepath.Join(tmp, ".jcode"))
	os.Setenv("XDG_CONFIG_HOME", filepath.Join(tmp, ".config"))
	os.Setenv("CODEX_HOME", filepath.Join(tmp, ".codex"))
	os.Setenv(guarded, tmp)
	code := tests()
	after := hashes(files)
	var changed []string
	for _, p := range files {
		if before[p] != after[p] {
			changed = append(changed, p)
		}
	}
	if len(changed) > 0 {
		fmt.Fprintf(os.Stderr, "homeguard: these tests changed %d of this machine's real configs:\n  %s\nfix: find the test that reads os.UserHomeDir or the process HOME instead of its own; the backups hug made sit next to each file as FILE.bak-hug-upgrade or FILE.*-bak-*\n", len(changed), strings.Join(changed, "\n  "))
		return 1
	}
	return code
}
