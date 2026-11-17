---
id: mv4vye
title: "TUI detail: visible focus, clear prompts, honest key hints"
state: "done"
type: feature
priority: "high"
assignee: ["Jørgen H. Fjeld"]
labels: []
created: 2026-11-16
updated: 2026-11-16
---

## Goal
Inside a task the reader always sees where focus is, scrolling behaves predictably, and a question from the tool cannot be missed

## Purpose
Jørgen 2026-11-16 review of the detail view: the heading 'ACCEPTANCE · [ / ] SELECT' made him press / (which opens search); Tab moved focus between acceptance and dependencies with nothing showing where focus went, and the tab bar still showed [overview] as if focus were there; the 'Confirm edit ID title?' line is bland, so he did not notice he was being asked something

## Principle
Show only keys that work, written so they cannot be misread. Focus is always visible in exactly one place. A prompt that waits for an answer is the most prominent thing on screen. Compact: no extra lines. Esc always backs out

## Relevant details
Code: huginn/internal/tui/detail.go (section headings, detailPane 0 = acceptance, 1 = dependencies), tui.go (detail key handling, j/k step through the focused list then scroll), render.go (confirm prompt near 'if m.confirm', tab bar near the views loop, keyBar in keys.go). Wanted behaviour: three focus states in detail: page (j/k scroll the page), acceptance (j/k step through checks), dependencies (j/k step through them); Tab cycles and skips an empty section; the focused section heading is highlighted and the others are not; with page focus no check carries the > pointer. Write the keys as '[ and ]' or similar, never '[ / ]'. The tab bar must not mark a view as current while a task is open; show the task as current instead (for example 'overview › s6ff5k'). The confirm line uses a strong style (warning colour or reverse) and starts with the question. Already shipped in v0.7.15: { and } step to the previous/next task, Space ticks only in detail, tick prompt reads 'Tick: ☐ 1. text'. Goldens: HUGINN_UPDATE_GOLDEN=1 go test -run TestTmuxGolden ./huginn/internal/tui/ then review the diff

## Depends on

## Acceptance
- [x] no heading or hint writes keys as '[ / ]'; pressing every key shown in a hint does what the hint says (test)
- [x] Tab cycles page, acceptance, dependencies and skips empty sections; the focused heading is highlighted and only it
- [x] with page focus j/k scroll the page and no check shows the pointer; with acceptance focus j/k step through checks
- [x] while a task is open the tab bar does not mark overview/list/board as focused
- [x] every confirm prompt is visually prominent and reads as a question; test asserts the style differs from normal text
- [x] full tests pass niced; goldens regenerated and reviewed

## Notes

2026-11-16 Jørgen, second screenshot: the bland-prompt problem is general. The tick prompt ('Tick item 2 of s6ff5k: ...?') is as easy to miss as the edit one. Every confirm line needs the prominent style, not just edit. Since v0.7.15 the tick prompt reads 'Tick: ☐ 2. text' (render.go, the m.confirm block); keep that wording and make the whole line stand out.

2026-11-16 Jørgen: { and } are hard on a Norwegian layout. Task stepping in detail moves to ',' (previous) and '.' (next), both unbound before; { and } are removed, Ctrl-P/Ctrl-N stay. Sent to the worker as an addition to this task.

2026-11-16 Jørgen, supersedes the ',' and '.' note: in detail n = next task, p = previous task; new task moves to N in every view; { and } removed; Ctrl-N/Ctrl-P stay; in lists n/p move the selection like j/k. Sent to the worker.

2026-11-16 Jørgen on the v0.7.15 board: compact rows are 'a lot better'. Addition: the legend must connect to the column labels, e.g. 'T type: F feature ... · P priority: C critical ...', with T and P styled like the header; narrow fallback 'T: ... · P: ...'. Sent to the worker.

2026-11-16 Jørgen, supersedes the 'T type:' legend note: the legend reads 'Type: F feature ... · Priority: C critical ...' with the initial T and P emphasised (bold or the column-header style), no separate letter before the word; narrow fallback 'T: ... · P: ...'. Sent to the worker.

2026-11-16 Jørgen on v0.7.15: 'Tick: ☐ 1. y' is better but needs more pop. Made concrete for check 5: the confirm line is a full-width bar in a strong background, carries the answer keys (Enter yes, Esc no), and stands out without colour too. Also: an empty priority renders as '[]' in the detail header; do not render empty badges. Both sent to the worker.

