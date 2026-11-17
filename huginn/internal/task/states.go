package task

import (
	"fmt"
	"sort"
	"strings"
)

// State and stage (6ephqk, Jørgen 2026-11-17 10:03-10:56): every task is in
// one state (does anyone have it?) and, within it, one stage (todo: why is
// nobody working on it? doing: what is happening with it now?). This table
// is the only place the tree is defined: hug task states prints it, lint
// holds every file to it, and the commands move only along its transitions.
// It is hard-coded because hug acts on every node; a configurable tree
// would need this one underneath anyway. Design: tmp/lanes/805p6y-states.md.

// Setter says who puts a task into a node or writes a field.
type Setter string

const (
	// ByHug: hug derives it from the task and the board and rewrites it on
	// every write; never set by hand.
	ByHug Setter = "hug"
	// ByAgent: the agent (or person) working on the board sets it with a
	// command.
	ByAgent Setter = "agent"
	// ByPerson: a person decides it.
	ByPerson Setter = "person"
	// ByReviewer: whoever takes a task for review.
	ByReviewer Setter = "reviewer"
)

// NodeField is a field that exists only while a task is in its node. Its
// name starts with the node's name: doing_since, waiting_on.
type NodeField struct {
	Name     string `json:"name"`
	Required bool   `json:"required,omitempty"`
	SetBy    Setter `json:"set_by"`
	Means    string `json:"means"`
}

// Node is a state or a stage in the tree.
type Node struct {
	Name   string      `json:"name"`
	Means  string      `json:"means"`
	SetBy  Setter      `json:"set_by,omitempty"`
	Fields []NodeField `json:"fields,omitempty"`
	// Stages are a state's children, in precedence order for todo: when
	// several hold, the first wins (Jørgen 10:18-10:23).
	Stages []Node `json:"stages,omitempty"`
}

// States is the tree: the four states and their stages.
var States = []Node{
	{Name: "todo", Means: "nobody is working on it", Stages: []Node{
		{Name: "deferred", SetBy: ByPerson, Means: "someone decided: not now (it may be complete or not)", Fields: []NodeField{
			{Name: "deferred_reason", Required: true, SetBy: ByPerson, Means: "why not now"},
			{Name: "deferred_until", SetBy: ByPerson, Means: "when to look again"},
		}},
		{Name: "depends", SetBy: ByHug, Means: "another task must be done first"},
		{Name: "design", SetBy: ByAgent, Means: "a question must be answered first", Fields: []NodeField{
			{Name: "design_question", Required: true, SetBy: ByAgent, Means: "the question"},
			{Name: "design_for", Required: true, SetBy: ByAgent, Means: "who must answer it (a person's name; crab 12:38: no implied person)"},
			{Name: "design_since", SetBy: ByHug, Means: "when the question was asked"},
			{Name: "design_asked_by", SetBy: ByHug, Means: "who asked it (the agent or person that ran hug task design)"},
		}},
		{Name: "draft", SetBy: ByHug, Means: "not described well enough to start (Goal or Acceptance missing)"},
		{Name: "review", SetBy: ByAgent, Means: "the work is done and waits for someone to check it", Fields: []NodeField{
			{Name: "review_by", SetBy: ByAgent, Means: "who should review it"},
			{Name: "review_since", SetBy: ByHug, Means: "since when it waits for review"},
		}},
		{Name: "ready", SetBy: ByHug, Means: "nothing stands in the way; anyone may take it"},
	}},
	{Name: "doing", Means: "someone has it", Fields: []NodeField{
		{Name: "doing_since", SetBy: ByHug, Means: "when it was taken"},
	}, Stages: []Node{
		{Name: "active", SetBy: ByAgent, Means: "the owner is working on it"},
		{Name: "waiting", SetBy: ByAgent, Means: "the owner waits on something outside the board (CI, a reply, a release)", Fields: []NodeField{
			{Name: "waiting_on", Required: true, SetBy: ByAgent, Means: "what it waits on"},
			{Name: "waiting_since", SetBy: ByHug, Means: "since when"},
			{Name: "waiting_asked_by", SetBy: ByHug, Means: "who waits (the agent or person that ran hug task wait), told when it is answered"},
		}},
		{Name: "depends", SetBy: ByHug, Means: "the owner can't go on until another task is done"},
		{Name: "review", SetBy: ByReviewer, Means: "someone is reviewing it", Fields: []NodeField{
			{Name: "review_by", Required: true, SetBy: ByReviewer, Means: "who is reviewing it"},
			{Name: "review_since", SetBy: ByHug, Means: "since when"},
		}},
	}},
	{Name: "done", Means: "finished: every check ticked", Fields: []NodeField{
		{Name: "done_at", SetBy: ByHug, Means: "when it was done"},
		{Name: "done_release", SetBy: ByHug, Means: "the release it shipped in, when tagged"},
	}},
	{Name: "dropped", Means: "decided: never", Fields: []NodeField{
		{Name: "dropped_reason", Required: true, SetBy: ByAgent, Means: "why never"},
		{Name: "dropped_at", SetBy: ByHug, Means: "when it was dropped"},
	}},
}

