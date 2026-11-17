package task

import (
	"fmt"
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
			{Name: "design_since", SetBy: ByHug, Means: "when the question was asked"},
		}},
		{Name: "draft", SetBy: ByHug, Means: "not described well enough to start (Goal or Acceptance missing)"},
		{Name: "review", SetBy: ByAgent, Means: "the work is done and waits for someone to check it", Fields: []NodeField{
			{Name: "review_by", SetBy: ByAgent, Means: "who should review it"},
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
		}},
		{Name: "depends", SetBy: ByHug, Means: "the owner can't go on until another task is done"},
		{Name: "review", SetBy: ByReviewer, Means: "someone is reviewing it", Fields: []NodeField{
			{Name: "review_by", Required: true, SetBy: ByReviewer, Means: "who is reviewing it"},
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
	{"doing/*", "todo/*", "hug task set ID --status todo (gives it back, keeping its stage)"},
	{"doing/*", "done", "hug task done ID"},
	{"any", "dropped", `hug task drop ID "reason"`},
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
	}
	return b.String()
}
