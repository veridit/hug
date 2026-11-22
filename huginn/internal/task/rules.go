package task

// ParseRules are the tables Parse reads and the prefix-to-code table whose
// codes a cached Finding carries: what a cache written by one hug holds as
// fresh for another. The cache's schema fingerprint hashes them with the
// parse of a fixed corpus, so a change here without a SchemaVersion bump
// fails a test (6q1d1t: 8a0ybb renamed Relevant details to Details under
// schema 30, and v0.7.103 served v0.7.101's section-order finding as its
// own). Codes (meaning, severity) is not here: severity is looked up again
// on every read, never cached.
func ParseRules() any {
	codes := make([][2]string, len(findingCodes))
	for i, c := range findingCodes {
		codes[i] = [2]string{c.prefix, c.code}
	}
	return struct {
		SectionOrder, TailSections, DescriptionSections []string
		OldDetails                                      string
		StateList, Types, BacklogFields                 []string
		States                                          []Node
		FindingCodes                                    [][2]string
	}{SectionOrder, TailSections, DescriptionSections, OldDetails, StateList, Types, BacklogFields, States, codes}
}
