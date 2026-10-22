// Package yamlmerge merges three versions of a workspace YAML file field by
// field, the way git merges lines. A field changed on one side only takes
// that change; a field both sides changed differently is a conflict the
// user settles by picking a side. Lists of items with an id, such as an
// environment's values or a request's headers, merge item by item.
package yamlmerge

import (
	"fmt"
	"reflect"
	"strings"

	"gopkg.in/yaml.v2"
)

// Side is a version a conflict can be settled with.
type Side int

const (
	// Unset means the user has not picked a side yet.
	Unset Side = iota
	Ours
	Theirs
)

// Conflict is a field both sides changed differently.
type Conflict struct {
	// Path names the field for people, such as
	// "spec.values[baseURL].value".
	Path string
	// Base, Ours and Theirs are the field's values; a missing field (one
	// side removed it) is Missing.
	Base, Ours, Theirs any
}

// missing marks a field or item a version doesn't have.
type missingT struct{}

// Missing is the value of a field a version doesn't have.
var Missing any = missingT{}

// slot is where a conflict's chosen value goes.
type slot struct{ index int }

// Result is a merge whose conflicts wait for a side to be picked.
type Result struct {
	Conflicts []Conflict
	tree      any
}

// Merge merges ours and theirs, the two edited versions of base. base is
// nil when both sides added the file. An error means a version is not
// YAML a field merge can work with.
func Merge(base, ours, theirs []byte) (*Result, error) {
	b, err := parse(base, "the common version")
	if err != nil {
		return nil, err
	}
	o, err := parse(ours, "this version")
	if err != nil {
		return nil, err
	}
	t, err := parse(theirs, "the incoming version")
	if err != nil {
		return nil, err
	}
	r := &Result{}
	r.tree = r.merge("", b, o, t)
	return r, nil
}

func parse(data []byte, which string) (any, error) {
	if data == nil {
		return Missing, nil
	}
	var v yaml.MapSlice
	if err := yaml.Unmarshal(data, &v); err != nil {
		return nil, fmt.Errorf("%s is not valid YAML: %w", which, err)
	}
	return v, nil
}

func equal(a, b any) bool { return reflect.DeepEqual(a, b) }

func (r *Result) merge(path string, b, o, t any) any {
	switch {
	case equal(o, t):
		return o
	case equal(b, o):
		return t
	case equal(b, t):
		return o
	}
	if om, ok := o.(yaml.MapSlice); ok {
		if tm, ok := t.(yaml.MapSlice); ok {
			bm, _ := b.(yaml.MapSlice)
			return r.mergeMaps(path, bm, om, tm)
		}
	}
	if ol, ok := idList(o); ok {
		if tl, ok := idList(t); ok {
			bl, _ := idList(b)
			return r.mergeLists(path, bl, ol, tl)
		}
	}
	r.Conflicts = append(r.Conflicts, Conflict{Path: path, Base: b, Ours: o, Theirs: t})
	return slot{len(r.Conflicts) - 1}
}

func get(m yaml.MapSlice, key any) any {
	for _, it := range m {
		if equal(it.Key, key) {
			return it.Value
		}
	}
	return Missing
}

// mergeMaps merges per key. Keys keep our order; keys only theirs has
// follow, in their order.
func (r *Result) mergeMaps(path string, b, o, t yaml.MapSlice) yaml.MapSlice {
	var keys []any
	seen := map[string]bool{}
	for _, m := range []yaml.MapSlice{o, t} {
		for _, it := range m {
			k := fmt.Sprint(it.Key)
			if !seen[k] {
				seen[k] = true
				keys = append(keys, it.Key)
			}
		}
	}
	out := make(yaml.MapSlice, 0, len(keys))
	for _, k := range keys {
		v := r.merge(join(path, fmt.Sprint(k)), get(b, k), get(o, k), get(t, k))
		out = append(out, yaml.MapItem{Key: k, Value: v})
	}
	return out
}

func join(path, key string) string {
	if path == "" {
		return key
	}
	return path + "." + key
}

