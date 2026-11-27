package task

import (
	"os"
	"regexp"
	"strconv"
	"strings"
)

// The step due now (erv680, Jørgen 2026-11-26: "the system must tell you
// what to do when it is due, whatever the agent or its instructions").
// StepOf derives it from the task's state alone, nothing memorised: one
// step, for one role, with one command. The CLI's next: line, the MCP
// result's next and the board cue (huginn/board, muninn recall) all read
// it, so the three say the same thing. hug names the stage, never a
// project's procedure: how to ship comes from the project (hug.toml
// [board] ship, else its AGENTS.md).

// Step kinds, in the order a task's life meets them.
const (
	StepTake   = "take"   // todo: someone takes it
	StepChecks = "checks" // doing with no acceptance check: the owner writes them
	StepCheck  = "check"  // doing with an open check: the owner works and ticks it
	StepReview = "review" // in review: the reviewer answers
	StepDone   = "done"   // every check holds: the owner closes it (5yh7hr: review starts here)
	StepShip   = "ship"   // a reviewer approved: ship it, the project's way
	StepAnswer = "answer" // every check holds, a question is open: its answer comes first
	StepNone   = "next"   // done or dropped: the next task
)

// Step is what is due on a task now. For is who it is due for (an
// assignee, the reviewer, the coordinator); "" anyone. Next is the next:
// line, naming exactly one hug command; Cue is the board's line for For,
// ending in Next.
type Step struct {
	Kind string `json:"kind"`
	For  string `json:"for,omitempty"`
	Next string `json:"next"`
	Cue  string `json:"cue,omitempty"`
}

// StepConfig is what a step reads from the project's hug.toml [board]:
// the coordinator (who ships an approved task) and the ship line (how).
type StepConfig struct {
	Coordinator, Ship string
}

// StepConfigOf reads root's hug.toml [board] coordinator and ship.
func StepConfigOf(root string) StepConfig {
	if root == "" {
		return StepConfig{}
	}
	c, _ := Coordinator(root)
	s, _ := ShipLine(root)
	return StepConfig{Coordinator: strings.TrimSpace(c), Ship: s}
}

// ShipLine is hug.toml [board] ship, or "" when unset (erv680 check 5):
// the project's own way to ship an approved task, in its words, e.g.
//
//	[board]
//	ship = "merge the branch to main"
//
// hug says it as written; it never names a procedure of its own.
func ShipLine(root string) (string, error) {
	b, e := os.ReadFile(configPath(root))
	if os.IsNotExist(e) {
		return "", nil
	}
	if e != nil {
		return "", e
	}
	in := false
	for _, line := range strings.Split(string(b), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		if strings.HasPrefix(line, "[") {
			in = strings.TrimSpace(strings.Trim(line, "[]")) == "board"
			continue
		}
		if key, value, ok := strings.Cut(line, "="); in && ok && strings.TrimSpace(key) == "ship" {
			return strings.Join(strings.Fields(scalar(value)), " "), nil
		}
	}
	return "", nil
}

// ShipHow is how the ship step says to ship: the project's ship line, or
// its AGENTS.md.
func ShipHow(line string) string {
	if line = strings.TrimSpace(line); line != "" {
		return "as hug.toml says: " + line
	}
	return "as your AGENTS.md says"
}

// AcceptFix is the command that writes a task's first acceptance check.
func AcceptFix(id string) string { return "hug task set " + id + ` --accept "observable check"` }

// NextFor is the next: line for who: the step's own command when the
// step is who's, or due for anyone; "hug task next" when it waits on
// another role (a review, ship it): the owner does other work meanwhile,
// and never answers its own review. who "" is anyone.
func (s Step) NextFor(who string) string {
	if (s.Kind == StepReview || s.Kind == StepShip) && s.For != "" && strings.TrimSpace(who) != "" && !strings.EqualFold(strings.TrimSpace(who), s.For) {
		return "hug task next"
	}
	return s.Next
}

// StepAt is StepOf with root's hug.toml, read only when the step needs it
// (an approved task): every write's next: goes through here.
func StepAt(t Task, root string) Step {
	var c StepConfig
	if t.Approval != nil {
		c = StepConfigOf(root)
	}
	return StepOf(t, c)
}

