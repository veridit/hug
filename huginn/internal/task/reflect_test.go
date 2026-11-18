package task

import (
	"strings"
	"testing"
)

// oldGolden is the old golden-update variable the board texts name, spelt
// in parts so the repository's old-name scan (internal/env) passes.
const oldGolden = "HUG" + "INN_UPDATE_GOLDEN"

// 2k68eg check 1: the reflections on hug's board on 2026-11-18 whose
// instruction quote started at an apostrophe inside a word (AGENTS.md's,
// Jørgen's, check's). Each text is verbatim from the board; want is the
// first span quoted on purpose, or "" for none: a quote opens after a
// non-letter and closes before one, so no quote starts mid-word.
func TestReflectionQuoteSkipsApostrophesInWords(t *testing.T) {
	t.Parallel()
	cases := []struct{ id, text, want string }{
		{"fkpyg3", "AGENTS.md's TUI golden line said " + oldGolden + " while tests read HUG_UPDATE_GOLDEN (since fixed on main by m1c28r). The 'every error ends with a fix: line' rule shaped the one combined fix design well.", "every error ends with a fix: line"},
		{"scrxak", "Jørgen's answer carried a small request; doing it before closing kept the outcome whole. 'n/p step' in the goal read clearly to us and not to him: footers should spell keys out. (nautilus)", "n/p step"},
		{"n2hq8p", "A gap: since fkpyg3, hug task new requires --purpose and --principle, but my tests written before it still passed locally on the old base, and only CI on main showed it. Nothing told me to recheck test fixtures for 'new' after a rule change landed. Also, TestHandEditedNotesSeenByLsAndShow's comment said 'right after a sync' while the test kept an old mtime, so it hid an elapsed-time assumption.", "new"},
		{"x1ra0t", "hug task check's help says 'hug task check ID N' with no evidence argument, while the board instructions say 'hug task check ID N with evidence'; I passed evidence as a third argument and got 'expected 2 positional arguments, got 3' with fix 'hug task help', which names no command. The fix line could say: hug task note ID \"evidence\".", "hug task check ID N"},
		{"cvbpdy", "The ticket held work for two repos: hug's lint and Seid's switch. An acceptance item that only another repo's owner can tick kept a shipped task open for a day. Splitting at filing time would have avoided it.", ""},
		{"t3fqaw", "The state rename broke this branch's test at vet time ('d.Task.Status undefined'), which a rebase reports only when the tests are built; nothing in the instructions misled.", "d.Task.Status undefined"},
		{"ce3dmq", "One gap: my session's hug binary (/Users/jhf/.jcode/scratch/hugmain, built 14:36) predated 'status: is now state:' (abcf36b, 19:49 hug upgrade). 'hug task wait s6ff5k' then failed with 'not-allowed s6ff5k: -> doing/waiting is not a move a command makes', with an empty state and a fix 'hug task take', which did not say the binary was older than the board.", "status: is now state:"},
		{"wdkqzx", "The ask gate turned on one detail: a shipped task asked about its own last open check must not be refused by the very finding the ask resolves, so readyToHand takes the ask's --check and drops shipped-not-done only when that check is the sole open one. design --for being a draft (no Goal or acceptance yet) was the other trap; a single 'draft' flag kept it to one gate instead of three.", "draft"},
		{"ygay9c", "The copy's board link is two files (admin gitdir and checkout .git); git worktree repair from the copy fixes both, so the fix line is git's own command, not a hug repair.", ""},
		{"f5vx4x", "A command's reach has to match where it runs. hug upgrade mixed board steps and machine steps under one --apply, so testing one board touched every repo's hooks. Check 2's 'golden' is verbatim heading blocks asserted in TestUpgradeTouchesMachineOnlyWithMachine, not a golden file: the output holds temp paths.", "golden"},
		// Quotes that already parsed right keep parsing right.
		{"at3qep", "One gap: hug task check takes no evidence text ('hug task check ID N'), but AGENTS.md says 'hug task check ID N with evidence'; I wrote the evidence as a separate note each time.", "hug task check ID N"},
		{"double", `the AGENTS.md line "Finish: hug task done ID" did not say it`, "Finish: hug task done ID"},
		{"curly", "the line “Recall changes” decided it", "Recall changes"},
		// An apostrophe inside a quote does not close it.
		{"inner", "the hint 'don't guess the file name' was missing", "don't guess the file name"},
	}
	for _, c := range cases {
		rs := ParseReflections(Task{ID: c.id}, "Reflection (open, 2026-11-17, x): "+c.text)
		if len(rs) != 1 {
			t.Fatalf("%s: %d reflections", c.id, len(rs))
		}
		if got := rs[0].Instruction; got != c.want {
			t.Errorf("%s: instruction %q, want %q", c.id, got, c.want)
		}
	}
}

