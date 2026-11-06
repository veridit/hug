package main

import (
	"strings"
	"testing"

	"github.com/veridit/muninn/internal/harness"
)

func TestFilterSecretsPreservesIdentityAndWithholdsWholeMessage(t *testing.T) {
	tx := harness.CaptureTranscript{Cwd: "/synthetic/project", Messages: []harness.CaptureMessage{
		{ID: "one", Role: "user", Text: "Use atomic writes for preferences."},
		{ID: "two", Role: "assistant", Text: "Never export this synthetic credential: sk-" + strings.Repeat("x", 24)},
		{ID: "three", Role: "tool", Text: ""},
	}}
	filterSecrets(&tx)
	if tx.Cwd != "/synthetic/project" || len(tx.Messages) != 3 {
		t.Fatal("transcript shape changed")
	}
	if tx.Messages[0].Text != "Use atomic writes for preferences." {
		t.Fatal("ordinary evidence changed")
	}
	if tx.Messages[1].ID != "two" || tx.Messages[1].Role != "assistant" || tx.Messages[1].Text != "" {
		t.Fatal("secret was not withheld with identity retained")
	}
	if tx.Messages[2].Text != "" {
		t.Fatal("empty message changed")
	}
}
