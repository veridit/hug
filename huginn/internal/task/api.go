package task

// TypeInfo is the short, agent-readable description used by `hug task types`.
type TypeInfo struct {
	Value       string `json:"value"`
	Description string `json:"description"`
}

var TypeDescriptions = []TypeInfo{
	{"feature", "Add a new user-visible capability."},
	{"bug", "Correct behavior that does not match expectations."},
	{"chore", "Maintain tooling, dependencies, or project infrastructure."},
	{"spike", "Investigate a question and record evidence."},
	{"docs", "Write or improve documentation."},
}

// NextHint returns the concrete follow-up command for this task's state.
func NextHint(t Task) string { return hint(t) }

// NextAt is NextHint on the board at root, for who: the ship step names
// the project's own way (hug.toml [board] ship, erv680), and a step due
// for another role is not who's (Step.NextFor).
func NextAt(t Task, root, who string) string { return StepAt(t, root).NextFor(who) }

// Get resolves human input once and reads the authoritative document.
func (s *Store) Get(ref string) (*Document, error) { d, _, e := s.load(ref); return d, e }

// Next syncs through Refresh before applying the rank-ordered filters.
func (s *Store) Next(f Filter) (Result, error) {
	ts, e := s.Tasks()
	if e != nil {
		return Result{}, e
	}
	r, e := Next(ts, f)
	if e == nil && r.Task.ID != "" && r.Next == hint(r.Task) {
		r.Next = StepAt(r.Task, s.Root).NextFor(f.Me)
	}
	return r, e
}
