package task

import (
	"bytes"
	"reflect"
	"strings"
	"testing"
)

// fileRefsPlain is FileRefs without the literal prefilter: every regexp on
// every document, as before 3mavpt.
func fileRefsPlain(doc []byte, self string) []Ref {
	var out []Ref
	for _, m := range fileRef.FindAllSubmatchIndex(doc, -1) {
		name := string(doc[m[2]:m[3]])
		if strings.HasPrefix(name, "/") || (!strings.Contains(name, ".") && !strings.Contains(name, "/")) || lostAfter(doc, m[1]) {
			continue
		}
		out = append(out, Ref{Task: self, Name: name, Raw: "file:" + name})
	}
	for _, m := range taskRef.FindAllSubmatchIndex(doc, -1) {
		ref, name := string(doc[m[2]:m[3]]), string(doc[m[4]:m[5]])
		if (ref == strings.ToUpper(ref) && strings.ToLower(ref) != ref && !strings.ContainsAny(ref, "0123456789")) || lostAfter(doc, m[1]) {
			continue
		}
		out = append(out, Ref{Task: ref, Name: name, Raw: "task:" + ref + "/" + name})
	}
	for _, m := range pathRef.FindAllSubmatchIndex(doc, -1) {
		if lostAfter(doc, m[1]) {
			continue
		}
		start := bytes.Index(doc[m[0]:m[1]], []byte(".hug/tasks/")) + m[0]
		out = append(out, Ref{Task: string(doc[m[2]:m[3]]), Name: string(doc[m[4]:m[5]]), Raw: string(doc[start:m[1]])})
	}
	return out
}

// 3mavpt: the literal prefilter in FileRefs changes no answer: each regexp
// needs its literal, so a document without it has no match.
func TestFileRefsPrefilterSameAnswer(t *testing.T) {
	docs := []string{
		"",
		"plain text, no refs",
		"see file:log.txt and (file:dir/a.csv) and [file:x.md]",
		"file:line is prose; file:///etc/x is a URL; file:gone.txt (lost)",
		"task:abc123/run.log and task:TASK-12/out.txt and task:REF/NAME",
		"`.hug/tasks/abc123/x.log` and \".hug/tasks/g/abc123/y.txt\" and .hug/tasks/abc123/z.txt (not kept: too large)",
		"mixed file:a.txt task:k2m4n6/b.txt .hug/tasks/k2m4n6/c.txt",
		"FILE:upper.txt Task:x/y.txt .HUG/TASKS/abc123/d.txt",
		"file:a.txt\r\ntask:abc123/b.txt\r\n",
	}
	for _, d := range docs {
		got, want := FileRefs([]byte(d), "self01"), fileRefsPlain([]byte(d), "self01")
		if !reflect.DeepEqual(got, want) {
			t.Errorf("%q:\n got  %v\n want %v", d, got, want)
		}
	}
}

func BenchmarkFileRefsNoRefs(b *testing.B) {
	doc := []byte(strings.Repeat("A long task body with no citations at all, only prose.\n", 400))
	for i := 0; i < b.N; i++ {
		FileRefs(doc, "self01")
	}
}
