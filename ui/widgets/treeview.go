package widgets

import (
	"image"
	"image/color"
	"sort"
	"strings"

	"gioui.org/font"
	"gioui.org/io/event"
	"gioui.org/io/input"
	"gioui.org/io/pointer"
	"gioui.org/layout"
	"gioui.org/op"
	"gioui.org/op/clip"
	"gioui.org/op/paint"
	"gioui.org/unit"
	"gioui.org/widget"
	"gioui.org/widget/material"
	"gioui.org/x/component"
	"github.com/oligo/gioview/misc"

	"github.com/chapar-rest/chapar/internal/safemap"
	"github.com/chapar-rest/chapar/ui/chapartheme"
)

type TreeView struct {
	nodes []*TreeNode
	list  widget.List

	onMenuItemClick func(tr *TreeNode, item string)

	filterText    string
	filteredNodes []*TreeNode

	onNodeDoubleClick func(tr *TreeNode)
	onNodeClick       func(tr *TreeNode)
}

type TreeNode struct {
	Text           string
	Prefix         string
	PrefixColor    color.NRGBA
	Identifier     string
	Children       []*TreeNode
	DiscloserState component.DiscloserState
	MenuOptions    []string

	menuContextArea component.ContextArea
	menu            component.MenuState
	menuClickables  []*widget.Clickable

	draggable widget.Draggable
	entered   bool

	menuInit bool
	isChild  bool
	expanded bool

	Meta *safemap.Map[string]
}

func NewTreeView(nodes []*TreeNode) *TreeView {
	// sort nodes alphabetically
	sort.Slice(nodes, func(i, j int) bool {
		sort.Slice(nodes[i].Children, func(k, l int) bool {
			return nodes[i].Children[k].Text < nodes[i].Children[l].Text
		})
		return nodes[i].Text < nodes[j].Text
	})

	return &TreeView{
		list: widget.List{
			List: layout.List{
				Axis: layout.Vertical,
			},
		},
		nodes: nodes,
	}
}

func (t *TreeView) OnNodeDoubleClick(fn func(tr *TreeNode)) {
	t.onNodeDoubleClick = fn
}

func (t *TreeView) OnNodeClick(fn func(tr *TreeNode)) {
	t.onNodeClick = fn
}

