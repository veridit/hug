package task

import (
	"regexp"
	"strings"
)

// SecretHint mirrors muninn's write-time check: credential-shaped tokens.
// "" when b looks clean. hug task attach and V in hug tui (q7n2zd) refuse
// what it names. A key starts at a word boundary, as in muninn and
// check-secrets.sh: 'task-900-...' is a file name, not 'sk-900...'
// (4c7327).
func SecretHint(b []byte) string {
	if secretPattern.Find(b) == nil && namedSecret(string(b)) {
		return "credential-shaped text (a secret-named key with a key-like value)"
	}
	if m := secretPattern.FindSubmatch(b); m != nil {
		s := string(m[0])
		if len(m[1]) > 0 {
			s = string(m[1]) // the key, without the boundary character
		}
		if len(s) > 12 {
			s = s[:6] + "..."
		}
		return "credential-shaped text near " + s
	}
	return ""
}

var secretPattern = regexp.MustCompile(`(?:(?:^|[^A-Za-z0-9])(sk-(?:proj-|ant-|or-)?[A-Za-z0-9_-]{20,})|gh[pousr]_[A-Za-z0-9]{30,}|github_pat_[A-Za-z0-9_]{40,}|AKIA[0-9A-Z]{16}|xox[abprs]-[A-Za-z0-9-]{10,}|AIza[0-9A-Za-z_-]{35}|-----BEGIN [A-Z ]*PRIVATE KEY-----)`)

// namedSecretRe mirrors muninn's check: NAME=VALUE where the name says
// secret and the value looks like a key (12+ characters, letters and
// digits). Plain names (NOPASSWD, *_API_KEY variable names) do not count.
var namedSecretRe = regexp.MustCompile(`(?i)[a-z0-9_]*(?:secret|password|api[_ -]?key|access[_ -]?token|auth[_ -]?token|private[_ -]?key)[a-z0-9_]*\s*[:=]\s*["']?([A-Za-z0-9+/_=.-]{12,})`)

func namedSecret(text string) bool {
	for _, m := range namedSecretRe.FindAllStringSubmatch(text, -1) {
		v := m[1]
		if strings.ContainsAny(v, "0123456789") && strings.IndexFunc(v, func(r rune) bool { return r >= 'A' && r <= 'Z' || r >= 'a' && r <= 'z' }) >= 0 {
			return true
		}
	}
	return false
}
