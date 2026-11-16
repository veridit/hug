package tui

import "testing"

func keysOf(names []string, reserved string) string {
	out := ""
	for _, s := range assignShortcuts(names, reserved) {
		if s.Pos < 0 {
			out += "-"
			continue
		}
		out += string(s.Key)
	}
	return out
}

// Each choice gets its first letter still free, in list order.
func TestAssignShortcuts(t *testing.T) {
	for _, tc := range []struct {
		name     string
		names    []string
		reserved string
		want     string
	}{
		{"statuses", []string{"todo", "doing", "blocked", "done", "dropped"}, "", "tdbor"},
		{"views", []string{"overview", "list", "board", "lint"}, "", "olbi"},
		// A leading sigil is the label's own key (2026-11-16, user).
		{"views with a sigil", []string{"overview", "list", "board", "!lint"}, "", "olb!"},
		{"a taken sigil falls back to a letter", []string{"!lint", "!later"}, "", "!l"},
		{"punctuation inside a name is skipped", []string{"a", "a-b"}, "", "ab"},
		{"types", []string{"feature", "bug", "chore", "spike", "docs"}, "", "fbcsd"},
		{"priorities", []string{"critical", "high", "medium", "low"}, "", "chml"},
		// The same letters the f filter picker already uses.
		{"filter fields", []string{"status", "type", "priority", "group", "label", "epic", "assignee"}, "", "stpglea"},
		{"reserved keys are skipped", []string{"quit", "query"}, "q", "ue"},
		{"upper case and spaces", []string{"Done", "do it"}, "", "do"},
		{"nothing left", []string{"aa", "a"}, "", "a-"},
	} {
		if got := keysOf(tc.names, tc.reserved); got != tc.want {
			t.Errorf("%s: got %q, want %q", tc.name, got, tc.want)
		}
	}
	// Pos points at the marked letter, counted in runes.
	got := assignShortcuts([]string{"doing", "done", "ødelagt", "øde"}, "")
	if got[1].Pos != 1 || got[1].Key != 'o' {
		t.Errorf("done: %+v, want o at 1", got[1])
	}
	if got[3].Pos != 2 || got[3].Key != 'e' {
		t.Errorf("øde: %+v, want e at rune 2", got[3])
	}
}