// idList returns v's items when v is a list whose items are all maps with
// an id: a list nobody reorders by position, so items merge by id. An
// empty list counts, as does a missing one.
func idList(v any) ([]yaml.MapSlice, bool) {
	if v == Missing {
		return nil, true
	}
	l, ok := v.([]any)
	if !ok {
		return nil, false
	}
	out := make([]yaml.MapSlice, 0, len(l))
	for _, it := range l {
		m, ok := it.(yaml.MapSlice)
		if !ok {
			return nil, false
		}
		if id, ok := get(m, "id").(string); !ok || id == "" {
			return nil, false
		}
		out = append(out, m)
	}
	return out, true
}

func byID(l []yaml.MapSlice, id string) any {
	for _, m := range l {
		if get(m, "id") == id {
			return m
		}
	}
	return Missing
}

// itemLabel names a list item in a path: its key or name when it has one,
// which people recognise, else its id.
func itemLabel(items ...any) string {
	for _, field := range []string{"key", "name"} {
		for _, it := range items {
			if m, ok := it.(yaml.MapSlice); ok {
				if s, ok := get(m, field).(string); ok && s != "" {
					return s
				}
			}
		}
	}
	for _, it := range items {
		if m, ok := it.(yaml.MapSlice); ok {
			return fmt.Sprint(get(m, "id"))
		}
	}
	return "?"
}

// mergeLists merges items by id. Items keep our order; items only theirs
// has follow, in their order.
func (r *Result) mergeLists(path string, b, o, t []yaml.MapSlice) []any {
	var ids []string
	seen := map[string]bool{}
	for _, l := range [][]yaml.MapSlice{o, t} {
		for _, m := range l {
			id := get(m, "id").(string)
			if !seen[id] {
				seen[id] = true
				ids = append(ids, id)
			}
		}
	}
	out := make([]any, 0, len(ids))
	for _, id := range ids {
		bv, ov, tv := byID(b, id), byID(o, id), byID(t, id)
		p := path + "[" + itemLabel(ov, tv, bv) + "]"
		out = append(out, r.merge(p, bv, ov, tv))
	}
	return out
}

// Apply builds the merged file with each conflict settled by sides[i]. It
// fails while a conflict has no side.
func (r *Result) Apply(sides []Side) ([]byte, error) {
	if len(sides) != len(r.Conflicts) {
		return nil, fmt.Errorf("%d conflicts, %d choices", len(r.Conflicts), len(sides))
	}
	for i, s := range sides {
		if s == Unset {
			return nil, fmt.Errorf("pick a version for %s", r.Conflicts[i].Path)
		}
	}
	v := r.fill(r.tree, sides)
	if v == Missing {
		return nil, nil
	}
	return yaml.Marshal(v)
}

// Deleted reports whether the merge, with sides, removes the whole file.
func (r *Result) Deleted(sides []Side) bool {
	return r.fill(r.tree, sides) == Missing
}

func (r *Result) fill(v any, sides []Side) any {
	switch x := v.(type) {
	case slot:
		c := r.Conflicts[x.index]
		if sides[x.index] == Theirs {
			return r.fill(c.Theirs, sides)
		}
		return r.fill(c.Ours, sides)
	case yaml.MapSlice:
		out := make(yaml.MapSlice, 0, len(x))
		for _, it := range x {
			if fv := r.fill(it.Value, sides); fv != Missing {
				out = append(out, yaml.MapItem{Key: it.Key, Value: fv})
			}
		}
		return out
	case []any:
		out := make([]any, 0, len(x))
		for _, it := range x {
			if fv := r.fill(it, sides); fv != Missing {
				out = append(out, fv)
			}
		}
		return out
	}
	return v
}

// Show writes a conflict value for people: a scalar as itself, anything
// bigger as YAML, a missing field as "".
func Show(v any) string {
	switch x := v.(type) {
	case missingT:
		return ""
	case string:
		return x
	case nil:
		return "null"
	case yaml.MapSlice, []any:
		out, err := yaml.Marshal(x)
		if err != nil {
			return fmt.Sprint(x)
		}
		return strings.TrimRight(string(out), "\n")
	}
	return fmt.Sprint(v)
}

// IsMissing reports whether v stands for a field a version doesn't have.
func IsMissing(v any) bool { return v == Missing }
