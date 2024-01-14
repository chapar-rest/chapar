package widgets

import (
	"sort"

	"gioui.org/layout"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
)

type TreeView struct {
	nodes []*TreeViewNode
	list  *widget.List
}

type TreeViewNode struct {
	collapsed bool
	collabser *widget.Clickable

	Parent *TreeViewNode
	Text   string
}

func NewTreeView() *TreeView {
	return &TreeView{
		list: &widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
	}
}

func (t *TreeView) AddNode(text string, collapsed bool, parent *TreeViewNode) {
	t.nodes = append(t.nodes, &TreeViewNode{
		Text:      text,
		collapsed: collapsed,
		collabser: &widget.Clickable{},
		Parent:    parent,
	})
}

func (t *TreeView) parentLayout(theme *material.Theme, gtx layout.Context, node *TreeViewNode) layout.Dimensions {
	return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceEnd}.Layout(gtx,
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			ib := &IconButton{
				Icon:      ForwardIcon,
				Color:     theme.ContrastFg,
				Size:      unit.Dp(16),
				Clickable: node.collabser,
			}
			return ib.Layout(theme, gtx)
		}),
		layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, node.Text).Layout(gtx)
			})
		}),
	)
}

func (t *TreeView) Layout(theme *material.Theme, gtx layout.Context) layout.Dimensions {
	// prepare parent and child nodes
	nodes := make(map[*TreeViewNode][]*TreeViewNode)
	for _, node := range t.nodes {
		// if node has no parent, it's a root node
		if node.Parent == nil {
			nodes[node] = []*TreeViewNode{}
			continue
		}

		// if node has parent, add it to the parent's children
		if _, ok := nodes[node.Parent]; !ok {
			nodes[node.Parent] = []*TreeViewNode{}
		} else {
			nodes[node.Parent] = append(nodes[node.Parent], node)
		}
	}

	var parents []*TreeViewNode
	for parent := range nodes {
		parents = append(parents, parent)
	}

	// sort parents by text
	sort.Slice(parents, func(i, j int) bool {
		return parents[i].Text < parents[j].Text
	})

	return material.List(theme, t.list).Layout(gtx, len(parents), func(gtx layout.Context, index int) layout.Dimensions {
		return layout.Inset{Bottom: unit.Dp(16)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return t.parentLayout(theme, gtx, parents[index])
		})
	})
}
