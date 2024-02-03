package widgets

import (
	"image"
	"time"

	"gioui.org/op/clip"

	"gioui.org/op/paint"

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
	clickable *widget.Clickable

	Children []*TreeViewNode
	Text     string

	lastClickAt time.Time
	order       int

	onDoubleClick func(tr *TreeViewNode)
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

func (t *TreeView) AddRootNode(text string, collapsed bool) {
	t.AddNode(NewNode(text, collapsed), nil)
}

func NewNode(text string, collapsed bool) *TreeViewNode {
	return &TreeViewNode{
		Text:      text,
		collapsed: collapsed,
		clickable: &widget.Clickable{},
		order:     1,
	}
}

func (tr *TreeViewNode) OnDoubleClick(f func(tr *TreeViewNode)) {
	tr.onDoubleClick = f
}

func (tr *TreeViewNode) AddChild(node *TreeViewNode) {
	tr.Children = append(tr.Children, node)
}

func (t *TreeView) AddNode(node *TreeViewNode, parent *TreeViewNode) {
	if parent == nil {
		t.nodes = append(t.nodes, node)
		return
	}

	parent.Children = append(parent.Children, node)
}

func (t *TreeView) childLayout(theme *material.Theme, gtx layout.Context, node *TreeViewNode) layout.Dimensions {
	background := theme.Palette.Bg
	for node.clickable.Clicked(gtx) {
		node.collapsed = !node.collapsed
	}

	return node.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				switch {
				case gtx.Queue == nil:
					background = Disabled(theme.Palette.Bg)
				case node.clickable.Hovered() || node.clickable.Focused():
					background = Hovered(theme.Palette.Bg)
				}
				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(48)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, node.Text).Layout(gtx)
				})
			},
		)
	})
}

func (t *TreeView) parentLayout(gtx layout.Context, theme *material.Theme, node *TreeViewNode) layout.Dimensions {
	background := theme.Palette.Bg
	for node.clickable.Clicked(gtx) {
		// is this a double click?
		if time.Since(node.lastClickAt) < 500*time.Millisecond {
			if node.onDoubleClick != nil {
				node.onDoubleClick(node)
			}
		} else {
			node.lastClickAt = time.Now()
			if node.Children == nil {
				continue
			}

			node.collapsed = !node.collapsed
		}
	}

	pr := node.clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				switch {
				case gtx.Queue == nil:
					background = Disabled(theme.Palette.Bg)
				case node.clickable.Hovered() || node.clickable.Focused():
					background = Hovered(theme.Palette.Bg)
				}
				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			func(gtx layout.Context) layout.Dimensions {
				return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle, Spacing: layout.SpaceEnd}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if node.Children == nil {
								s := gtx.Constraints.Min
								s.X = gtx.Dp(unit.Dp(16))
								return layout.Dimensions{Size: s}
							}

							gtx.Constraints.Min.X = gtx.Dp(16)
							if !node.collapsed {
								return ExpandIcon.Layout(gtx, theme.ContrastFg)
							}
							return ForwardIcon.Layout(gtx, theme.ContrastFg)
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							return layout.Inset{Left: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								return material.Label(theme, theme.TextSize, node.Text).Layout(gtx)
							})
						}),
					)
				})
			},
		)
	})

	if node.collapsed {
		return pr
	}

	var children []layout.FlexChild
	children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
		return pr
	}))
	for _, child := range node.Children {
		child := child
		children = append(children, layout.Rigid(func(gtx layout.Context) layout.Dimensions {
			return t.childLayout(theme, gtx, child)
		}))
	}

	return layout.Flex{Axis: layout.Vertical, Alignment: layout.Middle, Spacing: layout.SpaceEnd}.Layout(gtx, children...)

}

func (t *TreeView) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return material.List(theme, t.list).Layout(gtx, len(t.nodes), func(gtx layout.Context, index int) layout.Dimensions {
		return t.parentLayout(gtx, theme, t.nodes[index])
	})
}
