package main

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"testing"
	"time"

	sdk "github.com/modelcontextprotocol/go-sdk/mcp"
)

// builtHugs caches one build per version for the package's tests: each rig
// copies it, so every binary is its own file (its own inode).
var (
	buildsMu  sync.Mutex
	buildsDir string
	builds    = map[string]string{}
)

// cachedHug builds hug at version once and copies it to out.
func cachedHug(t *testing.T, out, version string) {
	t.Helper()
	buildsMu.Lock()
	src, ok := builds[version]
	if !ok {
		if buildsDir == "" {
			d, err := os.MkdirTemp("", "hug-test-builds-")
			if err != nil {
				buildsMu.Unlock()
				t.Fatal(err)
			}
			buildsDir = d
		}
		src = filepath.Join(buildsDir, version, "hug"+exeSuffix())
		os.MkdirAll(filepath.Dir(src), 0o755)
		buildHug(t, src, version)
		builds[version] = src
	}
	buildsMu.Unlock()
	b, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	os.MkdirAll(filepath.Dir(out), 0o755)
	if err := os.WriteFile(out, b, 0o755); err != nil {
		t.Fatal(err)
	}
}

func cleanBuilds() {
	if buildsDir != "" {
		os.RemoveAll(buildsDir)
	}
}

func exeSuffix() string {
	if runtime.GOOS == "windows" {
		return ".exe"
	}
	return ""
}

// brewRig lays hug out as Homebrew does: each version in its own keg
// (Cellar/hug/X/bin/hug) and bin/hug a relative symlink to the linked one.
// The app starts bin/hug mcp, as hug apps install writes it.
type brewRig struct {
	t      *testing.T
	dir    string
	link   string // dir/bin/hug
	stderr *lockedBuf
	cmd    *exec.Cmd
	cs     *sdk.ClientSession
	ctx    context.Context
}

func keg(dir, v string) string { return filepath.Join(dir, "Cellar", "hug", v, "bin", "hug"+exeSuffix()) }

func mcpEnv(dir, home string) []string {
	return append(os.Environ(), "HOME="+home, "USERPROFILE="+home, "HUG_HOME="+filepath.Join(dir, "mem"), "JCODE_HOME="+filepath.Join(home, "no-jcode"),
		"HUG_NO_EMBED=1", "HUG_SYNC=", "GOMAXPROCS=2", "GIT_CONFIG_NOSYSTEM=1")
}

func gitRepo(t *testing.T, repo string) {
	t.Helper()
	for _, args := range [][]string{{"init", "-q", repo}, {"-C", repo, "config", "user.email", "t@example.com"}, {"-C", repo, "config", "user.name", "t"}} {
		if o, err := exec.Command("git", args...).CombinedOutput(); err != nil {
			t.Skipf("git unavailable: %v %s", err, o)
		}
	}
}

// startMCP starts bin mcp in a throwaway repository and home and connects a
// client to it, as an app does.
func startMCP(t *testing.T, dir, bin string) (*exec.Cmd, *sdk.ClientSession, *lockedBuf, context.Context) {
	t.Helper()
	repo, home := filepath.Join(dir, "repo"), filepath.Join(dir, "home")
	gitRepo(t, repo)
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	cmd := exec.Command(bin, "mcp")
	cmd.Dir = repo
	cmd.Env = mcpEnv(dir, home)
	stderr := &lockedBuf{}
	cmd.Stderr = stderr
	cs, err := sdk.NewClient(&sdk.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, &sdk.CommandTransport{Command: cmd}, nil)
	if err != nil {
		t.Fatalf("connect: %v", err)
	}
	t.Cleanup(func() { cs.Close() })
	return cmd, cs, stderr, ctx
}

