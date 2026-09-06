package ui

import (
	"strings"

	"github.com/mirzakhany/yoga/layout"
)

type tagData struct {
	tags   []string
	onTags func([]string)
}

type tagState struct {
	draft string
}

// TagEdit is a chip input. tags are controlled by the app; draft lives in the store.
func TagEdit(id string, tags []string) *Node {
	return &Node{kind: kindTagEdit, id: id, extra: &tagData{tags: append([]string(nil), tags...)}}
}

// OnTags is called with the new tag list after add/remove.
func (n *Node) OnTags(fn func([]string)) *Node {
	if d, ok := n.extra.(*tagData); ok {
		d.onTags = fn
	}
	return n
}

func (n *Node) layoutTagEdit(c *Ctx) *layout.Element {
	id := n.id
	if id == "" {
		id = autoID(c, "tags")
	}
	d, _ := n.extra.(*tagData)
	if d == nil {
		d = &tagData{}
	}
	st := c.Widget(id, func() any { return &tagState{} }).(*tagState)
	th := c.Theme()
	onTags := d.onTags
	tags := d.tags

	emit := func(next []string) {
		if onTags != nil {
			onTags(next)
		}
	}
	add := func(s string) {
		s = strings.TrimSpace(s)
		if s == "" {
			return
		}
		for _, existing := range tags {
			if strings.EqualFold(existing, s) {
				st.draft = ""
				return
			}
		}
		next := append(append([]string(nil), tags...), s)
		st.draft = ""
		emit(next)
	}
	chips := make([]View, 0, len(tags)+1)
	for i, tag := range tags {
		i, tag := i, tag
		chips = append(chips, Button(id+"-chip-"+tag, Text(tag+" ×")).Subtle().Disabled(n.disabled).OnClick(func() {
			next := append([]string(nil), tags...)
			next = append(next[:i], next[i+1:]...)
			emit(next)
		}))
	}
	chips = append(chips, TextField(id+"-in", st.draft).
		Placeholder("Add tag...").
		OnChange(func(s string) { st.draft = s }).
		OnSubmit(func(s string) { add(s) }).
		Disabled(n.disabled).
		Width(80))
	return Row(chips...).
		Gap(th.Spacing.XS).
		Padding(th.Spacing.S).
		Wrap().
		Align(layout.AlignCenter).
		Style(n.spec).
		Layout(c)
}
