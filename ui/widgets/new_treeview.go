package widgets

import (
	"image"

	"gioui.org/op"

	"gioui.org/io/input"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type TreeViewV2 struct {
	nodes []*TreeNodeV2
	list  widget.List
}

type TreeNodeV2 struct {
	Text           string
	Identifier     string
	Children       []*TreeNodeV2
	DiscloserState component.DiscloserState

	menuClickable   widget.Clickable
	menuContextArea component.ContextArea
	menu            component.MenuState
	menuOptions     []string

	menuCache op.CallOp
	menuDim   layout.Dimensions
	menuInit  bool
}

func NewTreeViewV2(nodes []*TreeNodeV2) *TreeViewV2 {
	return &TreeViewV2{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		nodes: nodes,
	}
}

func (t *TreeViewV2) clickableWrap(gtx layout.Context, theme *material.Theme, node *TreeNodeV2, widget layout.Widget) layout.Dimensions {
	return node.DiscloserState.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				background := theme.Palette.Bg
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				if gtx.Source == (input.Source{}) {
					background = Disabled(theme.Palette.Bg)
				} else if node.DiscloserState.Clickable.Hovered() || gtx.Focused(node.DiscloserState.Clickable) {
					background = Hovered(theme.Palette.Bg)
				}

				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			widget,
		)
	})
}

func (t *TreeViewV2) controlLayout(gtx layout.Context, theme *material.Theme, node *TreeNodeV2) layout.Dimensions {
	return t.clickableWrap(gtx, theme, node, func(gtx layout.Context) layout.Dimensions {
		var icon = ExpandIcon
		if !node.DiscloserState.Visible() {
			icon = ForwardIcon
		}
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(4)}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return icon.Layout(gtx, theme.ContrastFg)
			},
		)
	})
}

func (t *TreeViewV2) itemLayout(gtx layout.Context, theme *material.Theme, node *TreeNodeV2) layout.Dimensions {
	leftPadding := unit.Dp(4)
	if len(node.Children) == 0 {
		leftPadding = unit.Dp(28)
	}
	return t.clickableWrap(gtx, theme, node, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: leftPadding}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return material.Label(theme, theme.TextSize, node.Text).Layout(gtx)
			})
	})
}

// LayoutTreeNode recursively lays out a tree of widgets described by
// TreeNodes.
func (t *TreeViewV2) LayoutTreeNode(gtx layout.Context, theme *material.Theme, node *TreeNodeV2) layout.Dimensions {
	if len(node.Children) == 0 {
		return t.itemLayout(gtx, theme, node)
	}
	children := make([]layout.FlexChild, 0, len(node.Children))
	for i := range node.Children {
		child := node.Children[i]
		children = append(children, layout.Rigid(
			func(gtx layout.Context) layout.Dimensions {
				return t.LayoutTreeNode(gtx, theme, child)
			}))
	}
	return component.Discloser(theme, &node.DiscloserState).Layout(gtx,
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Dp(16.4)
			return t.controlLayout(gtx, theme, node)
		},
		func(gtx layout.Context) layout.Dimensions {
			gtx.Constraints.Min.X = gtx.Constraints.Max.X
			return t.itemLayout(gtx, theme, node)
		},
		func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Vertical}.Layout(gtx, children...)
		},
	)
}

func (t *TreeViewV2) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	return material.List(theme, &t.list).Layout(gtx, len(t.nodes), func(gtx layout.Context, index int) layout.Dimensions {
		// return layout.UniformInset(unit.Dp(4)).Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return t.LayoutTreeNode(gtx, theme, t.nodes[index])
		//})
	})
}
