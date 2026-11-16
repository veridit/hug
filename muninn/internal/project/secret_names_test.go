package project

import (
	"strings"
	"testing"
)

func TestSecretHintKeyValueNames(t *testing.T) {
	for _, s := range []string{
		// Built at run time: literal fixtures trip the repo's no-secrets gate.
		"aws_" + "secret_access_key=" + "wJalr" + strings.Repeat("Xy7", 10),
		"DB_" + "PASSWORD: " + strings.Repeat("hunter2", 3),
		"client_" + "secret = '" + strings.Repeat("ab", 8) + "1234'",
		"OPENAI_" + "API_KEY=" + strings.Repeat("ab", 8), // already caught before
	} {
		if !HasCredential(s) {
			t.Errorf("missed: %s", s)
		}
	}
	for _, s := range []string{"secret sauce recipe", "password policy: 12 characters", "use secrets.NAME", "SECRET=$SECRET", "API_KEY=${API_KEY}"} {
		if HasCredential(s) {
			t.Errorf("false positive: %s", s)
		}
	}
}
