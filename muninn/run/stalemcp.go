package run

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/veridit/hug/muninn/internal/config"
	"github.com/veridit/hug/muninn/internal/harness"
)

// staleMCPScan looks at the machine's hug mcp servers (z77pem); tests
// replace it so a machine's own servers never show.
var staleMCPScan = harness.ScanMCP

// staleMCPListed says this system lists processes the way the scan reads
// them (ps, lsof, /proc): macOS and Linux. Elsewhere the first turn does
// nothing at all, no marker and no scan (crab: Windows costs nothing).
var staleMCPListed = runtime.GOOS == "darwin" || runtime.GOOS == "linux"

// staleMCPMarker says session sid was checked for stale hug mcp servers.
func staleMCPMarker(sid string) string {
	return filepath.Join(config.DataDir(), "stale-mcp", sid)
}

// staleMCPOnce returns the notice for session sid the first time the session
// asks, "" after. It runs on a turn's prompt (the recall path), never on a
// tool call: one ps (and one lsof on macOS) per session. The agent hears
// only of the server serving its own session (jcodeHome for jcode; else the
// app this hook runs under), the one it can reload; every stale server goes
// to hook.log. Markers older than a week are pruned.
func staleMCPOnce(ctx context.Context, sid, jcodeHome string) string {
	if !staleMCPListed || sid == "" || strings.ContainsAny(sid, `/\`) || sid == "." || sid == ".." {
		return ""
	}
	marker := staleMCPMarker(sid)
	if _, err := os.Stat(marker); err == nil {
		return ""
	}
	dir := filepath.Dir(marker)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return ""
	}
	// The marker first: two hooks of one session never both say it.
	f, err := os.OpenFile(marker, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0o600)
	if err != nil {
		return ""
	}
	f.Close()
	if ents, err := os.ReadDir(dir); err == nil {
		for _, e := range ents {
			if info, err := e.Info(); err == nil && time.Since(info.ModTime()) > 7*24*time.Hour {
				os.Remove(filepath.Join(dir, e.Name()))
			}
		}
	}
	sc := staleMCPScan(ctx)
	own, others := sc.Own(jcodeHome)
	if l := harness.StaleList(own); l != "" {
		logHook("%s: stale hug mcp serving this session: %s", short(sid), l)
	}
	if l := harness.StaleList(others); l != "" {
		logHook("%s: stale hug mcp of other apps (not told to this agent): %s", short(sid), l)
	}
	return harness.OwnStaleMessage(own)
}

// withStaleMCP puts the notice before what recall found.
func withStaleMCP(notice, block string) string {
	switch {
	case notice == "":
		return block
	case block == "":
		return notice
	}
	return notice + "\n\n" + block
}