func (t *TreeView) SetNodes(nodes []*TreeNode) {
	t.nodes = nodes
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

func (tr *TreeNode) AddChildNode(child *TreeNode) {
	child.isChild = true
	tr.Children = append(tr.Children, child)
}

func (tr *TreeNode) SetPrefix(prefix string, color color.NRGBA) {
	tr.Prefix = prefix
	tr.PrefixColor = color
}

func (t *TreeView) ExpandNode(identifier string) {
	for _, n := range t.nodes {
		if n.Identifier == identifier {
			n.expanded = true
			return
		}
	}
}

func (t *TreeView) AddChildNode(parentIdentifier string, child *TreeNode) {
	for _, n := range t.nodes {
		if n.Identifier == parentIdentifier {
			child.isChild = true
			n.Children = append(n.Children, child)
			return
		}
	}
}

func (t *TreeView) RemoveNode(identifier string) {
	for i, n := range t.nodes {
		if n.Identifier == identifier {
			t.nodes = append(t.nodes[:i], t.nodes[i+1:]...)
			return
		}

		for j, c := range n.Children {
			if c.Identifier == identifier {
				n.Children = append(n.Children[:j], n.Children[j+1:]...)
				return
			}
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

func (t *TreeView) clickableWrap(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode, widget layout.Widget) layout.Dimensions {
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

func (t *TreeView) controlLayout(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode) layout.Dimensions {
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

func (t *TreeView) itemLayout(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode) layout.Dimensions {
	leftPadding := 4
	if len(node.Children) == 0 {
		leftPadding = 24
	}

	for {
		click, ok := node.DiscloserState.Clickable.Update(gtx)
		if !ok {
			break
		}
		switch click.NumClicks {
		case 1:
			if t.onNodeClick != nil {
				go t.onNodeClick(node)
				gtx.Execute(op.InvalidateCmd{})
			}
		case 2:
			if t.onNodeDoubleClick != nil {
				go t.onNodeDoubleClick(node)
				gtx.Execute(op.InvalidateCmd{})
			}
		default:
			if node.Children == nil {
				continue
			}
		}
	}

	if node.expanded {
		node.expanded = false
		node.DiscloserState.Appear(gtx.Now)
	}

	return t.clickableWrap(gtx, theme, node, func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(8), Bottom: unit.Dp(8), Left: unit.Dp(8 + leftPadding)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
				layout.Flexed(1, func(gtx layout.Context) layout.Dimensions {
					return layout.Flex{Axis: layout.Horizontal, Alignment: layout.Middle}.Layout(gtx,
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							if node.Prefix == "" {
								return layout.Dimensions{}
							}
							return layout.Inset{Right: unit.Dp(4)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
								lb := material.Label(theme.Material(), unit.Sp(13), node.Prefix)
								lb.Font.Weight = font.SemiBold
								lb.Color = node.PrefixColor
								lb.TextSize = unit.Sp(11)
								lb.MaxLines = 1
								return lb.Layout(gtx)
							})
						}),
						layout.Rigid(func(gtx layout.Context) layout.Dimensions {
							lb := material.Label(theme.Material(), unit.Sp(13), node.Text)
							lb.Font.Weight = font.SemiBold
							lb.MaxLines = 1
							return lb.Layout(gtx)
						}),
					)
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
									m := component.Menu(theme.Material(), &node.menu)
									m.SurfaceStyle.Fill = theme.MenuBgColor
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
func (t *TreeView) layoutTreeNode(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode) layout.Dimensions {
	if !node.menuInit {
		node.menuInit = true
		node.menuContextArea = component.ContextArea{
			Activation:       pointer.ButtonPrimary,
			AbsolutePosition: true,
		}
		node.menu = component.MenuState{
			Options: func() []func(gtx layout.Context) layout.Dimensions {

				out := make([]func(gtx layout.Context) layout.Dimensions, 0, len(node.MenuOptions))
				node.menuClickables = make([]*widget.Clickable, 0, len(node.MenuOptions))
				for i, opt := range node.MenuOptions {
					opt := opt
					i := i
					node.menuClickables = append(node.menuClickables, new(widget.Clickable))

					if opt == "-" {
						out = append(out, component.Divider(theme.Material()).Layout)
						continue
					}

					out = append(out, component.MenuItem(theme.Material(), node.menuClickables[i], opt).Layout)
				}
				return out
			}(),
		}
	}

	for i := range node.menuClickables {
		if node.menuClickables[i].Clicked(gtx) {
			if t.onMenuItemClick != nil {
				t.onMenuItemClick(node, node.MenuOptions[i])
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

	return component.Discloser(theme.Material(), &node.DiscloserState).Layout(gtx,
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

func (tr *TreeNode) droppable() bool {
	return tr.entered && !tr.draggable.Dragging()
}

func (t *TreeView) LayoutTreeNode(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode) layout.Dimensions {
	for {
		ke, ok := gtx.Event(pointer.Filter{Target: node, Kinds: pointer.Enter | pointer.Leave})
		if !ok {
			break
		}

		switch event := ke.(type) {
		case pointer.Event:
			if event.Kind == pointer.Enter {
				node.entered = true
			} else if event.Kind == pointer.Leave {
				node.entered = false
			}
		}
	}

	macro := op.Record(gtx.Ops)
	dims := func() layout.Dimensions {
		if len(node.Children) != 0 {
			return t.layoutTreeNode(gtx, theme, node)
		}

		return node.draggable.Layout(gtx,
			func(gtx layout.Context) layout.Dimensions {
				return t.layoutTreeNode(gtx, theme, node)
			},
			func(gtx layout.Context) layout.Dimensions {
				return t.layoutDraggingBox(gtx, theme, node)
			},
		)
	}()
	call := macro.Stop()

	defer pointer.PassOp{}.Push(gtx.Ops).Pop()
	defer clip.Rect(image.Rectangle{Max: dims.Size}).Push(gtx.Ops).Pop()
	if node.droppable() {
		paint.ColorOp{Color: misc.WithAlpha(theme.ContrastFg, 0xb6)}.Add(gtx.Ops)
		paint.PaintOp{}.Add(gtx.Ops)
	}

	if m, ok := node.draggable.Update(gtx); ok {
		node.draggable.Offer(gtx, m, node)
	}

	event.Op(gtx.Ops, node)
	call.Add(gtx.Ops)

	return dims
}

// Implelments io.ReadCloser for widget.Draggable.
func (tr *TreeNode) Read(p []byte) (n int, err error) {
	return 0, nil
}

func (tr *TreeNode) Close() error {
	return nil
}

func (t *TreeView) layoutDraggingBox(gtx layout.Context, theme *chapartheme.Theme, node *TreeNode) layout.Dimensions {
	if !node.draggable.Dragging() {
		return layout.Dimensions{}
	}

	offset := node.draggable.Pos()
	if offset.Round().X == 0 && offset.Round().Y == 0 {
		return layout.Dimensions{}
	}

	macro := op.Record(gtx.Ops)
	dims := func(gtx layout.Context) layout.Dimensions {
		return layout.Inset{Top: unit.Dp(6), Bottom: unit.Dp(6), Left: unit.Dp(8), Right: unit.Dp(8)}.Layout(gtx, func(gtx layout.Context) layout.Dimensions {
			lb := material.Label(theme.Material(), theme.TextSize, node.Text)
			lb.Color = theme.ContrastFg
			return lb.Layout(gtx)
		})
	}(gtx)
	call := macro.Stop()

	defer clip.UniformRRect(image.Rectangle{Max: dims.Size}, gtx.Dp(unit.Dp(0))).Push(gtx.Ops).Pop()
	paint.ColorOp{Color: theme.MenuBgColor}.Add(gtx.Ops)
	paint.PaintOp{}.Add(gtx.Ops)
	defer paint.PushOpacity(gtx.Ops, 1).Pop()
	call.Add(gtx.Ops)

	return dims
}

func (t *TreeView) Layout(gtx layout.Context, theme *chapartheme.Theme) layout.Dimensions {
	nodes := t.nodes
	if t.filterText != "" {
		nodes = t.filteredNodes
	}

	if len(nodes) == 0 {
		return layout.Center.Layout(gtx, material.Label(theme.Material(), unit.Sp(14), "No items").Layout)
	}

	return material.List(theme.Material(), &t.list).Layout(gtx, len(nodes), func(gtx layout.Context, index int) layout.Dimensions {
		return t.LayoutTreeNode(gtx, theme, nodes[index])
	})
}