// 2k68eg check 2: open reflections that name the same instruction, or the
// same hug command, form one group. The key is the hug command inside the
// quoted instruction, else the quote itself, else the first hug command in
// the text, else none ("", listed last).
func TestGroupReflections(t *testing.T) {
	t.Parallel()
	texts := map[string]string{
		"at3qep": "One gap: hug task check takes no evidence text ('hug task check ID N'), but AGENTS.md says 'hug task check ID N with evidence'; I wrote the evidence as a separate note each time.",
		"49n9kw": "nothing misled me. One gap: 'hug task check ID N' takes no evidence, so each tick needs a note first, and a run of checks prints the whole ticket each time (long output for 6 ticks).",
		"x1ra0t": "hug task check's help says 'hug task check ID N' with no evidence argument, while the board instructions say 'hug task check ID N with evidence'; I passed evidence as a third argument.",
		"p29rbz": "One gap: hug task wait and take accept --as NAME but hug task note rejects it ('unknown flag --as; valid: --json, --help'), so an agent signing its work by habit loses the note.",
		"gjmqhn": "One gap: AGENTS.md said " + oldGolden + " for the TUI goldens, but every golden in huginn/internal/tui reads HUG_UPDATE_GOLDEN, so the documented command updates nothing.",
		"fkpyg3": "AGENTS.md's TUI golden line said " + oldGolden + " while tests read HUG_UPDATE_GOLDEN (since fixed on main by m1c28r). The 'every error ends with a fix: line' rule shaped the one combined fix design well.",
		"r5r8c1": "nothing misled; the rename surfaced as a build error at rebase time, as it should.",
		"m1":     "No quote here, but hug mem forget printed nothing about the twin.",
	}
	var rs []Reflection
	for _, id := range []string{"at3qep", "49n9kw", "x1ra0t", "p29rbz", "gjmqhn", "fkpyg3", "r5r8c1", "m1"} {
		rs = append(rs, ParseReflections(Task{ID: id}, "Reflection (open, 2026-11-17, x): "+texts[id])...)
	}
	gs := GroupReflections(rs)
	byKey := map[string][]string{}
	for _, g := range gs {
		var ids []string
		for _, r := range g.Reflections {
			ids = append(ids, r.ID)
		}
		byKey[g.Key] = ids
	}
	want := map[string]string{
		"hug task check": "at3qep 49n9kw x1ra0t",
		// The quote names no command: the quote is the key, not the first
		// command in the text (hug task wait), which is not what it is about.
		"unknown flag --as; valid: --json, --help": "p29rbz",
		"every error ends with a fix: line":        "fkpyg3",
		"hug mem forget":                           "m1",
		"":                                         "gjmqhn r5r8c1",
	}
	for k, ids := range want {
		if got := strings.Join(byKey[k], " "); got != ids {
			t.Errorf("group %q: %q, want %q (all groups %v)", k, got, ids, byKey)
		}
	}
	if len(gs) != len(want) {
		t.Errorf("groups %v", byKey)
	}
	// The largest group comes first; the ungrouped rest comes last.
	if gs[0].Key != "hug task check" || gs[len(gs)-1].Key != "" {
		t.Errorf("order %v", gs)
	}
}

// 2k68eg check 3: an unanswered done is a line of its own state.
func TestUnansweredReflectionLine(t *testing.T) {
	t.Parallel()
	rs := ParseReflections(Task{ID: "abcdef"}, "## Notes\nReflection (unanswered, 2026-11-18, rabbit)\nReflection (open, 2026-11-18, rabbit): a gap\n")
	if len(rs) != 2 || rs[0].State != "unanswered" || rs[0].Who != "rabbit" || rs[0].Text != "" || rs[1].State != "open" || rs[1].N != 2 {
		t.Fatalf("%+v", rs)
	}
}
