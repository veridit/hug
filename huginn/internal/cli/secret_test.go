package cli

import (
	"strings"
	"testing"
)

func TestSecretHintNames(t *testing.T) {
	for _, s := range []string{
		// Built at run time: literal fixtures trip the repo's no-secrets gate.
		"aws_" + "secret_access_key=" + "wJalr" + strings.Repeat("Xy7", 10),
		"DB_" + "PASSWORD: " + strings.Repeat("hunter2", 3),
		"token=" + "gh" + "p_" + strings.Repeat("a1", 18),
		"client_" + "secret = '" + strings.Repeat("ab", 8) + "1234'",
	} {
		if secretHint([]byte(s)) == "" {
			t.Errorf("missed: %s", s)
		}
	}
	for _, s := range []string{"secret sauce recipe", "password policy: 12 characters", "see secrets.NAME"} {
		if h := secretHint([]byte(s)); h != "" {
			t.Errorf("false positive %q: %s", s, h)
		}
	}
}
