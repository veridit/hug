package task

import (
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"testing"
)

// TestSeveralQuestionsOnRealBoard replays every question ever asked on a
// real board (HUG_REAL_BOARD = a .hug/tasks checkout) through the check-4
// rule and prints which it would refuse; a manual probe, skipped in CI.
func TestSeveralQuestionsOnRealBoard(t *testing.T) {
	dir := os.Getenv("HUG_REAL_BOARD")
	if dir == "" {
		t.Skip("HUG_REAL_BOARD not set")
	}
	out, err := exec.Command("git", "-C", dir, "log", "-p", "--format=", "--", "*.md").Output()
	if err != nil {
		t.Fatal(err)
	}
	seen := map[string]bool{}
	field := regexp.MustCompile(`^\+(?:asked_what|waiting_ask_what|design_question): (".*")$`)
	block := regexp.MustCompile(`^\+  > (.*)$`)
	var qs []string
	for _, l := range strings.Split(string(out), "\n") {
		if m := field.FindStringSubmatch(l); m != nil {
			if q, e := strconv.Unquote(m[1]); e == nil {
				qs = append(qs, q)
			}
		} else if m := block.FindStringSubmatch(l); m != nil {
			qs = append(qs, m[1])
		}
	}
	refused := 0
	for _, q := range qs {
		if seen[q] {
			continue
		}
		seen[q] = true
		if severalQuestions(q) != nil {
			refused++
			t.Logf("REFUSE: %s", q)
		}
	}
	t.Logf("%d distinct questions, %d refused (%s)", len(seen), refused, filepath.Base(dir))
}