// Transition is a move a command makes. From and To are "state/stage"; a
// From of "todo/*" means any todo stage, "any" any node.
type Transition struct {
	From    string `json:"from"`
	To      string `json:"to"`
	Command string `json:"command"`
}

// Transitions are the only moves allowed.
var Transitions = []Transition{
	{"todo/review", "doing/review", "hug task take ID"},
	{"todo/*", "doing/active", "hug task take ID"},
	{"doing/active", "doing/waiting", `hug task wait ID "on what"`},
	{"doing/waiting", "doing/active", "hug task resume ID"},
	{"todo/*", "todo/design", `hug task design ID "question"`},
	{"todo/design", "todo/*", "hug task design ID --clear"},
	{"doing/*", "todo/*", "hug task set ID --status todo (gives it back, keeping its stage)"},
	{"doing/*", "done", "hug task done ID"},
	{"any", "dropped", `hug task drop ID "reason"`},
	{"done", "todo/*", "hug task set ID --status todo (reopens it)"},
	{"dropped", "todo/*", "hug task set ID --status todo (reopens it)"},
}

// StateNode is the state named name, if it is one.
func StateNode(name string) (Node, bool) {
	for _, n := range States {
		if n.Name == name {
			return n, true
		}
	}
	return Node{}, false
}

// StageNode is the stage named stage under state, if it is one.
func StageNode(state, stage string) (Node, bool) {
	s, ok := StateNode(state)
	if !ok {
		return Node{}, false
	}
	for _, n := range s.Stages {
		if n.Name == stage {
			return n, true
		}
	}
	return Node{}, false
}

// StateNames are the states in tree order.
func StateNames() []string {
	var out []string
	for _, n := range States {
		out = append(out, n.Name)
	}
	return out
}

// StageNames are state's stages in precedence order.
func StageNames(state string) []string {
	s, _ := StateNode(state)
	var out []string
	for _, n := range s.Stages {
		out = append(out, n.Name)
	}
	return out
}

// NodeFields are every node field in the tree, by name, with the nodes
// ("state" or "state/stage") that own it. review_by belongs to both
// todo/review and doing/review.
func NodeFields() map[string][]string {
	out := map[string][]string{}
	for _, s := range States {
		for _, f := range s.Fields {
			out[f.Name] = append(out[f.Name], s.Name)
		}
		for _, g := range s.Stages {
			for _, f := range g.Fields {
				out[f.Name] = append(out[f.Name], s.Name+"/"+g.Name)
			}
		}
	}
	return out
}

