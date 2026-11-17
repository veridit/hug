package task

import (
	"bytes"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
)

// sfa8w4 check 5: the stored form renders as a crossed-out checklist line
// in a standard Markdown renderer (GitHub-flavoured: task list plus
// strikethrough, as GitHub, Gitea and most editors show it).
func TestStruckFormRendersCrossedOut(t *testing.T) {
	src := "## Acceptance\n- [x] done one\n- [ ] ~~no longer applies~~ (not relevant: the API changed, kestrel, 2026-11-17)\n- [ ] still open\n"
	var out bytes.Buffer
	if err := goldmark.New(goldmark.WithExtensions(extension.GFM)).Convert([]byte(src), &out); err != nil {
		t.Fatal(err)
	}
	html := out.String()
	want := `<li><input disabled="" type="checkbox"> <del>no longer applies</del> (not relevant: the API changed, kestrel, 2026-11-17)</li>`
	if !strings.Contains(html, want) {
		t.Fatalf("rendered:\n%s\nwant %s", html, want)
	}
	t.Log(html)
}
