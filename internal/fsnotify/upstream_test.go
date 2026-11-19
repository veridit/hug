//go:build darwin || linux

package fsnotify

import (
	"os/exec"
	"testing"
)

// TestUpstreamDiff holds the copy to its reviewed diff from fsnotify v1.10.1
// (README.md): upstream.patch applied to v1.10.1 gives the copy byte for
// byte. It compares code, not diff output, whose hunk layout differs between
// GNU and BSD diff. It needs v1.10.1 in the module cache or the network.
func TestUpstreamDiff(t *testing.T) {
	if testing.Short() {
		t.Skip("downloads fsnotify v1.10.1 when it is not in the module cache")
	}
	out, err := exec.Command("bash", "upstream-diff.sh", "--check").CombinedOutput()
	if err != nil {
		t.Fatalf("%v\n%s", err, out)
	}
}