// StatesText is the tree as hug task states prints it: each node, what it
// means and who sets it, its fields indented under it, then the
// transitions.
func StatesText() string {
	var b strings.Builder
	field := func(indent string, f NodeField) {
		req := ""
		if f.Required {
			req = ", required"
		}
		fmt.Fprintf(&b, "%s%-16s %s (by %s%s)\n", indent, f.Name, f.Means, f.SetBy, req)
	}
	for _, s := range States {
		fmt.Fprintf(&b, "%-20s %s\n", s.Name, s.Means)
		for _, f := range s.Fields {
			field("      ", f)
		}
		for _, g := range s.Stages {
			fmt.Fprintf(&b, "  %-18s %s (set by %s)\n", g.Name, g.Means, g.SetBy)
			for _, f := range g.Fields {
				field("      ", f)
			}
		}
	}
	b.WriteString("\ntransitions:\n")
	for _, t := range Transitions {
		fmt.Fprintf(&b, "  %-14s -> %-14s %s\n", t.From, t.To, t.Command)
	}
	if s, _ := StateNode("todo"); len(s.Stages) > 0 {
		b.WriteString("\nwhen several todo stages hold, the first wins: " + strings.Join(StageNames("todo"), ", ") + "\n")
		b.WriteString("when several doing stages hold, the first wins: " + strings.Join(DoingOrder, ", ") + "\n")
		b.WriteString("a stage holds while its fields are set (deferred_reason, design_question, review_by or review_since,\nwaiting_on), or, for hug's stages, while its condition holds; leaving a node moves its fields to ## History\n")
	}
	return b.String()
}

// Stage findings (6ephqk check 2). Four rules hold a file to the tree:
//
//  1. the stage is a stage of the task's state;
//  2. a node's fields exist only while the node's condition holds (crab
//     11:37, Proposal A, restating "only while the task is in that node"
//     so Jørgen's precedence can hold): the fields of another state are
//     never allowed, a stage's fields are allowed while that stage still
//     holds, even when an earlier one wins (design_question on a todo
//     task that now depends);
//  3. the required fields of the task's state and stage are present and
//     not empty;
//  4. the stage is the first in order whose condition holds (Stage).
//
// Rules 1-3 need only the file and run in Parse; rule 4 needs the other
// tasks (a dependency's state) and runs in Lint. A file with no stage line
// and no node field is not held to rules 3 and 4: hug upgrade writes them
// (step 4 of 6ephqk), and until then no file has them.

// DoingOrder is the precedence of doing stages: what the owner waits on
// outside the board first, then another task, then a reviewer.
var DoingOrder = []string{"waiting", "depends", "review", "active"}

// Staged is whether the file has stages yet: a stage line or a node field.
func Staged(t Task) bool { return t.Stage != "" || len(t.Fields) > 0 }

// stageFindings checks rules 1-3 on the parsed frontmatter.
func (d *Document) stageFindings() {
	t := d.Task
	state, ok := StateNode(t.Status)
	if !ok {
		return // unknown status is its own finding
	}
	node := t.Status
	var stage Node
	if t.Stage != "" {
		if stage, ok = StageNode(t.Status, t.Stage); !ok {
			valid := StageNames(t.Status)
			p := fmt.Sprintf("unknown stage %s for %s: a %s task's stage is one of %s", t.Stage, t.Status, t.Status, strings.Join(valid, ", "))
			if len(valid) == 0 {
				p = fmt.Sprintf("unknown stage %s for %s: a %s task has no stage; remove the stage line", t.Stage, t.Status, t.Status)
			}
			d.Findings = append(d.Findings, Finding{Path: t.Path, Line: d.fieldLine("stage"), Problem: p, Fix: "hug task set " + t.ID + " --status " + t.Status})
		} else {
			node += "/" + t.Stage
		}
	}
	owners := NodeFields()
	for _, k := range sortedKeys(t.Fields) {
		if !fieldHolds(t, k, owners[k]) {
			d.Findings = append(d.Findings, Finding{Path: t.Path, Line: d.fieldLine(k),
				Problem: fmt.Sprintf("node field %s belongs to %s, and this task is %s: hug moves it to ## History", k, strings.Join(owners[k], " and "), node),
				Fix:     "hug task set " + t.ID + " --status " + t.Status})
		}
	}
	if !Staged(t) {
		return
	}
	for _, f := range append(append([]NodeField(nil), state.Fields...), stage.Fields...) {
		if f.Required && strings.TrimSpace(t.Fields[f.Name]) == "" {
			d.Findings = append(d.Findings, Finding{Path: t.Path, Line: d.fieldLine("stage"),
				Problem: fmt.Sprintf("missing node field %s: a %s task needs it (%s)", f.Name, node, f.Means),
				Fix:     requiredFix(t, f.Name)})
		}
	}
}

