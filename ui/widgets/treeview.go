package widgets

import (
	"image"
	"strings"
	"time"

	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
)

type TreeView struct {
	nodes []*TreeNode
	list  widget.List

	ChildMenuOptions  []string
	ParentMenuOptions []string

	onMenuItemClick func(tr *TreeNode, item string)

	filterText    string
	filteredNodes []*TreeNode

	onDoubleClick func(tr *TreeNode)
}

type TreeNode struct {
	Text           string
	Identifier     string
	Children       []*TreeNode
	DiscloserState component.DiscloserState

	menuContextArea component.ContextArea
	menu            component.MenuState
	menuOptions     []string
	menuClickables  []*widget.Clickable

	menuInit bool
	isChild  bool

	lastClickAt time.Time
}

func NewTreeView(nodes []*TreeNode) *TreeView {
	return &TreeView{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		nodes: nodes,
	}
}

func (t *TreeView) OnDoubleClick(fn func(tr *TreeNode)) {
	t.onDoubleClick = fn
}

func (tr *TreeNode) SetIdentifier(identifier string) {
	tr.Identifier = identifier
}

func (t *TreeView) SetOnMenuItemClick(fn func(tr *TreeNode, item string)) {
	t.onMenuItemClick = fn
}

func (t *TreeView) AddNode(node *TreeNode) {
	t.nodes = append(t.nodes, node)
}

func (t *TreeView) RemoveNode(identifier string) {
	for i, n := range t.nodes {
		if n.Identifier == identifier {
			t.nodes = append(t.nodes[:i], t.nodes[i+1:]...)
			return
		}
	}
}

func (t *TreeView) Filter(text string) {
	t.filterText = text

	if text == "" {
		t.filteredNodes = make([]*TreeNode, 0)
		return
	}

	var items = make([]*TreeNode, 0)
	for _, item := range t.nodes {
		if strings.Contains(item.Text, text) {
			items = append(items, item)
		}

		for _, child := range item.Children {
			if strings.Contains(child.Text, text) {
				items = append(items, child)
			}
		}
	}

	t.filteredNodes = items
}

func (t *TreeView) clickableWrap(gtx layout.Context, theme *material.Theme, node *TreeNode, widget layout.Widget) layout.Dimensions {
	return node.DiscloserState.Clickable.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
		return layout.Background{}.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				background := theme.Palette.Bg
				defer clip.UniformRRect(image.Rectangle{Max: gtx.Constraints.Min}, 0).Push(gtx.Ops).Pop()
				if gtx.Source == (input.Source{}) {
					background = Disabled(theme.Palette.Bg)
				} else if node.DiscloserState.Clickable.Hovered() || gtx.Focused(node.DiscloserState.Clickable) || node.menuContextArea.Active() {
					background = Hovered(theme.Palette.Bg)
				}

				paint.Fill(gtx.Ops, background)
				return layout.Dimensions{Size: gtx.Constraints.Min}
			},
			widget,
		)
	})
}

func (t *TreeView) controlLayout(gtx layout.Context, theme *material.Theme, node *TreeNode) layout.Dimensions {
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

func (t *TreeView) itemLayout(gtx layout.Context, theme *material.Theme, node *TreeNode) layout.Dimensions {
	leftPadding := 4
	if len(node.Children) == 0 {
		leftPadding = 24
	}

	if node.isChild {
		leftPadding = 36
	}

	for node.DiscloserState.Clickable.Clicked(gtx) {
		// is this a double click?
		if time.Since(node.lastClickAt) < 500*time.Millisecond {
			if t.onDoubleClick != nil {
				t.onDoubleClick(node)
			}
		} else {
			node.lastClickAt = time.Now()
			if node.Children == nil {
				continue
			}
		}
	}

	return t.clickableWrap(gtx, theme, node, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8 + leftPadding)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return material.Label(theme, theme.TextSize, node.Text).Layout(gtx)
				}),
				layout.Rigid(func(gtx layout.Context) layout.Dimensions {
					iconBtn := layout.Inset{Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
						if !node.DiscloserState.Clickable.Hovered() && !node.menuContextArea.Active() {
							return layout.Dimensions{}
						}
						gtx.Constraints.Min.X = gtx.Dp(16)
						return MoreVertIcon.Layout(gtx, theme.ContrastFg)
					})
					return layout.Stack{}.Layout(gtx,
						layout.Stacked(func(gtx layout.Context) layout.Dimensions {
							return iconBtn
						}),
						layout.Expanded(func(gtx layout.Context) layout.Dimensions {
							return node.menuContextArea.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								offset := layout.Inset{
									Top:  unit.Dp(float32(iconBtn.Size.Y)/gtx.Metric.PxPerDp + 1),
									Left: unit.Dp(4),
								}
								return offset.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
									gtx.Constraints.Min = image.Point{}
									m := component.Menu(theme, &node.menu)
									m.SurfaceStyle.Fill = Gray300
									return m.Layout(gtx)
								})
							})
						}),
					)
				}),
			)
		})
	})
}

// LayoutTreeNode recursively lays out a tree of widgets described by
// TreeNodes.
func (t *TreeView) LayoutTreeNode(gtx layout.Context, theme *material.Theme, node *TreeNode) layout.Dimensions {
	if !node.menuInit {
		node.menuInit = true
		node.menuContextArea = component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		}
		node.menu = component.MenuState{
			Options: func() []func(gtx layout.Context) layout.Dimensions {
				options := t.ParentMenuOptions
				if node.isChild {
					options = t.ChildMenuOptions
				}

				out := make([]func(gtx layout.Context) layout.Dimensions, 0, len(options))
				node.menuClickables = make([]*widget.Clickable, 0, len(options))
				for i, opt := range options {
					opt := opt
					i := i
					node.menuClickables = append(node.menuClickables, new(widget.Clickable))

					if opt == "-" {
						out = append(out, component.Divider(theme).Layout)
						continue
					}

					out = append(out, component.MenuItem(theme, node.menuClickables[i], opt).Layout)
				}
				return out
			}(),
		}
	}

	for i := range node.menuClickables {
		if node.menuClickables[i].Clicked(gtx) {
			if t.onMenuItemClick != nil {
				t.onMenuItemClick(node, t.ParentMenuOptions[i])
			}
		}
	}

	if len(node.Children) == 0 {
		return t.itemLayout(gtx, theme, node)
	}

	children := make([]layout.FlexChild, 0, len(node.Children))
	for i := range node.Children {
		child := node.Children[i]
		child.isChild = true
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

func (t *TreeView) Layout(gtx layout.Context, theme *material.Theme) layout.Dimensions {
	nodes := t.nodes
	if t.filterText != "" {
		nodes = t.filteredNodes
	}

	if len(nodes) == 0 {
		return layout.Center.Layout(gtx, material.Label(theme, unit.Sp(14), "No items").Layout)
	}

	return material.List(theme, &t.list).Layout(gtx, len(nodes), func(gtx layout.Context, index int) layout.Dimensions {
		return t.LayoutTreeNode(gtx, theme, nodes[index])
	})
}
