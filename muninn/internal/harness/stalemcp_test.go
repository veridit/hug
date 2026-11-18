package harness

import (
	"strings"
	"testing"
)

// z77pem: what Jørgen's machine showed on 2026-11-18, judged from ps and
// lsof output: servers on a Cellar hug older than v0.7.54 are named with
// their app; newer ones, and paths with no version, are not.
func TestJudgeStaleMCP(t *testing.T) {
	ps := []byte(` 3168     1 /Users/jhf/ssb/.jcode/builds/shared-server/jcode serve
 3175  3168 /opt/homebrew/bin/hug mcp --jcode-home /Users/jhf/ssb/.jcode
18560 54786 /opt/homebrew/bin/hug mcp --jcode-home /Users/jhf/.jcode
51988     1 /Applications/Claude.app/Contents/MacOS/Claude
77273 77272 /opt/homebrew/bin/hug mcp
77272 51988 /Applications/Claude.app/Contents/Helpers/disclaimer --pgroup -- /bin/sh -c exec /opt/homebrew/bin/hug mcp
12550     1 /Users/jhf/Veridit/src/seid-lang/.jcode/builds/shared-server/jcode serve
90712 12550 hug mcp
 4000     1 hug mcp
 4001     1 /usr/bin/vim hug mcp
 4002     1 /opt/homebrew/bin/hug task ls
`)
	procs := ParsePS(ps)
	servers := MCPServers(procs, 18560)
	var pids []int
	for _, s := range servers {
		pids = append(pids, s.PID)
	}
	if len(pids) != 4 || pids[0] != 3175 || pids[1] != 77273 || pids[2] != 90712 || pids[3] != 4000 {
		t.Fatalf("hug mcp servers (self 18560 left out): %v", pids)
	}
	lsof := []byte("p3175\nftxt\nn/opt/homebrew/Cellar/hug/0.7.2/bin/hug\nftxt\nn/usr/lib/dyld\np77273\nftxt\nn/opt/homebrew/Cellar/hug/0.7.43/bin/hug\np90712\nftxt\nn/opt/homebrew/Cellar/hug/0.7.54/bin/hug\np4000\nftxt\nn/hug\n")
	exe := ParseLsofTxt(lsof)
	if exe[3175] != "/opt/homebrew/Cellar/hug/0.7.2/bin/hug" {
		t.Fatalf("lsof: the first txt is the executable, got %q", exe[3175])
	}
	// 4000: macOS names a removed file by its base name only; started on
	// a hug that reloads (56357 on 2026-11-18, 13:08, after brew cleanup)
	// or not, nothing says, so it is not named.
	stale := JudgeMCP(servers, procs, exe)
	var got []string
	for _, s := range stale {
		got = append(got, s.Version+"|"+s.App)
	}
	want := []string{"0.7.2|jcode /Users/jhf/ssb/.jcode", "0.7.43|Claude"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("stale: %v, want %v (0.7.54 reloads itself)", got, want)
	}
	// Linux: a removed Cellar file keeps its path and version.
	if l := JudgeMCP([]Proc{{PID: 4000, PPID: 1, Args: "hug mcp"}}, nil, map[int]string{4000: "/home/linuxbrew/.linuxbrew/Cellar/hug/0.7.14/bin/hug (deleted)"}); len(l) != 1 || l[0].Version != "0.7.14" {
		t.Fatalf("a deleted 0.7.14 on Linux: %+v", l)
	}
	stale = append(stale, StaleMCP{PID: 4000, Version: "0.7.14", Parents: []int{1}})
	// A person sees all of them (hook.log, hug mem stats).
	if l := StaleList(stale); l != "pid 3175 hug 0.7.2 in jcode /Users/jhf/ssb/.jcode; pid 77273 hug 0.7.43 in Claude; pid 4000 hug 0.7.14" {
		t.Fatalf("list: %q", l)
	}
	// The agent hears only of its own: the ssb jcode session of 3175.
	sc := MCPScan{Stale: stale}
	own, others := sc.Own("/Users/jhf/ssb/.jcode/")
	if len(own) != 1 || own[0].PID != 3175 || len(others) != 2 {
		t.Fatalf("ssb session: own %+v, others %+v", own, others)
	}
	msg := OwnStaleMessage(own)
	if msg != "hug: your hug mcp server (pid 3175) runs hug 0.7.2, older than v0.7.54: it cannot reload itself, so its calls run old rules and can fail (e.g. memory_remember: cannot modify graph_state because it is a view)\nfix: mcp action=reload server=hug" {
		t.Fatalf("own message:\n%s", msg)
	}
	// hug's own jcode home serves 18560 (0.7.66): nothing for its agent.
	if own, _ := sc.Own("/Users/jhf/.jcode"); len(own) != 0 || OwnStaleMessage(own) != "" {
		t.Fatalf("a current session is told of others: %+v", own)
	}
	// Claude Code (no jcode home): its server is the one whose app (above
	// the disclaimer wrapper) is an ancestor of the hook.
	sc.Self = []int{99001, 99000, 51988, 1}
	own, _ = sc.Own("")
	if len(own) != 1 || own[0].PID != 77273 || !strings.HasSuffix(OwnStaleMessage(own), "\nfix: restart hug's MCP server in Claude (quit and reopen it)") {
		t.Fatalf("Claude session: %+v\n%s", own, OwnStaleMessage(own))
	}
	// Not identified (no home, no ancestor in common, or only pid 1, which
	// every process has: the orphaned 4000 is nobody's): nothing.
	sc.Self = []int{99001, 1}
	if own, _ := sc.Own(""); len(own) != 0 {
		t.Fatalf("an unknown session got %+v", own)
	}
}

// A server whose parent is jcode but has no --jcode-home (seid's bare
// "hug mcp") is named by the jcode home in its parent's path.
func TestStaleMCPAppFromParent(t *testing.T) {
	s := Proc{PID: 90712, PPID: 12550, Args: "hug mcp"}
	p := Proc{PID: 12550, Args: "/Users/jhf/Veridit/src/seid-lang/.jcode/builds/shared-server/jcode serve"}
	if got := appOf(s, p); got != "jcode /Users/jhf/Veridit/src/seid-lang/.jcode" {
		t.Fatalf("app: %q", got)
	}
	if !versionBefore("0.7.9", "0.7.54") || versionBefore("0.7.54", "0.7.54") || versionBefore("0.8.0", "0.7.54") {
		t.Fatal("versions compare as numbers")
	}
}