// requiredFix is the command that writes a missing required field, or
// leaves the node that needs it, without an editor (crab 11:32).
func requiredFix(t Task, field string) string {
	switch field {
	case "waiting_on":
		return "hug task wait " + t.ID + ` "on what"`
	case "design_question", "design_for":
		return "hug task design " + t.ID + ` "question" --for NAME`
	case "dropped_reason":
		return "hug task drop " + t.ID + ` "reason"`
	case "review_by":
		return "hug task take " + t.ID + " --model MODEL"
	}
	return "hug task set " + t.ID + " --status todo"
}

// fieldHolds says whether node field k may stand on t: one of its owners
// is t's state, or a stage of t's state whose condition holds.
func fieldHolds(t Task, k string, owners []string) bool {
	for _, o := range owners {
		state, stage, staged := strings.Cut(o, "/")
		if state != t.Status {
			continue
		}
		if !staged || stage == t.Stage || ownHolds(t, stage) {
			return true
		}
	}
	return false
}

// ownHolds says whether a stage someone sets still holds on t, by its
// fields: deferred while deferred_reason is set, design while
// design_question is, todo/review while review_by or review_since is,
// waiting while waiting_on is, doing/review while review_by is.
func ownHolds(t Task, stage string) bool {
	set := func(k string) bool { return strings.TrimSpace(t.Fields[k]) != "" }
	switch stage {
	case "deferred":
		return set("deferred_reason")
	case "design":
		return set("design_question")
	case "review":
		return set("review_by") || (t.Status == "todo" && set("review_since"))
	case "waiting":
		return set("waiting_on")
	}
	return false
}

// Incomplete says whether t is not described well enough to start: the
// draft stage's condition (no Goal, or no acceptance item).
func Incomplete(t Task) bool {
	return strings.TrimSpace(t.Goal) == "" || len(t.Acceptance) == 0
}

// Stage is the stage t is in: the first, in precedence order, whose
// condition holds (Jørgen 10:18, Proposal A). todo: deferred, depends,
// design, draft, review, else ready. doing: waiting, depends, review, else
// active. done and dropped have none. tasks is the board, for the
// dependencies' states.
func Stage(t Task, tasks []Task) string {
	open := len(Blocked(t, tasks)) > 0
	switch t.Status {
	case "todo":
		for _, g := range StageNames("todo") {
			switch {
			case g == "depends" && open, g == "draft" && Incomplete(t), g != "depends" && g != "draft" && ownHolds(t, g):
				return g
			}
		}
		return "ready"
	case "doing":
		for _, g := range DoingOrder {
			switch {
			case g == "depends" && open, g != "depends" && ownHolds(t, g):
				return g
			}
		}
		return "active"
	}
	return ""
}

// DerivedStage is Stage; kept for the lint tests' name.
func DerivedStage(t Task, tasks []Task) string { return Stage(t, tasks) }

// derivedFindings checks rule 4: the stage is the first whose condition
// holds.
func derivedFindings(tasks []Task) []Finding {
	var out []Finding
	for _, t := range tasks {
		if !Staged(t) {
			continue
		}
		g, ok := StageNode(t.Status, t.Stage)
		if t.Stage != "" && !ok {
			continue // rule 1 reports it
		}
		missing := false
		for _, f := range g.Fields {
			missing = missing || (f.Required && strings.TrimSpace(t.Fields[f.Name]) == "")
		}
		if missing {
			continue // rule 3 reports it: the fix writes the field
		}
		want := Stage(t, tasks)
		if want == t.Stage {
			continue
		}
		why := ""
		switch want {
		case "depends":
			b := Blocked(t, tasks)[0]
			why = "it depends on " + b.ID + " (" + b.Status + ")"
		case "draft":
			why = "it is not complete (no Goal or no acceptance item)"
		case "ready", "active":
			why = "nothing holds it: no open dependency, and none of its stage's fields is set"
		case "":
			why = "a " + t.Status + " task has no stage"
		default:
			why = "its " + want + " fields are set"
		}
		p := fmt.Sprintf("stage %s disagrees with the task: %s, so its stage is %s", t.Stage, why, want)
		if want == "" {
			p = fmt.Sprintf("stage %s disagrees with the task: %s", t.Stage, why)
		}
		out = append(out, Finding{Path: t.Path, Line: 1, Problem: p, Fix: "hug task set " + t.ID + " --status " + t.Status})
	}
	return out
}

func sortedKeys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}
