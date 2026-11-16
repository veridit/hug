package project

import "testing"

func TestSecretHintKeyValueNames(t *testing.T) {
	for _, s := range []string{
		"aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"DB_PASSWORD: hunter2hunter2hunter2",
		"client_secret = 'abcdefghijklmnop1234'",
		"OPENAI_API_KEY=abcdefghijklmnop", // already caught before
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
