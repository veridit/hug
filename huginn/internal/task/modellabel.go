package task

// ModelLabel is how t's model is shown: "claude-opus-5-5/high" with the
// effort it was taken at (jr55s8), the id alone without one.
func (t Task) ModelLabel() string {
	if t.Model != "" && t.Effort != "" {
		return t.Model + "/" + t.Effort
	}
	return t.Model
}
