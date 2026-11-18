//go:build darwin || linux

package fsnotify

import (
	"os/exec"
	"testing"
)

// TestUpstreamDiff holds the copy to its reviewed diff from fsnotify v1.10.1
// (README.md): the four import lines and the PR 789 hunks, nothing else.
// It needs v1.10.1 in the module cache or the network to fetch it.
func TestUpstreamDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("downloads fsnotify v1.10.1 when it is not in the module cache")
	}
	out, err := exec.Command("bash", "upstream-diff.sh", "--check").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