// StepOf is the step due on t now (see Step).
func StepOf(t Task, c StepConfig) Step {
	owner := ""
	if len(t.Assignee) > 0 {
		owner = t.Assignee[0]
	}
	title := oneLineShort(t.Title, 60)
	switch t.State {
	case "todo":
		return Step{Kind: StepTake, Next: "hug task take " + t.ID}
	case "doing":
	default:
		return Step{Kind: StepNone, Next: "hug task next"}
	}
	// A task taken with no acceptance check (erv680 check 1): its checks
	// are its work items (Jørgen 2026-11-26: one list), so they come
	// first; done would have nothing to hold.
	if len(t.Acceptance) == 0 {
		next := AcceptFix(t.ID)
		return Step{Kind: StepChecks, For: owner, Next: next,
			Cue: t.ID + " (" + title + ") has no acceptance checks: write them first, one observable check each, they are its work items. " + next}
	}
	// In review the reviewer's answer is due, whatever is open: the owner
	// asked for it (6w0gve).
	if ownHolds(t, "review") {
		by := strings.TrimSpace(t.Fields["review_by"])
		next := "hug task answer " + t.ID + ` "looks right"`
		who := owner
		if who == "" {
			who = "its owner"
		}
		return Step{Kind: StepReview, For: by, Next: next,
			Cue: who + " asks you to review " + t.ID + " (" + title + "): read it, then answer looks right, or what to change: " + next}
	}
	for n, a := range t.Acceptance {
		if !a.Settled() {
			return Step{Kind: StepCheck, For: owner, Next: "hug task check " + t.ID + " " + itoa(n+1)}
		}
	}
	// Every check holds but a question is open (72n8m4): done would
	// refuse, so the next thing is other work while the answer comes.
	if HasOpenQuestion(t) {
		return Step{Kind: StepAnswer, Next: "hug task next"}
	}
	// A reviewer approved (erv680 check 5): the stage is ship it, the
	// project's way, by the coordinator (else the owner). Once it has
	// shipped the task closes; a project whose ship closes nothing closes
	// it by hand.
	if a := t.Approval; a != nil && (t.Stage == "" || t.Stage == "active") {
		who := c.Coordinator
		if who == "" {
			who = owner
		}
		next := "ship it, " + ShipHow(c.Ship) + "; once it has shipped: hug task done " + t.ID
		return Step{Kind: StepShip, For: who, Next: next,
			Cue: a.By + " approved " + t.ID + " (" + title + "): " + next}
	}
	// 5yh7hr: when review starts by itself on the last tick, this is the
	// step that hands the task to its reviewer instead.
	next := "hug task done " + t.ID
	return Step{Kind: StepDone, For: owner, Next: next,
		Cue: "every check of " + t.ID + " holds (" + title + "). Close it: " + next}
}

// Approval is a reviewer's answer that approves the task (erv680): the
// newest review answer, when it says so, with nothing since that sends the
// task back (a new review, back to todo).
type Approval struct {
	By   string `json:"by"`
	At   string `json:"at"`
	Text string `json:"text"`
}

// approvalWords are the first clauses of a review answer that approve it
// ("looks right", the answer hug task review's help names). Anything else
// ("looks right but rename X") is what to change.
var approvalWords = map[string]bool{"looks right": true, "looks good": true, "approved": true, "approve": true, "lgtm": true, "ok": true, "okay": true, "yes": true, "ship it": true}

var firstClause = regexp.MustCompile(`^[^,.;:!\n]*`)

// Approves reports whether a review answer approves: its first clause,
// case and spaces aside, is one of approvalWords.
func Approves(text string) bool {
	c := strings.Join(strings.Fields(strings.ToLower(firstClause.FindString(strings.TrimSpace(text)))), " ")
	return approvalWords[c]
}

// moveLine is a History move: "- STAMP FROM -> TO ...".
var moveLine = regexp.MustCompile(`^- \S+ [a-z]+(?:/[a-z]+)? -> ([a-z]+)(?:/([a-z]+))?\b`)

// readApproval is the approval History holds now, or nil.
func readApproval(history []string) *Approval {
	var cur *Approval
	for i := 0; i < len(history); i++ {
		line := strings.TrimRight(history[i], "\r\n")
		if m := answeredHead.FindStringSubmatch(line); m != nil {
			// The question's lines ("> ") come first, then the answer's, as
			// scanAnswers reads them.
			var text []string
			for i+1 < len(history) && strings.HasPrefix(history[i+1], "  ") {
				i++
				l := strings.TrimRight(history[i], "\r\n")[2:]
				if len(text) == 0 && (l == ">" || strings.HasPrefix(l, "> ")) {
					continue
				}
				text = append(text, l)
			}
			if m[2] == "review" {
				cur = nil
				if t := strings.Join(text, "\n"); Approves(t) {
					cur = &Approval{By: m[4], At: m[1], Text: t}
				}
			}
			continue
		}
		if m := answeredLine.FindStringSubmatch(line); m != nil {
			if m[2] == "review" {
				cur = nil
				if a, e := strconv.Unquote(m[4]); e == nil && Approves(a) {
					cur = &Approval{By: m[5], At: m[1], Text: a}
				}
			}
			continue
		}
		// A new review, or the task given back, ends the approval.
		if m := moveLine.FindStringSubmatch(line); m != nil && (m[2] == "review" || m[1] == "todo") {
			cur = nil
		}
	}
	return cur
}
