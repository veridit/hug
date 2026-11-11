package tidy

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	"github.com/veridit/muninn/internal/extract"
	"github.com/veridit/muninn/internal/store"
)

func TestPlaceSecretSafetyMustBeSubstantive(t *testing.T) {
	for _, reason := range []string{"", "   ", "\t", "explanation\nwith newline", "TOKEN=" + strings.Repeat("a", 20)} {
		a := answerFor("sid", "project:alpha")
		a["safe_reason"] = reason
		raw, _ := json.Marshal([]map[string]any{a})
		p := Plan{Scopes: []string{"global", "project:alpha"}, Entries: []Entry{{SID: "sid", Source: "jcode@global", Decision: "unsure"}}}
		applyPlaceAnswers(string(raw), &p, []int{0}, []placeInput{{SID: "sid", SecretHint: "capability URL"}})
		if p.Entries[0].Decision != "unsure" || p.Entries[0].Target != "" {
			t.Fatalf("unsafe safety reason %q accepted: %+v", reason, p.Entries)
		}
	}
}

func TestPlaceEffectiveClientAndDisabledProviderStayReadOnly(t *testing.T) {
	g := Graph{Units: []store.Unit{{ID: 1, SID: "client", Scope: "jcode@legacy:folder", Body: "A client implementation rule", Active: true}, {ID: 2, SID: "plain", Scope: "global", Body: "A personal preference", Active: true}}, EffectiveScopes: map[int64]string{1: "client:acme:project"}}
	p, err := Place(context.Background(), g, g.Units, placeCatalogFixture(), extract.Config{Provider: "off"})
	if err != nil || p.Placement.Calls != 0 || p.Entries[0].Decision != "stay" || len(p.Entries[0].Flags) != 1 || p.Entries[1].Decision != "unsure" {
		t.Fatalf("plan=%+v err=%v", p, err)
	}
}
