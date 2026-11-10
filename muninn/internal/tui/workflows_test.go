package tui

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/x/ansi"
	"github.com/veridit/muninn/internal/app"
	"github.com/veridit/muninn/internal/config"
	"github.com/veridit/muninn/internal/ingest"
	"github.com/veridit/muninn/internal/search"
	"github.com/veridit/muninn/internal/store"
	"github.com/veridit/muninn/internal/tidy"
)

func TestPaletteJumpAndFilterChips(t *testing.T) {
	a := testApp(t)
	var tm tea.Model = startModel(t, a, 140, 40)
	tm = typeText(t, tm, ":category pref")
	m := tm.(model)
	if len(m.completions) != 1 || m.completions[0] != "category preference" {
		t.Fatalf("value completion: %v", m.completions)
	}
	tm = keypress(t, tm, tea.KeyEnter)
	if tm.(model).focus != paletteInput || tm.(model).input.Value() != "category preference" {
		t.Fatal("first Enter must complete")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = typeText(t, tm, "fc")
	if !strings.Contains(tm.View(), "fact") || !strings.Contains(tm.View(), "3") {
		t.Fatal("counted categories missing")
	}
	tm = keypress(t, tm, tea.KeyEnter)
	if len(tm.(model).chips) != 1 || !strings.Contains(tm.View(), "[category:fact]") {
		t.Fatal("filter chip missing")
	}
	tm = typeText(t, tm, "/banana")
	tm = keypress(t, tm, tea.KeyEnter)
	if len(tm.(model).filtered(tm.(model).results)) != 1 {
		t.Fatal("search did not combine with chip")
	}
	tm = keypress(t, tm, tea.KeyBackspace)
	if len(tm.(model).chips) != 0 {
		t.Fatal("Backspace did not remove last chip")
	}
	tm = typeText(t, tm, "F3")
	global, _ := tm.(model).binding.Placement("", "global")
	if len(tm.(model).chips) != 1 || tm.(model).chips[0].value != global.Scope {
		t.Fatal("global shortcut")
	}
	tm = typeText(t, tm, "4")
	if len(tm.(model).chips) != 0 {
		t.Fatal("all shortcut did not remove place chip")
	}
	m = tm.(model)
	u := m.snapshot.units[0]
	tm = typeText(t, tm, "#"+u.SID[:3])
	if len(tm.(model).jumpIDs) == 0 || !strings.Contains(tm.View(), u.SID) {
		t.Fatal("id prefix completion missing")
	}
	tm = keypress(t, tm, tea.KeyEnter)
	if tm.(model).screen != detail || tm.(model).unit.SID != u.SID {
		t.Fatal("jump Enter did not open")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = typeText(t, tm, fmt.Sprintf("#%d", u.ID))
	tm = keypress(t, tm, tea.KeyEnter)
	if tm.(model).unit.SID != u.SID {
		t.Fatal("numeric jump did not open")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = typeText(t, tm, ":search q")
	if tm.(model).input.Value() != "search q" {
		t.Fatal("q quit from input")
	}
}
func TestDetailsRenderMarkdownFixedHeaderAndFullText(t *testing.T) {
	a := testApp(t)
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	body := "<!-- SECTION:TEXT:BEGIN -->\n# Heading\n\n**Bold** and *italic*, `code`, [site](https://example.invalid). <b>HTML</b>\n\n- [ ] first\n- [x] second\n\n```go\nfmt.Println(\"hello\")\n```\n" + strings.Repeat("Long line with 界 🦉 é words and readable wrapping.\n", 60) + "FINAL TAIL SENTINEL"
	mems[0].Content = body
	if err = ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	for _, size := range [][2]int{{80, 24}, {140, 40}, {200, 60}} {
		for _, no := range []string{"", "1"} {
			t.Run(fmt.Sprintf("%dx%d/no=%s", size[0], size[1], no), func(t *testing.T) {
				t.Setenv("NO_COLOR", no)
				m := startModel(t, a, size[0], size[1])
				m.openUnit(mems[0].SID)
				if m.unit.SID == "" {
					u, _ := a.DB.Resolve(mems[0].ID)
					m.openUnit(u.SID)
				}
				var tm tea.Model = m
				v := ansi.Strip(tm.View())
				for _, bad := range []string{"<!--", "SECTION:TEXT", "**Bold**", "`code`", "<b>"} {
					if strings.Contains(v, bad) {
						t.Fatalf("raw markup %q:\n%s", bad, v)
					}
				}
				for _, good := range []string{"[fact]", "[strength 1]", "Heading", "Bold"} {
					if !strings.Contains(v, good) {
						t.Fatalf("missing %q:\n%s", good, v)
					}
				}
				tm = typeText(t, tm, "G")
				for i := 0; i < 20 && !strings.Contains(flat(tm.(model).viewport.View()), "FINAL TAIL SENTINEL"); i++ {
					tm = typeText(t, tm, "k")
				}
				if !strings.Contains(flat(tm.(model).viewport.View()), "FINAL TAIL SENTINEL") || !strings.Contains(ansi.Strip(tm.View()), "[strength 1]") {
					t.Fatalf("full text not reachable or header scrolled away:\n%s", ansi.Strip(tm.View()))
				}
				for _, scene := range []tea.Model{m, tm, typeText(t, tm, "?"), typeText(t, m, ":"), typeText(t, m, "fc"), typeText(t, m, "#")} {
					lines := strings.Split(scene.View(), "\n")
					if len(lines) != size[1] {
						t.Fatalf("height=%d want %d", len(lines), size[1])
					}
					for i, line := range lines {
						if ansi.StringWidth(line) > size[0] {
							t.Fatalf("line %d width %d > %d", i, ansi.StringWidth(line), size[0])
						}
					}
				}
			})
		}
	}
}
func TestCorrectionIsConfirmedAndPreservesHistory(t *testing.T) {
	a := testApp(t)
	m := startModel(t, a, 140, 40)
	m.showList("Recent", m.allResults())
	old := m.results[0].Unit
	before, _ := os.ReadFile(app.NativePath())
	var tm tea.Model = typeText(t, m, "eThe corrected deployment policy uses a written staging checklist.")
	tm = keypress(t, tm, tea.KeyCtrlS)
	now, _ := os.ReadFile(app.NativePath())
	if !bytes.Equal(before, now) {
		t.Fatal("edited before owner confirmed")
	}
	if tm.(model).focus != confirmInput {
		t.Fatal("correction confirmation missing")
	}
	tm = typeText(t, tm, "y")
	if strings.Contains(tm.(model).status, "error:") {
		t.Fatal(tm.(model).status)
	}
	new := tm.(model).unit
	if new.SID == old.SID || new.Category != "correction" {
		t.Fatalf("not a new correction: %+v", new)
	}
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	foundOld, foundNew := false, false
	for _, mem := range mems {
		if mem.ID == old.Key {
			foundOld = true
			if mem.Content != old.Body || mem.Active || !mem.Updated.Equal(old.Updated) {
				t.Fatal("old text or date was rewritten")
			}
		}
		if mem.CaptureSupersedes == old.SID {
			foundNew = true
		}
	}
	if !foundOld || !foundNew {
		t.Fatal("correction lineage missing")
	}
	backups, _ := filepath.Glob(filepath.Join(config.DataDir(), "tidy", "correction-*", "files.json"))
	if len(backups) != 1 {
		t.Fatal("correction backup missing")
	}
	info, _ := os.Stat(backups[0])
	if info.Mode().Perm() != 0600 {
		t.Fatal("backup not private")
	}
}
func TestTidyReviewSkipAndApplyUseExistingBackup(t *testing.T) {
	a := testApp(t)
	m := startModel(t, a, 140, 40)
	u := m.snapshot.units[0]
	target := "personal"
	p := tidy.Plan{Version: 1, Database: a.DB.Path, Created: time.Now(), Entries: []tidy.Entry{{SID: u.SID, Origin: u.Origin, Fingerprint: tidy.Fingerprint(u), Source: u.Scope, Target: target, Decision: "unsure"}}}
	dir := filepath.Join(config.DataDir(), "tidy")
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	raw, _ := json.Marshal(p)
	if err := os.WriteFile(filepath.Join(dir, "plan-test.json"), raw, 0600); err != nil {
		t.Fatal(err)
	}
	m = startModel(t, a, 140, 40)
	m.startReview("moves")
	if m.screen != review {
		t.Fatalf("review missing: %s", m.status)
	}
	var tm tea.Model = typeText(t, m, "n")
	got, _ := a.DB.Resolve(u.SID)
	if got.Scope != u.Scope {
		t.Fatal("skip wrote memory")
	}
	m = startModel(t, a, 140, 40)
	m.startReview("moves")
	tm = typeText(t, m, "y")
	got, _ = a.DB.Resolve(u.SID)
	if got.Scope != target {
		t.Fatalf("apply did not move: %s", tm.(model).status)
	}
	receipts, _ := filepath.Glob(filepath.Join(dir, "apply-*", "receipt.json"))
	if len(receipts) != 1 {
		t.Fatal("existing tidy backup/receipt missing")
	}
}
func TestSearchIgnoresStaleResponses(t *testing.T) {
	a := testApp(t)
	m := startModel(t, a, 80, 24)
	m.screen = listing
	m.filter = "latest"
	m.searchSeq = 2
	old := resultsMsg{seq: 1, query: "older", res: []search.Result{{Unit: m.snapshot.units[0]}}}
	tm := drive(t, m, old).(model)
	if len(tm.results) != 0 {
		t.Fatal("stale search overwrote results")
	}
}

func TestBatchedTerminalInput(t *testing.T) {
	a := testApp(t)
	var tm tea.Model = startModel(t, a, 140, 40)
	packet := func(s string) { tm = drive(t, tm, tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune(s)}) }
	packet(":category pref")
	if tm.(model).input.Value() != "category pref" || tm.(model).completions[0] != "category preference" {
		t.Fatal("palette prefix was swallowed in a PTY packet")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	packet("fc")
	if tm.(model).focus != filterInput || tm.(model).filterField != "category" {
		t.Fatal("quick-filter field was swallowed in a PTY packet")
	}
	tm = keypress(t, tm, tea.KeyEnter)
	packet("F/banana")
	if len(tm.(model).chips) != 0 || tm.(model).input.Value() != "banana" || len(tm.(model).results) != 1 {
		t.Fatal("find did not combine command prefix and one input packet")
	}
	packet("?q")
	if !tm.(model).help || tm.(model).input.Value() != "banana" {
		t.Fatal("help entered the input or accepted unrelated help-overlay keys")
	}
	tm = keypress(t, tm, tea.KeyEsc)
	tm = keypress(t, tm, tea.KeyEsc)
	packet("gg")
	if tm.(model).cursor != 0 {
		t.Fatal("gg navigation was swallowed in a PTY packet")
	}
}

func TestFindCancelPreservesChipsAndPreviewPaging(t *testing.T) {
	a := testApp(t)
	mems, err := ingest.LoadNative(app.NativePath())
	if err != nil {
		t.Fatal(err)
	}
	mems[1].Category = "preference"
	if err = ingest.SaveNative(app.NativePath(), mems); err != nil {
		t.Fatal(err)
	}
	if _, err = a.Reindex(context.Background()); err != nil {
		t.Fatal(err)
	}
	var tm tea.Model = startModel(t, a, 140, 40)
	tm = typeText(t, tm, "fc")
	tm = keypress(t, tm, tea.KeyEnter)
	tm = typeText(t, tm, "/banana")
	tm = keypress(t, tm, tea.KeyEsc)
	m := tm.(model)
	if len(m.chips) != 1 || m.filter != "" || len(m.filtered(m.results)) != 2 || m.focus != noInput {
		t.Fatal("cancelling text find must retain the category chip and restore its list")
	}
	tm = typeText(t, tm, "/")
	tm = keypress(t, tm, tea.KeyEnter)
	tm = keypress(t, tm, tea.KeyBackspace)
	m = tm.(model)
	if len(m.filtered(m.results)) != 3 {
		t.Fatal("removing a chip after an empty find must restore every visible row")
	}
	m.screen = preview
	m.block = strings.Repeat("A readable recall preview line.\n", 100)
	m.sizeViewport()
	m.viewport.SetContent(wrapText(m.block, m.width-4))
	tm = typeText(t, m, "G")
	if !tm.(model).viewport.AtBottom() {
		t.Fatal("G did not scroll recall preview to the bottom")
	}
	tm = typeText(t, tm, "gg")
	if !tm.(model).viewport.AtTop() {
		t.Fatal("gg did not scroll recall preview to the top")
	}
	for _, screen := range []screen{review, preview} {
		m = tm.(model)
		m.screen = screen
		for _, action := range []string{"e", "m", "x"} {
			got := typeText(t, m, action).(model)
			if got.focus != noInput {
				t.Fatal("review/preview must not edit a stale, invisible list selection")
			}
		}
	}
}

func TestJumpExactIDPrecedesPrefixes(t *testing.T) {
	m := newModel(testApp(t))
	m.snapshot.units = []store.Unit{{ID: 17850, SID: "1785qr"}, {ID: 1785, SID: "k7m2ab"}}
	m.input.SetValue("1785")
	m.jumpChoices()
	if len(m.jumpIDs) != 2 || m.jumpIDs[0] != "k7m2ab" {
		t.Fatal("exact numeric id must win over longer row numbers and SID prefixes")
	}
	m.snapshot.units = append(m.snapshot.units, store.Unit{ID: 12, SID: "1785"})
	m.jumpChoices()
	if m.jumpIDs[0] != "1785" {
		t.Fatal("an exact stable id must win over a numeric row id")
	}
}

func TestDetailsBadgesKeepFixedFieldsWithLongPlace(t *testing.T) {
	m := newModel(testApp(t))
	m.width = 80
	m.unit = store.Unit{SID: "k7m2ab", Scope: "folder:/very/long/place", Category: "preference", Updated: time.Date(2026, 11, 10, 12, 0, 0, 0, time.Local)}
	m.snapshot.places = []placeCount{{Place: app.Place{Name: strings.Repeat("long-place-", 20), Scope: m.unit.Scope}}}
	m.metadata.strength = 7
	badges := ansi.Strip(m.detailBadges())
	for _, required := range []string{"[preference]", "[2026-11-10]", "[k7m2ab]", "[strength 7]"} {
		if !strings.Contains(badges, required) {
			t.Fatalf("fixed badge %s hidden: %s", required, badges)
		}
	}
	if ansi.StringWidth(badges) > m.width-2 {
		t.Fatal("detail badges overflow the pane")
	}
}

func TestTUIHelpQuitAndLongMovePicker(t *testing.T) {
	m := startModel(t, testApp(t), 80, 24)
	m.help = true
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Fatal("q outside input must still quit from help")
	}
	if _, ok := cmd().(tea.QuitMsg); !ok {
		t.Fatal("help q did not quit")
	}
	m.busy = true
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("help must not abandon a confirmed write")
	}
	m.busy = false
	m.focus = findInput
	_, cmd = m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd != nil {
		t.Fatal("q must not quit while an input retains focus behind help")
	}
	m.help = false
	m.focus = moveInput
	for i := range 40 {
		m.pending.places = append(m.pending.places, placeCount{Place: app.Place{Name: fmt.Sprintf("destination-%02d", i), Scope: fmt.Sprintf("scope-%02d", i)}})
	}
	m.pending.choice = 39
	view := ansi.Strip(m.View())
	if !strings.Contains(view, "destination-39") || strings.Contains(view, "destination-00") {
		t.Fatal("move picker must keep its selected destination in view")
	}
}

func TestTUIEscUnwindsOpenScreensBeforeUnderlyingFilters(t *testing.T) {
	base := startModel(t, testApp(t), 140, 40)
	base.chips = []chip{{field: "category", value: "fact"}}
	base.filter = "banana"
	base.screen = listing
	base.title = "Find"
	base.results = base.allResults()
	for _, open := range []screen{review, preview} {
		m := base
		m.push()
		m.screen = open
		m.title = "Open review or preview"
		m = keypress(t, m, tea.KeyEsc).(model)
		if m.screen != listing || m.filter != "banana" || len(m.chips) != 1 {
			t.Fatal("Esc must close the open screen before clearing its underlying query or chips")
		}
		m = keypress(t, m, tea.KeyEsc).(model)
		if m.filter != "" || len(m.chips) != 1 {
			t.Fatal("the next Esc must clear only the query")
		}
		m = keypress(t, m, tea.KeyEsc).(model)
		if len(m.chips) != 0 {
			t.Fatal("the next Esc must remove the chip")
		}
	}
	base.command("overview")
	base = keypress(t, base, tea.KeyEsc).(model)
	if len(base.chips) != 0 {
		t.Fatal("Esc must not leave an active overview filter open")
	}
}
