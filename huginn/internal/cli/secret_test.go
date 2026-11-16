package cli

import "testing"

func TestSecretHintNames(t *testing.T) {
	for _, s := range []string{
		"aws_secret_access_key=wJalrXUtnFEMI/K7MDENG/bPxRfiCYEXAMPLEKEY",
		"DB_PASSWORD: hunter2hunter2hunter2",
		"token=ghp_abcdefghijklmnopqrstuvwxyz0123456789",
		"client_secret = 'abcdefghijklmnop1234'",
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