func newBrewRig(t *testing.T) *brewRig {
	if testing.Short() {
		t.Skip("builds the hug binary twice")
	}
	dir := t.TempDir()
	r := &brewRig{t: t, dir: dir, link: filepath.Join(dir, "bin", "hug"+exeSuffix())}
	cachedHug(t, keg(dir, "0.0.1"), "0.0.1-old")
	os.MkdirAll(filepath.Dir(r.link), 0o755)
	if err := os.Symlink(filepath.Join("..", "Cellar", "hug", "0.0.1", "bin", "hug"+exeSuffix()), r.link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	r.cmd, r.cs, r.stderr, r.ctx = startMCP(t, dir, r.link)
	return r
}

// upgrade does what brew upgrade does: pour the new keg, move the bin/hug
// symlink to it (ln -sf: remove, then link), and clean up the old keg.
func (r *brewRig) upgrade() {
	r.t.Helper()
	cachedHug(r.t, keg(r.dir, "0.0.2"), "0.0.2-new")
	if err := os.Remove(r.link); err != nil {
		r.t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join("..", "Cellar", "hug", "0.0.2", "bin", "hug"+exeSuffix()), r.link); err != nil {
		r.t.Fatal(err)
	}
	if err := os.RemoveAll(filepath.Join(r.dir, "Cellar", "hug", "0.0.1")); err != nil {
		r.t.Fatal(err)
	}
}

func callVersion(ctx context.Context, cs *sdk.ClientSession, name string) (string, error) {
	res, err := cs.CallTool(ctx, &sdk.CallToolParams{Name: name, Arguments: map[string]any{}})
	if err != nil {
		return "", err
	}
	v, _ := res.GetMeta()["hug/version"].(string)
	return v, nil
}

// waitFor polls s until it holds want, at most d.
func waitFor(s *lockedBuf, want string, d time.Duration) bool {
	for deadline := time.Now().Add(d); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		if strings.Contains(s.String(), want) {
			return true
		}
	}
	return strings.Contains(s.String(), want)
}

// rvb3j5 check 1: brew upgrades hug under an idle hug mcp (new keg, symlink
// moved, old keg removed). The client sends nothing; within 60 s the same
// process runs the new hug, and it answers on the same pipes.
func TestMCPReloadsWhileIdleAfterBrewUpgrade(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no exec on Windows: an idle server there reloads at its next request (TestMCPStopsForRestartWithoutExec)")
	}
	r := newBrewRig(t)
	pid := r.cmd.Process.Pid
	if v, err := callVersion(r.ctx, r.cs, "task_list"); err != nil || v != "0.0.1-old" {
		t.Fatalf("before the upgrade: version %q, err %v", v, err)
	}
	start := time.Now()
	r.upgrade()
	// Nothing is sent: the server must notice by itself.
	if !waitFor(r.stderr, "reloaded as hug 0.0.2-new (was 0.0.1-old), same connection", 60*time.Second) {
		t.Fatalf("no reload within 60 s of the upgrade, with no request sent; stderr:\n%s", r.stderr)
	}
	t.Logf("reloaded %.1f s after the upgrade, idle", time.Since(start).Seconds())
	if exe := runningExe(t, pid); exe != "" && !strings.Contains(exe, filepath.Join("Cellar", "hug", "0.0.2")) {
		t.Fatalf("pid %d runs %s, want the 0.0.2 keg", pid, exe)
	}
	if v, err := callVersion(r.ctx, r.cs, "task_list"); err != nil || v != "0.0.2-new" {
		t.Fatalf("after the idle reload: version %q, err %v; stderr:\n%s", v, err, r.stderr)
	}
	if r.cmd.Process.Pid != pid {
		t.Fatal("pid changed")
	}
}

// runningExe is the file pid runs, as the system names it ("" when it
// cannot say): lsof on macOS, /proc on Linux.
func runningExe(t *testing.T, pid int) string {
	switch runtime.GOOS {
	case "linux":
		p, _ := os.Readlink("/proc/" + strconv.Itoa(pid) + "/exe")
		return p
	case "darwin":
		out, _ := exec.Command("lsof", "-a", "-p", strconv.Itoa(pid), "-d", "txt", "-Fn").Output()
		for _, l := range strings.Split(string(out), "\n") {
			if strings.HasPrefix(l, "n") {
				return l[1:]
			}
		}
	}
	return ""
}

// rvb3j5: hug upgrade (or a person) can nudge a hug mcp with SIGHUP. It
// does not kill the server: an unchanged hug says so and serves on.
func TestMCPHangupKeepsServing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no SIGHUP on Windows")
	}
	r := newBrewRig(t)
	if err := r.cmd.Process.Signal(syscall.SIGHUP); err != nil {
		t.Fatal(err)
	}
	if !waitFor(r.stderr, "is unchanged", 10*time.Second) {
		t.Fatalf("no answer to SIGHUP; stderr:\n%s", r.stderr)
	}
	if v, err := callVersion(r.ctx, r.cs, "task_list"); err != nil || v != "0.0.1-old" {
		t.Fatalf("after SIGHUP: version %q, err %v; stderr:\n%s", v, err, r.stderr)
	}
}