check 1 (e6f84c0): go test -run TestFocusHintsAreHonest ./huginn/internal/tui/ -> '--- PASS: TestFocusHintsAreHonest (0.51s)'. Asserts no '[ / ]' or '{ / }' in View at 80 and 140 in all three focus states nor in help, then presses every key detailHints shows in each focus and checks its effect. Headings now read '▸ ACCEPTANCE · j/k or [ ] step · Space ticks' (golden 80x24-no-color/detail.txt).

check 2 (e6f84c0): '--- PASS: TestFocusTabCyclesAndHighlightsOneHeading (0.08s)'. Tab: acceptance->deps->page->acceptance, Shift-Tab reverse; task without deps cycles page/acceptance; task without checks opens on page and Tab goes deps->page. Exactly one '▸ ' heading with list focus, zero with page focus (then the trail id is bracketed); colour: focus style only on the focused heading.

check 3 (e6f84c0): '--- PASS: TestFocusPageScrollsAcceptanceSteps (0.03s)'. Acceptance focus: j j -> check 3, k -> check 2. Page focus: no line starts with '>', no selected-tint SGR in detail text, j j down -> YOffset+3 with the check unchanged, k up -> +1; Space on page focus gives no confirm, message 'Space ticks the check marked >: Tab focuses the checks'.

check 4 (e6f84c0): '--- PASS: TestFocusTabBarShowsTrailInTask (0.03s)'. In a task no '[overview]'/'[list]'/'[board]'/'[!lint]'; tab bar reads ' overview    list    board    !lint   list › 000001' (golden 80x24-no-color/detail.txt line 2), 'list › A › B' after following a dependency, '[list]' back after Esc; View still has m.height lines.

check 5 (e6f84c0): '--- PASS: TestFocusConfirmPromptsStandOut (0.17s)'. For all 8 actionNames (t d x J K space e n): the line above the key bar starts '▶ ' and ends '?', differs from styles.normal.Render of its text, and carries a 48;2; background and bold SGR under TrueColor. Wording: 'Tick: ☐ 1. one', 'Confirm new Brand new?', 'Confirm take <id> Main?'. At width 30 the '?' survives truncation.

2026-11-16 Jørgen: the edit prompt does not say which editor it opens. Added to check 5: the prompt names the command from editor.Command() ('Edit qawmnb in subl --wait?'), and help says how to choose it ($VISUAL, then $EDITOR, then git's core.editor, then vi). On his machine ~/.zshrc line 108 sets EDITOR='subl --wait' and git core.editor is 'zed --new --wait'. Sent to the worker.

check 6 (e6f84c0): 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./huginn/...' -> 10 ok lines, last 'ok  	github.com/veridit/hug/huginn/internal/tui	22.268s', no FAIL. Goldens: HUGINN_UPDATE_GOLDEN=1 regenerated 10 files (detail, markdown-warning x4 sizes/colour modes, help x2 at 140x40); diff reviewed: tab bar trail, focused acceptance heading, n/p footer, focus-aware key bar, help rows for [ and ] / n and p / N / Tab. No unintended change.

Choices (worker hog): (1) page focus is shown by bracketing the task id in the tab-bar trail ('list › [id]'), so focus is always in exactly one place; (2) h/l/left/right keep cycling focus like Tab; (3) [ and ] with page focus do nothing and say 'Tab focuses the checks or dependencies'; (4) Space with page/deps focus does not tick (no check is marked) and says why; (5) confirm line is '▶ … ?' in a bold amber bar (ask style), the '?' kept on truncation; Enter/Esc stay in the key bar below, not repeated on the line; (6) following a dependency and Esc back re-apply the start focus.

Additions (274f3d1): check 5 extended per notes: confirm line is '▶ <question>? Enter yes · Esc no', full width (test asserts width == m.width) and bold amber background; edit prompt reads 'Edit <id> in subl --wait?' with EDITOR='subl --wait' (TestFocusConfirmPromptsStandOut, '--- PASS: TestFocusConfirmPromptsStandOut (0.16s)'); help row 'e  edit in $VISUAL, else $EDITOR, git core.editor, vi'. Board legend 'Type: F feature … · Priority: C critical …' (T and P bold+underlined), narrow 'T: … · P: …' ('--- PASS: TestBoardLegendNamesColumns (0.02s)', golden 80x24-no-color/board.txt). Detail header skips unset type/priority badges ('--- PASS: TestDetailHeaderHasNoEmptyBadge (0.02s)'). Full suite: 'GOMAXPROCS=2 GOFLAGS=-p=2 nice -n 19 go test -count=1 ./huginn/...' -> 10 ok, last 'ok  	github.com/veridit/hug/huginn/internal/tui	21.929s'.
