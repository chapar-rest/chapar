package treeview

import (
	"errors"
	"image/color"
	"slices"

	"gioui.org/widget"
)

type NodeKind uint8

const (
	RequestNode NodeKind = iota
	CollectionNode
	FolderNode
)

type Info struct {
	Id          string
	Title       string
	Prefix      string
	PrefixColor color.NRGBA
	Icon        widget.Icon
	IconColor   color.NRGBA

	Meta map[string]interface{}
}

type EntryNode struct {
	Parent   *EntryNode
	children []*EntryNode

	Info Info
	kind NodeKind
}

func NewEntryNode(info Info, kind NodeKind) *EntryNode {
	return &EntryNode{
		Info: info,
		kind: kind,
	}
}

func (n *EntryNode) Kind() NodeKind {
	return n.kind
}

func (n *EntryNode) Children() []*EntryNode {
	if n.kind == RequestNode {
		return nil
	}

	if n.children == nil {
		// Try to reload the children if the reload function is set
		n.Refresh()
	}

	return n.children
}

func (n *EntryNode) UpdateTitle(title string) error {
	if title == "" {
		return nil
	}

	n.Info.Title = title
	return nil
}

func (n *EntryNode) AddChild(info Info, kind NodeKind) error {
	if !n.IsFolder() && !n.IsCollection() {
		return nil
	}

	if info.Id == "" {
		return errors.New("invalid entry id")
	}

	if info.Title == "" {
		return errors.New("invalid entry title")
	}

	if n.children == nil {
		n.children = make([]*EntryNode, 0)
	}

	entry := &EntryNode{
		Parent: n,
		kind:   kind,
		Info:   info,
	}

	// insert at the beginning of the children.
	n.children = slices.Insert(n.children, 0, entry)
	return nil
}

func (n *EntryNode) RemoveChild(id string) error {
	if !n.IsFolder() && !n.IsCollection() {
		return nil
	}

	if n.children == nil {
		return nil
	}

	for i, child := range n.children {
		if child.Info.Id == id {
			n.children = slices.Delete(n.children, i, i+1)
			return nil
		}
	}

	return errors.New("child not found")
}

func (n *EntryNode) Refresh() {

}

func (n *EntryNode) IsCollection() bool {
	return n.kind == CollectionNode
}

func (n *EntryNode) IsFolder() bool {
	return n.kind == FolderNode
}

func (n *EntryNode) Title() string {
	return n.Info.Title
}

func (n *EntryNode) Delete() error {
	return nil
}
