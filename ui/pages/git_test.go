package pages

import (
	"testing"

	"github.com/mirzakhany/yoga/highlight"

	"github.com/chapar-rest/chapar/internal/gitx"
)

func TestItemFor(t *testing.T) {
	req := []byte("apiVersion: v1\nkind: Request\nmetadata:\n  id: r1\n  name: Get objects\n  type: http\n")
	cases := []struct {
		path string
		data []byte
		want gitItem
	}{
		{"collections/Sample/Get objects.yaml", req,
			gitItem{Kind: "Request", ID: "r1", Name: "Get objects", Type: "Request", Where: "Sample"}},
		{"requests/old name.yaml", req, gitItem{Kind: "Request", ID: "r1", Name: "Get objects", Type: "Request"}},
		{"collections/Sample/_collection.yaml", nil, gitItem{Name: "Sample", Type: "Collection"}},
		{"envs/Stage.yaml", []byte("kind: Environment\nmetadata:\n  id: e1\n  name: Stage\n"),
			gitItem{Kind: "Environment", ID: "e1", Name: "Stage", Type: "Environment"}},
		{"testcases/Smoke.yaml", []byte("not: [valid"), gitItem{Name: "Smoke", Type: "Test case"}},
		{"_workspace.yaml", nil, gitItem{Name: "_workspace", Type: "Workspace"}},
		{".gitignore", nil, gitItem{Name: ".gitignore", Type: "File"}},
		{"envs/notes.txt", nil, gitItem{Name: "envs/notes.txt", Type: "File"}},
	}
	for _, c := range cases {
		if got := itemFor(c.path, c.data); got != c.want {
			t.Errorf("itemFor(%q) = %+v, want %+v", c.path, got, c.want)
		}
	}
}

func TestDiffTokens(t *testing.T) {
	diff := "diff --git a/x b/x\nindex 1..2 100644\n--- a/x\n+++ b/x\n@@ -1 +1 @@\n-a: 1\n+a: 2\n ctx\n\\ No newline at end of file\n"
	toks := diffTokens([]byte(diff))
	classOf := func(sub string) highlight.ColorClass {
		for _, tk := range toks {
			if diff[tk.Start:tk.End] == sub {
				return tk.Class
			}
		}
		return 255
	}
	want := map[string]highlight.ColorClass{
		"diff --git a/x b/x":           highlight.ClassMuted,
		"--- a/x":                      highlight.ClassMuted,
		"+++ b/x":                      highlight.ClassMuted,
		"@@ -1 +1 @@":                  highlight.ClassType,
		"-a: 1":                        highlight.ClassError,
		"+a: 2":                        highlight.ClassSuccess,
		"\\ No newline at end of file": highlight.ClassMuted,
	}
	for line, class := range want {
		if got := classOf(line); got != class {
			t.Errorf("%q: class %v, want %v", line, got, class)
		}
	}
	if got := classOf(" ctx"); got != 255 {
		t.Errorf("context line got class %v", got)
	}
}

func TestSortEntries(t *testing.T) {
	es := []gitEntry{
		{item: gitItem{Type: "File", Name: ".gitignore"}},
		{item: gitItem{Type: "Environment", Name: "Stage"}},
		{item: gitItem{Type: "Request", Name: "b"}},
		{item: gitItem{Type: "Request", Name: "a", Where: "Col"}},
		{item: gitItem{Type: "Collection", Name: "Col"}},
	}
	sortEntries(es)
	var got []string
	for _, e := range es {
		got = append(got, e.item.label())
	}
	want := []string{"Col", "Col / a", "b", "Stage", ".gitignore"}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("order = %v, want %v", got, want)
		}
	}
}

func TestChangeLabel(t *testing.T) {
	if l := changeLabel(gitx.Change{Kind: gitx.Untracked}); l != "New" {
		t.Errorf("untracked = %q", l)
	}
	if l := changeLabel(gitx.Change{Kind: gitx.Renamed}); l != "Renamed" {
		t.Errorf("renamed = %q", l)
	}
}
