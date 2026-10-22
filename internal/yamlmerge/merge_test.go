package yamlmerge

import (
	"strings"
	"testing"
)

const baseEnv = `apiVersion: v1
kind: Environment
metadata:
  id: e1
  name: Stage
spec:
  values:
  - id: v1
    key: baseURL
    value: https://stage.example.com
    enable: true
  - id: v2
    key: token
    value: abc
    enable: true
`

func edit(s string, pairs ...string) string {
	return strings.NewReplacer(pairs...).Replace(s)
}

func TestMergeDisjointEdits(t *testing.T) {
	ours := edit(baseEnv, "value: https://stage.example.com", "value: https://stage2.example.com")
	theirs := edit(baseEnv, "value: abc", "value: xyz", "name: Stage", "name: Staging")
	r, err := Merge([]byte(baseEnv), []byte(ours), []byte(theirs))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	out, err := r.Apply(nil)
	if err != nil {
		t.Fatal(err)
	}
	want := edit(baseEnv, "value: https://stage.example.com", "value: https://stage2.example.com",
		"value: abc", "value: xyz", "name: Stage", "name: Staging")
	if string(out) != want {
		t.Errorf("merged =\n%s\nwant\n%s", out, want)
	}
}

func TestMergeConflictByItemID(t *testing.T) {
	ours := edit(baseEnv, "value: abc", "value: mine")
	theirs := edit(baseEnv, "value: abc", "value: yours")
	r, err := Merge([]byte(baseEnv), []byte(ours), []byte(theirs))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 1 {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	c := r.Conflicts[0]
	if c.Path != "spec.values[token].value" || Show(c.Ours) != "mine" || Show(c.Theirs) != "yours" || Show(c.Base) != "abc" {
		t.Errorf("conflict = %+v", c)
	}
	if _, err := r.Apply([]Side{Unset}); err == nil {
		t.Error("Apply with an unpicked conflict succeeded")
	}
	out, err := r.Apply([]Side{Theirs})
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != theirs {
		t.Errorf("merged =\n%s", out)
	}
	out, _ = r.Apply([]Side{Ours})
	if string(out) != ours {
		t.Errorf("merged with ours =\n%s", out)
	}
}

// Items added on both sides and removed on one merge by id, keeping our
// order and appending theirs.
func TestMergeListItems(t *testing.T) {
	ours := baseEnv + "  - id: v3\n    key: mine\n    value: \"1\"\n    enable: true\n"
	theirs := edit(baseEnv, "  - id: v2\n    key: token\n    value: abc\n    enable: true\n", "") +
		"  - id: v4\n    key: theirs\n    value: \"2\"\n    enable: false\n"
	r, err := Merge([]byte(baseEnv), []byte(ours), []byte(theirs))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 0 {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	out, _ := r.Apply(nil)
	s := string(out)
	if strings.Contains(s, "key: token") {
		t.Error("item theirs removed is still there")
	}
	if i, j := strings.Index(s, "key: mine"), strings.Index(s, "key: theirs"); i < 0 || j < 0 || i > j {
		t.Errorf("added items missing or out of order:\n%s", s)
	}
}

// One side edits a field the other removes: a conflict whose sides are a
// value and nothing.
func TestMergeEditVersusRemove(t *testing.T) {
	ours := edit(baseEnv, "value: abc", "value: mine")
	theirs := edit(baseEnv, "  - id: v2\n    key: token\n    value: abc\n    enable: true\n", "")
	r, err := Merge([]byte(baseEnv), []byte(ours), []byte(theirs))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 1 || !IsMissing(r.Conflicts[0].Theirs) || r.Conflicts[0].Path != "spec.values[token]" {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	out, _ := r.Apply([]Side{Theirs})
	if strings.Contains(string(out), "token") {
		t.Errorf("removed item kept:\n%s", out)
	}
}

func TestMergeWholeFile(t *testing.T) {
	// Both sides added the file with different content.
	r, err := Merge(nil, []byte("a: 1\nb: 1\n"), []byte("a: 1\nb: 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 1 || r.Conflicts[0].Path != "b" {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	// One side deleted it, the other edited it.
	r, err = Merge([]byte("a: 1\n"), nil, []byte("a: 2\n"))
	if err != nil {
		t.Fatal(err)
	}
	if len(r.Conflicts) != 1 || r.Conflicts[0].Path != "" {
		t.Fatalf("conflicts = %+v", r.Conflicts)
	}
	if !r.Deleted([]Side{Ours}) || r.Deleted([]Side{Theirs}) {
		t.Error("Deleted doesn't follow the side")
	}
	if out, err := r.Apply([]Side{Ours}); err != nil || out != nil {
		t.Errorf("Apply deleted = %q, %v", out, err)
	}
}

func TestMergeBadYAML(t *testing.T) {
	if _, err := Merge([]byte("a: 1\n"), []byte("a: [\n"), []byte("a: 2\n")); err == nil {
		t.Error("merged invalid YAML")
	}
}
